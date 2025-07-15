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

### Component A: [Name]
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
Curator AI's Curator Document is a YAML document providing a declarative document on where data is loaded from and how it is parsed. The document is loosely temporal, with each individual Processor configured in each catagory being run one by one. 

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

## 7. Error Handling & Monitoring

### Error Handling Strategy
- 
- [Error classification]
- [Error response format]
- [Retry mechanisms]

### Monitoring & Observability
- **Metrics:** [Key metrics to track]
- **Logging:** 
  - Runtime Logs: Each step in the flow should produce logs of each activity, including data. Full data logs are stored for the most recent run.
  - Evaluation Logs: Evaluation Logs are used for validating the quality of LLM outputs. The data should be both input to the LLM, and the output from it
  - Error Logs: Front end is captured from the /log/error endpoint, while backend is captured and stored.
- **Alerting:** [Alert conditions]

## 10. Testing Strategy

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