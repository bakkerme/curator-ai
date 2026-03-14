package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type EnvConfig struct {
	CuratorConfigPath        string
	FlowID                   string
	RunOnce                  bool
	AllowPartialSourceErrors bool
	SessionID                string
	OpenAI                   OpenAIEnvConfig
	OTel                     OTelEnvConfig
	Crawl4AI                 Crawl4AIEnvConfig
	Docling                  DoclingEnvConfig
	Arxiv                    ArxivEnvConfig
	Reddit                   RedditEnvConfig
	RSS                      RSSEnvConfig
	Scrape                   ScrapeEnvConfig
	SMTP                     SMTPEnvConfig
}

type OpenAIEnvConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature *float64
	// EnableThinking toggles provider-specific reasoning/thinking features for
	// OpenAI-compatible endpoints that support chat_template_kwargs.enable_thinking.
	EnableThinking bool
	OTel           OpenAIOTelEnvConfig
}

type OpenAIOTelEnvConfig struct {
	Enabled       bool
	CaptureBodies bool
	MaxBodyBytes  int
}

type OTelEnvConfig struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	Protocol    string // "grpc" or "http/protobuf"
	Headers     map[string]string
	Insecure    bool
	SampleRatio float64
}

type Crawl4AIEnvConfig struct {
	BaseURL     string        // CRAWL4AI_BASE_URL (e.g. http://crawl4ai:11235)
	HTTPTimeout time.Duration // CRAWL4AI_HTTP_TIMEOUT, default 60s
}

type DoclingEnvConfig struct {
	BaseURL     string        // DOCLING_BASE_URL (e.g. http://docling:8000)
	HTTPTimeout time.Duration // DOCLING_HTTP_TIMEOUT, default 60s
}

type ArxivEnvConfig struct {
	BaseURL     string
	HTTPTimeout time.Duration
	UserAgent   string
}

type RedditEnvConfig struct {
	HTTPTimeout  time.Duration
	UserAgent    string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	ProxyEnabled bool
	ProxyURL     string
}

type RSSEnvConfig struct {
	HTTPTimeout time.Duration
	UserAgent   string
}

type ScrapeEnvConfig struct {
	HTTPTimeout time.Duration
	UserAgent   string
}

type SMTPEnvConfig struct {
	Host               string
	Port               int
	User               string
	Password           string
	TLSMode            string
	InsecureSkipVerify bool
}

