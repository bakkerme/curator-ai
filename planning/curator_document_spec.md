# Curator Document Specification

## Overview
The Curator Document is a YAML file that declaratively defines a curation workflow. It specifies where data is loaded from, how it's processed, and where the results are sent. The document structure is designed to be extensible while maintaining clarity for the MVP scope.

## Document Structure

### Top-Level Structure
```yaml
workflow:
  name: string                    # Human-readable workflow name
  version: string                 # Optional: Document schema version (default: "1.0")
  
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
  prompt_template: string        # Reference to prompt template
  params:                        # Optional: Additional parameters for the prompt
    my_additional_param: [string]          # An additional example param
```

#### LLM Summary (Run-level)
Generates aggregate summaries across all posts in a run.

```yaml
llm:
  name: string                    # Unique identifier
  type: string                    # "llm" - processor type
  context: string                 # "flow" - operates on entire flow results
  model: string                   # Optional: Model override
  prompt_template: string         # Reference to prompt template
  params:                         # Optional: Additional parameters
    my_additional_param: [string] # An additional example param
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
-- Can come in any order --
2. **Sources** fetch and create PostBlocks with raw data
3. **Quality** processors filter posts (in order):
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

Templates are referenced by name and should be defined separately. The system will look for templates in:
1. Inline definitions within the curator document
2. External template files
3. System default templates

## Extensibility

The specification is designed to support future extensions:
- New trigger types (webhook, message queue)
- Additional sources (RSS, Twitter, HackerNews)
- More quality filters (spam detection, language detection)
- Alternative outputs (Slack, database, API webhook)

The `version` field allows for schema evolution while maintaining backward compatibility.