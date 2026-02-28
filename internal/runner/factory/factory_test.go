package factory

import (
	"log/slog"
	"net/url"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
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
		t.Fatalf("expected factory creation to fail for missing REDDIT_PROXY_URL")
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
		t.Fatalf("unexpected error creating factory with valid reddit proxy: %v", err)
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
