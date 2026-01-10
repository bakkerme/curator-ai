package dedupe

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreTracksSeenIDs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "seen.db")
	store, err := NewSQLiteStore(dbPath, "", 0)
	if err != nil {
		t.Fatalf("failed to init sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	seen, err := store.HasSeen(context.Background(), "abc")
	if err != nil {
		t.Fatalf("has seen failed: %v", err)
	}
	if seen {
		t.Fatalf("expected unseen id")
	}

	if err := store.MarkSeen(context.Background(), "abc"); err != nil {
		t.Fatalf("mark seen failed: %v", err)
	}

	seen, err = store.HasSeen(context.Background(), "abc")
	if err != nil {
		t.Fatalf("has seen failed: %v", err)
	}
	if !seen {
		t.Fatalf("expected seen id")
	}
}

func TestSQLiteStoreHonorsTTL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "seen.db")
	store, err := NewSQLiteStore(dbPath, "", 5*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to init sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.MarkSeen(context.Background(), "ttl-id"); err != nil {
		t.Fatalf("mark seen failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	seen, err := store.HasSeen(context.Background(), "ttl-id")
	if err != nil {
		t.Fatalf("has seen failed: %v", err)
	}
	if seen {
		t.Fatalf("expected id to expire")
	}
}

func TestSQLiteStoreMarkSeenBatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "seen.db")
	store, err := NewSQLiteStore(dbPath, "", 0)
	if err != nil {
		t.Fatalf("failed to init sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ids := []string{"a", "b", "c"}
	if err := store.MarkSeenBatch(context.Background(), ids); err != nil {
		t.Fatalf("mark seen batch failed: %v", err)
	}

	for _, id := range ids {
		seen, err := store.HasSeen(context.Background(), id)
		if err != nil {
			t.Fatalf("has seen failed: %v", err)
		}
		if !seen {
			t.Fatalf("expected id %q to be seen", id)
		}
	}
}
