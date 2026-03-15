# Curator Web UI Plan: Scrape Block Generator (v0)

## Goal

Build the first Curator web UI workflow: a guided experience that takes a target blog URL and outputs a working `scrape` block users can paste into a Curator document.

This first workflow should also establish reusable UI/backend patterns so we can expand into a full Curator web product over time.

## Product Scope (v0)

### User outcome

A user can:

1. Enter a target URL.
2. Run an agent-assisted inspection process.
3. See extracted sample results (discovery + extraction).
4. Approve the result.
5. Copy a generated Curator config snippet.

### Non-goals (v0)

- Full Curator document editing in-browser.
- Multi-user collaboration.
- Advanced auth/permissions beyond basic local/dev auth.
- Production-scale distributed job orchestration.

## Why Next.js + Node Backend

Use Next.js (App Router) with built-in Node.js route handlers/actions to keep architecture simple while learning requirements.

This enables:

- Fast iteration with a single deployable app.
- Shared types between frontend/backend.
- Easy progression from local process execution to queued workers later.

## High-level Architecture

```text
[Next.js Frontend]
   |
   | HTTP/SSE
   v
[Next.js API/Server Actions]
   |
   | spawn/stream
   v
[Codex CLI Orchestrator]
   |
   | uses browser automation (Puppeteer)
   v
[Inspection + selector proposal + extraction simulation]
   |
   v
[Results JSON + generated scrape block]
```

## Core Workflow

### Step 1: URL intake

UI form accepts:

- `targetUrl` (required)
- Optional hints:
  - content type (blog/news/docs)
  - preferred item count
  - include/exclude URL patterns

Validation:

- URL must be absolute and HTTP(S).
- Block localhost/private IP ranges for hosted deployments.

### Step 2: Agent run kickoff

Backend creates a `scrape_run` record with status `queued` then `running`.

Server invokes Codex CLI with a constrained prompt template:

- Visit URL with Puppeteer.
- Identify discovery selectors (list pages -> item links).
- Identify extraction selectors (title, url, author, published_at, body/summary).
- Return confidence scores and rationale for each selector.
- Return structured JSON matching a schema.

### Step 3: Runtime observation

Frontend subscribes via SSE/WebSocket for job progress events:

- page loaded
- discovery candidates found
- extraction candidates found
- sample extraction complete
- config generated

### Step 4: Review experience

Show side-by-side:

- Proposed selectors.
- Extracted sample rows (5-20 items).
- Field-level confidence badges.
- Raw HTML snippet preview for problematic fields.

Allow user edits before approval:

- Inline selector edits.
- Re-run extraction preview only (no full rediscovery unless requested).

### Step 5: Approval and output

On user approval:

- Freeze selected selectors/config.
- Render final Curator YAML snippet.
- Provide copy button and optional download.

## Data Contracts

## `ScrapeRun`

- `id`
- `targetUrl`
- `status` (`queued|running|needs_review|approved|failed`)
- `createdAt`, `updatedAt`
- `logs` (streamed events)
- `result` (`ScrapeProposal`)

## `ScrapeProposal`

- `discovery`
  - `listPageSelector`
  - `itemLinkSelector`
  - `paginationSelector` (optional)
- `extraction`
  - fields: `title`, `url`, `author`, `publishedAt`, `summary`, `content`
  - selector + transform hints per field
- `samples`
  - extracted sample records
- `confidence`
  - per selector + overall
- `generatedYaml`

Use Zod schemas both server and client side.

## Codex CLI Integration Pattern

Wrap Codex execution behind a backend interface:

- `runScrapeDiscovery(input): AsyncGenerator<RunEvent>`

Implementation details:

- Use child process spawning (`stdio` streaming).
- Parse structured JSON blocks from stdout.
- Enforce execution timeout and max tokens.
- Restrict tool access to browser + local sandboxed temp dir.

This abstraction allows future swap to a queue worker without UI changes.

## Puppeteer/Browser Strategy

In the Codex prompt contract, require the agent to:

1. Load page and wait for network idle.
2. Detect whether content is SSR vs client-rendered.
3. Gather multiple selector candidates.
4. Validate candidates across multiple items.
5. Prefer robust selectors (semantic attributes, stable classes, structural anchors).
6. Avoid brittle nth-child-only selectors unless unavoidable.

For reliability, include fallback modes:

- Retry with increased wait timeout.
- Scroll/load-more detection.
- Optional JS evaluation for shadow DOM or lazy content.

## UX Blueprint (v0)

### Pages

1. **`/scrape/new`**
   - URL entry + advanced options.
2. **`/scrape/runs/[id]`**
   - Live progress, logs, selector proposals.
3. **`/scrape/runs/[id]/review`**
   - editable selector panel + extraction preview grid.
4. **`/scrape/runs/[id]/result`**
   - final YAML + copy/export.

### Components

- `UrlIntakeForm`
- `RunTimeline`
- `SelectorEditor`
- `ExtractionPreviewTable`
- `YamlOutputPanel`

## Extensibility for Full Curator UI

Design choices now that support broader product evolution:

- Generic `Run` model that can later support RSS, Reddit, quality gates, summaries.
- Event stream infrastructure reusable for all long-running agent tasks.
- Proposal/review/approve pattern reusable across block generators.
- Config output abstraction to later support full document assembly.

## Security & Safety

- URL allow/deny policy (block internal networks, metadata endpoints).
- Navigation/request limits for browser automation.
- Sanitize and bound logs shown in UI.
- Store minimal scraped content in persistence; purge raw payloads by TTL.
- Add explicit user notice for site terms/compliance responsibility.

## Observability (deferred)

For v0, skip dedicated observability/telemetry infrastructure and keep runtime visibility simple via UI run logs and basic server logging.

Revisit full metrics/tracing once we validate the workflow and identify stable operational signals worth instrumenting.

## Suggested Milestones

### Milestone 1: Vertical slice (local dev)

- URL intake -> run -> static mock review -> YAML output.
- No persistence beyond in-memory or sqlite.

### Milestone 2: Real agent integration

- Codex CLI orchestration + Puppeteer-based selection.
- Streaming logs/events in UI.

### Milestone 3: Review loop

- Editable selectors.
- Re-run extraction preview.
- Approval flow + final snippet generation.

### Milestone 4: Hardening

- URL safety restrictions.
- Retry/fallback behavior.

## Open Questions

- Should we persist raw HTML snapshots for debugging, and for how long?
- What minimum extraction confidence is required before auto-advancing to review?
- Should approval store a reusable template keyed by domain?
- How do we handle login-gated or anti-bot protected blogs in v0?

## Recommended Next Action

Implement Milestone 1 in a separate `web/` package using Next.js App Router, TypeScript, and Zod schemas shared between server routes and client UI.
