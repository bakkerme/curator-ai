package config

import (
	"context"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/core"
	"gopkg.in/yaml.v3"
)

func TestParseExampleFlow(t *testing.T) {
	data := []byte(`
workflow:
  name: "Test Flow"
  trigger:
    - cron:
        schedule: "0 0 * * *"
  sources:
    - reddit:
        subreddits: ["MachineLearning"]
  output:
    email:
      template: "Hello"
      to: "test@example.com"
      from: "noreply@example.com"
      subject: "Daily Report"
`)

	var doc CuratorDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("Document validation failed: %v", err)
	}
	flow, err := doc.Parse()
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	if flow.Name == "" {
		t.Error("Flow name should not be empty")
	}
	if len(flow.Triggers) == 0 {
		t.Error("Flow should have at least one trigger")
	}
	if len(flow.Sources) == 0 {
		t.Error("Flow should have at least one source")
	}
	if len(flow.Outputs) == 0 {
		t.Error("Flow should have at least one output")
	}
}

func TestValidation(t *testing.T) {
	testCases := []struct {
		name        string
		doc         CuratorDocument
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing workflow name",
			doc: CuratorDocument{
				Workflow: Workflow{
					Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "* * * * *"}}},
					Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
					Output: map[string]any{"email": map[string]any{
						"template": "test",
						"to":       "test@test.com",
						"from":     "noreply@test.com",
						"subject":  "Test Subject",
					}},
				},
			},
			expectError: true,
			errorMsg:    "workflow name is required",
		},
		{
			name: "Missing trigger",
			doc: CuratorDocument{
				Workflow: Workflow{
					Name:    "Test",
					Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
					Output: map[string]any{"email": map[string]any{
						"template": "test",
						"to":       "test@test.com",
						"from":     "noreply@test.com",
						"subject":  "Test Subject",
					}},
				},
			},
			expectError: true,
			errorMsg:    "at least one trigger is required",
		},
		{
			name: "Missing source",
			doc: CuratorDocument{
				Workflow: Workflow{
					Name:    "Test",
					Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "* * * * *"}}},
					Output: map[string]any{"email": map[string]any{
						"template": "test",
						"to":       "test@test.com",
						"from":     "noreply@test.com",
						"subject":  "Test Subject",
					}},
				},
			},
			expectError: true,
			errorMsg:    "at least one source is required",
		},
		{
			name: "Missing output",
			doc: CuratorDocument{
				Workflow: Workflow{
					Name:    "Test",
					Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "* * * * *"}}},
					Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
				},
			},
			expectError: true,
			errorMsg:    "output configuration is required",
		},
		{
			name: "RSS source missing feeds",
			doc: CuratorDocument{
				Workflow: Workflow{
					Name:    "Test",
					Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "* * * * *"}}},
					Sources: []SourceConfig{{RSS: &RSSSource{}}},
					Output: map[string]any{"email": map[string]any{
						"template": "test",
						"to":       "test@test.com",
						"from":     "noreply@test.com",
						"subject":  "Test Subject",
					}},
				},
			},
			expectError: true,
			errorMsg:    "at least one rss feed is required",
		},
		{
			name: "Invalid quality rule action type",
			doc: CuratorDocument{
				Workflow: Workflow{
					Name:    "Test",
					Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "* * * * *"}}},
					Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
					Quality: []QualityConfig{{
						QualityRule: &QualityRule{
							Name:       "test",
							Rule:       "score > 10",
							ActionType: "invalid",
							Result:     "drop",
						},
					}},
					Output: map[string]any{"email": map[string]any{
						"template": "test",
						"to":       "test@test.com",
						"from":     "noreply@test.com",
						"subject":  "Test Subject",
					}},
				},
			},
			expectError: true,
			errorMsg:    "actionType must be 'pass_drop'",
		},
		{
			name: "Valid minimal configuration",
			doc: CuratorDocument{
				Workflow: Workflow{
					Name:    "Test Workflow",
					Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "0 0 * * *"}}},
					Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
					Output: map[string]any{"email": map[string]any{
						"template": "test",
						"to":       "test@test.com",
						"from":     "noreply@test.com",
						"subject":  "Test Subject",
					}},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.doc.Validate()
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s' but got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestParseToFlow(t *testing.T) {
	// Create a test document with all processor types
	doc := CuratorDocument{
		Workflow: Workflow{
			Name:    "Test Workflow",
			Version: "1.0",
			Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "0 0 * * *"}}},
			Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
			Quality: []QualityConfig{
				{QualityRule: &QualityRule{
					Name:       "min_score",
					Rule:       "score > 10",
					ActionType: "pass_drop",
					Result:     "pass",
				}},
				{LLM: &LLMQuality{
					Name:           "quality_check",
					PromptTemplate: "test_template",
					ActionType:     "pass_drop",
				}},
			},
			PostSummary: []SummaryConfig{{LLM: &LLMSummary{
				Name:           "post_summary",
				Type:           "llm",
				Context:        "post",
				PromptTemplate: "summary_template",
			}}},
			RunSummary: []SummaryConfig{{LLM: &LLMSummary{
				Name:           "run_summary",
				Type:           "llm",
				Context:        "flow",
				PromptTemplate: "run_summary_template",
			}}},
			Output: map[string]any{"email": map[string]any{
				"template": "test",
				"to":       "test@test.com",
				"from":     "noreply@test.com",
				"subject":  "Test Subject",
			}},
		},
	}

	// Parse to Flow
	flow, err := doc.ParseToFlowWithFactory(&mockFactory{})
	if err != nil {
		t.Fatalf("Failed to parse to flow: %v", err)
	}

	// Validate basic properties
	if flow.Name != "Test Workflow" {
		t.Errorf("Expected flow name 'Test Workflow', got '%s'", flow.Name)
	}

	if flow.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", flow.Version)
	}

	if flow.Status != core.FlowStatusWaiting {
		t.Errorf("Expected status '%s', got '%s'", core.FlowStatusWaiting, flow.Status)
	}

	// Validate OrderOfOperations sequence
	expectedOrder := []core.ProcessorType{
		core.TriggerProcessorType,
		core.SourceProcessorType,
		core.QualityProcessorType,
		core.QualityProcessorType,
		core.SummaryProcessorType,
		core.RunSummaryProcessorType,
		core.OutputProcessorType,
	}

	if len(flow.OrderOfOperations) != len(expectedOrder) {
		t.Errorf("Expected %d operations, got %d", len(expectedOrder), len(flow.OrderOfOperations))
	}

	for i, expected := range expectedOrder {
		if i >= len(flow.OrderOfOperations) {
			t.Errorf("Missing operation at index %d", i)
			continue
		}
		if flow.OrderOfOperations[i].Type != expected {
			t.Errorf("Expected operation %d to be %s, got %s", i, expected, flow.OrderOfOperations[i].Type)
		}
	}

	// Validate specific processor names in OrderOfOperations
	expectedNames := []string{"cron", "reddit", "min_score", "quality_check", "post_summary", "run_summary", "email"}
	for i, expectedName := range expectedNames {
		if i >= len(flow.OrderOfOperations) {
			t.Errorf("Missing operation at index %d", i)
			continue
		}
		if flow.OrderOfOperations[i].Name != expectedName {
			t.Errorf("Expected operation %d to have name '%s', got '%s'", i, expectedName, flow.OrderOfOperations[i].Name)
		}
	}

	// Validate that typed processor lists are populated (even if with nil pointers for now)
	if len(flow.Triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(flow.Triggers))
	}

	if len(flow.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(flow.Sources))
	}

	if len(flow.Quality) != 2 {
		t.Errorf("Expected 2 quality processors, got %d", len(flow.Quality))
	}

	if len(flow.PostSummary) != 1 {
		t.Errorf("Expected 1 post summary processor, got %d", len(flow.PostSummary))
	}

	if len(flow.RunSummary) != 1 {
		t.Errorf("Expected 1 run summary processor, got %d", len(flow.RunSummary))
	}

	if len(flow.Outputs) != 1 {
		t.Errorf("Expected 1 output processor, got %d", len(flow.Outputs))
	}

	// Validate that RawConfig is populated
	if flow.RawConfig == nil {
		t.Error("RawConfig should not be nil")
	}

	t.Logf("Successfully parsed flow with %d operations in correct order", len(flow.OrderOfOperations))
}

