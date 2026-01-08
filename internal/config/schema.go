package config

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/bakkerme/curator-ai/internal/core"
	"gopkg.in/yaml.v3"
)

// CuratorDocument represents the top-level structure of a curator.yaml file
type CuratorDocument struct {
	Workflow  Workflow             `yaml:"workflow"`
	Templates []TemplateDefinition `yaml:"templates,omitempty"`
}

// TemplateDefinition stores a named template that can be referenced by ID from processors.
// Templates use Go's standard library template syntax (text/template; email output uses html/template with the same syntax).
type TemplateDefinition struct {
	ID             string `yaml:"id"`
	SystemTemplate string `yaml:"system_template"`
	Template       string `yaml:"template"`
}

// Workflow contains the complete workflow configuration
type Workflow struct {
	Name           string          `yaml:"name"`
	Version        string          `yaml:"version,omitempty"`
	MaxConcurrency int             `yaml:"max_concurrency,omitempty"`
	Trigger        []TriggerConfig `yaml:"trigger"`
	Sources        []SourceConfig  `yaml:"sources"`
	Quality        []QualityConfig `yaml:"quality,omitempty"`
	PostSummary    []SummaryConfig `yaml:"post_summary,omitempty"`
	RunSummary     []SummaryConfig `yaml:"run_summary,omitempty"`
	Output         []OutputConfig  `yaml:"output"`
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
	RSS    *RSSSource    `yaml:"rss,omitempty"`
}

