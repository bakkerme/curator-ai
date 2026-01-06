package factory

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

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
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
	redditimpl "github.com/bakkerme/curator-ai/internal/sources/reddit/impl"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
	rssimpl "github.com/bakkerme/curator-ai/internal/sources/rss/impl"
)

type Factory struct {
	Logger        *slog.Logger
	LLMClient     llm.Client
	DefaultModel  string
	RedditFetcher reddit.Fetcher
	RSSFetcher    rss.Fetcher
	EmailSender   email.Sender
}

func NewFromEnv() *Factory {
	logger := slog.Default()
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	defaultModel := os.Getenv("OPENAI_MODEL")
	if defaultModel == "" {
		defaultModel = "gpt-4o-mini"
	}
	llmClient := llmopenai.NewClient(apiKey, baseURL)

	redditTimeout := envDuration("REDDIT_HTTP_TIMEOUT", 10*time.Second)
	redditUserAgent := os.Getenv("REDDIT_USER_AGENT")
	if redditUserAgent == "" {
		redditUserAgent = "curator-ai/0.1"
	}
	redditClientID := os.Getenv("REDDIT_CLIENT_ID")
	redditClientSecret := os.Getenv("REDDIT_CLIENT_SECRET")
	redditUsername := os.Getenv("REDDIT_USERNAME")
	redditPassword := os.Getenv("REDDIT_PASSWORD")

	rssTimeout := envDuration("RSS_HTTP_TIMEOUT", 10*time.Second)
	rssUserAgent := os.Getenv("RSS_USER_AGENT")
	if rssUserAgent == "" {
		rssUserAgent = "curator-ai/0.1"
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := envInt("SMTP_PORT", 587)
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpUseTLS := envBool("SMTP_USE_TLS", true)

	return &Factory{
		Logger:        logger,
		LLMClient:     llmClient,
		DefaultModel:  defaultModel,
		RedditFetcher: redditimpl.NewFetcher(logger, redditTimeout, redditUserAgent, redditClientID, redditClientSecret, redditUsername, redditPassword),
		RSSFetcher:    rssimpl.NewFetcher(rssTimeout, rssUserAgent),
		EmailSender:   smtp.NewSender(smtpHost, smtpPort, smtpUser, smtpPassword, smtpUseTLS),
	}
}

func (f *Factory) NewCronTrigger(cfg *config.CronTrigger) (core.TriggerProcessor, error) {
	return trigger.NewCronProcessor(cfg.Schedule, cfg.Timezone), nil
}

func (f *Factory) NewRedditSource(cfg *config.RedditSource) (core.SourceProcessor, error) {
	return source.NewRedditProcessor(cfg, f.RedditFetcher)
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
	merged := mergeEmailConfig(cfg)
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

func mergeEmailConfig(cfg *config.EmailOutput) *config.EmailOutput {
	if cfg == nil {
		return &config.EmailOutput{}
	}
	merged := *cfg
	if merged.SMTPHost == "" {
		merged.SMTPHost = os.Getenv("SMTP_HOST")
	}
	if merged.SMTPPort == 0 {
		merged.SMTPPort = envInt("SMTP_PORT", 587)
	}
	if merged.SMTPUser == "" {
		merged.SMTPUser = os.Getenv("SMTP_USER")
	}
	if merged.SMTPPassword == "" {
		merged.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	}
	if merged.UseTLS == nil {
		useTLS := envBool("SMTP_USE_TLS", true)
		merged.UseTLS = &useTLS
	}
	return &merged
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