func TestParseEmailOutputConfig(t *testing.T) {
	doc := CuratorDocument{
		Workflow: Workflow{
			Name:    "Email Test",
			Trigger: []TriggerConfig{{Cron: &CronTrigger{Schedule: "0 0 * * *"}}},
			Sources: []SourceConfig{{Reddit: &RedditSource{Subreddits: []string{"test"}}}},
			Output: map[string]any{"email": map[string]any{
				"template":      "test",
				"to":            "test@test.com",
				"from":          "noreply@test.com",
				"subject":       "Test Subject",
				"smtp_host":     "smtp.test.com",
				"smtp_port":     2525,
				"smtp_user":     "user",
				"smtp_password": "pass",
				"use_tls":       true,
			}},
		},
	}

	flow, err := doc.Parse()
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}
	if len(flow.Outputs) != 1 {
		t.Fatalf("Expected one output, got %d", len(flow.Outputs))
	}
	output := flow.Outputs[0]
	emailConfig, ok := output.Config.(*EmailOutput)
	if !ok {
		t.Fatalf("Expected email output config, got %T", output.Config)
	}
	if emailConfig.SMTPHost != "smtp.test.com" {
		t.Errorf("Expected smtp host, got %s", emailConfig.SMTPHost)
	}
	if emailConfig.SMTPPort != 2525 {
		t.Errorf("Expected smtp port 2525, got %d", emailConfig.SMTPPort)
	}
	if emailConfig.SMTPUser != "user" {
		t.Errorf("Expected smtp user, got %s", emailConfig.SMTPUser)
	}
	if emailConfig.SMTPPassword != "pass" {
		t.Errorf("Expected smtp password, got %s", emailConfig.SMTPPassword)
	}
	if emailConfig.UseTLS == nil || !*emailConfig.UseTLS {
		t.Errorf("Expected use_tls true")
	}
}

