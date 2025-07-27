# MVP – Curation Flow Runner

## 1. Purpose
The MVP is designed to build out the core Curation Flow Runner, getting to a point where it can load in a Curator Document, parse it and do the work.

## 2. Scope
The scope is solely the flow runner, with no UI, auth or evaluation

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

#### Out-of-scope
- Auth
- Dashboard
- Evaluation

## 3. Goals & Success Criteria
Goals:
- Gain a better understanding of how to construct a workflow system
- Validate the block system design and type system
- Compare a working version with the AI News Processor

## 4. Assumptions

## 5. Architecture Snapshot
### 5.1 Minimal Services
One single Docker container with a bespoke Go application.

### 5.2 Data Flow Overview

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

#### 6.4 Design Processor Data Type


### 6.4 Reddit Source
- Use the existing Reddit source from AI News Processor as the basis for this
- 

### 6.2 Processor Interfaces
### 6.3 Trigger Scheduler
### 6.4 Storage Layer
### 6.5 Output Adapter (Email)