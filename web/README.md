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

Optional CLI command override:

- `SCRAPE_CODEX_COMMAND` (default `codex`)

Example:

```bash
SCRAPE_ORCHESTRATOR_MODE=cli SCRAPE_CODEX_COMMAND=codex npm run dev
```

## Checks

```bash
npm run lint
npm run test
```
