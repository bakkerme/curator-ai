package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterDoc(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "My RSS Flow", "/path/to/rss.yaml")

	docs := r.ListDocs()
	require.Len(t, docs, 1)
	assert.Equal(t, "doc-1", docs[0].ID)
	assert.Equal(t, "My RSS Flow", docs[0].Name)
	assert.Equal(t, "/path/to/rss.yaml", docs[0].SourcePath)
	assert.False(t, docs[0].IsRunning)
	assert.Nil(t, docs[0].LastRun)
}

func TestRegistry_ListDocs(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow One", "/a.yaml")
	r.RegisterDoc("doc-2", "Flow Two", "/b.yaml")

	docs := r.ListDocs()
	assert.Len(t, docs, 2)
}

func TestRegistry_DocStatus(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")

	status, ok := r.DocStatus("doc-1")
	require.True(t, ok)
	assert.Equal(t, "doc-1", status.ID)
	assert.False(t, status.IsRunning)

	_, ok = r.DocStatus("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_LookupBySourcePath(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/path/to/doc.yaml")

	entry := r.LookupBySourcePath("/path/to/doc.yaml")
	require.NotNil(t, entry)
	assert.Equal(t, "doc-1", entry.ID)

	entry = r.LookupBySourcePath("/nonexistent")
	assert.Nil(t, entry)
}

func TestRegistry_PublishStatusTransition(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	entry := r.LookupBySourcePath("/f.yaml")
	require.NotNil(t, entry)

	var received []RunInfo
	entry.onStatusChange(func(info RunInfo) {
		received = append(received, info)
	})

	now := time.Now().UTC()
	entry.publishStatus(RunInfo{
		ID:        "run-1",
		FlowID:    "doc-1",
		StartedAt: now,
		Status:    RunStatusRunning,
	})

	require.Len(t, received, 1)
	assert.Equal(t, "run-1", received[0].ID)
	assert.Equal(t, RunStatusRunning, received[0].Status)
}

func TestRegistry_PublishLog(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	entry := r.LookupBySourcePath("/f.yaml")
	require.NotNil(t, entry)

	var received []LogEntry
	entry.onLog(func(entry LogEntry) {
		received = append(received, entry)
	})

	now := time.Now().UTC()
	entry.publishLog(LogEntry{
		Time:    now,
		Level:   "INFO",
		Message: "hello world",
		RunID:   "run-1",
		FlowID:  "doc-1",
	})

	require.Len(t, received, 1)
	assert.Equal(t, "hello world", received[0].Message)
}

func TestRegistry_BindRunToDocAndLookup(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	r.BindRunToDoc("run-abc", "doc-1")

	entry, ok := r.LookupByRunID("run-abc")
	require.True(t, ok)
	assert.Equal(t, "doc-1", entry.ID)

	_, ok = r.LookupByRunID("nonexistent-run")
	assert.False(t, ok)
}

func TestDocEntry_SubscribeLog(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	entry := r.LookupBySourcePath("/f.yaml")
	require.NotNil(t, entry)

	var received []LogEntry
	stop := entry.SubscribeLog(func(e LogEntry) {
		received = append(received, e)
	})

	entry.publishLog(LogEntry{Message: "first"})
	require.Len(t, received, 1)

	stop()
	entry.publishLog(LogEntry{Message: "second"})
	assert.Len(t, received, 1)
}

func TestMapCoreRunStatus(t *testing.T) {
	assert.Equal(t, RunStatusRunning, MapCoreRunStatus("running"))
	assert.Equal(t, RunStatusCompleted, MapCoreRunStatus("completed"))
	assert.Equal(t, RunStatusFailed, MapCoreRunStatus("failed"))
	assert.Equal(t, RunStatusCancelled, MapCoreRunStatus("cancelled"))
	assert.Equal(t, RunStatusPending, MapCoreRunStatus("unknown"))
	assert.Equal(t, RunStatusPending, MapCoreRunStatus(""))
}

func TestServer_HandleListDocs(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "RSS Feed", "/feeds/rss.yaml")
	r.RegisterDoc("doc-2", "Reddit", "/curator/reddit.yaml")

	logger := slog.Default()
	srv := NewServer(r, logger, OTelConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ListDocsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Docs, 2)
}

