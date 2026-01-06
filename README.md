# Curator AI (MVP)

This repo contains the MVP runner for Curator AI. It parses a Curator Document YAML and executes the flow using configured sources, processors, and output.

## Quick Start

1. Create a `curator.yaml` in the repo root (or set `CURATOR_CONFIG`).
2. Set the required environment variables (see below).
3. Run the entrypoint.

```bash
go run ./cmd/curator -config curator.yaml -run-once
```

## Required Environment Variables

### LLM (OpenAI-compatible)
- `OPENAI_API_KEY` (required for LLM processors)
- `OPENAI_BASE_URL` (optional for OpenAI-compatible endpoints)
- `OPENAI_MODEL` (optional, default: `gpt-4o-mini`)

### Email Output (SMTP)
- `SMTP_HOST` (required unless set in YAML)
- `SMTP_PORT` (optional, default: `587`)
- `SMTP_USER` (optional, depends on provider)
- `SMTP_PASSWORD` (optional, depends on provider)
- `SMTP_USE_TLS` (optional, default: `true`)

### HTTP Source Settings
- `REDDIT_HTTP_TIMEOUT` (optional, e.g. `10s`)
- `RSS_HTTP_TIMEOUT` (optional, e.g. `10s`)
- `RSS_USER_AGENT` (optional, default: `curator-ai/0.1`)

## Optional Runtime Flags / Envars

- `CURATOR_CONFIG` (default: `curator.yaml`)
- `FLOW_ID` (default: `flow-1`)
- `RUN_ONCE` (default: `false`)
- `ALLOW_PARTIAL_SOURCE_ERRORS` (default: `false`)

## Example (Run Once)

```bash
export OPENAI_API_KEY="..."
export SMTP_HOST="smtp.example.com"
export SMTP_USER="user"
export SMTP_PASSWORD="pass"

go run ./cmd/curator -config curator.yaml -run-once
```
