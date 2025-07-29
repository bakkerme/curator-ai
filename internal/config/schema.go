package config

import (
	"fmt"
	"time"

	"github.com/bakkerme/curator-ai/internal/core"
	"gopkg.in/yaml.v3"
)

// CuratorDocument represents the top-level structure of a curator.yaml file
type CuratorDocument struct {
	Workflow Workflow `yaml:"workflow"`
}

// Workflow contains the complete workflow configuration
type Workflow struct {
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version,omitempty"`
	Trigger     []TriggerConfig `yaml:"trigger"`
	Sources     []SourceConfig  `yaml:"sources"`
	Quality     []QualityConfig `yaml:"quality,omitempty"`
	PostSummary []SummaryConfig `yaml:"post_summary,omitempty"`
	RunSummary  []SummaryConfig `yaml:"run_summary,omitempty"`
	Output      map[string]any  `yaml:"output"`
}

// TriggerConfig wraps different trigger types
type TriggerConfig struct {
	Cron *CronTrigger `yaml:"cron,omitempty"`
}

// CronTrigger defines a scheduled trigger
type CronTrigger struct {
	Schedule string `yaml:"schedule"`
	Timezone string `yaml:"timezone,omitempty"`
}

// SourceConfig wraps different source types
type SourceConfig struct {
	Reddit *RedditSource `yaml:"reddit,omitempty"`
}

// RedditSource defines Reddit data source configuration
type RedditSource struct {
	Subreddits      []string `yaml:"subreddits"`
	Limit           int      `yaml:"limit,omitempty"`
	Sort            string   `yaml:"sort,omitempty"`
	TimeFilter      string   `yaml:"time_filter,omitempty"`
	IncludeComments bool     `yaml:"include_comments,omitempty"`
	IncludeWeb      bool     `yaml:"include_web,omitempty"`
	IncludeImages   bool     `yaml:"include_images,omitempty"`
	MinScore        int      `yaml:"min_score,omitempty"`
}

// QualityConfig wraps different quality processor types
type QualityConfig struct {
	QualityRule *QualityRule `yaml:"quality_rule,omitempty"`
	LLM         *LLMQuality  `yaml:"llm,omitempty"`
}

// QualityRule defines rule-based quality filtering
type QualityRule struct {
	Name       string `yaml:"name"`
	Rule       string `yaml:"rule"`
	ActionType string `yaml:"actionType"`
	Result     string `yaml:"result"`
}

// LLMQuality defines AI-powered quality evaluation
type LLMQuality struct {
	Name           string   `yaml:"name"`
	Model          string   `yaml:"model,omitempty"`
	PromptTemplate string   `yaml:"prompt_template"`
	Evaluations    []string `yaml:"evaluations,omitempty"`
	Exclusions     []string `yaml:"exclusions,omitempty"`
	ActionType     string   `yaml:"action_type"`
	Threshold      float64  `yaml:"threshold,omitempty"`
}

// SummaryConfig wraps LLM summary processors
type SummaryConfig struct {
	LLM *LLMSummary `yaml:"llm,omitempty"`
}

// LLMSummary defines LLM-based summarization
type LLMSummary struct {
	Name           string                 `yaml:"name"`
	Type           string                 `yaml:"type"`
	Context        string                 `yaml:"context"`
	Model          string                 `yaml:"model,omitempty"`
	PromptTemplate string                 `yaml:"prompt_template"`
	Params         map[string]interface{} `yaml:"params,omitempty"`
}

// EmailOutput defines email delivery configuration
type EmailOutput struct {
	Template     string `yaml:"template"`
	To           string `yaml:"to"`
	From         string `yaml:"from"`
	Subject      string `yaml:"subject"`
	SMTPHost     string `yaml:"smtp_host,omitempty"`
	SMTPPort     int    `yaml:"smtp_port,omitempty"`
	SMTPUser     string `yaml:"smtp_user,omitempty"`
	SMTPPassword string `yaml:"smtp_password,omitempty"`
	UseTLS       *bool  `yaml:"use_tls,omitempty"`
}

// ProcessorType identifies the type of processor
type ProcessorType string

const (
	ProcessorTriggerCron   ProcessorType = "trigger_cron"
	ProcessorSourceReddit  ProcessorType = "source_reddit"
	ProcessorQualityRule   ProcessorType = "quality_rule"
	ProcessorQualityLLM    ProcessorType = "quality_llm"
	ProcessorSummaryLLM    ProcessorType = "summary_llm"
	ProcessorRunSummaryLLM ProcessorType = "run_summary_llm"
	ProcessorOutputEmail   ProcessorType = "output_email"
)

// ParsedFlow represents the internal structure after parsing
type ParsedFlow struct {
	Name       string
	Version    string
	Triggers   []ParsedProcessor
	Sources    []ParsedProcessor
	Processors []ParsedProcessor // Quality, Summary, RunSummary in order
	Outputs    []ParsedProcessor
}

// ParsedProcessor represents a configured processor instance
type ParsedProcessor struct {
	Type   ProcessorType
	Name   string
	Config interface{} // Points to the specific config struct
}

