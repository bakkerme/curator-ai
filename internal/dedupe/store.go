package dedupe

import "context"

// SeenStore tracks previously emitted item identifiers.
type SeenStore interface {
	HasSeen(ctx context.Context, id string) (bool, error)
	MarkSeen(ctx context.Context, id string) error
	MarkSeenBatch(ctx context.Context, ids []string) error
	Close() error
}
