package config

import (
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
	Jina                     JinaEnvConfig
	Reddit                   RedditEnvConfig
	RSS                      RSSEnvConfig
	SMTP                     SMTPEnvConfig
}

type OpenAIEnvConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature *float64
	OTel        OpenAIOTelEnvConfig
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

type JinaEnvConfig struct {
	APIKey      string
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
}

type RSSEnvConfig struct {
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

func LoadEnv() EnvConfig {
	cfgPath := envString("CURATOR_CONFIG", "curator.yaml")
	flowID := envString("FLOW_ID", "flow-1")

	otlpEndpoint := strings.TrimSpace(envString("OTEL_EXPORTER_OTLP_ENDPOINT", ""))

	openAIModel := strings.TrimSpace(envString("OPENAI_MODEL", ""))
	if openAIModel == "" {
		openAIModel = "gpt-4o-mini"
	}

	return EnvConfig{
		CuratorConfigPath:        cfgPath,
		FlowID:                   flowID,
		RunOnce:                  envBool("RUN_ONCE", false),
		AllowPartialSourceErrors: envBool("ALLOW_PARTIAL_SOURCE_ERRORS", false),
		SessionID:                strings.TrimSpace(envString("SESSION_ID", "")),
		OpenAI: OpenAIEnvConfig{
			APIKey:      strings.TrimSpace(envString("OPENAI_API_KEY", "")),
			BaseURL:     strings.TrimSpace(envString("OPENAI_BASE_URL", "")),
			Model:       openAIModel,
			Temperature: envFloatPtr("OPENAI_TEMPERATURE"),
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
		Jina: JinaEnvConfig{
			APIKey:      strings.TrimSpace(envString("JINA_API_KEY", "")),
			BaseURL:     strings.TrimSpace(envString("JINA_BASE_URL", "")),
			HTTPTimeout: envDuration("JINA_HTTP_TIMEOUT", 15*time.Second),
			UserAgent:   envString("JINA_USER_AGENT", "curator-ai/0.1"),
		},
		Reddit: RedditEnvConfig{
			HTTPTimeout:  envDuration("REDDIT_HTTP_TIMEOUT", 10*time.Second),
			UserAgent:    envString("REDDIT_USER_AGENT", "curator-ai/0.1"),
			ClientID:     envString("REDDIT_CLIENT_ID", ""),
			ClientSecret: envString("REDDIT_CLIENT_SECRET", ""),
			Username:     envString("REDDIT_USERNAME", ""),
			Password:     envString("REDDIT_PASSWORD", ""),
		},
		RSS: RSSEnvConfig{
			HTTPTimeout: envDuration("RSS_HTTP_TIMEOUT", 10*time.Second),
			UserAgent:   envString("RSS_USER_AGENT", "curator-ai/0.1"),
		},
		SMTP: SMTPEnvConfig{
			Host:               envString("SMTP_HOST", ""),
			Port:               envInt("SMTP_PORT", 587),
			User:               envString("SMTP_USER", ""),
			Password:           envString("SMTP_PASSWORD", ""),
			TLSMode:            envString("SMTP_TLS_MODE", ""),
			InsecureSkipVerify: envBool("SMTP_INSECURE_SKIP_VERIFY", false),
		},
	}
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
	d, err := parseDurationExtended(v)
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