// RedditSource defines Reddit data source configuration
type RedditSource struct {
	Subreddits      []string             `yaml:"subreddits"`
	Limit           int                  `yaml:"limit,omitempty"`
	Sort            string               `yaml:"sort,omitempty"`
	TimeFilter      string               `yaml:"time_filter,omitempty"`
	IncludeComments bool                 `yaml:"include_comments,omitempty"`
	IncludeWeb      bool                 `yaml:"include_web,omitempty"`
	IncludeImages   bool                 `yaml:"include_images,omitempty"`
	MinScore        int                  `yaml:"min_score,omitempty"`
	Snapshot        *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// RSSSource defines RSS/Atom feed configuration
type RSSSource struct {
	Feeds          []string             `yaml:"feeds"`
	Limit          int                  `yaml:"limit,omitempty"`
	IncludeContent *bool                `yaml:"include_content,omitempty"`
	UserAgent      string               `yaml:"user_agent,omitempty"`
	Snapshot       *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// QualityConfig wraps different quality processor types
type QualityConfig struct {
	QualityRule *QualityRule `yaml:"quality_rule,omitempty"`
	LLM         *LLMQuality  `yaml:"llm,omitempty"`
}

// QualityRule defines rule-based quality filtering
type QualityRule struct {
	Name       string               `yaml:"name"`
	Rule       string               `yaml:"rule"`
	ActionType string               `yaml:"actionType"`
	Result     string               `yaml:"result"`
	Snapshot   *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// LLMQuality defines AI-powered quality evaluation
type LLMQuality struct {
	Name            string   `yaml:"name"`
	Model           string   `yaml:"model,omitempty"`
	Temperature     *float64 `yaml:"temperature,omitempty"`
	TopP            *float64 `yaml:"top_p,omitempty"`
	PresencePenalty *float64 `yaml:"presence_penalty,omitempty"`
	TopK            *int     `yaml:"top_k,omitempty"`
	SystemTemplate  string   `yaml:"system_template"`
	PromptTemplate  string   `yaml:"prompt_template"`
	Evaluations     []string `yaml:"evaluations,omitempty"`
	Exclusions      []string `yaml:"exclusions,omitempty"`
	ActionType      string   `yaml:"action_type"`
	Threshold       float64  `yaml:"threshold,omitempty"`
	MaxConcurrency  int      `yaml:"max_concurrency,omitempty"`
	// InvalidJSONRetries retries the LLM call when the response can't be parsed as JSON.
	InvalidJSONRetries int                  `yaml:"invalid_json_retries,omitempty"`
	Snapshot           *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// SummaryConfig wraps LLM summary processors
type SummaryConfig struct {
	LLM      *LLMSummary      `yaml:"llm,omitempty"`
	Markdown *MarkdownSummary `yaml:"markdown,omitempty"`
}

// LLMSummary defines LLM-based summarization
type LLMSummary struct {
	Name            string                 `yaml:"name"`
	Type            string                 `yaml:"type"`
	Context         string                 `yaml:"context"`
	Model           string                 `yaml:"model,omitempty"`
	Temperature     *float64               `yaml:"temperature,omitempty"`
	TopP            *float64               `yaml:"top_p,omitempty"`
	PresencePenalty *float64               `yaml:"presence_penalty,omitempty"`
	TopK            *int                   `yaml:"top_k,omitempty"`
	SystemTemplate  string                 `yaml:"system_template"`
	PromptTemplate  string                 `yaml:"prompt_template"`
	Params          map[string]interface{} `yaml:"params,omitempty"`
	MaxConcurrency  int                    `yaml:"max_concurrency,omitempty"`
	Snapshot        *core.SnapshotConfig   `yaml:"snapshot,omitempty"`
}

// MarkdownSummary defines markdown-to-HTML summarization
type MarkdownSummary struct {
	Name     string               `yaml:"name"`
	Type     string               `yaml:"type"`
	Context  string               `yaml:"context"`
	Snapshot *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

type OutputConfig struct {
	Email *EmailOutput `yaml:"email,omitempty"`
}

// EmailOutput defines email delivery configuration
type EmailOutput struct {
	Template     string               `yaml:"template"`
	To           string               `yaml:"to"`
	From         string               `yaml:"from"`
	Subject      string               `yaml:"subject"`
	SMTPHost     string               `yaml:"smtp_host,omitempty"`
	SMTPPort     int                  `yaml:"smtp_port,omitempty"`
	SMTPUser     string               `yaml:"smtp_user,omitempty"`
	SMTPPassword string               `yaml:"smtp_password,omitempty"`
	UseTLS       *bool                `yaml:"use_tls,omitempty"`
	Snapshot     *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// ProcessorType identifies the type of processor
type ProcessorType string

const (
	ProcessorTriggerCron   ProcessorType = "trigger_cron"
	ProcessorSourceReddit  ProcessorType = "source_reddit"
	ProcessorSourceRSS     ProcessorType = "source_rss"
	ProcessorQualityRule   ProcessorType = "quality_rule"
	ProcessorQualityLLM    ProcessorType = "quality_llm"
	ProcessorSummaryLLM    ProcessorType = "summary_llm"
	ProcessorRunSummaryLLM ProcessorType = "run_summary_llm"
	ProcessorSummaryMD     ProcessorType = "summary_markdown"
	ProcessorRunSummaryMD  ProcessorType = "run_summary_markdown"
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

// ProcessorFactory constructs concrete processor implementations for a parsed document.
type ProcessorFactory interface {
	NewCronTrigger(config *CronTrigger) (core.TriggerProcessor, error)
	NewRedditSource(config *RedditSource) (core.SourceProcessor, error)
	NewRSSSource(config *RSSSource) (core.SourceProcessor, error)
	NewQualityRule(config *QualityRule) (core.QualityProcessor, error)
	NewLLMQuality(config *LLMQuality) (core.QualityProcessor, error)
	NewLLMSummary(config *LLMSummary) (core.SummaryProcessor, error)
	NewLLMRunSummary(config *LLMSummary) (core.RunSummaryProcessor, error)
	NewMarkdownSummary(config *MarkdownSummary) (core.SummaryProcessor, error)
	NewMarkdownRunSummary(config *MarkdownSummary) (core.RunSummaryProcessor, error)
	NewEmailOutput(config *EmailOutput) (core.OutputProcessor, error)
}

// Validate performs validation on the curator document
func (d *CuratorDocument) Validate() error {
	if err := d.resolveTemplateReferences(); err != nil {
		return err
	}
	if err := d.validateTemplateTypes(); err != nil {
		return err
	}

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

	for _, output := range d.Workflow.Output {
		emailConfig, err := decodeEmailOutput(output.Email)
		if err != nil {
			return fmt.Errorf("output email: %w", err)
		}
		requiredFields := map[string]string{
			"template": emailConfig.Template,
			"to":       emailConfig.To,
			"subject":  emailConfig.Subject,
		}
		for field, value := range requiredFields {
			if value == "" {
				return fmt.Errorf("output email: '%s' field is required. Provided %+v", field, emailConfig)
			}
		}
		if _, err := mail.ParseAddress(emailConfig.To); err != nil {
			return fmt.Errorf("output email: invalid to address")
		}

		if emailConfig.From != "" { // From is optional, but if provided must be valid
			if _, err := mail.ParseAddress(emailConfig.From); err != nil {
				return fmt.Errorf("output email: invalid from address")
			}
		}
		if err := validateSnapshotConfig("output email", emailConfig.Snapshot); err != nil {
			return err
		}
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
		if source.Reddit == nil && source.RSS == nil {
			return fmt.Errorf("source %d: unsupported source type", i)
		}
		if source.Reddit != nil && len(source.Reddit.Subreddits) == 0 {
			return fmt.Errorf("source %d: at least one subreddit is required", i)
		}
		if source.RSS != nil && len(source.RSS.Feeds) == 0 {
			return fmt.Errorf("source %d: at least one rss feed is required", i)
		}
		if source.Reddit != nil {
			if err := validateSnapshotConfig(fmt.Sprintf("source %d reddit", i), source.Reddit.Snapshot); err != nil {
				return err
			}
		}
		if source.RSS != nil {
			if err := validateSnapshotConfig(fmt.Sprintf("source %d rss", i), source.RSS.Snapshot); err != nil {
				return err
			}
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
			if err := validateSnapshotConfig(fmt.Sprintf("quality %d rule", i), quality.QualityRule.Snapshot); err != nil {
				return err
			}
		}

		if quality.LLM != nil {
			if quality.LLM.Name == "" || quality.LLM.PromptTemplate == "" {
				return fmt.Errorf("quality %d: LLM name and prompt_template are required", i)
			}
			if err := validateLLMParams(fmt.Sprintf("quality %d llm", i), quality.LLM.Temperature, quality.LLM.TopP, quality.LLM.PresencePenalty, quality.LLM.TopK); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("quality %d llm", i), quality.LLM.Snapshot); err != nil {
				return err
			}
		}
	}

	// Validate summaries
	for i, summary := range d.Workflow.PostSummary {
		if summary.LLM == nil && summary.Markdown == nil {
			return fmt.Errorf("post_summary %d: unsupported summary type", i)
		}
		if summary.LLM != nil && summary.LLM.Context != "post" {
			return fmt.Errorf("post_summary %d: context must be 'post'", i)
		}
		if summary.Markdown != nil && summary.Markdown.Context != "post" {
			return fmt.Errorf("post_summary %d: context must be 'post'", i)
		}
		if summary.LLM != nil {
			if err := validateLLMParams(fmt.Sprintf("post_summary %d llm", i), summary.LLM.Temperature, summary.LLM.TopP, summary.LLM.PresencePenalty, summary.LLM.TopK); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("post_summary %d llm", i), summary.LLM.Snapshot); err != nil {
				return err
			}
		}
		if summary.Markdown != nil {
			if err := validateSnapshotConfig(fmt.Sprintf("post_summary %d markdown", i), summary.Markdown.Snapshot); err != nil {
				return err
			}
		}
	}

	for i, summary := range d.Workflow.RunSummary {
		if summary.LLM == nil && summary.Markdown == nil {
			return fmt.Errorf("run_summary %d: unsupported summary type", i)
		}
		if summary.LLM != nil && summary.LLM.Context != "flow" {
			return fmt.Errorf("run_summary %d: context must be 'flow'", i)
		}
		if summary.Markdown != nil && summary.Markdown.Context != "flow" {
			return fmt.Errorf("run_summary %d: context must be 'flow'", i)
		}
		if summary.LLM != nil {
			if err := validateLLMParams(fmt.Sprintf("run_summary %d llm", i), summary.LLM.Temperature, summary.LLM.TopP, summary.LLM.PresencePenalty, summary.LLM.TopK); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("run_summary %d llm", i), summary.LLM.Snapshot); err != nil {
				return err
			}
		}
		if summary.Markdown != nil {
			if err := validateSnapshotConfig(fmt.Sprintf("run_summary %d markdown", i), summary.Markdown.Snapshot); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateSnapshotConfig(label string, cfg *core.SnapshotConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Snapshot && cfg.Restore {
		return fmt.Errorf("%s: snapshot and restore cannot both be true", label)
	}
	if (cfg.Snapshot || cfg.Restore) && cfg.Path == "" {
		return fmt.Errorf("%s: snapshot path is required", label)
	}
	return nil
}

func validateLLMParams(label string, temperature, topP, presencePenalty *float64, topK *int) error {
	if temperature != nil {
		if *temperature < 0 || *temperature > 2 {
			return fmt.Errorf("%s: temperature must be between 0 and 2", label)
		}
	}
	if topP != nil {
		if *topP < 0 || *topP > 1 {
			return fmt.Errorf("%s: top_p must be between 0 and 1", label)
		}
	}
	if presencePenalty != nil {
		if *presencePenalty < -2 || *presencePenalty > 2 {
			return fmt.Errorf("%s: presence_penalty must be between -2 and 2", label)
		}
	}
	if topK != nil {
		if *topK < 0 {
			return fmt.Errorf("%s: top_k must be >= 0", label)
		}
	}
	return nil
}

func (d *CuratorDocument) resolveTemplateReferences() error {
	if len(d.Templates) == 0 {
		return nil
	}

	byID := make(map[string]TemplateDefinition, len(d.Templates))
	for i, t := range d.Templates {
		if t.ID == "" {
			return fmt.Errorf("templates %d: id is required", i)
		}
		if _, exists := byID[t.ID]; exists {
			return fmt.Errorf("templates: duplicate id %q", t.ID)
		}
		if t.SystemTemplate == "" {
			return fmt.Errorf("templates %q: system_template is required", t.ID)
		}
		if t.Template == "" {
			return fmt.Errorf("templates %q: template is required", t.ID)
		}
		byID[t.ID] = t
	}

	// Quality LLM
	for i := range d.Workflow.Quality {
		q := d.Workflow.Quality[i].LLM
		if q == nil {
			continue
		}
		if resolved, ok := byID[q.PromptTemplate]; ok {
			q.SystemTemplate = resolved.SystemTemplate
			q.PromptTemplate = resolved.Template
		}
	}

	// Post summary LLM
	for i := range d.Workflow.PostSummary {
		s := d.Workflow.PostSummary[i].LLM
		if s == nil {
			continue
		}
		if resolved, ok := byID[s.PromptTemplate]; ok {
			s.SystemTemplate = resolved.SystemTemplate
			s.PromptTemplate = resolved.Template
		}
	}

	// Run summary LLM
	for i := range d.Workflow.RunSummary {
		s := d.Workflow.RunSummary[i].LLM
		if s == nil {
			continue
		}
		if resolved, ok := byID[s.PromptTemplate]; ok {
			s.SystemTemplate = resolved.SystemTemplate
			s.PromptTemplate = resolved.Template
		}
	}

	// Outputs (currently only email).
	for i := range d.Workflow.Output {
		o := d.Workflow.Output[i].Email
		if o == nil {
			continue
		}
		if resolved, ok := byID[o.Template]; ok {
			o.Template = resolved.Template
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
		if source.RSS != nil {
			flow.Sources = append(flow.Sources, ParsedProcessor{
				Type:   ProcessorSourceRSS,
				Name:   "rss",
				Config: source.RSS,
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
		} else if summary.Markdown != nil {
			flow.Processors = append(flow.Processors, ParsedProcessor{
				Type:   ProcessorSummaryMD,
				Name:   summary.Markdown.Name,
				Config: summary.Markdown,
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
		} else if summary.Markdown != nil {
			flow.Processors = append(flow.Processors, ParsedProcessor{
				Type:   ProcessorRunSummaryMD,
				Name:   summary.Markdown.Name,
				Config: summary.Markdown,
			})
		}
	}

	// Parse outputs
	for _, output := range d.Workflow.Output {
		if output.Email != nil {
			flow.Outputs = append(flow.Outputs, ParsedProcessor{
				Type:   ProcessorOutputEmail,
				Name:   "email",
				Config: output.Email,
			})
		}
	}

	return flow, nil
}

// ParseToFlow converts the document into a core.Flow structure with OrderOfOperations
func (d *CuratorDocument) ParseToFlow() (*core.Flow, error) {
	return d.ParseToFlowWithFactory(nil)
}

// ParseToFlowWithFactory converts the document into a core.Flow structure with OrderOfOperations.
// When factory is nil, the flow will be created without concrete processors.
func (d *CuratorDocument) ParseToFlowWithFactory(factory ProcessorFactory) (*core.Flow, error) {
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

	// Build OrderOfOperations in the correct sequence
	// 1. Triggers (always first)
	for _, trigger := range d.Workflow.Trigger {
		if trigger.Cron != nil {
			var triggerProcessor core.TriggerProcessor
			if factory != nil {
				var err error
				triggerProcessor, err = factory.NewCronTrigger(trigger.Cron)
				if err != nil {
					return nil, err
				}
			}
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
			var sourceProcessor core.SourceProcessor
			if factory != nil {
				var err error
				sourceProcessor, err = factory.NewRedditSource(source.Reddit)
				if err != nil {
					return nil, err
				}
			}
			flow.Sources = append(flow.Sources, sourceProcessor)

			processRef := core.ProcessReference{
				Name:   "reddit",
				Type:   core.SourceProcessorType,
				Source: sourceProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
		if source.RSS != nil {
			var sourceProcessor core.SourceProcessor
			if factory != nil {
				var err error
				sourceProcessor, err = factory.NewRSSSource(source.RSS)
				if err != nil {
					return nil, err
				}
			}
			flow.Sources = append(flow.Sources, sourceProcessor)

			processRef := core.ProcessReference{
				Name:   "rss",
				Type:   core.SourceProcessorType,
				Source: sourceProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 3. Quality processors (in order defined in document)
	for _, quality := range d.Workflow.Quality {
		if quality.QualityRule != nil {
			var qualityProcessor core.QualityProcessor
			if factory != nil {
				var err error
				qualityProcessor, err = factory.NewQualityRule(quality.QualityRule)
				if err != nil {
					return nil, err
				}
			}
			flow.Quality = append(flow.Quality, qualityProcessor)

			processRef := core.ProcessReference{
				Name:    quality.QualityRule.Name,
				Type:    core.QualityProcessorType,
				Quality: qualityProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		} else if quality.LLM != nil {
			quality.LLM.MaxConcurrency = d.Workflow.MaxConcurrency

			var qualityProcessor core.QualityProcessor
			if factory != nil {
				var err error
				qualityProcessor, err = factory.NewLLMQuality(quality.LLM)
				if err != nil {
					return nil, err
				}
			}
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
			summary.LLM.MaxConcurrency = d.Workflow.MaxConcurrency

			var summaryProcessor core.SummaryProcessor
			if factory != nil {
				var err error
				summaryProcessor, err = factory.NewLLMSummary(summary.LLM)
				if err != nil {
					return nil, err
				}
			}
			flow.PostSummary = append(flow.PostSummary, summaryProcessor)

			processRef := core.ProcessReference{
				Name:        summary.LLM.Name,
				Type:        core.SummaryProcessorType,
				PostSummary: summaryProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		} else if summary.Markdown != nil {
			var summaryProcessor core.SummaryProcessor
			if factory != nil {
				var err error
				summaryProcessor, err = factory.NewMarkdownSummary(summary.Markdown)
				if err != nil {
					return nil, err
				}
			}
			flow.PostSummary = append(flow.PostSummary, summaryProcessor)

			processRef := core.ProcessReference{
				Name:        summary.Markdown.Name,
				Type:        core.SummaryProcessorType,
				PostSummary: summaryProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 5. Run Summary processors
	for _, summary := range d.Workflow.RunSummary {
		if summary.LLM != nil {
			summary.LLM.MaxConcurrency = d.Workflow.MaxConcurrency

			var runSummaryProcessor core.RunSummaryProcessor
			if factory != nil {
				var err error
				runSummaryProcessor, err = factory.NewLLMRunSummary(summary.LLM)
				if err != nil {
					return nil, err
				}
			}
			flow.RunSummary = append(flow.RunSummary, runSummaryProcessor)

			processRef := core.ProcessReference{
				Name:       summary.LLM.Name,
				Type:       core.RunSummaryProcessorType,
				RunSummary: runSummaryProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		} else if summary.Markdown != nil {
			var runSummaryProcessor core.RunSummaryProcessor
			if factory != nil {
				var err error
				runSummaryProcessor, err = factory.NewMarkdownRunSummary(summary.Markdown)
				if err != nil {
					return nil, err
				}
			}
			flow.RunSummary = append(flow.RunSummary, runSummaryProcessor)

			processRef := core.ProcessReference{
				Name:       summary.Markdown.Name,
				Type:       core.RunSummaryProcessorType,
				RunSummary: runSummaryProcessor,
			}
			flow.OrderOfOperations = append(flow.OrderOfOperations, processRef)
		}
	}

	// 6. Output processors (always last)
	for _, output := range d.Workflow.Output {
		if output.Email != nil {
			emailConfig, err := decodeEmailOutput(output.Email)
			if err != nil {
				return nil, fmt.Errorf("output email: %w", err)
			}

			var outputProcessor core.OutputProcessor
			if factory != nil {
				outputProcessor, err = factory.NewEmailOutput(emailConfig)
				if err != nil {
					return nil, err
				}
			}
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

func decodeEmailOutput(value interface{}) (*EmailOutput, error) {
	if value == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	raw, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal output config: %w", err)
	}
	var emailConfig EmailOutput
	if err := yaml.Unmarshal(raw, &emailConfig); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &emailConfig, nil
}
