// Package mcp is a minimal Model Context Protocol client: enough to launch
// a local MCP server over stdio, complete the initialize handshake,
// discover its tools, and call them. It intentionally implements only the
// subset of MCP ORH needs — not the full protocol (resources, prompts,
// sampling, HTTP/SSE transport, ...).
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Client is a connection to one MCP server, speaking JSON-RPC 2.0 as
// newline-delimited JSON messages over stdin/stdout.
type Client struct {
	stdin  io.WriteCloser
	writeM sync.Mutex

	nextID int64

	mu      sync.Mutex
	pending map[int64]chan rpcResponse
	closed  chan struct{}
	readErr error

	closeFn func() error
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error response from the server.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return fmt.Sprintf("mcp error %d: %s", e.Code, e.Message) }

// Start launches command (run through "sh -c") as a child process and
// speaks MCP to it over its stdin/stdout. dir sets the child's working
// directory (relevant since a toolbox's runtime.server command commonly
// references files, like a server script, relative to the toolbox
// package's own directory) — pass "" to inherit the caller's cwd. The
// child's stderr is passed through to this process's stderr so server-side
// failures are visible.
func Start(ctx context.Context, command, dir string) (*Client, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("opening stdin for %q: %w", command, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("opening stdout for %q: %w", command, err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting MCP server %q: %w", command, err)
	}

	return New(stdin, stdout, func() error {
		stdin.Close()
		return cmd.Wait()
	}), nil
}

// New builds a Client around an already-connected transport. closeFn is
// called by Close after the write side is shut down. This lower-level
// constructor exists so tests can drive a fake in-process server instead of
// spawning a real subprocess.
func New(stdin io.WriteCloser, stdout io.Reader, closeFn func() error) *Client {
	c := &Client{
		stdin:   stdin,
		pending: map[int64]chan rpcResponse{},
		closed:  make(chan struct{}),
		closeFn: closeFn,
	}
	go c.readLoop(bufio.NewReader(stdout))
	return c
}

func (c *Client) readLoop(r *bufio.Reader) {
	defer close(c.closed)

	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			var resp rpcResponse
			if jsonErr := json.Unmarshal(line, &resp); jsonErr == nil && resp.ID != 0 {
				c.mu.Lock()
				ch, ok := c.pending[resp.ID]
				if ok {
					delete(c.pending, resp.ID)
				}
				c.mu.Unlock()
				if ok {
					ch <- resp
				}
			}
		}
		if err != nil {
			c.mu.Lock()
			c.readErr = err
			c.mu.Unlock()
			return
		}
	}
}

func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.nextID, 1)

	data, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("encoding %s request: %w", method, err)
	}
	data = append(data, '\n')

	ch := make(chan rpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	c.writeM.Lock()
	_, werr := c.stdin.Write(data)
	c.writeM.Unlock()
	if werr != nil {
		return nil, fmt.Errorf("writing %s request: %w", method, werr)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		c.mu.Lock()
		readErr := c.readErr
		c.mu.Unlock()
		return nil, fmt.Errorf("mcp server connection closed: %w", readErr)
	}
}

func (c *Client) notify(method string, params any) error {
	data, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("encoding %s notification: %w", method, err)
	}
	data = append(data, '\n')

	c.writeM.Lock()
	defer c.writeM.Unlock()
	_, err = c.stdin.Write(data)
	return err
}

// Close shuts down the connection (and, for a Start-launched server, waits
// for the child process to exit).
func (c *Client) Close() error {
	if c.closeFn != nil {
		return c.closeFn()
	}
	return nil
}