func LoadEnv() (EnvConfig, error) {
	cfgPath := envString("CURATOR_CONFIG", "")
	if cfgPath == "" {
		return EnvConfig{}, errors.New("CURATOR_CONFIG environment variable is required")
	}

	flowID := envString("FLOW_ID", "flow-1")

	otlpEndpoint := strings.TrimSpace(envString("OTEL_EXPORTER_OTLP_ENDPOINT", ""))

	openAIModel := strings.TrimSpace(envString("OPENAI_MODEL", ""))
	if openAIModel == "" {
		return EnvConfig{}, errors.New("OPENAI_MODEL environment variable is required")
	}

	return EnvConfig{
		CuratorConfigPath:        cfgPath,
		FlowID:                   flowID,
		RunOnce:                  envBool("RUN_ONCE", false),
		AllowPartialSourceErrors: envBool("ALLOW_PARTIAL_SOURCE_ERRORS", false),
		SessionID:                strings.TrimSpace(envString("SESSION_ID", "")),
		OpenAI: OpenAIEnvConfig{
			APIKey:         strings.TrimSpace(envString("OPENAI_API_KEY", "")),
			BaseURL:        strings.TrimSpace(envString("OPENAI_BASE_URL", "")),
			Model:          openAIModel,
			Temperature:    envFloatPtr("OPENAI_TEMPERATURE"),
			EnableThinking: envBool("OPENAI_ENABLE_THINKING", true),
			OTel: OpenAIOTelEnvConfig{
				Enabled:       envBool("OTEL_OPENAI_ENABLED", true),
				CaptureBodies: envBool("OTEL_CAPTURE_OPENAI_BODIES", false),
				MaxBodyBytes:  envInt("OTEL_OPENAI_MAX_BODY_BYTES", 64*1024),
			},
		},
		OTel: OTelEnvConfig{
			Enabled:     envBool("OTEL_ENABLED", false),
			ServiceName: strings.TrimSpace(envString("OTEL_SERVICE_NAME", "curator-ai")),
			Endpoint:    otlpEndpoint,
			Protocol:    strings.ToLower(strings.TrimSpace(envString("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"))),
			Headers:     parseHeaders(envString("OTEL_EXPORTER_OTLP_HEADERS", "")),
			Insecure:    envBool("OTEL_EXPORTER_OTLP_INSECURE", defaultInsecure(otlpEndpoint)),
			SampleRatio: clamp01(envFloat("OTEL_TRACES_SAMPLE_RATIO", 1.0)),
		},
		Crawl4AI: Crawl4AIEnvConfig{
			BaseURL:     strings.TrimSpace(envString("CRAWL4AI_BASE_URL", "")),
			HTTPTimeout: envDuration("CRAWL4AI_HTTP_TIMEOUT", 60*time.Second),
		},
		Docling: DoclingEnvConfig{
			BaseURL:     strings.TrimSpace(envString("DOCLING_BASE_URL", "")),
			HTTPTimeout: envDuration("DOCLING_HTTP_TIMEOUT", 60*time.Second),
		},
		Arxiv: ArxivEnvConfig{
			BaseURL:     strings.TrimSpace(envString("ARXIV_BASE_URL", "")),
			HTTPTimeout: envDuration("ARXIV_HTTP_TIMEOUT", 10*time.Second),
			UserAgent:   envString("ARXIV_USER_AGENT", "curator-ai/0.1"),
		},
		Reddit: RedditEnvConfig{
			HTTPTimeout:  envDuration("REDDIT_HTTP_TIMEOUT", 10*time.Second),
			UserAgent:    envString("REDDIT_USER_AGENT", "curator-ai/0.1"),
			ClientID:     envString("REDDIT_CLIENT_ID", ""),
			ClientSecret: envString("REDDIT_CLIENT_SECRET", ""),
			Username:     envString("REDDIT_USERNAME", ""),
			Password:     envString("REDDIT_PASSWORD", ""),
			ProxyEnabled: envBool("REDDIT_PROXY_ENABLED", false),
			ProxyURL:     strings.TrimSpace(envString("REDDIT_PROXY_URL", "")),
		},
		RSS: RSSEnvConfig{
			HTTPTimeout: envDuration("RSS_HTTP_TIMEOUT", 10*time.Second),
			UserAgent:   envString("RSS_USER_AGENT", "curator-ai/0.1"),
		},
		Scrape: ScrapeEnvConfig{
			HTTPTimeout: envDuration("SCRAPE_HTTP_TIMEOUT", 10*time.Second),
			UserAgent:   envString("SCRAPE_USER_AGENT", "curator-ai/0.1"),
		},
		SMTP: SMTPEnvConfig{
			Host:               envString("SMTP_HOST", ""),
			Port:               envInt("SMTP_PORT", 587),
			User:               envString("SMTP_USER", ""),
			Password:           envString("SMTP_PASSWORD", ""),
			TLSMode:            envString("SMTP_TLS_MODE", ""),
			InsecureSkipVerify: envBool("SMTP_INSECURE_SKIP_VERIFY", false),
		},
	}, nil
}

func envString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func envFloat(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envFloatPtr(key string) *float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &f
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := ParseDurationExtended(v)
	if err != nil {
		return fallback
	}
	return d
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func parseHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func defaultInsecure(endpoint string) bool {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return true
	}
	if strings.Contains(endpoint, "://") {
		u, err := url.Parse(endpoint)
		if err != nil {
			return false
		}
		return u.Scheme == "http"
	}
	return strings.HasPrefix(endpoint, "localhost:") ||
		strings.HasPrefix(endpoint, "127.0.0.1:") ||
		strings.HasPrefix(endpoint, "0.0.0.0:")
}
