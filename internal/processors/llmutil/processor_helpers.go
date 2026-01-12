package llmutil

import (
	"context"
	"fmt"
	"log/slog"
	"text/template"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

// ParseProcessorTemplates parses the primary system/prompt templates and optional image caption templates
// for LLM processors so the callers can share consistent setup and error handling.
func ParseProcessorTemplates(name, systemTemplate, promptTemplate string, images *config.LLMImages) (*template.Template, *template.Template, CaptionTemplates, error) {
	systemTmpl, promptTmpl, err := ParseSystemAndPromptTemplates(name, systemTemplate, promptTemplate)
	if err != nil {
		return nil, nil, CaptionTemplates{}, err
	}

	captionTemplates, err := ParseImageCaptionTemplates(name, images)
	if err != nil {
		return nil, nil, CaptionTemplates{}, err
	}

	return systemTmpl, promptTmpl, captionTemplates, nil
}

// ParseImageCaptionTemplates parses the caption system/prompt templates if image captions are enabled.
func ParseImageCaptionTemplates(name string, images *config.LLMImages) (CaptionTemplates, error) {
	if images == nil || !images.Enabled || images.Mode != config.ImageModeCaption {
		return CaptionTemplates{}, nil
	}
	if images.Caption == nil {
		return CaptionTemplates{}, fmt.Errorf("image caption config is required when images.mode=caption")
	}

	systemTmpl, promptTmpl, err := ParseSystemAndPromptTemplates(name+"-image-caption", images.Caption.SystemTemplate, images.Caption.PromptTemplate)
	if err != nil {
		return CaptionTemplates{}, err
	}

	return CaptionTemplates{System: systemTmpl, Prompt: promptTmpl}, nil
}

// DefaultLogger ensures a non-nil logger is always available for processors.
func DefaultLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

// ProcessorLogger returns a context-aware logger that includes common processor metadata.
func ProcessorLogger(ctx context.Context, logger *slog.Logger, name string, processor any) *slog.Logger {
	if ctxLogger := core.LoggerFromContext(ctx); ctxLogger != nil {
		logger = ctxLogger
	}
	logger = DefaultLogger(logger)
	return logger.With("processor", name, "processor_type", fmt.Sprintf("%T", processor))
}
