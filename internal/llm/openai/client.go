package openai

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/bakkerme/curator-ai/internal/llm"
)

type Client struct {
	client openai.Client
}

func NewClient(apiKey, baseURL string) *Client {
	options := []option.RequestOption{}
	if apiKey != "" {
		options = append(options, option.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}
	return &Client{client: openai.NewClient(options...)}
}

func (c *Client) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(request.Messages))
	for _, msg := range request.Messages {
		switch msg.Role {
		case llm.RoleSystem:
			messages = append(messages, openai.SystemMessage(msg.Content))
		default:
			messages = append(messages, openai.UserMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(request.Model),
		Messages: messages,
	}
	if request.Temperature > 0 {
		params.Temperature = openai.Float(request.Temperature)
	}
	if request.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(request.MaxTokens))
	}

	response, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return llm.ChatResponse{}, err
	}
	if len(response.Choices) == 0 {
		return llm.ChatResponse{}, fmt.Errorf("openai: empty response")
	}

	return llm.ChatResponse{Content: response.Choices[0].Message.Content}, nil
}