// Validate performs validation on the curator document
func (d *CuratorDocument) Validate() error {
	if d.Workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(d.Workflow.Trigger) == 0 {
		return fmt.Errorf("at least one trigger is required")
	}

	if len(d.Workflow.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}

	if len(d.Workflow.Output) == 0 {
		return fmt.Errorf("output configuration is required")
	}

	// Validate triggers
	for i, trigger := range d.Workflow.Trigger {
		if trigger.Cron == nil {
			return fmt.Errorf("trigger %d: unsupported trigger type", i)
		}
		if trigger.Cron.Schedule == "" {
			return fmt.Errorf("trigger %d: cron schedule is required", i)
		}
	}

	// Validate sources
	for i, source := range d.Workflow.Sources {
		if source.Reddit == nil {
			return fmt.Errorf("source %d: unsupported source type", i)
		}
		if len(source.Reddit.Subreddits) == 0 {
			return fmt.Errorf("source %d: at least one subreddit is required", i)
		}
	}

	// Validate quality processors
	for i, quality := range d.Workflow.Quality {
		if quality.QualityRule == nil && quality.LLM == nil {
			return fmt.Errorf("quality %d: unsupported quality type", i)
		}

		if quality.QualityRule != nil {
			if quality.QualityRule.Name == "" || quality.QualityRule.Rule == "" {
				return fmt.Errorf("quality %d: rule name and expression are required", i)
			}
			if quality.QualityRule.ActionType != "pass_drop" {
				return fmt.Errorf("quality %d: actionType must be 'pass_drop'", i)
			}
			if quality.QualityRule.Result != "pass" && quality.QualityRule.Result != "drop" {
				return fmt.Errorf("quality %d: result must be 'pass' or 'drop'", i)
			}
		}

		if quality.LLM != nil {
			if quality.LLM.Name == "" || quality.LLM.PromptTemplate == "" {
				return fmt.Errorf("quality %d: LLM name and prompt_template are required", i)
			}
		}
	}

	// Validate summaries
	for i, summary := range d.Workflow.PostSummary {
		if summary.LLM == nil {
			return fmt.Errorf("post_summary %d: unsupported summary type", i)
		}
		if summary.LLM.Context != "post" {
			return fmt.Errorf("post_summary %d: context must be 'post'", i)
		}
	}

	for i, summary := range d.Workflow.RunSummary {
		if summary.LLM == nil {
			return fmt.Errorf("run_summary %d: unsupported summary type", i)
		}
		if summary.LLM.Context != "flow" {
			return fmt.Errorf("run_summary %d: context must be 'flow'", i)
		}
	}

	return nil
}

// Parse converts the document into an internal ParsedFlow structure
func (d *CuratorDocument) Parse() (*ParsedFlow, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	flow := &ParsedFlow{
		Name:    d.Workflow.Name,
		Version: d.Workflow.Version,
	}

	if flow.Version == "" {
		flow.Version = "1.0"
	}

	// Parse triggers
	for _, trigger := range d.Workflow.Trigger {
		if trigger.Cron != nil {
			flow.Triggers = append(flow.Triggers, ParsedProcessor{
				Type:   ProcessorTriggerCron,
				Name:   "cron",
				Config: trigger.Cron,
			})
		}
	}

	// Parse sources
	for _, source := range d.Workflow.Sources {
		if source.Reddit != nil {
			flow.Sources = append(flow.Sources, ParsedProcessor{
				Type:   ProcessorSourceReddit,
				Name:   "reddit",
				Config: source.Reddit,
			})
		}
	}

	// Parse quality processors
	for _, quality := range d.Workflow.Quality {
		if quality.QualityRule != nil {
			flow.Processors = append(flow.Processors, ParsedProcessor{
				Type:   ProcessorQualityRule,
				Name:   quality.QualityRule.Name,
				Config: quality.QualityRule,
			})
		} else if quality.LLM != nil {
			flow.Processors = append(flow.Processors, ParsedProcessor{
				Type:   ProcessorQualityLLM,
				Name:   quality.LLM.Name,
				Config: quality.LLM,
			})
		}
	}

	// Parse post summaries
	for _, summary := range d.Workflow.PostSummary {
		if summary.LLM != nil {
			flow.Processors = append(flow.Processors, ParsedProcessor{
				Type:   ProcessorSummaryLLM,
				Name:   summary.LLM.Name,
				Config: summary.LLM,
			})
		}
	}

	// Parse run summaries
	for _, summary := range d.Workflow.RunSummary {
		if summary.LLM != nil {
			flow.Processors = append(flow.Processors, ParsedProcessor{
				Type:   ProcessorRunSummaryLLM,
				Name:   summary.LLM.Name,
				Config: summary.LLM,
			})
		}
	}

	// Parse outputs
	for outputType, outputConfig := range d.Workflow.Output {
		if outputType == "email" {
			// Convert map to EmailOutput struct
			emailConfig := &EmailOutput{}
			if configMap, ok := outputConfig.(map[string]interface{}); ok {
				// Manual mapping since we're dealing with interface{}
				if v, ok := configMap["template"].(string); ok {
					emailConfig.Template = v
				}
				if v, ok := configMap["to"].(string); ok {
					emailConfig.To = v
				}
				if v, ok := configMap["from"].(string); ok {
					emailConfig.From = v
				}
				if v, ok := configMap["subject"].(string); ok {
					emailConfig.Subject = v
				}
			}

			flow.Outputs = append(flow.Outputs, ParsedProcessor{
				Type:   ProcessorOutputEmail,
				Name:   "email",
				Config: emailConfig,
			})
		}
	}

	return flow, nil
}

