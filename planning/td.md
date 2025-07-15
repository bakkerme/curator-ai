# Technical Design Template

## 1. Overview

### System Purpose

Curator AI is an AI-driven Personal Intelligence Platform that is designed to pull news, posts and feeds into a simple, customisable workflow engine. By using AI, we can filter, summarise and process information, drawing out the most meaningful insights and filtering out brainrot.

### Key Requirements

- Focused workflow engine, specifically for processing posts and feeds
- AI options for filtering and summarisation
- Workflow can be entirely built automatically using AI
- Email, Slack and MCP output options
- Operational with small, open weight LLMs
- Self-hostable
- Automatic Evalutiona Suite

### Success Criteria

- The user can provide a single URL plus an email address and the service will spin up an entire workflow for harvesting the data. If the site is a feed or contains news, this should work 100% of the time.
- The system can be 100% self hosted, with no external LLM infrastructure or APIs if the user doesn't want to

## 2. Architecture Overview

### High-Level Components

- UI
  - Login, Account Management etc.
  - Curation Flow List
  - Curation Flow Builder
  - Settings
  - Logs
  - Evaluation
- Backend
  - CRUD
  - Curation Flow Creator
  - Curation Flow Runner
  - Evaluation Runner
  - Log Manager

```
[Component A] --> [Component B] --> [Component C]
     |                 |                 |
     v                 v                 v
[Storage A]       [Storage B]       [Output]
```

## 3. Core Components

#### Curation Flow Engine

The Curation Flow Engine consists of the Parser, Execution Framework and the code for each Processor.

The Parser is designed to take in a Curator Document and convert it into an internal structure that includes the correct configuration for the Execution Framework, with the needed Processors attached. Once set up, the Curation Flow is set to be in Waiting state, until the required conditions for the Trigger is met, which will then start the loading from the Source and further process the data output it via the Output Processor.

**Responsibilities:**
- Accept a Curator Document
- Configure and stage a Curation run
- Execute a Curation run
- Handle requests to the LLM
- Send data to the User for most Outputs, store in cache for MCP

**Interfaces:**
- Input: Curator Document (see @./curator_doc_spec.yml), Runtime control (Start, Stop, etc)
- Output: Provides monitoring

#### Curation Flow Creator

The Curation Flow Creator is an AI-powered tool, designed to take a URL for a Website or Platform, and apply a number of Rule and AI-powered analysis to produce a Curator Document that curates the data from the website for the user.

**Responsibilities:**
- Accepts a URL initially, then lets the User specify other behaviour, like Quality metrics, Summarisation rules and required Outputs
- Outputs a valid Curator Document that successfully curates data from the site the user provided

### Component B: [Name]

**Purpose:** [What this component does]

**Responsibilities:**

- [Responsibility 1]
- [Responsibility 2]

**Interfaces:**

- Input: [Data format/API]
- Output: [Data format/API]

**Implementation Notes:**

- [Key technical decision]
- [Alternative considered]

## 4. Data Models

### Primary Entities

```
Entity A {
  id: string
  field1: type
  field2: type
  relationships: []Entity
}

Entity B {
  id: string
  field1: type
  field2: type
}
```

### Data Relationships

- [Entity A] has many [Entity B]
- [Entity B] belongs to [Entity A]

## 5. Curator Documents

Curator AI's Curator Document is a YAML document providing a declarative document on where data is loaded from and how it is curated. The document is loosely temporal, with each individual Processor configured in each catagory being run one by one.

Trigger Processors - Define when processing runs
Source Processors - Fetch and ingest data
Quality Processors - Filter and evaluate content
Summary Processors - Transform and summarize content
Output Processors - Deliver results

See @./planning/example_flow.yml for an example.

## 5. API Design

### Core Endpoints

#### Error Logging

```
POST /api/v1/log/error - Log an error
{
    "error": "Some error",
    "stack": "Stack trace"
}
```

## 6. Technology Stack

### Core Technologies

#### Front End

- **Language:** Typescript
- **Framework:** NextJS

#### Back End

- **Language:** Go
- **Framework:** Echo
- **Database:** PostgreSQL
- **Cache:** Redis

### Infrastructure

- **Deployment:** Docker Compose
- **Hosting:** Self-Hosted
- **Monitoring:** Internal Monitoring Dashboard

## 7. Security Considerations

### Authentication & Authorization

- **Auth mechanism**: JWT (JSON Web Tokens) with middleware-based validation
  - Alternative: Session-based auth with secure cookies
  - OAuth 2.0/OpenID Connect for third-party integration
  - API keys for service-to-service communication
  - github.com/golang-jwt/jwt/v5
- **Permission model**:
  - Resource + User-level permissions with middleware guards

## 8. Error Handling & Monitoring

### Error Handling Strategy

- **Error Classification:**

  - System Errors: Database failures, network timeouts, service unavailability
  - LLM Errors: Model failures, rate limits, malformed responses, timeout errors
  - Data Errors: Invalid feeds, parsing failures, malformed content
  - User Errors: Invalid configurations, authentication failures
  - Workflow Errors: Step failures, dependency issues, validation errors
- **Error Response Format:**

  - Structured JSON with error code, message, context, and correlation ID
  - Stack traces captured for system errors (development/debug mode)
  - User-friendly messages for client-facing errors
- **Retry Mechanisms:**

  - Exponential backoff for LLM requests with configurable max attempts
  - Circuit breaker pattern for external service calls
  - Immediate retry for transient network errors

### Monitoring & Observability

- **Metrics:**
  - LLM-specific metrics for evaluation engine
- **Logging:**
  - Runtime Logs: Each step in the flow should produce logs of each activity, including data. Full data logs are stored for the most recent run.
  - Evaluation Logs: Evaluation Logs are used for validating the quality of LLM outputs. The data should be both input to the LLM, and the output from it
  - Error Logs: Front end is captured from the /log/error endpoint, while backend is captured and stored.

## 9. Testing Strategy

### Test Levels

- **Unit Tests:** Standard Unit Tests, Front and Backend.
- **Integration Tests:** Use integration tests where appropriate.
- **End-to-End Tests:** For critical user journeys.

### Test Data Management

- Avoid any testing with real LLMs, always mock data

## 11. Deployment & Operations

### Deployment Strategy

- Use Docker Compose with Docker images available on GitHub Container Registry

### Configuration Management

- Environment Variables or .env config only
