package components

import (
	"context"
	"fmt"
	"strings"

	"github.com/pertevniyalai/orh/internal/commands"
	"github.com/pertevniyalai/orh/internal/memory"
	"github.com/pertevniyalai/orh/internal/providers"
	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/tools"
)

// defaultMaxToolIterations bounds the tool-calling loop so a model that
// never stops asking for tools can't hang a run forever.
const defaultMaxToolIterations = 8

// AgentComponent receives an input event, builds a prompt from its
// configured system prompt plus the incoming payload, calls a model
// provider, and emits the result on its output port.
//
// When Tools is empty it makes one plain call through Provider. When Tools
// is non-empty it instead runs a tool-calling loop through ChatProvider:
// call the model, execute any tools it asks for, feed the results back, and
// repeat until it gives a final answer (or MaxToolIterations is hit).
type AgentComponent struct {
	Name     string
	Prompt   string
	Provider providers.ModelProvider

	ChatProvider      providers.ToolCallingProvider
	Tools             []tools.Tool
	MaxToolIterations int

	// Memory, when set, makes this agent load prior turns before calling
	// the model and save the updated history afterward — keyed by
	// SessionKey, so different sessions/components never share history.
	Memory     memory.Store
	SessionKey string
	// Trace observes model and tool calls for debugging and run provenance.
	Trace func(kind, value string)
}

// Handle implements Component.
func (a *AgentComponent) Handle(ctx context.Context, event events.Event) ([]commands.Command, error) {
	if len(a.Tools) == 0 {
		return a.handleSimple(ctx, event)
	}
	return a.handleWithTools(ctx, event)
}

func (a *AgentComponent) handleSimple(ctx context.Context, event events.Event) ([]commands.Command, error) {
	history, err := a.loadHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("component %q: loading memory: %w", a.Name, err)
	}

	prompt := strings.TrimRight(a.Prompt, "\n")
	for _, turn := range history {
		prompt += fmt.Sprintf("\n\n%s: %s", turn.Role, turn.Content)
	}
	text := event.Text()
	if text != "" {
		prompt += "\n\nuser: " + text
	}

	a.trace("ModelCall", a.Name)
	resp, err := a.Provider.Generate(ctx, providers.Request{Prompt: prompt})
	if err != nil {
		return nil, fmt.Errorf("component %q: %w", a.Name, err)
	}

	if err := a.saveHistory(ctx, history, text, resp.Text); err != nil {
		return nil, fmt.Errorf("component %q: saving memory: %w", a.Name, err)
	}

	return []commands.Command{a.output(resp.Text)}, nil
}

func (a *AgentComponent) handleWithTools(ctx context.Context, event events.Event) ([]commands.Command, error) {
	maxIter := a.MaxToolIterations
	if maxIter == 0 {
		maxIter = defaultMaxToolIterations
	}

	toolSpecs := make([]providers.ToolSpec, len(a.Tools))
	toolByName := make(map[string]tools.Tool, len(a.Tools))
	for i, t := range a.Tools {
		toolSpecs[i] = providers.ToolSpec{Name: t.Name(), Description: t.Description(), Parameters: t.InputSchema()}
		toolByName[t.Name()] = t
	}

	history, err := a.loadHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("component %q: loading memory: %w", a.Name, err)
	}

	messages := []providers.Message{{Role: "system", Content: strings.TrimRight(a.Prompt, "\n")}}
	messages = append(messages, history...)
	text := event.Text()
	if text != "" {
		messages = append(messages, providers.Message{Role: "user", Content: text})
	}

	for i := 0; i < maxIter; i++ {
		a.trace("ModelCall", a.Name)
		resp, err := a.ChatProvider.Chat(ctx, providers.ChatRequest{Messages: messages, Tools: toolSpecs})
		if err != nil {
			return nil, fmt.Errorf("component %q: %w", a.Name, err)
		}

		if len(resp.Message.ToolCalls) == 0 {
			if err := a.saveHistory(ctx, history, text, resp.Message.Content); err != nil {
				return nil, fmt.Errorf("component %q: saving memory: %w", a.Name, err)
			}
			return []commands.Command{a.output(resp.Message.Content)}, nil
		}

		messages = append(messages, resp.Message)

		for j, call := range resp.Message.ToolCalls {
			id := call.ID
			if id == "" {
				id = fmt.Sprintf("call_%d_%d", i, j)
			}

			t, ok := toolByName[call.Name]
			if !ok {
				messages = append(messages, providers.Message{
					Role:       "tool",
					ToolCallID: id,
					Content:    fmt.Sprintf("error: component %q has no tool named %q configured", a.Name, call.Name),
				})
				continue
			}

			a.trace("ToolCall", a.Name+":"+call.Name)
			result, err := t.Execute(ctx, call.Arguments)
			if err != nil {
				result = fmt.Sprintf("error: %s", err)
			}
			messages = append(messages, providers.Message{Role: "tool", ToolCallID: id, Content: result})
		}
	}

	return nil, fmt.Errorf("component %q: exceeded %d tool-calling iterations without a final answer", a.Name, maxIter)
}

func (a *AgentComponent) trace(kind, value string) {
	if a.Trace != nil {
		a.Trace(kind, value)
	}
}

func (a *AgentComponent) output(text string) commands.Command {
	return commands.EmitEvent{Event: events.Event{Source: a.Name, Port: "output", Payload: text}}
}

// loadHistory returns this agent's prior turns, or nil if Memory isn't
// configured.
func (a *AgentComponent) loadHistory(ctx context.Context) ([]providers.Message, error) {
	if a.Memory == nil {
		return nil, nil
	}
	return a.Memory.Load(ctx, a.SessionKey)
}

// saveHistory appends the just-completed turn (user text, if any, plus the
// assistant's reply) to history and persists it. A no-op if Memory isn't
// configured.
func (a *AgentComponent) saveHistory(ctx context.Context, history []providers.Message, userText, assistantText string) error {
	if a.Memory == nil {
		return nil
	}

	updated := history
	if userText != "" {
		updated = append(updated, providers.Message{Role: "user", Content: userText})
	}
	updated = append(updated, providers.Message{Role: "assistant", Content: assistantText})

	return a.Memory.Save(ctx, a.SessionKey, updated)
}
