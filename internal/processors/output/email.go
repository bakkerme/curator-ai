package output

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/outputs/email"
)

type EmailProcessor struct {
	name   string
	config config.EmailOutput
	sender email.Sender
}

func NewEmailProcessor(cfg *config.EmailOutput, sender email.Sender) (*EmailProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("email config is required")
	}
	return &EmailProcessor{
		name:   "email",
		config: *cfg,
		sender: sender,
	}, nil
}

func (p *EmailProcessor) Name() string {
	return p.name
}

func (p *EmailProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *EmailProcessor) Validate() error {
	if p.sender == nil {
		return fmt.Errorf("email sender is required")
	}
	if p.config.Template == "" || p.config.To == "" || p.config.Subject == "" {
		return fmt.Errorf("email template, to, from, subject are required")
	}
	return nil
}

func (p *EmailProcessor) Deliver(ctx context.Context, blocks []*core.PostBlock, runSummary *core.RunSummary) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("email processor validation failed: %w", err)
	}
	body, err := renderEmailTemplate(p.config.Template, blocks, runSummary)
	if err != nil {
		return fmt.Errorf("render email template failed: %w", err)
	}
	return p.sender.Send(ctx, email.Message{
		From:    p.config.From,
		To:      p.config.To,
		Subject: p.config.Subject,
		Body:    body,
	})
}

func renderEmailTemplate(templateText string, blocks []*core.PostBlock, runSummary *core.RunSummary) (string, error) {
	tmpl, err := template.New("email").Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("parse email template failed: %w", err)
	}
	var builder strings.Builder
	data := struct {
		Blocks     []*core.PostBlock
		RunSummary *core.RunSummary
	}{
		Blocks:     blocks,
		RunSummary: runSummary,
	}
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("execute email template failed: %w", err)
	}
	return builder.String(), nil
}
