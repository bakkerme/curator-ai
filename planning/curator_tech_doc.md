# Curator MVP Technical Document

## 1. MVP Focus: Pipeline Engine Architecture

**Core objective:** Design and build a flexible, configurable pipeline engine that can ingest content, process it through multiple AI-powered stages, and provide comprehensive feedback mechanisms to optimize performance.

**Key design principles:**
- **Modularity**: Each pipeline stage is a discrete, configurable component
- **Measurability**: Every processing decision is logged and can be evaluated
- **Adaptability**: Pipeline configurations can be iteratively improved based on feedback
- **Transparency**: Users understand exactly why content was filtered or promoted

## 2. Workflow System Architecture Overview

**Terminology Clarification:**
- **Workflow**: The complete end-to-end process from content discovery to delivery (user-facing concept)
- **Pipeline**: The technical execution engine that processes content through stages (system concept)  
- **Component**: The element that produces or acts upon data, making up the pipeline
- A single workflow may contain multiple pipelines or processing paths

**Component Hierarchy:**
1. **Sources**: Complete interfaces to external content systems with schema discovery
2. **Processors**: Discrete transformation, analysis, or enrichment units
3. **Evaluators**: Specialized processors for quality assessment and scoring
4. **Routers**: Flow control components that direct content through different paths
5. **Aggregators**: Collection and grouping components for processed content
6. **Formatters**: Output generation components for deliverable formats

See planning/curator_workflow_framework.md for complete conceptual framework.

## 3. Core Pipeline Components

### Data Ingestion Layer
**Question: How is data ingested?**

**Dynamic Schema Discovery Pattern**: Each content source generates its own schema
- **Input Adapters**: Reddit, RSS, Twitter, Forums, Documents
- **Schema Discovery Phase**: Analyze sample data to generate JSON Schema
- **Runtime Type Safety**: JSON Schema enables Go/TypeScript validation
- **Rate Limiting**: Built-in backoff and throttling for API compliance

**Schema Discovery Workflow**:
1. **Sample Data Collection**: Adapter fetches representative content samples
2. **Field Analysis**: Analyze available fields, types, and data patterns
3. **JSON Schema Generation**: Create formal schema describing the data structure
4. **Schema Attachment**: Associate schema with adapter configuration
5. **Runtime Validation**: Use schema for type checking during processing

```
// Generated JSON Schema Example (Reddit)
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "title": {"type": "string"},
    "selftext": {"type": "string"},
    "author": {"type": "string"},
    "created_utc": {"type": "number"},
    "score": {"type": "integer"},
    "num_comments": {"type": "integer"},
    "subreddit": {"type": "string"},
    "url": {"type": "string", "format": "uri"},
    "thumbnail": {"type": "string"},
    "is_video": {"type": "boolean"}
  },
  "required": ["id", "title", "created_utc"]
}
```

**Benefits of Schema Discovery**:
- **Type Safety**: Runtime validation in both Go and TypeScript
- **Field Awareness**: Processors know exactly what fields are available
- **Schema Evolution**: Detect when source APIs change structure
- **Pipeline Configuration**: UI can show available fields for filtering/processing
- **Custom Processing**: Users can reference specific fields in pipeline rules

### Workflow Component Architecture
**Question: What are the various workflow components and processing steps?**

**Component Types from Workflow Framework**:
1. **Sources**: Complete interfaces to external content systems that output normalized ContentItem objects
   - Handle authentication, rate limiting, pagination, error handling
   - Auto-discover and attach JSON schemas describing their data structure
   - Examples: RedditSource, RSSSource, TwitterSource, ForumScraperSource

2. **Processors**: Discrete transformation, analysis, or enrichment units
   - **Extractors**: Parse text, extract entities, clean formatting
   - **Filters**: Rule-based quality checks, spam detection, keyword matching  
   - **Enrichers**: Add metadata, resolve links, fetch additional context
   - **Analyzers**: LLM-powered quality scoring, summarization, classification
   - Characteristics: Stateless, composable, dependency-aware

