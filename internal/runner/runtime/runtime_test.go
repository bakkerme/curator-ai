package runtime

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/dedupe"
)

func TestParseRedditProxyURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.RedditEnvConfig
		wantErr bool
	}{
		{
			name: "disabled ignores empty url",
			cfg: config.RedditEnvConfig{
				ProxyEnabled: false,
				ProxyURL:     "",
			},
			wantErr: false,
		},
		{
			name: "enabled requires url",
			cfg: config.RedditEnvConfig{
				ProxyEnabled: true,
				ProxyURL:     "",
			},
			wantErr: true,
		},
		{
			name: "enabled rejects malformed url",
			cfg: config.RedditEnvConfig{
				ProxyEnabled: true,
				ProxyURL:     "not-a-url",
			},
			wantErr: true,
		},
		{
			name: "enabled rejects non-http scheme",
			cfg: config.RedditEnvConfig{
				ProxyEnabled: true,
				ProxyURL:     "socks5://proxy.example.com:1080",
			},
			wantErr: true,
		},
		{
			name: "enabled accepts valid url",
			cfg: config.RedditEnvConfig{
				ProxyEnabled: true,
				ProxyURL:     "http://user:pass@proxy.example.com:12321",
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRedditProxyURL(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.cfg.ProxyEnabled {
				if got == nil {
					t.Fatalf("expected parsed URL when proxy is enabled")
				}
			}
		})
	}
}

