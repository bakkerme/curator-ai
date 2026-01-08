# Curator Document Specification

## Overview
The Curator Document is a YAML file that declaratively defines a curation workflow. It specifies where data is loaded from, how it's processed, and where the results are sent. The document structure is designed to be extensible while maintaining clarity for the MVP scope.

## Document Structure

### Top-Level Structure
```yaml
workflow:
  name: string                    # Human-readable workflow name
  version: string                 # Optional: Document schema version (default: "1.0")
  max_concurrency: number         # Optional: max in-flight LLM calls for per-block processors
  
  trigger:                        # When to execute the workflow
    - <trigger_processor>         # Array of trigger configurations
    
  sources:                        # Data ingestion
    - <source_processor>          # Array of source configurations
    
  quality:                        # Content filtering and evaluation
    - <quality_processor>         # Array of quality configurations
    
  post_summary:                   # Per-post summarization
    - <summary_processor>         # Array of summary configurations
    
  run_summary:                    # Aggregate summarization
    - <summary_processor>         # Array of run summary configurations
    
  output:                         # Result delivery
    <output_processor>            # Single output configuration
```

## Processor Definitions
### Snapshot/Restore (Per-Processor)
Any processor can optionally include a `snapshot` block that controls saving its output to disk or restoring inputs from disk before the processor runs.

```yaml
snapshot:
  snapshot: boolean             # Optional: write output to disk after this processor
  restore: boolean              # Optional: load input from disk before this processor runs
  path: string                  # Required when snapshot or restore is true
```

When `restore` is enabled, the runner should skip upstream work and use the data loaded from `path` for this processor.

### Trigger Processors

#### Cron Trigger
Executes the workflow on a schedule using standard cron syntax.

```yaml
cron:
  schedule: string               # Cron expression (e.g., "0 0 * * *")
  timezone: string               # Optional: Timezone (default: "UTC")
```

### Source Processors

#### Reddit Source
Fetches posts from specified subreddits with optional enrichment.

```yaml
reddit:
  subreddits: [string]           # List of subreddit names (without r/ prefix)
  limit: number                  # Optional: Max posts per subreddit (default: 25)
  sort: string                   # Optional: "hot", "new", "top" (default: "hot")
  time_filter: string            # Optional: For "top" sort - "hour", "day", "week", "month", "year", "all"
  include_comments: boolean      # Optional: Fetch comment data (default: false)
  include_web: boolean           # Optional: Extract and process linked URLs (default: false)
  include_images: boolean        # Optional: Extract and process image URLs (default: false)
  min_score: number              # Optional: Minimum post score filter
```

### Quality Processors

#### Quality Rule
Rule-based filtering using expressions evaluated against post data.

```yaml
quality_rule:
  name: string                   # Unique identifier for the rule
  rule: string                   # Expression to evaluate (e.g., "comments.count > 5")
  actionType: string             # "pass_drop" - determines what happens on rule match
  result: string                 # "drop" or "pass" - action when rule evaluates to true
```

#### LLM Quality
AI-powered content evaluation for relevance and quality. By default, this is intended to take a score of 0-1 on quality.

```yaml
llm:
  name: string                   # Unique identifier
  model: string                  # Optional: Model override (default: system default)
  temperature: number            # Optional: Sampling temperature (uses OPENAI_TEMPERATURE when omitted)
  top_p: number                  # Optional: Nucleus sampling (uses OPENAI_TOP_P when omitted)
  presence_penalty: number       # Optional: Penalize new tokens (uses OPENAI_PRESENCE_PENALTY when omitted)
  top_k: number                  # Optional: Top-k sampling (uses OPENAI_TOP_K when omitted)
  prompt_template: string        # Reference to prompt template
  evaluations: [string]          # Positive criteria - content should match these
  exclusions: [string]           # Negative criteria - content matching these is dropped
  action_type: string            # "pass_drop" - binary decision
  threshold: number              # Optional: Score threshold (0-1) for pass/drop decision
```

### Summary Processors

#### LLM Summary (Post-level)
Generates summaries for individual posts.

```yaml
llm:
  name: string                   # Unique identifier
  type: string                   # "llm" - processor type
  context: string                # "post" - operates on individual posts
  model: string                  # Optional: Model override
  temperature: number            # Optional: Sampling temperature (uses OPENAI_TEMPERATURE when omitted)
  top_p: number                  # Optional: Nucleus sampling (uses OPENAI_TOP_P when omitted)
  presence_penalty: number       # Optional: Penalize new tokens (uses OPENAI_PRESENCE_PENALTY when omitted)
  top_k: number                  # Optional: Top-k sampling (uses OPENAI_TOP_K when omitted)
  prompt_template: string        # Reference to prompt template
  params:                        # Optional: Additional parameters for the prompt
    my_additional_param: [string]          # An additional example param
```

#### Markdown Summary (Post-level)
Converts markdown summaries on posts into HTML (GitHub Flavored Markdown; raw HTML is not rendered).

```yaml
markdown:
  name: string                   # Unique identifier
  type: string                   # "markdown" - processor type
  context: string                # "post" - operates on individual posts
```

#### LLM Summary (Run-level)
Generates aggregate summaries across all posts in a run.

