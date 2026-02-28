package config

import "testing"

func TestLoadEnv_RedditProxyDefaults(t *testing.T) {
	t.Setenv("REDDIT_PROXY_ENABLED", "")
	t.Setenv("REDDIT_PROXY_URL", "")

	env := LoadEnv()
	if env.Reddit.ProxyEnabled {
		t.Fatalf("expected REDDIT_PROXY_ENABLED default to be false")
	}
	if env.Reddit.ProxyURL != "" {
		t.Fatalf("expected REDDIT_PROXY_URL default to be empty, got %q", env.Reddit.ProxyURL)
	}
}

func TestLoadEnv_RedditProxyConfigured(t *testing.T) {
	t.Setenv("REDDIT_PROXY_ENABLED", "true")
	t.Setenv("REDDIT_PROXY_URL", "  http://user:pass@proxy.example.com:12321 \t")

	env := LoadEnv()
	if !env.Reddit.ProxyEnabled {
		t.Fatalf("expected REDDIT_PROXY_ENABLED to be true")
	}
	if env.Reddit.ProxyURL != "http://user:pass@proxy.example.com:12321" {
		t.Fatalf("unexpected REDDIT_PROXY_URL: %q", env.Reddit.ProxyURL)
	}
}
