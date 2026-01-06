# Curator AI — Agent Notes

This repository is a Go-based MVP “flow runner” for Curator AI. It loads a Curator Document (YAML), builds a processor pipeline (trigger → sources → quality → summaries → output), and executes runs on a schedule or once.

## Start Here

- Project overview and env vars: `README.md`
- Go version/toolchain: `go.mod` (currently `go 1.24.4`)
- Curator Document YAML reference: `planning/curator_document_spec.md`
- Example Curator Document: `planning/example_flow.yml` (treat as a starting point; it may need required fields like `output.email.from`)
- Runner execution model: `planning/runner_logic.md`
- Block model notes: `planning/block_design.md` and `internal/core/blocks.go`

## Repository Structure

- Entrypoint (CLI):
  - `cmd/curator/main.go`: loads YAML (`-config` / `CURATOR_CONFIG`), builds a `core.Flow`, and runs (`-run-once` / cron trigger).
- Core domain types:
  - `internal/core/processor.go`: processor interfaces (Trigger/Source/Quality/Summary/RunSummary/Output).
  - `internal/core/flow.go`: `Flow` and `Run` models.
  - `internal/core/blocks.go`: `PostBlock`, `RunSummary`, and related structs.
- Curator Document parsing and validation:
  - `internal/config/schema.go`: YAML schema structs, template reference resolution, and validation.
- Orchestration:
  - `internal/runner/runner.go`: executes sources → quality → post summaries → run summaries → outputs; logs stage timing and block counts.
  - `internal/runner/factory/factory.go`: wires concrete processors from env (LLM client, fetchers, SMTP sender).
- Concrete implementations:
  - `internal/processors/trigger`: cron trigger processor.
  - `internal/processors/source`: Reddit + RSS processors (wrap fetchers).
  - `internal/processors/quality`: rule-based quality (`expr`) + LLM quality.
  - `internal/processors/summary`: LLM + markdown-to-HTML processors (post + run).
  - `internal/processors/output`: email output processor.
  - `internal/sources/*`: fetcher implementations + mocks (Reddit, RSS).
  - `internal/llm/*`: LLM client abstraction and OpenAI-compatible implementation.
  - `internal/outputs/email/*`: email templating/sending plumbing (SMTP).
- Observability:
  - `internal/observability/otelx/otelx.go`: OpenTelemetry setup (OTLP exporter).

## Common Commands

- Run once:
  - `go run ./cmd/curator -config curator.yaml -run-once`
- Run continuously (cron trigger keeps it alive):
  - `go run ./cmd/curator -config curator.yaml`
- Tests:
  - `go test ./...`

## Configuration Notes

### CLI Flags / Envars

`cmd/curator/main.go` supports both flags and env fallbacks:
- `-config` / `CURATOR_CONFIG` (default `curator.yaml`)
- `-flow-id` / `FLOW_ID` (default `flow-1`)
- `-run-once` / `RUN_ONCE` (default `false`)
- `-allow-partial` / `ALLOW_PARTIAL_SOURCE_ERRORS` (default `false`)

### Runtime Dependencies (Env)

Factory wiring lives in `internal/runner/factory/factory.go`:
- LLM:
  - `OPENAI_API_KEY` (required when using LLM processors)
  - `OPENAI_BASE_URL` (optional; OpenAI-compatible endpoint)
  - `OPENAI_MODEL` (optional; default `gpt-4o-mini`)
- Reddit:
  - `REDDIT_HTTP_TIMEOUT` (duration; default `10s`)
  - `REDDIT_USER_AGENT` (default `curator-ai/0.1`)
  - Optional API creds (enables API mode vs public `.json`):
    - `REDDIT_CLIENT_ID`, `REDDIT_CLIENT_SECRET`
    - `REDDIT_USERNAME`, `REDDIT_PASSWORD` (optional password grant)
- RSS:
  - `RSS_HTTP_TIMEOUT` (duration; default `10s`)
  - `RSS_USER_AGENT` (default `curator-ai/0.1`)
- Email (SMTP):
  - `SMTP_HOST` (required unless set in YAML)
  - `SMTP_PORT` (default `587`), `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_USE_TLS` (default `true`)

### Templates

Curator Documents can define reusable templates and reference them by ID.
- Template definitions live under `templates:` in the YAML.
- Processors can reference templates by ID (e.g. `prompt_template: myTemplate`).
- Resolution and validation lives in `internal/config/schema.go`.

## Observability (Optional)

`internal/observability/otelx/otelx.go` enables OTEL tracing when:
- `CURATOR_OTEL_ENABLED=true` (or `OTEL_EXPORTER_OTLP_ENDPOINT` is set)
- Configure exporter via standard OTEL env vars like `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_PROTOCOL`, `OTEL_EXPORTER_OTLP_HEADERS`.

## Contribution Tips for Agents

- Prefer adding/adjusting processors by following the existing pattern:
  - define config in `internal/config/schema.go` → implement processor in `internal/processors/...` → wire in `internal/runner/factory/factory.go`.
- Keep secrets out of the repo; prefer env vars or an ignored `.env`/`.envrc`.
- Avoid touching workspace caches like `.gocache/`, `.gomodcache/`, `.gopath/` (they are ignored by `.gitignore`).
- Don't scatter os.Getenv around the codebase. Use a 
