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
	name     string
	config   config.EmailOutput
	sender   email.Sender
	template *template.Template
}

func NewEmailProcessor(cfg *config.EmailOutput, sender email.Sender) (*EmailProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("email config is required")
	}
	compiled, err := parseEmailTemplate(cfg.Template)
	if err != nil {
		return nil, err
	}
	return &EmailProcessor{
		name:     "email",
		config:   *cfg,
		sender:   sender,
		template: compiled,
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
	if p.template == nil {
		return fmt.Errorf("email template must be parsed")
	}
	return nil
}

func (p *EmailProcessor) Deliver(ctx context.Context, blocks []*core.PostBlock, runSummary *core.RunSummary) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("email processor validation failed: %w", err)
	}
	body, err := executeEmailTemplate(p.template, blocks, runSummary)
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
	tmpl, err := parseEmailTemplate(templateText)
	if err != nil {
		return "", err
	}
	return executeEmailTemplate(tmpl, blocks, runSummary)
}

// parseEmailTemplate compiles the email template with Curator's template helpers.
func parseEmailTemplate(templateText string) (*template.Template, error) {
	tmpl, err := template.New("email").Funcs(email.TemplateFuncs()).Parse(templateText)
	if err != nil {
		return nil, fmt.Errorf("parse email template failed: %w", err)
	}
	return tmpl, nil
}

// executeEmailTemplate renders a compiled email template against the current run data.
func executeEmailTemplate(tmpl *template.Template, blocks []*core.PostBlock, runSummary *core.RunSummary) (string, error) {
	var builder strings.Builder
	data := newEmailTemplateData(blocks, runSummary)
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("execute email template failed: %w", err)
	}
	return builder.String(), nil
}

type emailTemplateData struct {
	Blocks     []*emailPostBlock
	RunSummary *emailRunSummary
}

type emailPostBlock struct {
	*core.PostBlock
	Summary *emailSummaryResult
}

type emailSummaryResult struct {
	*core.SummaryResult
	HTML template.HTML
}

type emailRunSummary struct {
	*core.RunSummary
	HTML template.HTML
}

func newEmailTemplateData(blocks []*core.PostBlock, runSummary *core.RunSummary) emailTemplateData {
	emailBlocks := make([]*emailPostBlock, 0, len(blocks))
	for _, block := range blocks {
		var summary *emailSummaryResult
		if block != nil && block.Summary != nil {
			summary = &emailSummaryResult{
				SummaryResult: block.Summary,
				HTML:          template.HTML(block.Summary.HTML),
			}
		}
		emailBlocks = append(emailBlocks, &emailPostBlock{
			PostBlock: block,
			Summary:   summary,
		})
	}

	var emailRun *emailRunSummary
	if runSummary != nil {
		emailRun = &emailRunSummary{
			RunSummary: runSummary,
			HTML:       template.HTML(runSummary.HTML),
		}
	}

	return emailTemplateData{
		Blocks:     emailBlocks,
		RunSummary: emailRun,
	}
}
