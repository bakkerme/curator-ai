package config

import "testing"

// setRequiredEnv sets the env vars that LoadEnv requires so tests that
// exercise other fields do not fail on the mandatory checks.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CURATOR_CONFIG", "test.yaml")
	t.Setenv("OPENAI_MODEL", "gpt-4o-mini")
}

func TestLoadEnv_RedditProxyDefaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("REDDIT_PROXY_ENABLED", "")
	t.Setenv("REDDIT_PROXY_URL", "")

	env, err := LoadEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Reddit.ProxyEnabled {
		t.Fatalf("expected REDDIT_PROXY_ENABLED default to be false")
	}
	if env.Reddit.ProxyURL != "" {
		t.Fatalf("expected REDDIT_PROXY_URL default to be empty, got %q", env.Reddit.ProxyURL)
	}
}

func TestLoadEnv_RedditProxyConfigured(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("REDDIT_PROXY_ENABLED", "true")
	t.Setenv("REDDIT_PROXY_URL", "  http://user:pass@proxy.example.com:12321 \t")

	env, err := LoadEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !env.Reddit.ProxyEnabled {
		t.Fatalf("expected REDDIT_PROXY_ENABLED to be true")
	}
	if env.Reddit.ProxyURL != "http://user:pass@proxy.example.com:12321" {
		t.Fatalf("unexpected REDDIT_PROXY_URL: %q", env.Reddit.ProxyURL)
	}
}

func TestLoadEnv_OpenAIEnableThinkingDefault(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("OPENAI_ENABLE_THINKING", "")

	env, err := LoadEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !env.OpenAI.EnableThinking {
		t.Fatalf("expected OPENAI_ENABLE_THINKING default to be true")
	}
}

func TestLoadEnv_OpenAIEnableThinkingConfigured(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("OPENAI_ENABLE_THINKING", "false")

	env, err := LoadEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.OpenAI.EnableThinking {
		t.Fatalf("expected OPENAI_ENABLE_THINKING to be false")
	}
}
