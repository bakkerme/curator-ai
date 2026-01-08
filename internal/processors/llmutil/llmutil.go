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

// ChatCompletionWithRetries retries the request when decode fails. The request is not modified between attempts.
// If decode is nil, this behaves like a single ChatCompletion call.
func ChatCompletionWithRetries(
	ctx context.Context,
	client llm.Client,
	request llm.ChatRequest,
	decodeRetries int,
	decode ResponseDecoder,
) (llm.ChatResponse, error) {
	attempts := decodeRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastResp llm.ChatResponse
	var lastDecodeErr error
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := client.ChatCompletion(ctx, request)
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
