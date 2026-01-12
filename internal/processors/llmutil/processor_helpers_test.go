package llmutil

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

type captureHandler struct {
	store  *captureStore
	attrs  []slog.Attr
	groups []string
}

type captureStore struct {
	mu     sync.Mutex
	events []capturedEvent
}

type capturedEvent struct {
	message string
	level   slog.Level
	attrs   map[string]any
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	attrs := make(map[string]any)
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	h.store.events = append(h.store.events, capturedEvent{
		message: record.Message,
		level:   record.Level,
		attrs:   attrs,
	})
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &captureHandler{
		store:  h.store,
		attrs:  append(append([]slog.Attr{}, h.attrs...), attrs...),
		groups: append([]string{}, h.groups...),
	}
	return next
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	next := &captureHandler{
		store:  h.store,
		attrs:  append([]slog.Attr{}, h.attrs...),
		groups: append(append([]string{}, h.groups...), name),
	}
	return next
}

func TestParseImageCaptionTemplates_GuardsAndErrors(t *testing.T) {
	t.Run("disabled and non-caption are no-ops", func(t *testing.T) {
		for _, cfg := range []*config.LLMImages{
			nil,
			{Enabled: false, Mode: config.ImageModeCaption, Caption: &config.LLMImageCaption{}},
			{Enabled: true, Mode: config.ImageModeMultimodal, Caption: &config.LLMImageCaption{}},
		} {
			tmpls, err := ParseImageCaptionTemplates("x", cfg)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tmpls.System != nil || tmpls.Prompt != nil {
				t.Fatalf("expected empty templates, got %#v", tmpls)
			}
		}
	})

	t.Run("missing caption config errors", func(t *testing.T) {
		_, err := ParseImageCaptionTemplates("x", &config.LLMImages{Enabled: true, Mode: config.ImageModeCaption})
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "image caption config is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("template parse error propagates", func(t *testing.T) {
		_, err := ParseImageCaptionTemplates("x", &config.LLMImages{
			Enabled: true,
			Mode:    config.ImageModeCaption,
			Caption: &config.LLMImageCaption{
				SystemTemplate: "{{",
				PromptTemplate: "ok",
			},
		})
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "parse system template:") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseImageCaptionTemplates_Success(t *testing.T) {
	tmpls, err := ParseImageCaptionTemplates("myproc", &config.LLMImages{
		Enabled: true,
		Mode:    config.ImageModeCaption,
		Caption: &config.LLMImageCaption{
			SystemTemplate: "system",
			PromptTemplate: "prompt",
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tmpls.System == nil || tmpls.Prompt == nil {
		t.Fatalf("expected non-nil templates")
	}
	if tmpls.System.Name() != "myproc-image-caption" || tmpls.Prompt.Name() != "myproc-image-caption" {
		t.Fatalf("unexpected template names: %q / %q", tmpls.System.Name(), tmpls.Prompt.Name())
	}
}

func TestParseProcessorTemplates_SuccessAndErrorPropagation(t *testing.T) {
	t.Run("success without image captions", func(t *testing.T) {
		systemTmpl, promptTmpl, captions, err := ParseProcessorTemplates("myproc", "sys", "prompt", nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if systemTmpl == nil || promptTmpl == nil {
			t.Fatalf("expected non-nil templates")
		}
		if captions.System != nil || captions.Prompt != nil {
			t.Fatalf("expected no caption templates, got %#v", captions)
		}
	})

	t.Run("success with image captions", func(t *testing.T) {
		systemTmpl, promptTmpl, captions, err := ParseProcessorTemplates("myproc", "sys", "prompt", &config.LLMImages{
			Enabled: true,
			Mode:    config.ImageModeCaption,
			Caption: &config.LLMImageCaption{
				SystemTemplate: "imgsys",
				PromptTemplate: "imgprompt",
			},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if systemTmpl == nil || promptTmpl == nil {
			t.Fatalf("expected non-nil primary templates")
		}
		if captions.System == nil || captions.Prompt == nil {
			t.Fatalf("expected non-nil caption templates")
		}
	})

	t.Run("primary template parse error returns nil templates", func(t *testing.T) {
		systemTmpl, promptTmpl, captions, err := ParseProcessorTemplates("myproc", "{{", "ok", nil)
		if err == nil {
			t.Fatalf("expected error")
		}
		if systemTmpl != nil || promptTmpl != nil {
			t.Fatalf("expected nil templates on error")
		}
		if captions.System != nil || captions.Prompt != nil {
			t.Fatalf("expected empty caption templates, got %#v", captions)
		}
	})

	t.Run("caption template errors return nil templates", func(t *testing.T) {
		systemTmpl, promptTmpl, captions, err := ParseProcessorTemplates("myproc", "sys", "prompt", &config.LLMImages{
			Enabled: true,
			Mode:    config.ImageModeCaption,
		})
		if err == nil {
			t.Fatalf("expected error")
		}
		if systemTmpl != nil || promptTmpl != nil {
			t.Fatalf("expected nil templates on error")
		}
		if captions.System != nil || captions.Prompt != nil {
			t.Fatalf("expected empty caption templates, got %#v", captions)
		}
	})
}

func TestDefaultLogger_ReturnsNonNil(t *testing.T) {
	// Ensure the slog default logger is deterministic for this test.
	orig := slog.Default()
	defer slog.SetDefault(orig)

	defaultHandler := &captureHandler{store: &captureStore{}}
	defaultLogger := slog.New(defaultHandler)
	slog.SetDefault(defaultLogger)

	if got := DefaultLogger(nil); got != defaultLogger {
		t.Fatalf("expected default logger to be returned")
	}

	customLogger := slog.New(&captureHandler{store: &captureStore{}})
	if got := DefaultLogger(customLogger); got != customLogger {
		t.Fatalf("expected custom logger to be returned")
	}
}

func TestProcessorLogger_UsesContextLoggerAndAddsMetadata(t *testing.T) {
	type testProcessor struct{}

	ctxHandler := &captureHandler{store: &captureStore{}}
	ctxLogger := slog.New(ctxHandler)
	ctx := core.WithLogger(context.Background(), ctxLogger)

	passedHandler := &captureHandler{store: &captureStore{}}
	passedLogger := slog.New(passedHandler)

	processor := &testProcessor{}
	logger := ProcessorLogger(ctx, passedLogger, "unit-test", processor)
	logger.Info("hello")

	ctxHandler.store.mu.Lock()
	defer ctxHandler.store.mu.Unlock()

	if len(ctxHandler.store.events) != 1 {
		t.Fatalf("expected 1 log event from context logger, got %d", len(ctxHandler.store.events))
	}

	event := ctxHandler.store.events[0]
	if event.message != "hello" {
		t.Fatalf("unexpected message: %q", event.message)
	}

	if got := event.attrs["processor"]; got != "unit-test" {
		t.Fatalf("missing/incorrect processor attr: %v", got)
	}
	wantType := fmt.Sprintf("%T", processor)
	if got := event.attrs["processor_type"]; got != wantType {
		t.Fatalf("missing/incorrect processor_type attr: %v (want %q)", got, wantType)
	}

	passedHandler.store.mu.Lock()
	defer passedHandler.store.mu.Unlock()
	if len(passedHandler.store.events) != 0 {
		t.Fatalf("expected passed logger to be ignored when context logger exists")
	}
}

func TestProcessorLogger_FallsBackToDefaultLoggerWhenNoContextLogger(t *testing.T) {
	orig := slog.Default()
	defer slog.SetDefault(orig)

	defaultHandler := &captureHandler{store: &captureStore{}}
	defaultLogger := slog.New(defaultHandler)
	slog.SetDefault(defaultLogger)

	passedHandler := &captureHandler{store: &captureStore{}}
	passedLogger := slog.New(passedHandler)

	logger := ProcessorLogger(context.Background(), passedLogger, "unit-test", struct{}{})
	logger.Info("hello")

	defaultHandler.store.mu.Lock()
	defer defaultHandler.store.mu.Unlock()
	if len(defaultHandler.store.events) != 1 {
		t.Fatalf("expected 1 log event from default logger, got %d", len(defaultHandler.store.events))
	}

	passedHandler.store.mu.Lock()
	defer passedHandler.store.mu.Unlock()
	if len(passedHandler.store.events) != 0 {
		t.Fatalf("expected passed logger to be ignored when no context logger exists")
	}
}
