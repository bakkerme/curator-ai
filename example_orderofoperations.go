package main

import (
	"fmt"
	"log"

	"github.com/bakkerme/curator-ai/internal/config"
	"gopkg.in/yaml.v3"
)

func main() {
	// Example YAML configuration
	yamlConfig := `
workflow:
  name: "Example Flow"
  version: "1.0"
  
  trigger:
    - cron:
        schedule: "0 0 * * *"
        
  sources:
    - reddit:
        subreddits: ["MachineLearning"]
        limit: 10
        
  quality:
    - quality_rule:
        name: min_score
        rule: "score > 5"
        actionType: pass_drop
        result: pass
    - llm:
        name: relevance_check
        prompt_template: relevance_template
        action_type: pass_drop
        
  post_summary:
    - llm:
        name: post_summarizer
        type: llm
        context: post
        prompt_template: summary_template
        
  run_summary:
    - llm:
        name: run_summarizer
        type: llm
        context: flow
        prompt_template: run_summary_template
        
  output:
    email:
      template: email_template
      to: user@example.com
      from: curator@example.com
      subject: "Daily AI Research Summary"
`

	// Parse the YAML into our config structure
	var doc config.CuratorDocument
	err := yaml.Unmarshal([]byte(yamlConfig), &doc)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// Convert to Flow with OrderOfOperations
	flow, err := doc.ParseToFlow()
	if err != nil {
		log.Fatalf("Failed to parse to Flow: %v", err)
	}

	// Display the OrderOfOperations
	fmt.Printf("Flow: %s (v%s)\n", flow.Name, flow.Version)
	fmt.Printf("Status: %s\n", flow.Status)
	fmt.Printf("\nOrder of Operations (%d steps):\n", len(flow.OrderOfOperations))
	fmt.Printf("%-3s %-20s %-25s\n", "#", "Name", "Type")
	fmt.Printf("%-3s %-20s %-25s\n", "---", "--------------------", "-------------------------")

	for i, op := range flow.OrderOfOperations {
		fmt.Printf("%-3d %-20s %-25s\n", i+1, op.Name, op.Type)
	}

	// Display processor counts by type
	fmt.Printf("\nProcessor Counts:\n")
	fmt.Printf("  Triggers: %d\n", len(flow.Triggers))
	fmt.Printf("  Sources: %d\n", len(flow.Sources))
	fmt.Printf("  Quality: %d\n", len(flow.Quality))
	fmt.Printf("  Post Summary: %d\n", len(flow.PostSummary))
	fmt.Printf("  Run Summary: %d\n", len(flow.RunSummary))
	fmt.Printf("  Outputs: %d\n", len(flow.Outputs))
}