func TestNewFromEnvConfig_RecordReplayMutuallyExclusive(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	_, err := NewFromEnvConfig(logger, config.EnvConfig{
		OpenAI: config.OpenAIEnvConfig{
			Model: "gpt-4o-mini",
		},
		LLMRecordPath: "/tmp/record.json",
		LLMReplayPath: "/tmp/replay.json",
	})
	if err == nil {
		t.Fatal("expected error when both CURATOR_LLM_RECORD and CURATOR_LLM_REPLAY are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
}

func TestNewFromEnvConfig_RedditProxyValidation(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	_, err := NewFromEnvConfig(logger, config.EnvConfig{
		Reddit: config.RedditEnvConfig{
			ProxyEnabled: true,
			ProxyURL:     "",
		},
	})
	if err == nil {
		t.Fatalf("expected runtime creation to fail for missing REDDIT_PROXY_URL")
	}

	proxyURL := "http://user:pass@proxy.example.com:12321"
	f, err := NewFromEnvConfig(logger, config.EnvConfig{
		OpenAI: config.OpenAIEnvConfig{
			Model: "gpt-4o-mini",
		},
		Reddit: config.RedditEnvConfig{
			ProxyEnabled: true,
			ProxyURL:     proxyURL,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating runtime with valid reddit proxy: %v", err)
	}
	if f == nil {
		t.Fatalf("expected non-nil factory")
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		t.Fatalf("parse expected proxy URL: %v", err)
	}
	if parsed.Host != "proxy.example.com:12321" {
		t.Fatalf("unexpected parsed host %q", parsed.Host)
	}
}

type closerStub struct {
	closed bool
	err    error
}

func (c *closerStub) Close() error {
	c.closed = true
	return c.err
}

type seenStoreStub struct {
	closed bool
	err    error
}

func (s *seenStoreStub) HasSeen(context.Context, string) (bool, error) { return false, nil }
func (s *seenStoreStub) MarkSeen(context.Context, string) error         { return nil }
func (s *seenStoreStub) MarkSeenBatch(context.Context, []string) error  { return nil }
func (s *seenStoreStub) Close() error {
	s.closed = true
	return s.err
}

func TestRuntimeCloneForFlow_IsolatesDedupeStore(t *testing.T) {
	t.Parallel()

	root := &Runtime{
		Logger: slog.Default(),
	}
	a := root.CloneForFlow()
	b := root.CloneForFlow()

	dbPath := filepath.Join(t.TempDir(), "seen.db")
	if err := a.ConfigureDedupeStore(&config.DedupeStoreConfig{
		Driver: "sqlite",
		DSN:    dbPath,
		Table:  "seen_posts",
	}); err != nil {
		t.Fatalf("configure dedupe for clone A: %v", err)
	}
	if a.SeenStore == nil {
		t.Fatalf("expected clone A to have a dedupe store")
	}
	if b.SeenStore != nil {
		t.Fatalf("expected clone B to remain without dedupe store")
	}

	if err := b.ConfigureDedupeStore(nil); err != nil {
		t.Fatalf("configure nil dedupe for clone B: %v", err)
	}
	if a.SeenStore == nil {
		t.Fatalf("expected clone A dedupe store to remain configured")
	}

	// Clone A store must still be live after clone B reset.
	if err := a.SeenStore.MarkSeen(context.Background(), "rss-id-1"); err != nil {
		t.Fatalf("expected clone A store to stay usable, got error: %v", err)
	}
	if seen, err := a.SeenStore.HasSeen(context.Background(), "rss-id-1"); err != nil || !seen {
		t.Fatalf("expected clone A store to read back key (seen=%v, err=%v)", seen, err)
	}

	_ = a.Close()
	_ = b.Close()
}

func TestRuntimeClose_ClosesSeenStoreAndClosers(t *testing.T) {
	t.Parallel()

	store := &seenStoreStub{}
	closer := &closerStub{}
	f := &Runtime{
		SeenStore: store,
		closers:   []Closer{closer},
	}

	if err := f.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.closed {
		t.Fatalf("expected seen store to be closed")
	}
	if !closer.closed {
		t.Fatalf("expected closer to be closed")
	}
}

func TestRuntimeClose_CollectsCloseErrors(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("store close failed")
	closerErr := errors.New("closer close failed")
	f := &Runtime{
		SeenStore: &seenStoreStub{err: storeErr},
		closers:   []Closer{&closerStub{err: closerErr}},
	}

	err := f.Close()
	if err == nil {
		t.Fatalf("expected close error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "runtime close") || !strings.Contains(msg, storeErr.Error()) || !strings.Contains(msg, closerErr.Error()) {
		t.Fatalf("expected combined close errors, got: %v", err)
	}
}

func TestRuntimeCloneForFlow_DoesNotCopyClosersOrSeenStore(t *testing.T) {
	t.Parallel()

	root := &Runtime{
		SeenStore: &seenStoreStub{},
		closers:   []Closer{&closerStub{}},
	}

	clone := root.CloneForFlow()
	if clone == root {
		t.Fatalf("expected new instance")
	}
	if clone.SeenStore != nil {
		t.Fatalf("expected clone seen store to be reset")
	}
	if len(clone.closers) != 0 {
		t.Fatalf("expected clone closers to be reset")
	}
}

func TestRuntimeCloneForFlow_NilReceiver(t *testing.T) {
	t.Parallel()

	var f *Runtime
	clone := f.CloneForFlow()
	if clone == nil {
		t.Fatalf("expected non-nil clone")
	}
	if clone.SeenStore != nil || len(clone.closers) != 0 {
		t.Fatalf("expected empty clone state")
	}
}

func TestRuntimeCloneForFlow_LeavesRootUntouched(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "root.db")
	store, err := dedupe.NewSQLiteStore(dbPath, "seen_posts", 0)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	root := &Runtime{
		SeenStore: store,
		closers:   []Closer{&closerStub{}},
	}

	clone := root.CloneForFlow()
	if root.SeenStore == nil {
		t.Fatalf("expected root store to remain configured")
	}
	if len(root.closers) != 1 {
		t.Fatalf("expected root closers to remain configured")
	}
	if clone.SeenStore != nil || len(clone.closers) != 0 {
		t.Fatalf("expected clone to reset mutable fields")
	}

	_ = root.Close()
}
