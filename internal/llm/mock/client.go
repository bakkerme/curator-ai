package mock

import (
	"context"
	"sync"

	"github.com/bakkerme/curator-ai/internal/llm"
)

type Client struct {
	mu        sync.Mutex
	Responses []llm.ChatResponse
	Err       error
	Calls     []llm.ChatRequest
}

func (c *Client) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, request)
	if c.Err != nil {
		return llm.ChatResponse{}, c.Err
	}
	if len(c.Responses) == 0 {
		return llm.ChatResponse{}, nil
	}
	response := c.Responses[0]
	if len(c.Responses) > 1 {
		c.Responses = c.Responses[1:]
	}
	return response, nil
}
