package runner

import (
	"context"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/llm"
	llmmock "github.com/bakkerme/curator-ai/internal/llm/mock"
	emailmock "github.com/bakkerme/curator-ai/internal/outputs/email/mock"
	"github.com/bakkerme/curator-ai/internal/runner/factory"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
	rssmock "github.com/bakkerme/curator-ai/internal/sources/rss/mock"
)

func TestRunnerEndToEnd(t *testing.T) {
	llmMock := &llmmock.Client{
		Responses: []llm.ChatResponse{
			{Content: `{"score":0.9,"reason":"ok"}`},
			{Content: "Post summary"},
			{Content: "Run summary"},
		},
	}
	emailMock := &emailmock.Sender{}
	rssFetcher := &rssmock.Fetcher{
		ItemsByFeed: map[string][]rss.Item{
			"https://example.com/feed.xml": {
				{
					ID:          "1",
					Title:       "Test Title",
					Link:        "https://example.com/post",
					Description: "Summary",
					Content:     "Full content",
				},
			},
		},
	}

	factory := &factory.Factory{
		LLMClient:    llmMock,
		DefaultModel: "gpt-4o-mini",
		RSSFetcher:   rssFetcher,
		EmailSender:  emailMock,
	}

	doc := config.CuratorDocument{
		Workflow: config.Workflow{
			Name:    "Test Flow",
			Trigger: []config.TriggerConfig{{Cron: &config.CronTrigger{Schedule: "0 0 * * *"}}},
			Sources: []config.SourceConfig{{RSS: &config.RSSSource{Feeds: []string{"https://example.com/feed.xml"}}}},
			Quality: []config.QualityConfig{{LLM: &config.LLMQuality{
				Name:           "quality",
				PromptTemplate: `{"title":"{{.Title}}"}`,
				ActionType:     "pass_drop",
				Threshold:      0.5,
			}}},
			PostSummary: []config.SummaryConfig{{LLM: &config.LLMSummary{
				Name:           "post_summary",
				Type:           "llm",
				Context:        "post",
				PromptTemplate: "{{.Title}}",
			}}},
			RunSummary: []config.SummaryConfig{{LLM: &config.LLMSummary{
				Name:           "run_summary",
				Type:           "llm",
				Context:        "flow",
				PromptTemplate: "{{len .Blocks}} posts",
			}}},
			Output: map[string]any{"email": map[string]any{
				"template": "Posts: {{range .Blocks}}{{.Title}}{{end}}",
				"to":       "test@example.com",
				"from":     "noreply@example.com",
				"subject":  "Daily",
			}},
		},
	}

	flow, err := doc.ParseToFlowWithFactory(factory)
	if err != nil {
		t.Fatalf("failed to build flow: %v", err)
	}
	flow.ID = "flow-1"

	runner := New(nil)
	run, err := runner.RunOnce(context.Background(), flow)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if run.Status != "completed" {
		t.Fatalf("expected completed run, got %s", run.Status)
	}
	if len(emailMock.Messages) != 1 {
		t.Fatalf("expected one email message, got %d", len(emailMock.Messages))
	}
	if emailMock.Messages[0].Subject != "Daily" {
		t.Errorf("expected subject 'Daily', got %s", emailMock.Messages[0].Subject)
	}
	if run.RunSummary == nil || run.RunSummary.Summary != "Run summary" {
		t.Fatalf("expected run summary, got %#v", run.RunSummary)
	}
}
