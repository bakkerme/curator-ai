package llmutil

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/bakkerme/curator-ai/internal/llm"
)

type ResponseDecoder func(content string) error

func ParseSystemAndPromptTemplates(name, systemTemplate, promptTemplate string) (*template.Template, *template.Template, error) {
	if name == "" {
		name = "llm"
	}
	systemTmpl, err := template.New(name).Parse(systemTemplate)
	if err != nil {
		return nil, nil, fmt.Errorf("parse system template: %w", err)
	}
	promptTmpl, err := template.New(name).Parse(promptTemplate)
	if err != nil {
		return nil, nil, fmt.Errorf("parse prompt template: %w", err)
	}
	return systemTmpl, promptTmpl, nil
}

func ExecuteTemplate(tmpl *template.Template, data any) (string, error) {
	builder := &strings.Builder{}
	if err := tmpl.Execute(builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func ModelOrDefault(model, defaultModel string) string {
	if model != "" {
		return model
	}
	return defaultModel
}

func ChatSystemUser(ctx context.Context, client llm.Client, model, systemPrompt, userPrompt string, temperature *float64) (llm.ChatResponse, error) {
	return client.ChatCompletion(ctx, llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: systemPrompt},
			{Role: llm.RoleUser, Content: userPrompt},
		},
		Temperature: temperature,
	})
}

func ChatUser(ctx context.Context, client llm.Client, model, userPrompt string, temperature *float64) (llm.ChatResponse, error) {
	return client.ChatCompletion(ctx, llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: userPrompt},
		},
		Temperature: temperature,
	})
}

func ChatCompletionWithRetries(
	ctx context.Context,
	client llm.Client,
	model string,
	messages []llm.Message,
	decodeRetries int,
	decode ResponseDecoder,
	temperature *float64,
) (llm.ChatResponse, error) {
	attempts := decodeRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastResp llm.ChatResponse
	var lastDecodeErr error
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
			Model:       model,
			Messages:    messages,
			Temperature: temperature,
		})
		if err != nil {
			return llm.ChatResponse{}, err
		}
		lastResp = resp
		if decode == nil {
			return resp, nil
		}
		if err := decode(resp.Content); err != nil {
			lastDecodeErr = err
			continue
		}
		return resp, nil
	}

	return lastResp, fmt.Errorf("decode response after %d attempt(s): %w; content=%q", attempts, lastDecodeErr, lastResp.Content)
}

// ChatSystemUserWithRetries retries the request when decode fails. The prompt is not modified between attempts.
// If decode is nil, this behaves like ChatSystemUser with a single attempt.
func ChatSystemUserWithRetries(
	ctx context.Context,
	client llm.Client,
	model, systemPrompt, userPrompt string,
	decodeRetries int,
	decode ResponseDecoder,
	temperature *float64,
) (llm.ChatResponse, error) {
	return ChatCompletionWithRetries(ctx, client, model, []llm.Message{
		{Role: llm.RoleSystem, Content: systemPrompt},
		{Role: llm.RoleUser, Content: userPrompt},
	}, decodeRetries, decode, temperature)
}

// ChatUserWithRetries retries the request when decode fails. The prompt is not modified between attempts.
// If decode is nil, this behaves like ChatUser with a single attempt.
func ChatUserWithRetries(
	ctx context.Context,
	client llm.Client,
	model, userPrompt string,
	decodeRetries int,
	decode ResponseDecoder,
	temperature *float64,
) (llm.ChatResponse, error) {
	return ChatCompletionWithRetries(ctx, client, model, []llm.Message{
		{Role: llm.RoleUser, Content: userPrompt},
	}, decodeRetries, decode, temperature)
}
