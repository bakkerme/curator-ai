# Curator MVP Technical Document

## 1. MVP Focus: Pipeline Engine Architecture

**Core objective:** Design and build a flexible, configurable pipeline engine that can ingest content, process it through multiple AI-powered stages, and provide comprehensive feedback mechanisms to optimize performance.

**Key design principles:**
- **Modularity**: Each pipeline stage is a discrete, configurable component
- **Measurability**: Every processing decision is logged and can be evaluated
- **Adaptability**: Pipeline configurations can be iteratively improved based on feedback
- **Transparency**: Users understand exactly why content was filtered or promoted

## 2. Pipeline Architecture Overview

```
Content Sources → Data Processing Pipeline → Quality Assessment → Output Generation
       ↓                    ↓                      ↓               ↓
   [Adapters]          [Processors]           [Evaluators]    [Formatters]
       ↓                    ↓                      ↓               ↓
                    Feedback & Benchmarking System
                           ↓
                    Pipeline Optimization
```

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

### Data Processing Pipeline
**Question: What are the various data processing steps?**

**Flexible Processor Chain**: Configurable sequence with dependency management
1. **Content Extraction**: Clean text, extract entities, parse structure
2. **Algorithmic Filtering**: Rule-based quality checks (length, spam patterns, etc.)
3. **Enrichment**: Add context, resolve links, fetch additional metadata  
4. **Image Recognition**: LLM-powered analysis of visual content
5. **Quality Assessment**: Multi-dimensional LLM scoring
6. **Content Summarization**: LLM-generated summaries and key insights
7. **Final Filtering**: Rule-based decisions based on LLM outputs
8. **Custom Processors**: User-defined processing logic

**Pipeline Execution Model**:
- **Dependency-aware**: Processors declare input requirements
- **Fault-tolerant**: Failed processors skip individual items, continue pipeline
- **Retry Logic**: LLM failures trigger automatic retries with backoff
- **Error Tracking**: All processing failures logged for analysis
- **Flexible Ordering**: Pipeline validates dependencies but allows custom flows

**Example Multi-LLM Pipeline**:
```
Reddit Post → Text Extraction → Spam Filter → 
Image Analysis (LLM) → Quality Assessment (LLM) → 
Summary Generation (LLM) → Final Rules Filter → Output
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

### Pipeline Execution & Error Handling

**Cron-Based Processing**: Pipeline runs on scheduled intervals
- **Batch Processing**: Each run processes all available new content
- **State Management**: Track processing state between runs
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
- **Gin/Fiber**: HTTP API for frontend communication
- **JSON Schema**: Runtime validation and TypeScript type generation

### Frontend (TypeScript/React)
- **React**: Management interface for pipeline configuration and monitoring
- **TypeScript**: Type-safe development with generated types from backend schemas
- **Vite**: Fast development and build tooling
- **TailwindCSS**: Utility-first styling for rapid UI development
- **React Query**: Efficient API state management and caching

### LLM Integration
- **Ollama**: Local model serving with multiple model support
- **OpenAI API**: Fallback for benchmark comparisons
- **Custom Adapters**: Pluggable LLM provider interface

### Benchmarking Infrastructure  
- **Test Harness**: Automated pipeline testing and model comparison
- **Metrics Collection**: Detailed performance and quality tracking
- **Reporting Dashboard**: Visual pipeline performance analytics (React-based)

### Deployment & DevOps
- **Docker Compose**: Unified deployment of backend, frontend, and dependencies
- **Multi-stage Builds**: Optimized containers for production deployment
- **Shared Volumes**: Configuration and data persistence across services

## 5. MVP Iterations

### Iteration 1: Core Pipeline Engine + Basic UI
- **Backend**: Basic adapter system (Reddit, RSS)
- **Backend**: Configurable processor chain
- **Backend**: LLM quality assessment integration
- **Frontend**: Basic React app with pipeline status monitoring
- **DevOps**: Docker Compose setup for unified deployment
- **Integration**: JSON Schema to TypeScript type generation

### Iteration 2: Pipeline Configuration Interface
- **Frontend**: Visual pipeline configuration interface
- **Frontend**: Real-time pipeline execution monitoring
- **Backend**: Pipeline configuration API endpoints
- **Backend**: WebSocket support for real-time updates
- **Integration**: Shared configuration validation

### Iteration 3: Benchmarking Foundation
- **Backend**: Golden dataset creation tools
- **Backend**: Multi-model comparison framework
- **Frontend**: Benchmarking dashboard and visualizations
- **Frontend**: Performance metrics and analytics interface
- **Integration**: Real-time benchmark result streaming

### Iteration 4: Advanced Features & Polish
- **Frontend**: User rating interface for content quality
- **Frontend**: Advanced pipeline analytics and optimization suggestions
- **Backend**: Automated prompt optimization
- **Backend**: Model size vs. quality analysis tools
- **DevOps**: Production-ready deployment configurations

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