func TestServer_HandleTriggerRun_DocNotFound(t *testing.T) {
	r := NewRegistry()
	logger := slog.Default()
	srv := NewServer(r, logger, OTelConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/docs/nonexistent/run", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServer_HandleTriggerRun_UsesRunner(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")

	logger := slog.Default()
	srv := NewServer(r, logger, OTelConfig{})
	srv.SetTriggerRunHandler(func(ctx context.Context, docID string) (string, time.Time, error) {
		return "run-1", time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/docs/doc-1/run", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp TriggerRunResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "run-1", resp.RunID)
	assert.Equal(t, "doc-1", resp.FlowID)
}

func TestServer_HandleRunLogs_NotFound(t *testing.T) {
	r := NewRegistry()
	logger := slog.Default()
	srv := NewServer(r, logger, OTelConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/runs/nonexistent/logs", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServer_HandleRunLogs_WebSocket(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	r.BindRunToDoc("run-1", "doc-1")

	entry, _ := r.LookupByRunID("run-1")
	logger := slog.Default()
	srv := NewServer(r, logger, OTelConfig{})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		srv.ServeHTTP(w, req)
	}))
	defer ts.Close()

	wsURL := "ws://" + ts.Listener.Addr().String() + "/api/runs/run-1/logs"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	entry.publishLog(LogEntry{
		Time:    time.Now().UTC(),
		Level:   "INFO",
		Message: "hello from websocket",
		RunID:   "run-1",
		FlowID:  "doc-1",
	})

	var received LogEntry
	err = conn.ReadJSON(&received)
	require.NoError(t, err)
	assert.Equal(t, "hello from websocket", received.Message)
	assert.Equal(t, "run-1", received.RunID)
}

func TestServer_HandleRunLogs_WebSocketBacklog(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	r.BindRunToDoc("run-1", "doc-1")

	entry, _ := r.LookupByRunID("run-1")
	entry.publishLog(LogEntry{
		Time:    time.Now().UTC(),
		Level:   "INFO",
		Message: "buffered entry",
		RunID:   "run-1",
		FlowID:  "doc-1",
	})

	logger := slog.Default()
	srv := NewServer(r, logger, OTelConfig{})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		srv.ServeHTTP(w, req)
	}))
	defer ts.Close()

	wsURL := "ws://" + ts.Listener.Addr().String() + "/api/runs/run-1/logs"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	var received LogEntry
	err = conn.ReadJSON(&received)
	require.NoError(t, err)
	assert.Equal(t, "buffered entry", received.Message)
	assert.Equal(t, "run-1", received.RunID)
}

func TestLogWriter_Buffered(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")

	lw := newLogWriter(r, "/f.yaml", "run-1")
	lw.Publish(LogEntry{Level: "INFO", Message: "one"})
	lw.Publish(LogEntry{Level: "INFO", Message: "two"})

	entries := lw.Buffered()
	require.Len(t, entries, 2)
	assert.Equal(t, "one", entries[0].Message)
	assert.Equal(t, "two", entries[1].Message)
	assert.Equal(t, "run-1", entries[0].RunID)
}

func TestRegistryLogHandler_PublishesToDocByID(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")

	entry, ok := r.LookupDocByID("doc-1")
	require.True(t, ok)

	var received []LogEntry
	stop := entry.SubscribeLog(func(e LogEntry) {
		received = append(received, e)
	})
	defer stop()

	logger := NewRegistryLogHandler(slog.Default(), r, "doc-1", "run-1")
	logger.Info("hello from handler")

	require.Len(t, received, 1)
	assert.Equal(t, "hello from handler", received[0].Message)
	assert.Equal(t, "run-1", received[0].RunID)
	assert.Equal(t, "doc-1", received[0].FlowID)
}

func TestStructuredLogEmitter_PublishesToDocByID(t *testing.T) {
	r := NewRegistry()
	r.RegisterDoc("doc-1", "Flow", "/f.yaml")
	SetGlobalRegistry(r)
	t.Cleanup(func() {
		SetGlobalRegistry(nil)
	})

	entry, ok := r.LookupDocByID("doc-1")
	require.True(t, ok)

	var received []LogEntry
	stop := entry.SubscribeLog(func(e LogEntry) {
		received = append(received, e)
	})
	defer stop()

	emitter := NewStructuredLogEmitter("doc-1", "run-1")
	emitter.Emit("INFO", "structured message", map[string]interface{}{"stage": "source"})

	require.Len(t, received, 1)
	assert.Equal(t, "run-1", received[0].RunID)
	assert.Equal(t, "doc-1", received[0].FlowID)
	assert.Equal(t, "source", received[0].Stage)
	assert.Contains(t, received[0].Message, "structured message")
}

func TestSlugifyName(t *testing.T) {
	assert.Equal(t, "my-rss-flow", slugifyName("My RSS Flow"))
	assert.Equal(t, "ai-ml-daily", slugifyName("AI/ML (Daily)"))
	assert.Equal(t, "flow", slugifyName(""))
	assert.Equal(t, "hello-world", slugifyName("Hello   World!"))
}

func TestFlowIDForPath(t *testing.T) {
	assert.Equal(t, "some-path-yaml", flowIDForPath("/some/path.yaml", nil))
	assert.Equal(t, "test-flow", flowIDForPath("/test.yaml", &config.CuratorDocument{Workflow: config.Workflow{Name: "Test Flow"}}))
}