type mockFactory struct{}

func (m *mockFactory) NewCronTrigger(config *CronTrigger) (core.TriggerProcessor, error) {
	return &mockTrigger{}, nil
}

func (m *mockFactory) NewRedditSource(config *RedditSource) (core.SourceProcessor, error) {
	return &mockSource{}, nil
}

func (m *mockFactory) NewRSSSource(config *RSSSource) (core.SourceProcessor, error) {
	return &mockSource{}, nil
}

func (m *mockFactory) NewQualityRule(config *QualityRule) (core.QualityProcessor, error) {
	return &mockQuality{}, nil
}

func (m *mockFactory) NewLLMQuality(config *LLMQuality) (core.QualityProcessor, error) {
	return &mockQuality{}, nil
}

func (m *mockFactory) NewLLMSummary(config *LLMSummary) (core.SummaryProcessor, error) {
	return &mockSummary{}, nil
}

func (m *mockFactory) NewLLMRunSummary(config *LLMSummary) (core.RunSummaryProcessor, error) {
	return &mockRunSummary{}, nil
}

func (m *mockFactory) NewEmailOutput(config *EmailOutput) (core.OutputProcessor, error) {
	return &mockOutput{}, nil
}

type mockTrigger struct{}

func (m *mockTrigger) Name() string                                  { return "mock_trigger" }
func (m *mockTrigger) Configure(config map[string]interface{}) error { return nil }
func (m *mockTrigger) Validate() error                               { return nil }
func (m *mockTrigger) Start(ctx context.Context, flowID string) (<-chan core.TriggerEvent, error) {
	return make(chan core.TriggerEvent), nil
}
func (m *mockTrigger) Stop() error { return nil }

type mockSource struct{}

func (m *mockSource) Name() string                                         { return "mock_source" }
func (m *mockSource) Configure(config map[string]interface{}) error        { return nil }
func (m *mockSource) Validate() error                                      { return nil }
func (m *mockSource) Fetch(ctx context.Context) ([]*core.PostBlock, error) { return nil, nil }

type mockQuality struct{}

func (m *mockQuality) Name() string                                  { return "mock_quality" }
func (m *mockQuality) Configure(config map[string]interface{}) error { return nil }
func (m *mockQuality) Validate() error                               { return nil }
func (m *mockQuality) Evaluate(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	return blocks, nil
}

type mockSummary struct{}

func (m *mockSummary) Name() string                                  { return "mock_summary" }
func (m *mockSummary) Configure(config map[string]interface{}) error { return nil }
func (m *mockSummary) Validate() error                               { return nil }
func (m *mockSummary) Summarize(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	return blocks, nil
}

type mockRunSummary struct{}

func (m *mockRunSummary) Name() string                                  { return "mock_run_summary" }
func (m *mockRunSummary) Configure(config map[string]interface{}) error { return nil }
func (m *mockRunSummary) Validate() error                               { return nil }
func (m *mockRunSummary) SummarizeRun(ctx context.Context, blocks []*core.PostBlock) (*core.RunSummary, error) {
	return &core.RunSummary{}, nil
}

type mockOutput struct{}

func (m *mockOutput) Name() string                                  { return "mock_output" }
func (m *mockOutput) Configure(config map[string]interface{}) error { return nil }
func (m *mockOutput) Validate() error                               { return nil }
func (m *mockOutput) Deliver(ctx context.Context, blocks []*core.PostBlock, runSummary *core.RunSummary) error {
	return nil
}
