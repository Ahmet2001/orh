package memory

import (
	"context"
	"sync"

	"github.com/pertevniyalai/orh/internal/providers"
)

// InMemoryStore keeps history in a process-local map — never touches disk,
// and dies with the process. Used as the default Store for a run that
// doesn't pass --session: a cyclic graph that revisits the same agent
// several times within one run still accumulates history, but nothing
// persists once the run ends.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string][]providers.Message
}

// NewInMemoryStore returns an empty InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: map[string][]providers.Message{}}
}

// Load implements Store.
func (s *InMemoryStore) Load(ctx context.Context, key string) ([]providers.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]providers.Message(nil), s.data[key]...), nil
}

// Save implements Store.
func (s *InMemoryStore) Save(ctx context.Context, key string, messages []providers.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append([]providers.Message(nil), messages...)
	return nil
}