3. **Evaluators**: Specialized processors that assign quality scores for routing decisions
   - Multi-dimensional scoring: Substantiveness, constructiveness, novelty, accuracy, relevance
   - Output numerical scores, confidence levels, reasoning explanations
   - Support multiple LLM backends (local Ollama, OpenAI API, etc.)

4. **Routers**: Components that direct content through different processing paths
   - If/then rules, score thresholds, content type detection
   - Examples: "High-quality content → summarization path", "Low-quality → discard path"

5. **Aggregators**: Components that collect processed content for output generation
   - Deduplication, topic clustering, priority ranking
   - Output structured collections ready for formatting

6. **Formatters**: Transform aggregated content into deliverable formats
   - Email digest, JSON feed, knowledge graph, dashboard data
   - User-customizable output formatting templates

**Pipeline Execution Model**:
- **Batch Processing**: Workflows run on cron schedules, process all new content since last run
- **Dependency Resolution**: Processors declare input requirements, system validates before execution
- **Parallel Execution**: Independent processors run concurrently
- **Fault-tolerant**: Failed processors skip individual items, pipeline continues
- **Retry Logic**: LLM failures trigger automatic retries with backoff
- **State Management**: Track processing checkpoints, support reprocessing failed items

**Example Workflow: AI Research Daily Digest**:
```
RedditSource (50 posts) + RSSSource (20 papers) → 
TextExtractor → SpamFilter → LLMQuality (Evaluator) → 
QualityRouter → [High: LLM Summary → TechAnalyze] + [Medium: Quick Summary] → 
TopicClustering (Aggregator) → EmailDigest (Formatter) → HTML Email Output
```

### Type System & Data Flow
**Question: How are data types handled in a configurable system?**

**JSON Schema-Driven Type System**: Dynamic schemas enable flexible yet type-safe processing

**Runtime Type Management**:
- **Schema Registry**: Store discovered schemas for each adapter configuration
- **Go Integration**: Use JSON Schema validation libraries for runtime type checking
- **TypeScript Generation**: Generate TypeScript types from JSON Schemas for frontend
- **Field Access Validation**: Processors validate field availability before accessing
- **Schema Versioning**: Track schema changes over time for debugging

**Content Processing Flow**:
```
Raw Content + JSON Schema → Validated Content → Processor Chain

// Go runtime validation
func (p *Processor) Process(content map[string]interface{}, schema *jsonschema.Schema) error {
    if err := schema.Validate(content); err != nil {
        return fmt.Errorf("content validation failed: %w", err)
    }
    // Process with type confidence
}

// TypeScript frontend gets generated types
interface RedditContent {
    id: string;
    title: string;
    selftext?: string;
    score: number;
    // ... generated from JSON Schema
}
```

**Schema Evolution Handling**:
- **Change Detection**: Compare new discoveries with stored schemas
- **Backward Compatibility**: Validate pipelines still work with schema changes
- **User Notification**: Alert when source schemas change unexpectedly
- **Pipeline Migration**: Help users update configurations for new schemas

### Content Object Structure & Pipeline Execution

**ContentItem Structure**: Every piece of content flowing through the system carries:
```go
type ContentItem struct {
    // Core identification
    ID          string                 `json:"id"`
    SourceType  string                 `json:"source_type"`  // "reddit", "rss", etc.
    SourceID    string                 `json:"source_id"`    // Original source identifier
    
    // Raw data with schema
    RawData     map[string]interface{} `json:"raw_data"`
    Schema      *JSONSchema            `json:"schema"`
    
    // Processing metadata
    ProcessedAt time.Time              `json:"processed_at"`
    Pipeline    string                 `json:"pipeline"`     // Which workflow processed this
    
    // Analysis results
    Scores      map[string]float64     `json:"scores"`       // Quality dimensions
    Tags        []string               `json:"tags"`         // Classification labels
    Summary     string                 `json:"summary"`      // LLM-generated summary
    
    // Processing trail
    ProcessingLog []ProcessingStep     `json:"processing_log"`
    Errors        []ProcessingError    `json:"errors"`
}
```

