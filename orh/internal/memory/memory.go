// Package memory gives an AgentComponent a place to persist conversation
// history across Handle calls — within a single run (so a component
// revisited in a cyclic graph remembers earlier turns) and, when a session
// id is set, across separate `orh run` invocations too.
package memory

import (
	"context"

	"github.com/pertevniyalai/orh/internal/providers"
)

// Store loads and saves the full turn history for a key. A key identifies
// one conversation — typically a session id plus a component name, so
// different agents (and different sessions) never see each other's
// history.
type Store interface {
	// Load returns the stored history for key, or (nil, nil) if nothing
	// has been saved for it yet.
	Load(ctx context.Context, key string) ([]providers.Message, error)
	// Save overwrites the stored history for key.
	Save(ctx context.Context, key string, messages []providers.Message) error
}
