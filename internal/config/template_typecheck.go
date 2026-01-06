package config

import (
	"bytes"
	"fmt"
	htmltmpl "html/template"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/core"
)

func (d *CuratorDocument) validateTemplateTypes() error {
	post := samplePostBlockForTemplateValidation()
	run := sampleRunSummaryForTemplateValidation()
	blocks := []*core.PostBlock{post}

	// Quality LLM templates.
	for i := range d.Workflow.Quality {
		q := d.Workflow.Quality[i].LLM
		if q == nil {
			continue
		}
		data := struct {
			*core.PostBlock
			Evaluations []string
			Exclusions  []string
		}{
			PostBlock:   post,
			Evaluations: q.Evaluations,
			Exclusions:  q.Exclusions,
		}
		if q.SystemTemplate != "" {
			if err := typeCheckTextTemplate(fmt.Sprintf("quality[%d].system_template", i), q.SystemTemplate, data); err != nil {
				return fmt.Errorf("quality %d (%s): system_template type check failed: %w", i, q.Name, err)
			}
		}
		if q.PromptTemplate != "" {
			if err := typeCheckTextTemplate(fmt.Sprintf("quality[%d].prompt_template", i), q.PromptTemplate, data); err != nil {
				return fmt.Errorf("quality %d (%s): prompt_template type check failed: %w", i, q.Name, err)
			}
		}
	}

	// Post summary LLM templates.
	for i := range d.Workflow.PostSummary {
		s := d.Workflow.PostSummary[i].LLM
		if s == nil {
			continue
		}
		data := struct {
			*core.PostBlock
			Params map[string]interface{}
		}{
			PostBlock: post,
			Params:    s.Params,
		}
		if s.SystemTemplate != "" {
			if err := typeCheckTextTemplate(fmt.Sprintf("post_summary[%d].system_template", i), s.SystemTemplate, data); err != nil {
				return fmt.Errorf("post_summary %d (%s): system_template type check failed: %w", i, s.Name, err)
			}
		}
		if s.PromptTemplate != "" {
			if err := typeCheckTextTemplate(fmt.Sprintf("post_summary[%d].prompt_template", i), s.PromptTemplate, data); err != nil {
				return fmt.Errorf("post_summary %d (%s): prompt_template type check failed: %w", i, s.Name, err)
			}
		}
	}

	// Run summary LLM templates.
	for i := range d.Workflow.RunSummary {
		s := d.Workflow.RunSummary[i].LLM
		if s == nil {
			continue
		}
		data := struct {
			Blocks []*core.PostBlock
			Params map[string]interface{}
		}{
			Blocks: blocks,
			Params: s.Params,
		}
		if s.SystemTemplate != "" {
			if err := typeCheckTextTemplate(fmt.Sprintf("run_summary[%d].system_template", i), s.SystemTemplate, data); err != nil {
				return fmt.Errorf("run_summary %d (%s): system_template type check failed: %w", i, s.Name, err)
			}
		}
		if s.PromptTemplate != "" {
			if err := typeCheckTextTemplate(fmt.Sprintf("run_summary[%d].prompt_template", i), s.PromptTemplate, data); err != nil {
				return fmt.Errorf("run_summary %d (%s): prompt_template type check failed: %w", i, s.Name, err)
			}
		}
	}

	// Output templates (currently only email).
	for i := range d.Workflow.Output {
		o := d.Workflow.Output[i].Email
		if o == nil {
			continue
		}
		if o.Template == "" {
			continue
		}
		data := struct {
			Blocks     []*core.PostBlock
			RunSummary *core.RunSummary
		}{
			Blocks:     blocks,
			RunSummary: run,
		}
		if err := typeCheckHTMLTemplate(fmt.Sprintf("output[%d].email.template", i), o.Template, data); err != nil {
			return fmt.Errorf("output %d (email): template type check failed: %w", i, err)
		}
	}

	return nil
}

func typeCheckTextTemplate(name, templateText string, data any) error {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(templateText)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	return tmpl.Execute(&buf, data)
}

func typeCheckHTMLTemplate(name, templateText string, data any) error {
	tmpl, err := htmltmpl.New(name).Option("missingkey=error").Parse(templateText)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	return tmpl.Execute(&buf, data)
}

func samplePostBlockForTemplateValidation() *core.PostBlock {
	now := time.Unix(0, 0).UTC()
	return &core.PostBlock{
		FlowID:    "flow",
		ID:        "post",
		URL:       "https://example.com/post",
		Title:     "Example post",
		Content:   "Example content",
		Author:    "example",
		CreatedAt: now,
		Comments: []core.CommentBlock{
			{ID: "comment", Author: "commenter", Content: "comment", CreatedAt: now},
		},
		WebBlocks: []core.WebBlock{
			{URL: "https://example.com"},
		},
		ImageBlocks: []core.ImageBlock{
			{URL: "https://example.com/image.jpg"},
		},
		Summary: &core.SummaryResult{
			ProcessorName: "summary",
			Summary:       "Example summary",
			HTML:          "<p>Example summary</p>",
			ProcessedAt:   now,
		},
		Quality: &core.QualityResult{
			ProcessorName: "quality",
			Result:        "pass",
			Score:         1,
			Reason:        "ok",
			ProcessedAt:   now,
		},
		ProcessedAt: now,
	}
}

func sampleRunSummaryForTemplateValidation() *core.RunSummary {
	now := time.Unix(0, 0).UTC()
	return &core.RunSummary{
		ProcessorName: "run_summary",
		Summary:       "Example run summary",
		HTML:          "<p>Example run summary</p>",
		PostCount:     1,
		ProcessedAt:   now,
	}
}

