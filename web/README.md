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
- SDK mode (server-side TypeScript runner): `SCRAPE_ORCHESTRATOR_MODE=sdk`

In SDK mode, the web server calls Codex directly through the official
`@openai/codex-sdk` package. The prompt lives in the TypeScript runner:

- Runner file: `web/lib/orchestrator/codexSdkRunner.ts`
- Top-of-file variable to edit quickly:
  - `promptTemplate`

Optional runtime overrides:

- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`
- `SCRAPE_CODEX_MODEL`
- `SCRAPE_CODEX_WORKING_DIRECTORY` (defaults to the repo root)
- `SCRAPE_CODEX_WEB_SEARCH` (`live` or unset)

Example:

```bash
SCRAPE_ORCHESTRATOR_MODE=sdk SCRAPE_CODEX_MODEL=gpt-5.4-mini npm run dev
```

When you open a run page, you now get:

- A live timeline for high-level stage changes
- An agent console panel that streams Codex SDK events and command output as the run progresses

## Checks

```bash
npm run lint
npm run test
```