```yaml
llm:
  name: string                    # Unique identifier
  type: string                    # "llm" - processor type
  context: string                 # "flow" - operates on entire flow results
  model: string                   # Optional: Model override
  temperature: number             # Optional: Sampling temperature (uses OPENAI_TEMPERATURE when omitted)
  top_p: number                   # Optional: Nucleus sampling (uses OPENAI_TOP_P when omitted)
  presence_penalty: number        # Optional: Penalize new tokens (uses OPENAI_PRESENCE_PENALTY when omitted)
  top_k: number                   # Optional: Top-k sampling (uses OPENAI_TOP_K when omitted)
  prompt_template: string         # Reference to prompt template
  params:                         # Optional: Additional parameters
    my_additional_param: [string] # An additional example param
```

#### Markdown Summary (Run-level)
Converts markdown run summaries into HTML (GitHub Flavored Markdown; raw HTML is not rendered).

```yaml
markdown:
  name: string                    # Unique identifier
  type: string                    # "markdown" - processor type
  context: string                 # "flow" - operates on entire flow results
```

### Output Processors

#### Email Output
Sends results via email using SMTP.

```yaml
email:
  template: string               # Reference to email template
  to: string                     # Recipient email address
  from: string                   # Sender email address
  subject: string                # Email subject line
  smtp_host: string              # Optional: SMTP server (default: from config)
  smtp_port: number              # Optional: SMTP port (default: from config)
  smtp_user: string              # Optional: SMTP username (default: from config)
  smtp_password: string          # Optional: SMTP password (default: from config)
  use_tls: boolean               # Optional: Enable TLS (default: true)
```

## Data Flow and Processing Order

1. **Trigger** fires based on configured conditions. Always first.
2. **Sources** fetch and create PostBlocks with raw data
3. **Quality** processors filter posts
4. **Post Summary** processors enhance remaining posts
5. **Run Summary** processors create aggregate summaries
6. **Output** delivers results

## Expression Language for Rules
Uses [expr](https://expr-lang.org) library to evaluate rule expressions against PostBlock data.

Quality rules use a simple expression language:
- Field access: `field.subfield`
- Comparisons: `>`, `<`, `>=`, `<=`, `==`, `!=`
- Logical operators: `&&`, `||`, `!`
- Array access: `field[0]`, `field.length`

Examples:
- `comments.count > 5`
- `score >= 100 && author != "[deleted]"`
- `title.length < 200`

## Template References

Curator uses Go's standard library templating language (`text/template`) for LLM prompts; email output templates use `html/template` (same syntax, with HTML auto-escaping).

This is a good fit because:
- It's already used in the codebase.
- It supports basic control flow (`if`, `with`, `range`) and pipelines.
- It can iterate over slices like `Comments`, `WebBlocks`, and `ImageBlocks`.

### Where templates live

Templates are defined at the top-level of the Curator Document:

```yaml
templates:
  - id: myTemplate
    template: |-
      Hello {{.Title}}
```

### Referencing templates

Processors reference templates by ID:

```yaml
prompt_template: myTemplate
```

At load time, Curator resolves these IDs into the actual template text.

Notes:
- If `prompt_template` (or `email.template`) matches a template `id`, it is treated as a reference.
- Otherwise it is treated as inline template content.

### Template language basics (Go `text/template`)

- Interpolation: `{{.Title}}`
- Conditionals:

```gotemplate
{{if .Summary}}Has summary{{end}}
```

- Looping:

```gotemplate
{{range .Comments}}
- {{.Author}}: {{.Content}}
{{end}}
```

- Common helpers:
  - `len` for slice/map/string length: `{{len .Comments}}`
  - `index` for map access: `{{index .Params "interests"}}`

### Template data available

Templates are executed with different root objects depending on where they are used.

#### LLM Quality templates (`quality.llm.prompt_template`)

Root object contains:
- All `PostBlock` fields directly (e.g. `.Title`, `.Content`, `.Comments`, `.WebBlocks`)
- `.Evaluations []string` from the processor config
- `.Exclusions []string` from the processor config

Example:

```gotemplate
Title: {{.Title}}
Evals: {{range .Evaluations}}- {{.}}{{end}}
```

#### Post Summary templates (`post_summary.llm.prompt_template`)

Root object contains:
- All `PostBlock` fields directly
- `.Params map[string]any` from the processor config

Example:

```gotemplate
Interests:
{{range (index .Params "interests")}}- {{.}}{{end}}
```

#### Run Summary templates (`run_summary.llm.prompt_template`)

Root object contains:
- `.Blocks []*PostBlock`
- `.Params map[string]any`

Example:

```gotemplate
There are {{len .Blocks}} posts.
{{range .Blocks}}- {{.Title}}{{end}}
```

#### Email templates (`output.email.template`)

Email templates are rendered as HTML bodies.

Root object contains:
- `.Blocks []*PostBlock`
- `.RunSummary *RunSummary`
- `PostBlock.Summary.HTML` and `RunSummary.HTML` when markdown summary processors are used (inserted as raw HTML, not escaped)

Example:

```gotemplate
Daily Digest\n\n{{.RunSummary.Summary}}\n\n{{range .Blocks}}- {{.Title}} ({{.URL}})\n{{end}}
```

## Extensibility

The specification is designed to support future extensions:
- New trigger types (webhook, message queue)
- Additional sources (RSS, Twitter, HackerNews)
- More quality filters (spam detection, language detection)
- Alternative outputs (Slack, database, API webhook)

The `version` field allows for schema evolution while maintaining backward compatibility.
