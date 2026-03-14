package config

import (
	"fmt"
	"net/mail"
	"strings"
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
	Name           string             `yaml:"name"`
	Version        string             `yaml:"version,omitempty"`
	MaxConcurrency int                `yaml:"max_concurrency,omitempty"`
	DedupeStore    *DedupeStoreConfig `yaml:"dedupe_store,omitempty"`
	Trigger        []TriggerConfig    `yaml:"trigger"`
	Sources        []SourceConfig     `yaml:"sources"`
	Quality        []QualityConfig    `yaml:"quality,omitempty"`
	PostSummary    []SummaryConfig    `yaml:"post_summary,omitempty"`
	RunSummary     []SummaryConfig    `yaml:"run_summary,omitempty"`
	Output         []OutputConfig     `yaml:"output"`
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
	Reddit   *RedditSource   `yaml:"reddit,omitempty"`
	RSS      *RSSSource      `yaml:"rss,omitempty"`
	Arxiv    *ArxivSource    `yaml:"arxiv,omitempty"`
	Scrape   *ScrapeSource   `yaml:"scrape,omitempty"`
	TestFile *TestFileSource `yaml:"testfile,omitempty"`
}

// ScrapeSource defines generic web scraping source configuration.
type ScrapeSource struct {
	URL         string                 `yaml:"url"`
	PostLimit   int                    `yaml:"post_limit,omitempty"`
	Lookback    string                 `yaml:"lookback,omitempty"`
	Discovery   ScrapeDiscoveryConfig  `yaml:"discovery"`
	Extraction  ScrapeExtractionConfig `yaml:"extraction"`
	Markdown    ScrapeMarkdownConfig   `yaml:"markdown,omitempty"`
	Request     ScrapeRequestConfig    `yaml:"request,omitempty"`
	SummaryPlan *SummaryPlanConfig     `yaml:"summary_plan,omitempty"`
	Snapshot    *core.SnapshotConfig   `yaml:"snapshot,omitempty"`
}

// ScrapeDiscoveryConfig controls discovery of candidate post URLs from index pages.
type ScrapeDiscoveryConfig struct {
	ItemSelector     string `yaml:"item_selector"`
	LinkAttr         string `yaml:"link_attr,omitempty"`
	NextPageSelector string `yaml:"next_page_selector,omitempty"`
	MaxPages         int    `yaml:"max_pages,omitempty"`
}

// ScrapeExtractionConfig controls field extraction from each discovered post page.
type ScrapeExtractionConfig struct {
	TitleSelector   string   `yaml:"title_selector,omitempty"`
	TitleAttr       string   `yaml:"title_attr,omitempty"`
	AuthorSelector  string   `yaml:"author_selector,omitempty"`
	AuthorAttr      string   `yaml:"author_attr,omitempty"`
	DateSelector    string   `yaml:"date_selector,omitempty"`
	DateAttr        string   `yaml:"date_attr,omitempty"`
	ContentSelector string   `yaml:"content_selector"`
	RemoveSelectors []string `yaml:"remove_selectors,omitempty"`
}

