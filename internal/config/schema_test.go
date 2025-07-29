package config

import (
	"os"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/core"
	"gopkg.in/yaml.v3"
)

func TestParseExampleFlow(t *testing.T) {
	testCases := []struct {
		name     string
		filepath string
	}{
		{
			name:     "Original example flow",
			filepath: "../../planning/example_flow.yml",
		},
		{
			name:     "Complete example flow",
			filepath: "../../planning/example-flow-complete.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read the YAML file
			data, err := os.ReadFile(tc.filepath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", tc.filepath, err)
			}

			// Parse the YAML
			var doc CuratorDocument
			err = yaml.Unmarshal(data, &doc)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			// Validate the document
			err = doc.Validate()
			if err != nil {
				t.Fatalf("Document validation failed: %v", err)
			}

			// Parse into internal structure
			flow, err := doc.Parse()
			if err != nil {
				t.Fatalf("Failed to parse document: %v", err)
			}

			// Basic assertions
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

			// Log parsed structure for debugging
			t.Logf("Parsed flow: %s", flow.Name)
			t.Logf("Version: %s", flow.Version)
			t.Logf("Triggers: %d", len(flow.Triggers))
			t.Logf("Sources: %d", len(flow.Sources))
			t.Logf("Processors: %d", len(flow.Processors))
			t.Logf("Outputs: %d", len(flow.Outputs))
		})
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
					Output:  map[string]any{"email": map[string]any{"to": "test@test.com"}},
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
					Output:  map[string]any{"email": map[string]any{"to": "test@test.com"}},
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
					Output:  map[string]any{"email": map[string]any{"to": "test@test.com"}},
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
					Output: map[string]any{"email": map[string]any{"to": "test@test.com"}},
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
					Output:  map[string]any{"email": map[string]any{"to": "test@test.com"}},
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
				"to":      "test@test.com",
				"from":    "noreply@test.com",
				"subject": "Test Subject",
			}},
		},
	}

	// Parse to Flow
	flow, err := doc.ParseToFlow()
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
