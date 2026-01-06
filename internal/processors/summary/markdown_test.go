package summary

import (
	"context"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

func TestPostMarkdownProcessorSummarize(t *testing.T) {
	processor, err := NewPostMarkdownProcessor(&config.MarkdownSummary{
		Name:    "post_markdown",
		Type:    "markdown",
		Context: "post",
	})
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	blocks := []*core.PostBlock{
		{
			Title: "Test",
			Summary: &core.SummaryResult{
				Summary: "# Heading\n\nSome text.",
			},
		},
	}

	updated, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("summarize failed: %v", err)
	}
	if updated[0].Summary == nil {
		t.Fatal("expected summary to remain on block")
	}
	if !strings.Contains(updated[0].Summary.HTML, "<h1") || !strings.Contains(updated[0].Summary.HTML, "Heading</h1>") {
		t.Fatalf("expected HTML heading, got %q", updated[0].Summary.HTML)
	}
	if updated[0].Summary.ProcessorName != "post_markdown" {
		t.Fatalf("expected processor name to be updated, got %q", updated[0].Summary.ProcessorName)
	}
}

func TestRunMarkdownProcessorSummarizeRun(t *testing.T) {
	processor, err := NewRunMarkdownProcessor(&config.MarkdownSummary{
		Name:    "run_markdown",
		Type:    "markdown",
		Context: "flow",
	})
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	current := &core.RunSummary{
		Summary: "- First\n- Second",
	}

	updated, err := processor.SummarizeRun(context.Background(), nil, current)
	if err != nil {
		t.Fatalf("summarize run failed: %v", err)
	}
	if !strings.Contains(updated.HTML, "<ul>") {
		t.Fatalf("expected HTML list, got %q", updated.HTML)
	}
	if updated.ProcessorName != "run_markdown" {
		t.Fatalf("expected processor name to be updated, got %q", updated.ProcessorName)
	}
}

func TestMarkdownProcessorRendersGFMTable(t *testing.T) {
	processor, err := NewPostMarkdownProcessor(&config.MarkdownSummary{
		Name:    "post_markdown",
		Type:    "markdown",
		Context: "post",
	})
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	blocks := []*core.PostBlock{
		{
			Title: "Table",
			Summary: &core.SummaryResult{
				Summary: "| A | B |\n| --- | --- |\n| 1 | 2 |\n",
			},
		},
	}

	updated, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("summarize failed: %v", err)
	}
	if !strings.Contains(updated[0].Summary.HTML, "<table>") {
		t.Fatalf("expected HTML table, got %q", updated[0].Summary.HTML)
	}
}