**Workflow Execution Model**:
- **Batch Processing**: Workflows run on cron schedules, process all new content since last successful run
- **Dependency Resolution**: Processors declare input requirements, system validates dependencies before execution
- **Parallel Execution**: Independent processors can run concurrently 
- **State Management**: Track processing checkpoints between runs, support for reprocessing failed items
- **Incremental Updates**: Only process new content since last successful run

**Error Handling Strategy**:
- **Schema Validation**: Reject malformed content before processing
- **LLM Retry Logic**: Automatic retries for failed LLM calls (malformed output, timeouts)
- **Graceful Degradation**: Failed processors skip items, pipeline continues
- **Error Reporting**: Processing failures logged with context for user review
- **Recovery Mechanisms**: Items can be reprocessed individually or in failed batches

**Processing Flow**:
```
New Content Batch → Schema Validation → Process Item 1 → [Success/Retry/Skip] → 
Process Item 2 → [Success/Retry/Skip] → ... → 
Generate Report → Update State → Schedule Next Run
```

**Schema-Aware Processing Benefits**:
- **Early Validation**: Catch data issues before expensive LLM processing
- **Field-Specific Rules**: Users can create rules based on known schema fields
- **Type-Safe Templating**: Generate outputs using validated field access
- **Debugging Support**: Clear errors when accessing undefined fields

### LLM Integration & Prompting
**Question: How do we prompt the LLM?**

**Structured Prompt System**: Templated, configurable LLM interactions
- **Prompt Templates**: Modular prompts for different assessment types
- **Context Management**: Relevant content history and user preferences
- **Output Parsing**: Structured JSON responses with confidence scores
- **Model Abstraction**: Support multiple LLM providers and local models
- **Prompt Versioning**: Track prompt changes and their impact on quality

**Assessment Dimensions**:
- **Substantiveness**: Evidence-based vs opinion, depth of analysis
- **Constructiveness**: Contributes to discourse vs inflammatory  
- **Novelty**: New insights vs rehashing common points
- **Accuracy**: Factual correctness and logical consistency
- **Relevance**: Alignment with pipeline topic focus

### Quality Measurement & Feedback System
**Question: How do we measure the success of the software?**

**Multi-Layered Evaluation Framework**:

#### 1. Automated Benchmarking
- **Golden Datasets**: Curated test sets with human-labeled quality scores
- **Cross-Model Validation**: Compare assessments across different LLM sizes
- **Consistency Testing**: Same content, multiple evaluation runs
- **Prompt A/B Testing**: Systematic prompt optimization with measurable outcomes

#### 2. User Feedback Integration  
- **Explicit Ratings**: Users rate digest quality and individual content selections
- **Implicit Signals**: Reading time, sharing behavior, content engagement
- **Correction Interface**: Users can override filtering decisions with explanations
- **Preference Learning**: System adapts to user-specific quality definitions

#### 3. Pipeline Performance Analytics
- **Filter Effectiveness**: Precision/recall on user-defined quality thresholds
- **Processing Latency**: Time spent in each pipeline stage
- **Resource Utilization**: Cost per content item processed
- **Error Rates**: Failed processing attempts and failure reasons

#### 4. Comprehensive LLM Processor Evaluation
**Critical MVP Component**: Benchmarking across all LLM-powered pipeline stages

**Multi-Stage LLM Benchmarking**:
- **Image Recognition Performance**: Accuracy of visual content analysis across model sizes
- **Quality Assessment Consistency**: Reliability of content scoring with different models
- **Summarization Quality**: Coherence and insight quality in LLM-generated summaries
- **Processing Speed**: Latency impact of model size on each pipeline stage
- **Error Rate Analysis**: Frequency of malformed outputs requiring retries

**Pipeline-Wide Evaluation**:
- **End-to-End Quality**: Final output quality vs. individual processor performance
- **Bottleneck Identification**: Which LLM stages benefit most from larger models
- **Cost-Effectiveness**: Quality improvements vs. computational cost at each stage
- **Prompt Optimization**: How prompt engineering affects performance across model sizes

**Benchmark Scenarios**:
```
Test Pipeline Configurations:
1. All 7B models vs. All 13B models vs. Mixed sizing
2. Different prompt strategies per model size
3. Error handling effectiveness across model capabilities
4. Processing throughput with different model combinations
```

