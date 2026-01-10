# Curator AI — Agent Notes

This file is intentionally a short index. The authoritative behavior and design notes live in the docs in this repo.

## Start Here

- Product + setup overview (env vars, examples): [README.md](README.md)
- Curator Document (YAML) spec: [docs/curator_document_spec.md](docs/curator_document_spec.md)
- MVP/architecture notes: [docs/mvp.md](docs/mvp.md)
- Runner execution model (stage ordering): [docs/runner_logic.md](docs/runner_logic.md)
- Block model / terminology: [docs/block_design.md](docs/block_design.md)
- Additional design notes / drafts:
  - [docs/curator_pd_v2.md](docs/curator_pd_v2.md)
  - [docs/td.md](docs/td.md)

## How To Run

- Go toolchain version is defined in: [go.mod](go.mod)
- Run once (single execution of the flow):
  - `go run ./cmd/curator -config curator.yaml -run-once`
- Run continuously (cron trigger keeps it alive):
  - `go run ./cmd/curator -config curator.yaml`
- Run tests:
  - `go test ./...`

## Where Key Behavior Lives (Code Map)

- CLI entrypoint / flags / env fallbacks: [cmd/curator/main.go](cmd/curator/main.go)
- Curator Document parsing + validation + template resolution: [internal/config/schema.go](internal/config/schema.go)
- Environment variable helpers (prefer this over `os.Getenv`): [internal/config/env.go](internal/config/env.go)
- Core domain types:
  - Processor interfaces: [internal/core/processor.go](internal/core/processor.go)
  - Flow/run models: [internal/core/flow.go](internal/core/flow.go)
  - Block types (posts, summaries): [internal/core/blocks.go](internal/core/blocks.go)
- Orchestration (the actual stage pipeline execution): [internal/runner/runner.go](internal/runner/runner.go)
- Wiring concrete processors from config + env: [internal/runner/factory/factory.go](internal/runner/factory/factory.go)

## Processors (Where To Implement Things)

- Triggers (cron): [internal/processors/trigger](internal/processors/trigger)
- Sources (Reddit, RSS): [internal/processors/source](internal/processors/source)
- Quality gates (rule/expr + LLM): [internal/processors/quality](internal/processors/quality)
- Summaries (post + run, including markdown → HTML): [internal/processors/summary](internal/processors/summary)
- Outputs (email): [internal/processors/output](internal/processors/output)

## Integrations

- LLM abstraction + OpenAI-compatible client: [internal/llm](internal/llm)
- Source fetchers + mocks:
  - Reddit: [internal/sources/reddit](internal/sources/reddit)
  - RSS: [internal/sources/rss](internal/sources/rss)
- Email plumbing (templating + SMTP sender): [internal/outputs/email](internal/outputs/email)

## Observability

- OpenTelemetry setup: [internal/observability/otelx/otelx.go](internal/observability/otelx/otelx.go)
- Notes: enable via env (see [README.md](README.md) and standard OTEL env vars).

## Coding Standards (Project-Specific)

- Keep configuration/env access centralized in [internal/config/env.go](internal/config/env.go) (avoid sprinkling `os.Getenv`).
- Prefer the existing processor pattern:
  - add schema config in [internal/config/schema.go](internal/config/schema.go)
  - implement processor under [internal/processors](internal/processors)
  - wire it in [internal/runner/factory/factory.go](internal/runner/factory/factory.go)
- Keep secrets out of the repo; use env vars / local `.env` / `.envrc` (gitignored).
