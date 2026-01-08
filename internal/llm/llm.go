package llm

import "context"

type MessageRole string

const (
	RoleSystem MessageRole = "system"
	RoleUser   MessageRole = "user"
)

type MessagePartType string

const (
	MessagePartText     MessagePartType = "text"
	MessagePartImageURL MessagePartType = "image_url"
)

type MessagePart struct {
	Type     MessagePartType
	Text     string
	ImageURL string
}

type Message struct {
	Role    MessageRole
	Content string

	// Multi-part message content is only valid for user messages and
	// takes priority over Content in that case.
	Parts []MessagePart
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature *float64
	MaxTokens   int
}

type ChatResponse struct {
	Content string
}

type Client interface {
	ChatCompletion(ctx context.Context, request ChatRequest) (ChatResponse, error)
}
