package recording

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/bakkerme/curator-ai/internal/llm"
	"github.com/bakkerme/curator-ai/internal/llm/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordAndReplay(t *testing.T) {
	req := llm.ChatRequest{
		Model: "test-model",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helper."},
			{Role: llm.RoleUser, Content: "Hello"},
		},
	}
	resp := llm.ChatResponse{Content: "Hi there!"}

	inner := &mock.Client{Responses: []llm.ChatResponse{resp}}

	dir := t.TempDir()
	tapePath := filepath.Join(dir, "tape.json")

	// Record
	rec := NewRecordClient(inner, tapePath)
	got, err := rec.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, resp.Content, got.Content)

	err = rec.Close()
	require.NoError(t, err)

	// Verify tape was written
	_, err = os.Stat(tapePath)
	require.NoError(t, err)

	// Replay
	tape, err := LoadTape(tapePath)
	require.NoError(t, err)
	assert.Len(t, tape.Interactions, 1)

	replay := NewReplayClient(tape)
	got, err = replay.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, resp.Content, got.Content)
}

func TestReplayKeyMismatch(t *testing.T) {
	tape := &Tape{
		Interactions: []Interaction{
			{
				Key:      interactionKey(llm.ChatRequest{Model: "m", Messages: []llm.Message{{Role: llm.RoleUser, Content: "A"}}}),
				Request:  ChatRequestJSON{Model: "m"},
				Response: llm.ChatResponse{Content: "resp-A"},
			},
		},
	}

	replay := NewReplayClient(tape)

	// A different request should not match
	_, err := replay.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:    "m",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "B"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no tape interaction found")
}

func TestReplayExhausted(t *testing.T) {
	req := llm.ChatRequest{Model: "m", Messages: []llm.Message{{Role: llm.RoleUser, Content: "A"}}}
	tape := &Tape{
		Interactions: []Interaction{
			{
				Key:      interactionKey(req),
				Request:  ChatRequestJSON{Model: "m"},
				Response: llm.ChatResponse{Content: "resp"},
			},
		},
	}

	replay := NewReplayClient(tape)

	// First call succeeds
	_, err := replay.ChatCompletion(context.Background(), req)
	require.NoError(t, err)

	// Second call with same key should fail (exhausted)
	_, err = replay.ChatCompletion(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "have been consumed")
}

func TestReplayPreservesErrors(t *testing.T) {
	req := llm.ChatRequest{Model: "m", Messages: []llm.Message{{Role: llm.RoleUser, Content: "A"}}}
	tape := &Tape{
		Interactions: []Interaction{
			{
				Key:      interactionKey(req),
				Request:  ChatRequestJSON{Model: "m"},
				Response: llm.ChatResponse{},
				Error:    "upstream failure",
			},
		},
	}

	replay := NewReplayClient(tape)
	_, err := replay.ChatCompletion(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upstream failure")
}

func TestRecordPreservesErrors(t *testing.T) {
	req := llm.ChatRequest{Model: "m", Messages: []llm.Message{{Role: llm.RoleUser, Content: "A"}}}
	inner := &mock.Client{Err: fmt.Errorf("api error")}

	dir := t.TempDir()
	tapePath := filepath.Join(dir, "tape.json")

	rec := NewRecordClient(inner, tapePath)
	_, err := rec.ChatCompletion(context.Background(), req)
	assert.Error(t, err)

	err = rec.Close()
	require.NoError(t, err)

	tape, err := LoadTape(tapePath)
	require.NoError(t, err)
	assert.Len(t, tape.Interactions, 1)
	assert.Equal(t, "api error", tape.Interactions[0].Error)
}

func TestConcurrentReplay(t *testing.T) {
	// Create 20 distinct requests, each producing a unique key.
	const n = 20
	requests := make([]llm.ChatRequest, n)
	interactions := make([]Interaction, n)
	for i := 0; i < n; i++ {
		req := llm.ChatRequest{
			Model:    "m",
			Messages: []llm.Message{{Role: llm.RoleUser, Content: fmt.Sprintf("prompt-%d", i)}},
		}
		requests[i] = req
		interactions[i] = Interaction{
			Key:      interactionKey(req),
			Request:  chatRequestToJSON(req),
			Response: llm.ChatResponse{Content: fmt.Sprintf("response-%d", i)},
		}
	}

	tape := &Tape{Interactions: interactions}
	replay := NewReplayClient(tape)

	// Fire all requests concurrently.
	var wg sync.WaitGroup
	results := make([]llm.ChatResponse, n)
	errors := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx] = replay.ChatCompletion(context.Background(), requests[idx])
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errors[i], "request %d", i)
		assert.Equal(t, fmt.Sprintf("response-%d", i), results[i].Content, "request %d", i)
	}
}

