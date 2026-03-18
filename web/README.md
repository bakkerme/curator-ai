# Curator Web (v0)

Initial vertical slice of the scrape block generator workflow from `docs/web_ui_scrape_block_planning.md`.

## Run

```bash
cd web
npm install
npm run dev
```

Then open <http://localhost:3000>.

## Agent-layer mode

The run timeline/events route now uses an orchestrator abstraction.

- Default (safe local testing): `SCRAPE_ORCHESTRATOR_MODE=mock`
- CLI mode (first real adapter): `SCRAPE_ORCHESTRATOR_MODE=cli`

In CLI mode, the Node orchestrator calls a Go bridge command that keeps prompt and
command configuration in one place:

- Go bridge file: `cmd/scrape-agent/main.go`
- Top-of-file variables to edit quickly:
  - `codexExecCommand`
  - `promptTemplate`

Optional runtime overrides:

- `SCRAPE_AGENT_BRIDGE_COMMAND` (default `go`)
- `SCRAPE_AGENT_BRIDGE_ARGS` (default `run ../cmd/scrape-agent`)
- `SCRAPE_CODEX_COMMAND` (forwarded to bridge `-command`)
- `SCRAPE_CODEX_PROMPT_TEMPLATE` (forwarded to bridge `-prompt-template`)

Example:

```bash
SCRAPE_ORCHESTRATOR_MODE=cli SCRAPE_CODEX_COMMAND=codex npm run dev
```

## Checks

```bash
npm run lint
npm run test
```
