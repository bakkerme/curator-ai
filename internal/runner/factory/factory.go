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
	"github.com/bakkerme/curator-ai/internal/sources/jina"
	jinaimpl "github.com/bakkerme/curator-ai/internal/sources/jina/impl"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
	rssimpl "github.com/bakkerme/curator-ai/internal/sources/rss/impl"
)

type Factory struct {
	Logger        *slog.Logger
	LLMClient     llm.Client
	DefaultModel  string
	SMTPDefaults  config.SMTPEnvConfig
	JinaReader    jina.Reader
	RedditFetcher reddit.Fetcher
	RSSFetcher    rss.Fetcher
	EmailSender   email.Sender
}

func NewFromEnvConfig(logger *slog.Logger, env config.EnvConfig) *Factory {
	if logger == nil {
		logger = slog.Default()
	}
	llmClient := llmopenai.NewClient(env.OpenAI)
	return &Factory{
		Logger:        logger,
		LLMClient:     llmClient,
		DefaultModel:  env.OpenAI.Model,
		SMTPDefaults:  env.SMTP,
		JinaReader:    jinaimpl.NewReader(env.Jina.HTTPTimeout, env.Jina.UserAgent, env.Jina.BaseURL, env.Jina.APIKey),
		RedditFetcher: reddit.NewFetcher(logger, env.Reddit.HTTPTimeout, env.Reddit.UserAgent, env.Reddit.ClientID, env.Reddit.ClientSecret, env.Reddit.Username, env.Reddit.Password),
		RSSFetcher:    rssimpl.NewFetcher(env.RSS.HTTPTimeout, env.RSS.UserAgent),
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
	return source.NewRedditProcessor(cfg, f.RedditFetcher, f.JinaReader, f.Logger)
}

func (f *Factory) NewRSSSource(cfg *config.RSSSource) (core.SourceProcessor, error) {
	return source.NewRSSProcessor(cfg, f.RSSFetcher)
}

func (f *Factory) NewQualityRule(cfg *config.QualityRule) (core.QualityProcessor, error) {
	return quality.NewRuleProcessor(cfg)
}

func (f *Factory) NewLLMQuality(cfg *config.LLMQuality) (core.QualityProcessor, error) {
	return quality.NewLLMProcessorWithLogger(cfg, f.LLMClient, f.DefaultModel, f.Logger)
}

func (f *Factory) NewLLMSummary(cfg *config.LLMSummary) (core.SummaryProcessor, error) {
	return summary.NewPostLLMProcessorWithLogger(cfg, f.LLMClient, f.DefaultModel, f.Logger)
}

func (f *Factory) NewLLMRunSummary(cfg *config.LLMSummary) (core.RunSummaryProcessor, error) {
	return summary.NewRunLLMProcessorWithLogger(cfg, f.LLMClient, f.DefaultModel, f.Logger)
}

func (f *Factory) NewMarkdownSummary(cfg *config.MarkdownSummary) (core.SummaryProcessor, error) {
	return summary.NewPostMarkdownProcessor(cfg)
}

func (f *Factory) NewMarkdownRunSummary(cfg *config.MarkdownSummary) (core.RunSummaryProcessor, error) {
	return summary.NewRunMarkdownProcessor(cfg)
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
	return output.NewEmailProcessor(merged, sender)
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
