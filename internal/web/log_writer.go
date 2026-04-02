package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

type LogPublisher interface {
	PublishLog(entry LogEntry)
}

type logWriter struct {
	registry *Registry
	docID    string
	runID    string
	buf      []LogEntry
	mu       sync.Mutex
}

func newLogWriter(registry *Registry, docID, runID string) *logWriter {
	return &logWriter{
		registry: registry,
		docID:    docID,
		runID:    runID,
		buf:      make([]LogEntry, 0, 64),
	}
}

func (lw *logWriter) Publish(entry LogEntry) {
	entry.RunID = lw.runID
	entry.Time = time.Now().UTC()

	lw.mu.Lock()
	lw.buf = append(lw.buf, entry)
	lw.mu.Unlock()

	// Log publication is keyed by registered flow ID during web-triggered runs.
	// Looking up by source path drops those entries before they reach websocket subscribers.
	if doc, ok := lw.registry.LookupDocByID(lw.docID); ok && doc != nil {
		doc.publishLog(entry)
	}
}

func (lw *logWriter) Buffered() []LogEntry {
	lw.mu.Lock()
	cpy := make([]LogEntry, len(lw.buf))
	copy(cpy, lw.buf)
	lw.mu.Unlock()
	return cpy
}

type RegistryLogHandler struct {
	registry *Registry
	docID    string
	runID    string
	slog.Handler
}

func NewRegistryLogHandler(logger *slog.Logger, registry *Registry, docID, runID string) *slog.Logger {
	handler := &RegistryLogHandler{
		registry: registry,
		docID:    docID,
		runID:    runID,
		Handler:  logger.Handler(),
	}
	newLogger := slog.New(handler)
	return newLogger
}

func (h *RegistryLogHandler) Handle(ctx context.Context, r slog.Record) error {
	entry := LogEntry{
		Time:    r.Time,
		Level:   r.Level.String(),
		Message: r.Message,
		RunID:   h.runID,
		FlowID:  h.docID,
	}
	// Runtime loggers are created with a flow ID, not a source path, so the
	// registry lookup must use the doc ID to keep websocket log streaming alive.
	if doc, ok := h.registry.LookupDocByID(h.docID); ok && doc != nil {
		doc.publishLog(entry)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *RegistryLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &RegistryLogHandler{
		registry: h.registry,
		docID:    h.docID,
		runID:    h.runID,
		Handler:  h.Handler.WithAttrs(attrs),
	}
}

func (h *RegistryLogHandler) WithGroup(group string) slog.Handler {
	return &RegistryLogHandler{
		registry: h.registry,
		docID:    h.docID,
		runID:    h.runID,
		Handler:  h.Handler.WithGroup(group),
	}
}

type LogEmitter interface {
	Emit(level string, msg string, attrs map[string]interface{})
}

type structuredLogEmitter struct {
	docID string
	runID string
}

func NewStructuredLogEmitter(docID, runID string) *structuredLogEmitter {
	return &structuredLogEmitter{docID: docID, runID: runID}
}

func (e *structuredLogEmitter) Emit(level string, msg string, attrs map[string]interface{}) {
	entry := LogEntry{
		Time:    time.Now().UTC(),
		Level:   level,
		Message: msg,
		RunID:   e.runID,
		FlowID:  e.docID,
	}
	if stage, ok := attrs["stage"].(string); ok {
		entry.Stage = stage
	}
	data, _ := json.Marshal(attrs)
	if len(data) > 0 {
		entry.Message = msg + " " + string(data)
	}
	if globalRegistry == nil {
		return
	}
	if doc, ok := globalRegistry.LookupDocByID(e.docID); ok && doc != nil {
		doc.publishLog(entry)
	}
}

var globalRegistry *Registry

func SetGlobalRegistry(r *Registry) {
	globalRegistry = r
}
