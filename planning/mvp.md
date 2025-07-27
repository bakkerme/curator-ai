# MVP – Curation Flow Runner

## 1. Purpose
The MVP is designed to build out the core Curation Flow Runner, getting to a point where it can load in a Curator Document, parse it and do the work.

## 2. Scope
The scope is solely the flow runner, with no UI, auth or evaluation.

### In-Scope
- Curator Document parser
- Assembles the flow internally
- Triggers
    - Cron Trigger
- Sources
    - Reddit
- Quality
    - Quality Rule
    - LLM
- Summarisers
    - LLM
- Run Summariser
    - LLM
- Output
    - Email

### Out-of-scope
- Auth
- Dashboard
- Evaluation

## 3. Goals & Success Criteria
Goals:
- Gain a better understanding of how to construct a workflow system
- Validate the block system design and type system
- Compare a working version with the AI News Processor

## 4. Assumptions

## 5. Technology Choice
* Application
    * Go (latest)
    * Echo
* Infrastructure
    * Docker
    * Alpine Linux

## 6. Key Components
### 6.1 Block System Design
See #./planning/td.md for an overview of the block, and #./planning/block_design.md for specs. These should be represented as Go types.

### 6.2 Runner Orchestrator
Minimal system to instantiate the required services to start the flow.

### 6.3 Curator Document Parser
The document is in yaml, so the parser should:
- Parse the document
- Validate it's contents are accurate and well formed
- Convert the document into an internal format that contains:
    - Each Processor needed, and the order to execute them in

#### 6.4 Curator Flow Runner
1. Wait till a Trigger criteria is met
1. Load a batch from the Source, with any enrichment required
2. Send post to next Processor in the list
3. Process each Post into each Processor, one by one
4. If the next Processor is a Run Summary Processor, wait until every Post in the batch is processed up to the Run Summary
5. If the next step is an Output, execute the output step with all Posts up until this point

### 6.5 Reddit Source
- Use the existing Reddit source from AI News Processor as the basis for this
- Outputs a PostBlock for each Reddit post

## 6.6 LLM Foundation
- Provides the backbone for LLM-based Processors including:
    - LLM Quality
    - LLM Summary
    - LLM Run Summary
- Uses an OpenAI-compatible API

### 6.7 Output Processor (Email)
- Processes Post inputs, inserting them into a defined email template via Go's built in templating library
- Use SMTP details defined via config

### 6.8 Logging & Observability
- Centralised structured logging (zap or logrus) with log levels configurable via env vars  
- Basic metrics (Prometheus exporter) for trigger runs, processor latencies and error counts

### 6.9 Configuration & Secrets
- Config via environment variables + optional `.env` file  
- Secret values (API keys, SMTP creds) injected at runtime, never committed to VCS

### 6.10 Error Handling & Retry Logic
- Standard error type hierarchy for processors  
- Auto-retry for LLM parse fails with no backoff time
- Exponential backoff and retry for other fail types

### 6.11 Testing Strategy
- Unit tests for individual processors (Go’s `testing` pkg)  
- Integration test that loads a sample Curator Document and runs the entire flow in-memory  
- GitHub Actions workflow to run `go test ./...`