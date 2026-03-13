package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/llm"
)

func TestChatCompletion_SendsThinkingToggleFromEnvDefault(t *testing.T) {
	t.Parallel()

	requestBody := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		decoded := map[string]any{}
		_ = json.Unmarshal(body, &decoded)
		requestBody <- decoded
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","created":1,"model":"qwen","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	client := NewClient(config.OpenAIEnvConfig{BaseURL: server.URL, APIKey: "test", EnableThinking: false})
	_, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:    "Qwen/Qwen3.5-35B-A3B",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion returned error: %v", err)
	}

	select {
	case got := <-requestBody:
		chatTemplate, ok := got["chat_template_kwargs"].(map[string]any)
		if !ok {
			t.Fatalf("expected chat_template_kwargs object in request body")
		}
		if enabled, ok := chatTemplate["enable_thinking"].(bool); !ok || enabled {
			t.Fatalf("expected enable_thinking=false, got %#v", chatTemplate["enable_thinking"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for request body")
	}
}

func TestChatCompletion_RequestOverrideThinkingToggle(t *testing.T) {
	t.Parallel()

	var (
		mu   sync.Mutex
		body map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, _ := io.ReadAll(r.Body)
		decoded := map[string]any{}
		_ = json.Unmarshal(raw, &decoded)
		mu.Lock()
		body = decoded
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","created":1,"model":"qwen","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	client := NewClient(config.OpenAIEnvConfig{BaseURL: server.URL, APIKey: "test", EnableThinking: true})
	disableThinking := false
	_, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:          "Qwen/Qwen3.5-35B-A3B",
		Messages:       []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
		EnableThinking: &disableThinking,
	})
	if err != nil {
		t.Fatalf("ChatCompletion returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	chatTemplate, ok := body["chat_template_kwargs"].(map[string]any)
	if !ok {
		t.Fatalf("expected chat_template_kwargs object in request body")
	}
	if enabled, ok := chatTemplate["enable_thinking"].(bool); !ok || enabled {
		t.Fatalf("expected enable_thinking=false, got %#v", chatTemplate["enable_thinking"])
	}
}