// ScrapeMarkdownConfig controls optional HTML-to-Markdown conversion for extracted content.
type ScrapeMarkdownConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// ScrapeRequestConfig controls HTTP behavior for scrape source requests.
type ScrapeRequestConfig struct {
	UserAgent string `yaml:"user_agent,omitempty"`
	Timeout   string `yaml:"timeout,omitempty"`
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
	SummaryPlan     *SummaryPlanConfig   `yaml:"summary_plan,omitempty"`
	Snapshot        *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// RSSSource defines RSS/Atom feed configuration
type RSSSource struct {
	Feeds                   []string             `yaml:"feeds"`
	Limit                   int                  `yaml:"limit,omitempty"`
	IncludeContent          *bool                `yaml:"include_content,omitempty"`
	ConvertSourceToMarkdown bool                 `yaml:"convert_source_to_markdown,omitempty"`
	UserAgent               string               `yaml:"user_agent,omitempty"`
	SummaryPlan             *SummaryPlanConfig   `yaml:"summary_plan,omitempty"`
	Snapshot                *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// ArxivSource defines arXiv API configuration for paper discovery and ingestion.
type ArxivSource struct {
	Query      string   `yaml:"query,omitempty"`
	Categories []string `yaml:"categories,omitempty"`
	MaxResults int      `yaml:"max_results,omitempty"`
	SortBy     string   `yaml:"sort_by,omitempty"`
	SortOrder  string   `yaml:"sort_order,omitempty"`
	DateFrom   string   `yaml:"date_from,omitempty"`
	DateTo     string   `yaml:"date_to,omitempty"`
	// AbstractOnly forces PostBlock content/chunks to be built from the abstract only.
	// When enabled, the processor skips full-text fetches via Jina.
	AbstractOnly            *bool                `yaml:"abstract_only,omitempty"`
	IncludeAbstractInChunks *bool                `yaml:"include_abstract_in_chunks,omitempty"`
	Chunking                *ArxivChunkingConfig `yaml:"chunking,omitempty"`
	SummaryPlan             *SummaryPlanConfig   `yaml:"summary_plan,omitempty"`
	Snapshot                *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// ArxivChunkingConfig controls how arXiv paper content is split into chunks.
type ArxivChunkingConfig struct {
	Mode             string `yaml:"mode,omitempty"`
	FallbackMaxChars int    `yaml:"fallback_max_chars,omitempty"`
	MinSectionChars  int    `yaml:"min_section_chars,omitempty"`
}

// SummaryPlanConfig declares how summary processors should handle a post.
type SummaryPlanConfig struct {
	Mode          core.SummaryMode `yaml:"mode"`
	MaxChunkChars int              `yaml:"max_chunk_chars,omitempty"`
	ChunkLimit    int              `yaml:"chunk_limit,omitempty"`
}

// TestFileSource defines a file-backed source for local testing.
type TestFileSource struct {
	Path        string               `yaml:"path"`
	ChunkSize   int                  `yaml:"chunk_size,omitempty"`
	SummaryPlan *SummaryPlanConfig   `yaml:"summary_plan,omitempty"`
	Snapshot    *core.SnapshotConfig `yaml:"snapshot,omitempty"`
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
	ActionType string               `yaml:"action_type"`
	Result     string               `yaml:"result"`
	Snapshot   *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// LLMQuality defines AI-powered quality evaluation
type LLMQuality struct {
	Name           string   `yaml:"name"`
	Model          string   `yaml:"model,omitempty"`
	Temperature    *float64 `yaml:"temperature,omitempty"`
	SystemTemplate string   `yaml:"system_template"`
	PromptTemplate string   `yaml:"prompt_template"`
	Evaluations    []string `yaml:"evaluations,omitempty"`
	Exclusions     []string `yaml:"exclusions,omitempty"`
	ActionType     string   `yaml:"action_type"`
	Threshold      float64  `yaml:"threshold,omitempty"`
	// BlockErrorPolicy controls behavior when processing a single PostBlock fails.
	// Allowed values: "fail" (default) or "drop".
	BlockErrorPolicy string     `yaml:"block_error_policy,omitempty"`
	MaxConcurrency   int        `yaml:"max_concurrency,omitempty"`
	Images           *LLMImages `yaml:"images,omitempty"`
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
	Name           string                 `yaml:"name"`
	Type           string                 `yaml:"type"`
	Context        string                 `yaml:"context"`
	Model          string                 `yaml:"model,omitempty"`
	Temperature    *float64               `yaml:"temperature,omitempty"`
	SystemTemplate string                 `yaml:"system_template"`
	PromptTemplate string                 `yaml:"prompt_template"`
	ChunkSystem    string                 `yaml:"chunk_system_template,omitempty"`
	ChunkPrompt    string                 `yaml:"chunk_prompt_template,omitempty"`
	Params         map[string]interface{} `yaml:"params,omitempty"`
	// BlockErrorPolicy controls behavior when processing a single PostBlock fails.
	// Allowed values: "fail" (default) or "drop".
	// Only applies to context=post processors.
	BlockErrorPolicy string               `yaml:"block_error_policy,omitempty"`
	MaxConcurrency   int                  `yaml:"max_concurrency,omitempty"`
	Images           *LLMImages           `yaml:"images,omitempty"`
	Snapshot         *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

const (
	BlockErrorPolicyFail = "fail"
	BlockErrorPolicyDrop = "drop"
)

const (
	ImageModeMultimodal = "multimodal"
	ImageModeCaption    = "caption"
)

type LLMImages struct {
	Enabled              bool             `yaml:"enabled,omitempty"`
	Mode                 string           `yaml:"mode,omitempty"`
	MaxImages            int              `yaml:"max_images,omitempty"`
	IncludeCommentImages bool             `yaml:"include_comment_images,omitempty"`
	Caption              *LLMImageCaption `yaml:"caption,omitempty"`
}

type LLMImageCaption struct {
	Model string `yaml:"model,omitempty"`
	// This is the reference to the template that's actually used in the doc
	Template string `yaml:"template,omitempty"`
	// These two are for the resolved templates
	SystemTemplate string `yaml:"system_template"`
	PromptTemplate string `yaml:"prompt_template"`

	Temperature    *float64 `yaml:"temperature,omitempty"`
	MaxConcurrency int      `yaml:"max_concurrency,omitempty"`
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

type DedupeStoreConfig struct {
	Driver string        `yaml:"driver,omitempty"`
	DSN    string        `yaml:"dsn,omitempty"`
	Table  string        `yaml:"table,omitempty"`
	TTL    time.Duration `yaml:"ttl,omitempty"`
}

func (c *DedupeStoreConfig) UnmarshalYAML(value *yaml.Node) error {
	type temp struct {
		Driver string      `yaml:"driver,omitempty"`
		DSN    string      `yaml:"dsn,omitempty"`
		Table  string      `yaml:"table,omitempty"`
		TTL    interface{} `yaml:"ttl,omitempty"`
	}
	var t temp
	if err := value.Decode(&t); err != nil {
		return err
	}

	c.Driver = t.Driver
	c.DSN = t.DSN
	c.Table = t.Table

	if t.TTL == nil {
		c.TTL = 0
		return nil
	}

	s, ok := t.TTL.(string)
	if !ok {
		return fmt.Errorf("dedupe_store ttl must be a duration string")
	}
	if strings.TrimSpace(s) == "" {
		c.TTL = 0
		return nil
	}
	d, err := ParseDurationExtended(s)
	if err != nil {
		return fmt.Errorf("dedupe_store ttl: %w", err)
	}
	c.TTL = d
	return nil
}

// EmailOutput defines email delivery configuration
type EmailOutput struct {
	Template               string               `yaml:"template"`
	To                     string               `yaml:"to"`
	From                   string               `yaml:"from"`
	Subject                string               `yaml:"subject"`
	SMTPHost               string               `yaml:"smtp_host,omitempty"`
	SMTPPort               int                  `yaml:"smtp_port,omitempty"`
	SMTPUser               string               `yaml:"smtp_user,omitempty"`
	SMTPPassword           string               `yaml:"smtp_password,omitempty"`
	SMTPTLSMode            string               `yaml:"smtp_tls_mode,omitempty"`
	SMTPInsecureSkipVerify *bool                `yaml:"smtp_insecure_skip_verify,omitempty"`
	Snapshot               *core.SnapshotConfig `yaml:"snapshot,omitempty"`
}

// ProcessorType identifies the type of processor
type ProcessorType string

const (
	ProcessorTriggerCron   ProcessorType = "trigger_cron"
	ProcessorSourceReddit  ProcessorType = "source_reddit"
	ProcessorSourceRSS     ProcessorType = "source_rss"
	ProcessorSourceArxiv   ProcessorType = "source_arxiv"
	ProcessorSourceScrape  ProcessorType = "source_scrape"
	ProcessorSourceTest    ProcessorType = "source_testfile"
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
	NewArxivSource(config *ArxivSource) (core.SourceProcessor, error)
	NewScrapeSource(config *ScrapeSource) (core.SourceProcessor, error)
	NewTestFileSource(config *TestFileSource) (core.SourceProcessor, error)
	NewQualityRule(config *QualityRule) (core.QualityProcessor, error)
	NewLLMQuality(config *LLMQuality) (core.QualityProcessor, error)
	NewLLMSummary(config *LLMSummary) (core.SummaryProcessor, error)
	NewLLMRunSummary(config *LLMSummary) (core.RunSummaryProcessor, error)
	NewMarkdownSummary(config *MarkdownSummary) (core.SummaryProcessor, error)
	NewMarkdownRunSummary(config *MarkdownSummary) (core.RunSummaryProcessor, error)
	NewEmailOutput(config *EmailOutput) (core.OutputProcessor, error)
}

// DedupeStoreConfigurer supports configuring a shared dedupe store for processors.
type DedupeStoreConfigurer interface {
	ConfigureDedupeStore(config *DedupeStoreConfig) error
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

	if err := validateDedupeStoreConfig(d.Workflow.DedupeStore); err != nil {
		return err
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
		if source.Reddit == nil && source.RSS == nil && source.Arxiv == nil && source.Scrape == nil && source.TestFile == nil {
			return fmt.Errorf("source %d: unsupported source type", i)
		}
		if source.Reddit != nil && len(source.Reddit.Subreddits) == 0 {
			return fmt.Errorf("source %d: at least one subreddit is required", i)
		}
		if source.RSS != nil && len(source.RSS.Feeds) == 0 {
			return fmt.Errorf("source %d: at least one rss feed is required", i)
		}
		if source.Arxiv != nil && strings.TrimSpace(source.Arxiv.Query) == "" && len(source.Arxiv.Categories) == 0 {
			return fmt.Errorf("source %d: arxiv requires query or categories", i)
		}
		if source.TestFile != nil && strings.TrimSpace(source.TestFile.Path) == "" {
			return fmt.Errorf("source %d: testfile path is required", i)
		}
		if source.Scrape != nil {
			if strings.TrimSpace(source.Scrape.URL) == "" {
				return fmt.Errorf("source %d: scrape url is required", i)
			}
			if strings.TrimSpace(source.Scrape.Discovery.ItemSelector) == "" {
				return fmt.Errorf("source %d: scrape discovery.item_selector is required", i)
			}
			if strings.TrimSpace(source.Scrape.Extraction.ContentSelector) == "" {
				return fmt.Errorf("source %d: scrape extraction.content_selector is required", i)
			}
			if source.Scrape.Discovery.MaxPages < 0 {
				return fmt.Errorf("source %d: scrape discovery.max_pages must be >= 0", i)
			}
			if source.Scrape.PostLimit < 0 {
				return fmt.Errorf("source %d: scrape post_limit must be >= 0", i)
			}
			if strings.TrimSpace(source.Scrape.Lookback) != "" {
				if _, err := ParseDurationExtended(source.Scrape.Lookback); err != nil {
					return fmt.Errorf("source %d: scrape lookback: %w", i, err)
				}
			}
		}
		if source.Reddit != nil {
			if err := validateSummaryPlanConfig(fmt.Sprintf("source %d reddit", i), source.Reddit.SummaryPlan); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("source %d reddit", i), source.Reddit.Snapshot); err != nil {
				return err
			}
		}
		if source.RSS != nil {
			if err := validateSummaryPlanConfig(fmt.Sprintf("source %d rss", i), source.RSS.SummaryPlan); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("source %d rss", i), source.RSS.Snapshot); err != nil {
				return err
			}
		}
		if source.Arxiv != nil {
			if err := validateSummaryPlanConfig(fmt.Sprintf("source %d arxiv", i), source.Arxiv.SummaryPlan); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("source %d arxiv", i), source.Arxiv.Snapshot); err != nil {
				return err
			}
		}
		if source.TestFile != nil {
			if err := validateSummaryPlanConfig(fmt.Sprintf("source %d testfile", i), source.TestFile.SummaryPlan); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("source %d testfile", i), source.TestFile.Snapshot); err != nil {
				return err
			}
		}
		if source.Scrape != nil {
			if err := validateSummaryPlanConfig(fmt.Sprintf("source %d scrape", i), source.Scrape.SummaryPlan); err != nil {
				return err
			}
			if err := validateSnapshotConfig(fmt.Sprintf("source %d scrape", i), source.Scrape.Snapshot); err != nil {
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
			if err := validateBlockErrorPolicy(fmt.Sprintf("quality %d llm", i), quality.LLM.BlockErrorPolicy); err != nil {
				return err
			}
			if err := validateLLMTemperature(fmt.Sprintf("quality %d llm", i), quality.LLM.Temperature); err != nil {
				return err
			}
			if err := validateImagesConfig(fmt.Sprintf("quality %d llm", i), quality.LLM.Images); err != nil {
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
			if err := validateBlockErrorPolicy(fmt.Sprintf("post_summary %d llm", i), summary.LLM.BlockErrorPolicy); err != nil {
				return err
			}
			if err := validateLLMTemperature(fmt.Sprintf("post_summary %d llm", i), summary.LLM.Temperature); err != nil {
				return err
			}
			if err := validateChunkTemplates(fmt.Sprintf("post_summary %d llm", i), summary.LLM); err != nil {
				return err
			}
			if err := validateImagesConfig(fmt.Sprintf("post_summary %d llm", i), summary.LLM.Images); err != nil {
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
			if err := validateBlockErrorPolicy(fmt.Sprintf("run_summary %d llm", i), summary.LLM.BlockErrorPolicy); err != nil {
				return err
			}
			if err := validateLLMTemperature(fmt.Sprintf("run_summary %d llm", i), summary.LLM.Temperature); err != nil {
				return err
			}
			if err := validateChunkTemplates(fmt.Sprintf("run_summary %d llm", i), summary.LLM); err != nil {
				return err
			}
			if err := validateImagesConfig(fmt.Sprintf("run_summary %d llm", i), summary.LLM.Images); err != nil {
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

func validateDedupeStoreConfig(cfg *DedupeStoreConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.TTL < 0 {
		return fmt.Errorf("dedupe_store ttl must be >= 0")
	}
	switch strings.ToLower(cfg.Driver) {
	case "", "sqlite":
		return nil
	default:
		return fmt.Errorf("dedupe_store driver must be \"sqlite\"")
	}
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

func validateSummaryPlanConfig(label string, cfg *SummaryPlanConfig) error {
	if cfg == nil {
		// summary_plan is optional; when omitted, sources default to mode=full.
		return nil
	}
	switch cfg.Mode {
	case core.SummaryModeFull, core.SummaryModePerChunk, core.SummaryModeMapReduce:
		return nil
	case "":
		// Treat an empty mode as implicit full so existing docs can opt in gradually.
		cfg.Mode = core.SummaryModeFull
		return nil
	default:
		return fmt.Errorf("%s: summary_plan.mode must be %q, %q, or %q", label, core.SummaryModeFull, core.SummaryModePerChunk, core.SummaryModeMapReduce)
	}
}

func validateLLMTemperature(label string, temperature *float64) error {
	if temperature == nil {
		return nil
	}
	if *temperature < 0 || *temperature > 2 {
		return fmt.Errorf("%s: temperature must be between 0 and 2", label)
	}
	return nil
}

func validateBlockErrorPolicy(label string, policy string) error {
	if policy == "" {
		return nil
	}
	switch policy {
	case BlockErrorPolicyFail, BlockErrorPolicyDrop:
		return nil
	default:
		return fmt.Errorf("%s: block_error_policy must be %q or %q", label, BlockErrorPolicyFail, BlockErrorPolicyDrop)
	}
}

func validateImagesConfig(label string, cfg *LLMImages) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if cfg.Mode == "" {
		return fmt.Errorf("%s: images.mode is required when images.enabled is true", label)
	}
	switch cfg.Mode {
	case ImageModeCaption, ImageModeMultimodal:
	default:
		return fmt.Errorf("%s: images.mode must be %q or %q", label, ImageModeCaption, ImageModeMultimodal)
	}
	if cfg.MaxImages < 0 {
		return fmt.Errorf("%s: images.max_images must be >= 0", label)
	}
	if cfg.Mode == ImageModeCaption {
		if cfg.Caption == nil {
			return fmt.Errorf("%s: images.caption is required when images.mode=caption", label)
		}
		if cfg.Caption.SystemTemplate == "" || cfg.Caption.PromptTemplate == "" {
			return fmt.Errorf("%s: images.caption system_template and prompt_template are required", label)
		}
		if err := validateLLMTemperature(fmt.Sprintf("%s images.caption", label), cfg.Caption.Temperature); err != nil {
			return err
		}
	}
	return nil
}

func validateChunkTemplates(label string, cfg *LLMSummary) error {
	if cfg == nil {
		return nil
	}
	if cfg.ChunkSystem == "" && cfg.ChunkPrompt == "" {
		return nil
	}
	if cfg.ChunkSystem == "" || cfg.ChunkPrompt == "" {
		return fmt.Errorf("%s: chunk_system_template and chunk_prompt_template must both be set", label)
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
		if q.Images != nil && q.Images.Caption != nil {
			if resolved, ok := byID[q.Images.Caption.Template]; ok {
				q.Images.Caption.SystemTemplate = resolved.SystemTemplate
				q.Images.Caption.PromptTemplate = resolved.Template
			}
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
		if resolved, ok := byID[s.ChunkPrompt]; ok {
			s.ChunkSystem = resolved.SystemTemplate
			s.ChunkPrompt = resolved.Template
		}
		if s.Images != nil && s.Images.Caption != nil {
			if resolved, ok := byID[s.Images.Caption.Template]; ok {
				s.Images.Caption.SystemTemplate = resolved.SystemTemplate
				s.Images.Caption.PromptTemplate = resolved.Template
			}
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
		if s.Images != nil && s.Images.Caption != nil {
			if resolved, ok := byID[s.Images.Caption.Template]; ok {
				s.Images.Caption.SystemTemplate = resolved.SystemTemplate
				s.Images.Caption.PromptTemplate = resolved.Template
			}
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
		if source.Arxiv != nil {
			flow.Sources = append(flow.Sources, ParsedProcessor{
				Type:   ProcessorSourceArxiv,
				Name:   "arxiv",
				Config: source.Arxiv,
			})
		}
		if source.Scrape != nil {
			flow.Sources = append(flow.Sources, ParsedProcessor{
				Type:   ProcessorSourceScrape,
				Name:   "scrape",
				Config: source.Scrape,
			})
		}
		if source.TestFile != nil {
			flow.Sources = append(flow.Sources, ParsedProcessor{
				Type:   ProcessorSourceTest,
				Name:   "testfile",
				Config: source.TestFile,
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

	if factory != nil {
		if dedupeFactory, ok := factory.(DedupeStoreConfigurer); ok {
			if err := dedupeFactory.ConfigureDedupeStore(d.Workflow.DedupeStore); err != nil {
				return nil, err
			}
		}
	}

	flow := newFlowFromDocument(d)

	buildTriggers(flow, d.Workflow.Trigger, factory)
	buildSources(flow, d.Workflow.Sources, factory)
	buildQuality(flow, d.Workflow.Quality, d.Workflow.MaxConcurrency, factory)
	buildSummaries(flow, d.Workflow.PostSummary, d.Workflow.MaxConcurrency, factory, false)
	buildSummaries(flow, d.Workflow.RunSummary, d.Workflow.MaxConcurrency, factory, true)
	buildOutputs(flow, d.Workflow.Output, factory)

	configBytes, _ := yaml.Marshal(d)
	var rawConfig map[string]interface{}
	err := yaml.Unmarshal(configBytes, &rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config: %w", err)
	}
	flow.RawConfig = rawConfig

	return flow, nil
}

func newFlowFromDocument(d *CuratorDocument) *core.Flow {
	now := time.Now()
	flow := &core.Flow{
		ID:        "",
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
	return flow
}

func buildTriggers(flow *core.Flow, triggers []TriggerConfig, factory ProcessorFactory) {
	for _, trigger := range triggers {
		if trigger.Cron == nil {
			continue
		}
		var triggerProcessor core.TriggerProcessor
		if factory != nil {
			triggerProcessor, _ = factory.NewCronTrigger(trigger.Cron)
		}
		flow.Triggers = append(flow.Triggers, triggerProcessor)
		flow.OrderOfOperations = append(flow.OrderOfOperations, core.ProcessReference{
			Name:    "cron",
			Type:    core.TriggerProcessorType,
			Trigger: triggerProcessor,
		})
	}
}

func buildSources(flow *core.Flow, sources []SourceConfig, factory ProcessorFactory) {
	for _, source := range sources {
		if source.Reddit != nil {
			buildSourceProcessor(flow, "reddit", core.SourceProcessorType, source.Reddit,
				func(f ProcessorFactory, c *RedditSource) (core.SourceProcessor, error) {
					return f.NewRedditSource(c)
				}, factory)
		}
		if source.RSS != nil {
			buildSourceProcessor(flow, "rss", core.SourceProcessorType, source.RSS,
				func(f ProcessorFactory, c *RSSSource) (core.SourceProcessor, error) {
					return f.NewRSSSource(c)
				}, factory)
		}
		if source.Arxiv != nil {
			buildSourceProcessor(flow, "arxiv", core.SourceProcessorType, source.Arxiv,
				func(f ProcessorFactory, c *ArxivSource) (core.SourceProcessor, error) {
					return f.NewArxivSource(c)
				}, factory)
		}
		if source.Scrape != nil {
			buildSourceProcessor(flow, "scrape", core.SourceProcessorType, source.Scrape,
				func(f ProcessorFactory, c *ScrapeSource) (core.SourceProcessor, error) {
					return f.NewScrapeSource(c)
				}, factory)
		}
		if source.TestFile != nil {
			buildSourceProcessor(flow, "testfile", core.SourceProcessorType, source.TestFile,
				func(f ProcessorFactory, c *TestFileSource) (core.SourceProcessor, error) {
					return f.NewTestFileSource(c)
				}, factory)
		}
	}
}

func buildSourceProcessor[T any](flow *core.Flow, name string, ptype core.ProcessorType, cfg T, factoryFn func(ProcessorFactory, T) (core.SourceProcessor, error), factory ProcessorFactory) {
	var processor core.SourceProcessor
	if factory != nil {
		processor, _ = factoryFn(factory, cfg)
	}
	flow.Sources = append(flow.Sources, processor)
	flow.OrderOfOperations = append(flow.OrderOfOperations, core.ProcessReference{
		Name:   name,
		Type:   ptype,
		Source: processor,
	})
}

func buildQuality(flow *core.Flow, quality []QualityConfig, maxConcurrency int, factory ProcessorFactory) {
	for _, q := range quality {
		if q.QualityRule != nil {
			buildQualityProcessor(flow, q.QualityRule.Name, q.QualityRule,
				func(f ProcessorFactory, c *QualityRule) (core.QualityProcessor, error) {
					return f.NewQualityRule(c)
				}, factory)
		} else if q.LLM != nil {
			q.LLM.MaxConcurrency = maxConcurrency
			buildQualityProcessor(flow, q.LLM.Name, q.LLM,
				func(f ProcessorFactory, c *LLMQuality) (core.QualityProcessor, error) {
					return f.NewLLMQuality(c)
				}, factory)
		}
	}
}

func buildQualityProcessor[T any](flow *core.Flow, name string, cfg T, factoryFn func(ProcessorFactory, T) (core.QualityProcessor, error), factory ProcessorFactory) {
	var processor core.QualityProcessor
	if factory != nil {
		processor, _ = factoryFn(factory, cfg)
	}
	flow.Quality = append(flow.Quality, processor)
	flow.OrderOfOperations = append(flow.OrderOfOperations, core.ProcessReference{
		Name:    name,
		Type:    core.QualityProcessorType,
		Quality: processor,
	})
}

func buildSummaries(flow *core.Flow, summaries []SummaryConfig, maxConcurrency int, factory ProcessorFactory, isRunSummary bool) {
	for _, s := range summaries {
		if s.LLM != nil {
			s.LLM.MaxConcurrency = maxConcurrency
			if isRunSummary {
				buildRunSummaryProcessor(flow, s.LLM.Name, s.LLM,
					func(f ProcessorFactory, c *LLMSummary) (core.RunSummaryProcessor, error) {
						return f.NewLLMRunSummary(c)
					}, factory)
			} else {
				buildSummaryProcessor(flow, s.LLM.Name, s.LLM,
					func(f ProcessorFactory, c *LLMSummary) (core.SummaryProcessor, error) {
						return f.NewLLMSummary(c)
					}, factory)
			}
		} else if s.Markdown != nil {
			if isRunSummary {
				buildRunSummaryProcessor(flow, s.Markdown.Name, s.Markdown,
					func(f ProcessorFactory, c *MarkdownSummary) (core.RunSummaryProcessor, error) {
						return f.NewMarkdownRunSummary(c)
					}, factory)
			} else {
				buildSummaryProcessor(flow, s.Markdown.Name, s.Markdown,
					func(f ProcessorFactory, c *MarkdownSummary) (core.SummaryProcessor, error) {
						return f.NewMarkdownSummary(c)
					}, factory)
			}
		}
	}
}

func buildSummaryProcessor[T any](flow *core.Flow, name string, cfg T, factoryFn func(ProcessorFactory, T) (core.SummaryProcessor, error), factory ProcessorFactory) {
	var processor core.SummaryProcessor
	if factory != nil {
		processor, _ = factoryFn(factory, cfg)
	}
	flow.PostSummary = append(flow.PostSummary, processor)
	flow.OrderOfOperations = append(flow.OrderOfOperations, core.ProcessReference{
		Name:        name,
		Type:        core.SummaryProcessorType,
		PostSummary: processor,
	})
}

func buildRunSummaryProcessor[T any](flow *core.Flow, name string, cfg T, factoryFn func(ProcessorFactory, T) (core.RunSummaryProcessor, error), factory ProcessorFactory) {
	var processor core.RunSummaryProcessor
	if factory != nil {
		processor, _ = factoryFn(factory, cfg)
	}
	flow.RunSummary = append(flow.RunSummary, processor)
	flow.OrderOfOperations = append(flow.OrderOfOperations, core.ProcessReference{
		Name:       name,
		Type:       core.RunSummaryProcessorType,
		RunSummary: processor,
	})
}

func buildOutputs(flow *core.Flow, outputs []OutputConfig, factory ProcessorFactory) {
	for _, output := range outputs {
		if output.Email == nil {
			continue
		}
		emailConfig, err := decodeEmailOutput(output.Email)
		if err != nil {
			continue
		}
		var outputProcessor core.OutputProcessor
		if factory != nil {
			outputProcessor, _ = factory.NewEmailOutput(emailConfig)
		}
		flow.Outputs = append(flow.Outputs, outputProcessor)
		flow.OrderOfOperations = append(flow.OrderOfOperations, core.ProcessReference{
			Name:   "email",
			Type:   core.OutputProcessorType,
			Output: outputProcessor,
		})
	}
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
