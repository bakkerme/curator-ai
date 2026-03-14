package recording

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/bakkerme/curator-ai/internal/llm"
)

// Mode controls whether the client records or replays interactions.
type Mode string

const (
	ModeRecord Mode = "record"
	ModeReplay Mode = "replay"
)

// Client wraps an llm.Client and records or replays ChatCompletion calls.
// In record mode it proxies to the inner client and saves each interaction.
// In replay mode it returns previously recorded interactions matched by a
// stable key derived from the request content.
type Client struct {
	inner llm.Client // nil in replay mode
	mode  Mode
	tape  *Tape
	path  string // file path for saving the tape on Close

	mu    sync.Mutex
	index map[string][]int // key -> ordered indices into tape.Interactions
	used  map[int]bool     // tracks which interactions have been consumed
}

// NewRecordClient creates a recording client that proxies to inner and appends
// every interaction to a new tape. Call Close to write the tape to path.
func NewRecordClient(inner llm.Client, path string) *Client {
	return &Client{
		inner: inner,
		mode:  ModeRecord,
		tape:  NewTape(),
		path:  path,
		index: make(map[string][]int),
		used:  make(map[int]bool),
	}
}

// NewReplayClient creates a replay client that serves responses from the
// provided tape without making any real LLM calls.
func NewReplayClient(tape *Tape) *Client {
	c := &Client{
		mode:  ModeReplay,
		tape:  tape,
		index: make(map[string][]int),
		used:  make(map[int]bool),
	}
	c.buildIndex()
	return c
}

// buildIndex populates the key -> indices lookup from the tape.
func (c *Client) buildIndex() {
	for i, interaction := range c.tape.Interactions {
		c.index[interaction.Key] = append(c.index[interaction.Key], i)
	}
}

// ChatCompletion either records or replays a single LLM interaction.
func (c *Client) ChatCompletion(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	switch c.mode {
	case ModeRecord:
		return c.record(ctx, req)
	case ModeReplay:
		return c.replay(req)
	default:
		return llm.ChatResponse{}, fmt.Errorf("recording: unknown mode %q", c.mode)
	}
}

// record proxies to the inner client and stores the interaction.
func (c *Client) record(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	resp, err := c.inner.ChatCompletion(ctx, req)

	key := interactionKey(req)
	interaction := Interaction{
		Key:      key,
		Request:  chatRequestToJSON(req),
		Response: resp,
	}
	if err != nil {
		interaction.Error = err.Error()
	}

	c.mu.Lock()
	idx := len(c.tape.Interactions)
	c.tape.Interactions = append(c.tape.Interactions, interaction)
	c.index[key] = append(c.index[key], idx)
	c.mu.Unlock()

	return resp, err
}

// replay finds and returns the next unconsumed interaction matching the request key.
func (c *Client) replay(req llm.ChatRequest) (llm.ChatResponse, error) {
	key := interactionKey(req)

	c.mu.Lock()
	defer c.mu.Unlock()

	indices, ok := c.index[key]
	if !ok {
		return llm.ChatResponse{}, fmt.Errorf("recording: no tape interaction found for key %s", key)
	}

	for _, idx := range indices {
		if !c.used[idx] {
			c.used[idx] = true
			interaction := c.tape.Interactions[idx]
			if interaction.Error != "" {
				return interaction.Response, fmt.Errorf("%s", interaction.Error)
			}
			return interaction.Response, nil
		}
	}

	return llm.ChatResponse{}, fmt.Errorf("recording: all tape interactions for key %s have been consumed", key)
}

// Close writes the tape to disk in record mode. It is a no-op in replay mode.
func (c *Client) Close() error {
	if c.mode != ModeRecord {
		return nil
	}
	if c.path == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tape.SaveTo(c.path)
}

// interactionKey computes a deterministic SHA-256 hash from the request content.
// This produces a stable key regardless of goroutine scheduling order, making
// replay safe under concurrent access.
func interactionKey(req llm.ChatRequest) string {
	h := sha256.New()
	sep := []byte{0}
	h.Write([]byte(req.Model))
	h.Write(sep)
	for _, msg := range req.Messages {
		h.Write([]byte(msg.Role))
		h.Write(sep)
		h.Write([]byte(msg.Content))
		h.Write(sep)
		for _, part := range msg.Parts {
			h.Write([]byte(part.Type))
			h.Write(sep)
			h.Write([]byte(part.Text))
			h.Write(sep)
			h.Write([]byte(part.ImageURL))
			h.Write(sep)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