func TestConcurrentRecord(t *testing.T) {
	const n = 20
	requests := make([]llm.ChatRequest, n)
	responses := make([]llm.ChatResponse, n)
	for i := 0; i < n; i++ {
		requests[i] = llm.ChatRequest{
			Model:    "m",
			Messages: []llm.Message{{Role: llm.RoleUser, Content: fmt.Sprintf("prompt-%d", i)}},
		}
		responses[i] = llm.ChatResponse{Content: fmt.Sprintf("response-%d", i)}
	}

	inner := &mock.Client{Responses: responses}

	dir := t.TempDir()
	tapePath := filepath.Join(dir, "tape.json")

	rec := NewRecordClient(inner, tapePath)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = rec.ChatCompletion(context.Background(), requests[idx])
		}(i)
	}
	wg.Wait()

	err := rec.Close()
	require.NoError(t, err)

	tape, err := LoadTape(tapePath)
	require.NoError(t, err)
	assert.Len(t, tape.Interactions, n)
}

func TestInteractionKeyDeterministic(t *testing.T) {
	req := llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "system prompt"},
			{Role: llm.RoleUser, Content: "user prompt"},
		},
	}

	key1 := interactionKey(req)
	key2 := interactionKey(req)
	assert.Equal(t, key1, key2)

	// Different content should produce a different key
	req2 := llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "system prompt"},
			{Role: llm.RoleUser, Content: "different prompt"},
		},
	}
	key3 := interactionKey(req2)
	assert.NotEqual(t, key1, key3)
}

func TestInteractionKeyWithParts(t *testing.T) {
	req := llm.ChatRequest{
		Model: "m",
		Messages: []llm.Message{
			{
				Role: llm.RoleUser,
				Parts: []llm.MessagePart{
					{Type: llm.MessagePartText, Text: "describe this"},
					{Type: llm.MessagePartImageURL, ImageURL: "https://example.com/img.png"},
				},
			},
		},
	}

	key1 := interactionKey(req)

	// Different image URL should produce different key
	req2 := llm.ChatRequest{
		Model: "m",
		Messages: []llm.Message{
			{
				Role: llm.RoleUser,
				Parts: []llm.MessagePart{
					{Type: llm.MessagePartText, Text: "describe this"},
					{Type: llm.MessagePartImageURL, ImageURL: "https://example.com/other.png"},
				},
			},
		},
	}

	key2 := interactionKey(req2)
	assert.NotEqual(t, key1, key2)
}

func TestReplayRetries(t *testing.T) {
	// Simulate two calls with identical request (e.g., retry after JSON parse failure).
	req := llm.ChatRequest{Model: "m", Messages: []llm.Message{{Role: llm.RoleUser, Content: "same"}}}
	key := interactionKey(req)

	tape := &Tape{
		Interactions: []Interaction{
			{Key: key, Request: chatRequestToJSON(req), Response: llm.ChatResponse{Content: "first"}},
			{Key: key, Request: chatRequestToJSON(req), Response: llm.ChatResponse{Content: "second"}},
		},
	}

	replay := NewReplayClient(tape)

	// First call returns "first"
	got1, err := replay.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "first", got1.Content)

	// Second call returns "second" (FIFO within same key)
	got2, err := replay.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "second", got2.Content)
}

func TestTapeRoundTrip(t *testing.T) {
	tape := NewTape()
	tape.Interactions = []Interaction{
		{
			Key: "abc123",
			Request: ChatRequestJSON{
				Model: "gpt-4o-mini",
				Messages: []MessageJSON{
					{Role: "system", Content: "sys"},
					{Role: "user", Content: "usr"},
				},
			},
			Response: llm.ChatResponse{Content: "response"},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "tape.json")

	err := tape.SaveTo(path)
	require.NoError(t, err)

	loaded, err := LoadTape(path)
	require.NoError(t, err)
	assert.Len(t, loaded.Interactions, 1)
	assert.Equal(t, "abc123", loaded.Interactions[0].Key)
	assert.Equal(t, "response", loaded.Interactions[0].Response.Content)
	assert.Equal(t, "gpt-4o-mini", loaded.Interactions[0].Request.Model)
}

func TestCloseNoOpInReplayMode(t *testing.T) {
	tape := &Tape{}
	replay := NewReplayClient(tape)
	err := replay.Close()
	assert.NoError(t, err)
}
