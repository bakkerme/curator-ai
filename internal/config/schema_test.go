package config

import (
	"os"
	"strings"
	"testing"

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