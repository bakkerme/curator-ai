package llm

import "context"

type MessageRole string

const (
	RoleSystem MessageRole = "system"
	RoleUser   MessageRole = "user"
)

type Message struct {
	Role    MessageRole
	Content string
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

type ChatResponse struct {
	Content string
}

type Client interface {
	ChatCompletion(ctx context.Context, request ChatRequest) (ChatResponse, error)
}