// ParseToFlow converts the document into a core.Flow structure with OrderOfOperations
func (d *CuratorDocument) ParseToFlow() (*core.Flow, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	flow := &core.Flow{
		ID:        "", // Should be set by the caller
		Name:      d.Workflow.Name,
		Version:   d.Workflow.Version,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    core.FlowStatusWaiting,
		RawConfig: make(map[string]interface{}),
	}

	if flow.Version == "" {
		flow.Version = "1.0"
	}

	// Store processor configurations for OrderOfOperations lookup
	triggerMap := make(map[string]*core.TriggerProcessor)
	sourceMap := make(map[string]*core.SourceProcessor)
	qualityMap := make(map[string]*core.QualityProcessor)
	postSummaryMap := make(map[string]*core.SummaryProcessor)
	runSummaryMap := make(map[string]*core.RunSummaryProcessor)
	outputMap := make(map[string]*core.OutputProcessor)

	// Build OrderOfOperations in the correct sequence
	// 1. Triggers (always first)
	for _, trigger := range d.Workflow.Trigger {
		if trigger.Cron != nil {
			// Note: In a real implementation, these would be created by a processor factory
			// For now, we're just creating placeholder instances
			var triggerProcessor *core.TriggerProcessor
			triggerMap["cron"] = triggerProcessor
			flow.Triggers = append(flow.Triggers, triggerProcessor)

			processRef := core.ProcessReference{
				Name:    "cron",
				Type:    core.TriggerProcessorType,
				Trigger: triggerProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 2. Sources
	for _, source := range d.Workflow.Sources {
		if source.Reddit != nil {
			var sourceProcessor *core.SourceProcessor
			sourceMap["reddit"] = sourceProcessor
			flow.Sources = append(flow.Sources, sourceProcessor)

			processRef := core.ProcessReference{
				Name:   "reddit",
				Type:   core.SourceProcessorType,
				Source: sourceProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 3. Quality processors (in order defined in document)
	for _, quality := range d.Workflow.Quality {
		if quality.QualityRule != nil {
			var qualityProcessor *core.QualityProcessor
			qualityMap[quality.QualityRule.Name] = qualityProcessor
			flow.Quality = append(flow.Quality, qualityProcessor)

			processRef := core.ProcessReference{
				Name:    quality.QualityRule.Name,
				Type:    core.QualityProcessorType,
				Quality: qualityProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		} else if quality.LLM != nil {
			var qualityProcessor *core.QualityProcessor
			qualityMap[quality.LLM.Name] = qualityProcessor
			flow.Quality = append(flow.Quality, qualityProcessor)

			processRef := core.ProcessReference{
				Name:    quality.LLM.Name,
				Type:    core.QualityProcessorType,
				Quality: qualityProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 4. Post Summary processors
	for _, summary := range d.Workflow.PostSummary {
		if summary.LLM != nil {
			var summaryProcessor *core.SummaryProcessor
			postSummaryMap[summary.LLM.Name] = summaryProcessor
			flow.PostSummary = append(flow.PostSummary, summaryProcessor)

			processRef := core.ProcessReference{
				Name:        summary.LLM.Name,
				Type:        core.SummaryProcessorType,
				PostSummary: summaryProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 5. Run Summary processors
	for _, summary := range d.Workflow.RunSummary {
		if summary.LLM != nil {
			var runSummaryProcessor *core.RunSummaryProcessor
			runSummaryMap[summary.LLM.Name] = runSummaryProcessor
			flow.RunSummary = append(flow.RunSummary, runSummaryProcessor)

			processRef := core.ProcessReference{
				Name:       summary.LLM.Name,
				Type:       core.RunSummaryProcessorType,
				RunSummary: runSummaryProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 6. Output processors (always last)
	for outputType := range d.Workflow.Output {
		if outputType == "email" {
			var outputProcessor *core.OutputProcessor
			outputMap["email"] = outputProcessor
			flow.Outputs = append(flow.Outputs, outputProcessor)

			processRef := core.ProcessReference{
				Name:   "email",
				Type:   core.OutputProcessorType,
				Output: outputProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// Store raw configuration for reference
	configBytes, _ := yaml.Marshal(d)
	var rawConfig map[string]interface{}
	yaml.Unmarshal(configBytes, &rawConfig)
	flow.RawConfig = rawConfig

	return flow, nil
}
