package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pertevniyalai/orh/internal/providers"
)

// FileStore persists history as one JSON-lines file per key under Dir —
// used when a run passes --session, so history survives across separate
// `orh run` invocations.
type FileStore struct {
	Dir string
}

// NewFileStore returns a FileStore rooted at dir.
func NewFileStore(dir string) *FileStore {
	return &FileStore{Dir: dir}
}

// Load implements Store. A key with no file yet returns (nil, nil).
func (f *FileStore) Load(ctx context.Context, key string) ([]providers.Message, error) {
	path := f.path(key)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var messages []providers.Message
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var m providers.Message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// Save implements Store, overwriting the key's file with the full history.
func (f *FileStore) Save(ctx context.Context, key string, messages []providers.Message) error {
	path := f.path(key)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	var b strings.Builder
	for _, m := range messages {
		line, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("encoding message: %w", err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func (f *FileStore) path(key string) string {
	return filepath.Join(f.Dir, safeFilename(key)+".jsonl")
}

// safeFilename replaces path separators and ".." sequences in key so it
// can never escape Dir, while staying human-readable for debugging.
func safeFilename(key string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_")
	return replacer.Replace(key)
}
