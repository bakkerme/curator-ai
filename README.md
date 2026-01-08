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
- `OPENAI_TEMPERATURE` (optional, default: provider default; overridden by per-processor YAML `temperature`)
- `OPENAI_TOP_P` (optional; overridden by per-processor YAML `top_p`)
- `OPENAI_PRESENCE_PENALTY` (optional; overridden by per-processor YAML `presence_penalty`)
- `OPENAI_TOP_K` (optional; overridden by per-processor YAML `top_k`)

### Email Output (SMTP)
- `SMTP_HOST` (required unless set in YAML)
- `SMTP_PORT` (optional, default: `587`)
- `SMTP_USER` (optional, depends on provider)
- `SMTP_PASSWORD` (optional, depends on provider)
- `SMTP_USE_TLS` (optional, default: `true`)

### HTTP Source Settings
- `REDDIT_HTTP_TIMEOUT` (optional, e.g. `10s`)
- `REDDIT_USER_AGENT` (optional, default: `curator-ai/0.1`)
- `REDDIT_CLIENT_ID` (optional; when set with `REDDIT_CLIENT_SECRET`, Curator uses the Reddit API instead of the public `.json` endpoint)
- `REDDIT_CLIENT_SECRET` (optional; required with `REDDIT_CLIENT_ID`)
- `REDDIT_USERNAME` (optional; if set with `REDDIT_PASSWORD`, uses password grant instead of client credentials)
- `REDDIT_PASSWORD` (optional; required with `REDDIT_USERNAME`)
- `RSS_HTTP_TIMEOUT` (optional, e.g. `10s`)
- `RSS_USER_AGENT` (optional, default: `curator-ai/0.1`)

### Jina Reader (URL → markdown)
- `JINA_API_KEY` (required to use the Jina Reader client)
- `JINA_BASE_URL` (optional, default: `https://r.jina.ai/`)
- `JINA_HTTP_TIMEOUT` (optional, e.g. `15s`)
- `JINA_USER_AGENT` (optional, default: `curator-ai/0.1`)

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

## Local Email Dev (Mailpit)

Run Mailpit (SMTP sink + web UI/API):

```bash
docker compose -f docker-compose.yml up -d
open http://localhost:8025
```

## Docker (Test Deploy)

Build and run Curator + Mailpit:

```bash
docker compose up -d --build curator mailpit
```

Notes:
- The `curator` service expects `./curator.yaml` mounted at `/app/curator.yaml`.
- Provide any required runtime env vars (e.g. `OPENAI_API_KEY`) via your shell or a `.env` file.
- SMTP is pre-wired to Mailpit inside Compose (`SMTP_HOST=mailpit`, port `1025`, TLS off).

## Docker (Dev Container with Reload)

Run the reloadable dev container (uses `air` inside the container):

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

This mounts the repo into the container and rebuilds/restarts on `.go` and `.yml/.yaml` changes.

E2E test (local RSS fixture → Curator → Mailpit API assertion):

```bash
CURATOR_E2E=1 go test -tags=e2e ./internal/e2e -run TestMailpitE2E
```
