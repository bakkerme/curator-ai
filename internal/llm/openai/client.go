package openai

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type Client struct {
	client openai.Client
}

func NewClient(cfg config.OpenAIEnvConfig, opts ...option.RequestOption) *Client {
	options := []option.RequestOption{}
	if cfg.APIKey != "" {
		options = append(options, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		options = append(options, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.OTel.Enabled {
		options = append(options, option.WithMiddleware(openAIMiddleware(cfg.OTel)))
	}
	options = append(options, opts...)
	return &Client{client: openai.NewClient(options...)}
}

func (c *Client) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	tracer := otel.Tracer("curator-ai/llm/openai")
	ctx, span := tracer.Start(ctx, "llm.openai.chat.completions")
	attrs := []attribute.KeyValue{
		attribute.String("openinference.span.kind", "LLM"),
		attribute.String("llm.provider", "openai"),
		attribute.String("llm.model", request.Model),
		attribute.Int("llm.max_tokens", request.MaxTokens),
		attribute.Int("llm.input_messages", len(request.Messages)),
		attribute.String("flow.id", core.FlowIDFromContext(ctx)),
		attribute.String("run.id", core.RunIDFromContext(ctx)),
		attribute.String("session.id", core.RunIDFromContext(ctx)),
	}
	if request.Temperature != nil {
		attrs = append(attrs, attribute.Float64("llm.temperature", *request.Temperature))
	}
	span.SetAttributes(attrs...)
	defer span.End()

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(request.Messages))
	for _, msg := range request.Messages {
		if len(msg.Parts) > 0 {
			if msg.Role != llm.RoleUser {
				err := fmt.Errorf("openai: only user messages may include multipart content")
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return llm.ChatResponse{}, err
			}
			contentParts, err := openAIContentParts(msg.Parts)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return llm.ChatResponse{}, err
			}
			messages = append(messages, openai.UserMessage(contentParts))
			continue
		}
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
	if request.Temperature != nil {
		params.Temperature = openai.Float(*request.Temperature)
	}
	if request.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(request.MaxTokens))
	}

	response, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return llm.ChatResponse{}, err
	}
	if len(response.Choices) == 0 {
		err := fmt.Errorf("openai: empty response")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return llm.ChatResponse{}, err
	}

	span.SetStatus(codes.Ok, "")
	return llm.ChatResponse{Content: response.Choices[0].Message.Content}, nil
}

func openAIContentParts(parts []llm.MessagePart) ([]openai.ChatCompletionContentPartUnionParam, error) {
	contentParts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case llm.MessagePartText:
			if part.Text == "" {
				continue
			}
			contentParts = append(contentParts, openai.TextContentPart(part.Text))
		case llm.MessagePartImageURL:
			if part.ImageURL == "" {
				continue
			}
			contentParts = append(contentParts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: part.ImageURL,
			}))
		default:
			return nil, fmt.Errorf("openai: unsupported message part type %q", part.Type)
		}
	}
	return contentParts, nil
}
