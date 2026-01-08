package factory

import (
	"log/slog"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
	llmopenai "github.com/bakkerme/curator-ai/internal/llm/openai"
	"github.com/bakkerme/curator-ai/internal/outputs/email"
	"github.com/bakkerme/curator-ai/internal/outputs/email/smtp"
	"github.com/bakkerme/curator-ai/internal/processors/output"
	"github.com/bakkerme/curator-ai/internal/processors/quality"
	"github.com/bakkerme/curator-ai/internal/processors/source"
	"github.com/bakkerme/curator-ai/internal/processors/summary"
	"github.com/bakkerme/curator-ai/internal/processors/trigger"
	"github.com/bakkerme/curator-ai/internal/runner/snapshot"
	"github.com/bakkerme/curator-ai/internal/sources/jina"
	jinaimpl "github.com/bakkerme/curator-ai/internal/sources/jina/impl"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
	rssimpl "github.com/bakkerme/curator-ai/internal/sources/rss/impl"
)

type Factory struct {
	Logger             *slog.Logger
	LLMClient          llm.Client
	DefaultModel       string
	DefaultTemperature *float64
	DefaultTopP        *float64
	DefaultPresPenalty *float64
	DefaultTopK        *int
	SMTPDefaults       config.SMTPEnvConfig
	JinaReader         jina.Reader
	RedditFetcher      reddit.Fetcher
	RSSFetcher         rss.Fetcher
	EmailSender        email.Sender
}

func NewFromEnvConfig(logger *slog.Logger, env config.EnvConfig) *Factory {
	if logger == nil {
		logger = slog.Default()
	}
	llmClient := llmopenai.NewClient(env.OpenAI)
	return &Factory{
		Logger:             logger,
		LLMClient:          llmClient,
		DefaultModel:       env.OpenAI.Model,
		DefaultTemperature: env.OpenAI.Temperature,
		DefaultTopP:        env.OpenAI.TopP,
		DefaultPresPenalty: env.OpenAI.PresencePenalty,
		DefaultTopK:        env.OpenAI.TopK,
		SMTPDefaults:       env.SMTP,
		JinaReader:         jinaimpl.NewReader(env.Jina.HTTPTimeout, env.Jina.UserAgent, env.Jina.BaseURL, env.Jina.APIKey),
		RedditFetcher:      reddit.NewFetcher(logger, env.Reddit.HTTPTimeout, env.Reddit.UserAgent, env.Reddit.ClientID, env.Reddit.ClientSecret, env.Reddit.Username, env.Reddit.Password),
		RSSFetcher:         rssimpl.NewFetcher(env.RSS.HTTPTimeout, env.RSS.UserAgent),
		// Leave EmailSender nil so the output processor can build it from the merged
		// YAML config + env defaults. This allows per-flow SMTP overrides in the Curator
		// Document to take effect.
		EmailSender: nil,
	}
}

func (f *Factory) NewCronTrigger(cfg *config.CronTrigger) (core.TriggerProcessor, error) {
	return trigger.NewCronProcessor(cfg.Schedule, cfg.Timezone), nil
}

func (f *Factory) NewRedditSource(cfg *config.RedditSource) (core.SourceProcessor, error) {
	processor, err := source.NewRedditProcessor(cfg, f.RedditFetcher, f.JinaReader, f.Logger)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapSource(processor, cfg.Snapshot), nil
}

func (f *Factory) NewRSSSource(cfg *config.RSSSource) (core.SourceProcessor, error) {
	processor, err := source.NewRSSProcessor(cfg, f.RSSFetcher)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapSource(processor, cfg.Snapshot), nil
}

func (f *Factory) NewQualityRule(cfg *config.QualityRule) (core.QualityProcessor, error) {
	processor, err := quality.NewRuleProcessor(cfg)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapQuality(processor, cfg.Snapshot), nil
}

func (f *Factory) NewLLMQuality(cfg *config.LLMQuality) (core.QualityProcessor, error) {
	processor, err := quality.NewLLMProcessorWithLogger(cfg, f.LLMClient, f.DefaultModel, f.Logger, f.DefaultTemperature, f.DefaultTopP, f.DefaultPresPenalty, f.DefaultTopK)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapQuality(processor, cfg.Snapshot), nil
}

func (f *Factory) NewLLMSummary(cfg *config.LLMSummary) (core.SummaryProcessor, error) {
	processor, err := summary.NewPostLLMProcessorWithLogger(cfg, f.LLMClient, f.DefaultModel, f.Logger, f.DefaultTemperature, f.DefaultTopP, f.DefaultPresPenalty, f.DefaultTopK)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapSummary(processor, cfg.Snapshot), nil
}

func (f *Factory) NewLLMRunSummary(cfg *config.LLMSummary) (core.RunSummaryProcessor, error) {
	processor, err := summary.NewRunLLMProcessorWithLogger(cfg, f.LLMClient, f.DefaultModel, f.Logger, f.DefaultTemperature, f.DefaultTopP, f.DefaultPresPenalty, f.DefaultTopK)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapRunSummary(processor, cfg.Snapshot), nil
}

func (f *Factory) NewMarkdownSummary(cfg *config.MarkdownSummary) (core.SummaryProcessor, error) {
	processor, err := summary.NewPostMarkdownProcessor(cfg)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapSummary(processor, cfg.Snapshot), nil
}

func (f *Factory) NewMarkdownRunSummary(cfg *config.MarkdownSummary) (core.RunSummaryProcessor, error) {
	processor, err := summary.NewRunMarkdownProcessor(cfg)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapRunSummary(processor, cfg.Snapshot), nil
}

func (f *Factory) NewEmailOutput(cfg *config.EmailOutput) (core.OutputProcessor, error) {
	merged := f.mergeEmailConfig(cfg)
	sender := f.EmailSender
	if sender == nil {
		useTLS := true
		if merged.UseTLS != nil {
			useTLS = *merged.UseTLS
		}
		sender = smtp.NewSender(merged.SMTPHost, merged.SMTPPort, merged.SMTPUser, merged.SMTPPassword, useTLS)
	}
	processor, err := output.NewEmailProcessor(merged, sender)
	if err != nil {
		return nil, err
	}
	return snapshot.WrapOutput(processor, merged.Snapshot), nil
}

func (f *Factory) mergeEmailConfig(cfg *config.EmailOutput) *config.EmailOutput {
	if cfg == nil {
		return &config.EmailOutput{}
	}
	merged := *cfg
	if merged.SMTPHost == "" {
		merged.SMTPHost = f.SMTPDefaults.Host
	}
	if merged.SMTPPort == 0 {
		merged.SMTPPort = f.SMTPDefaults.Port
	}
	if merged.SMTPUser == "" {
		merged.SMTPUser = f.SMTPDefaults.User
	}
	if merged.SMTPPassword == "" {
		merged.SMTPPassword = f.SMTPDefaults.Password
	}
	if merged.UseTLS == nil {
		useTLS := f.SMTPDefaults.UseTLS
		merged.UseTLS = &useTLS
	}
	return &merged
}