## 4. Technology Stack & Architecture

### Monorepo Structure
**Decision**: Single repository containing both backend and frontend for:
- **Shared Type System**: JSON Schema generates TypeScript types for both Go and frontend
- **Atomic Deployments**: Pipeline engine and management UI deployed together
- **Development Velocity**: Tight integration between pipeline configuration and web interface
- **Unified Docker Deployment**: Single-command setup for complete system

### Backend (Go)
- **Go**: Pipeline orchestration, high-performance content processing
- **BadgerDB**: Pipeline state, content storage, benchmark results
- **YAML/JSON**: Pipeline configuration and prompt templates
- **Echo**: HTTP API for frontend communication
- **JSON Schema**: Runtime validation and TypeScript type generation

### Frontend (TypeScript/React)
- **NextJS**: UI Framework
- **TailwindCSS**: Utility-first styling for rapid UI development

### LLM Integration
- **OpenAI-compatible API**: Support for multiple LLM providers through standardized interface
- **Model Abstraction**: Pluggable backends (Ollama, OpenAI, local models)
- **Prompt Templates**: Configurable prompts per component type and model
- **Response Parsing**: Structured JSON output handling with validation

### Benchmarking Infrastructure  
- **Test Harness**: Automated pipeline testing and model comparison
- **Metrics Collection**: Detailed performance and quality tracking
- **Reporting Dashboard**: Visual pipeline performance analytics (React-based)

### Deployment & DevOps
- **Docker Compose**: Unified deployment of backend, frontend, and dependencies
- **Multi-stage Builds**: Optimized containers for production deployment
- **Shared Volumes**: Configuration and data persistence across services

## 5. MVP Iterations

### Iteration 1: Core Workflow Engine + Basic UI
- **Backend**: Basic source system (Reddit, RSS) implementing ContentItem structure
- **Backend**: Configurable processor chain with Sources, Processors, Evaluators architecture
- **Backend**: LLM quality assessment integration (Evaluators)
- **Frontend**: Basic React app with workflow status monitoring
- **DevOps**: Docker Compose setup for unified deployment
- **Integration**: JSON Schema to TypeScript type generation

### Iteration 2: Workflow Configuration Interface
- **Frontend**: Visual workflow configuration interface supporting component hierarchy
- **Frontend**: Real-time workflow execution monitoring with processing trail visualization
- **Backend**: Workflow configuration API endpoints with YAML/JSON support
- **Backend**: WebSocket support for real-time updates
- **Integration**: Shared configuration validation using workflow framework structure

### Iteration 3: Advanced Workflow Components
- **Backend**: Router implementation for flow control and content path routing
- **Backend**: Aggregator components for content collection and grouping
- **Backend**: Formatter components for multiple output generation
- **Frontend**: Component-aware configuration interface for Routers, Aggregators, Formatters
- **Integration**: End-to-end workflow execution with all component types

### Iteration 4: Benchmarking Foundation & Management
- **Backend**: Golden dataset creation tools
- **Backend**: Multi-model comparison framework across workflow components
- **Frontend**: Benchmarking dashboard and workflow performance visualizations
- **Frontend**: Performance metrics and analytics interface
- **Integration**: Real-time benchmark result streaming and workflow optimization suggestions

## 6. Success Metrics

### Pipeline Quality
- **Filter Accuracy**: >85% precision on curated test datasets
- **User Satisfaction**: >4.0/5.0 average rating on generated digests
- **Model Efficiency**: Acceptable quality with 7B models vs 70B+ models

### System Performance  
- **Processing Reliability**: <5% failure rate in content processing
- **Configuration Usability**: Users can create working pipelines in <30 minutes
- **Benchmark Coverage**: Comprehensive evaluation across 3+ model sizes

### Platform Capability
- **Pipeline Flexibility**: Support for 5+ different content processing workflows
- **Model Compatibility**: Validated performance across 3+ LLM families
- **Feedback Loop Effectiveness**: Measurable improvement in pipeline quality over time

---

This architecture prioritizes the pipeline engine design and feedback mechanisms essential for building a production-quality content curation platform.
