# Full-Flow E2E Testing System

## 1. Problem Statement

Curator AI currently lacks a system for:
- **Automated e2e testing of LLM stages**: The existing `mailpit_test.go` e2e test skips all LLM processors (quality, summary, run_summary). There is no automated validation that prompts render correctly, that LLM responses are parsed properly, or that the overall pipeline produces expected outputs when LLM stages are included.
- **Prompt iteration**: No structured way to change a prompt template, re-run the pipeline with controlled inputs, and validate the result.
- **Regression detection**: No way to know if a code change has broken LLM-stage behavior without manually running the pipeline.

### What already exists

| Capability | Current state |
|---|---|
| Mock LLM client | `internal/llm/mock` -- canned FIFO responses, no request validation |
| TestFile source | `internal/sources/testfile` -- loads fixture markdown from disk |
| Snapshot save/restore | `internal/runner/snapshot` -- saves `[]*PostBlock` + `RunSummary` as JSON at any pipeline stage |
| E2E test (Mailpit) | `internal/e2e/mailpit_test.go` -- RSS source -> email output only (no LLM stages) |
| Unit tests | Individual processor tests with mock LLM client |

These primitives are useful but disconnected. There is no harness that ties them together into a full pipeline test that exercises prompt rendering, LLM interaction, response parsing, and output validation in a single run.

## 2. Design Goals

1. **Deterministic CI tests**: Full-pipeline tests that run in CI without a live LLM endpoint, using recorded LLM interactions.
2. **Prompt iteration workflow**: Run the full pipeline against a real LLM with fixture data, record the interactions, and replay them deterministically afterward.
3. **Minimal new abstractions**: Build on existing types (`core.PostBlock`, `core.RunSummary`, `llm.Client`, snapshot JSON) rather than introducing new frameworks.
4. **Go-native**: Everything runs via `go test` with build tags; no external test runners.

## 3. Architecture Overview

```
                                    +--------------------+
                                    |  Eval Spec (YAML)  |
                                    |  curator doc ref,  |
                                    |  tape ref,         |
                                    |  assertions        |
                                    +---------+----------+
                                              |
                                 +------------v-----------+
                                 |    Eval Test Runner     |
                                 |  go test -tags=eval     |
                                 |                         |
                                 |  1. Load eval spec      |
                                 |  2. Parse curator doc   |
                                 |  3. Build pipeline via  |
                                 |     factory             |
                                 |  4. Inject recording    |
                                 |     LLM client          |
                                 |  5. Run pipeline        |
                                 |  6. Capture stage state |
                                 |     via observer        |
                                 |  7. Check assertions    |
                                 |  8. Write report        |
                                 +---+-------------+------+
                                     |             |
                          +----------+             +----------+
                          v                                   v
                 +--------+---------+              +----------+----------+
                 | Recording Client |              | Stage Observer      |
                 | record: proxy to |              | (wraps processors,  |
                 |   real LLM, save |              |  captures blocks    |
                 |   tape to disk   |              |  after each stage)  |
                 | replay: return   |              +---------------------+
                 |   saved tape     |
                 +------------------+
```

### How the recording client relates to snapshots

Snapshots and the recording client operate at **different levels** and compose together:

- **Snapshots** capture **pipeline state** (the `[]*PostBlock` and `RunSummary` between processor stages). They are configured per-processor in the Curator Document YAML. Use snapshot `restore` on a source to inject controlled fixture data into the pipeline without hitting a live source.

- **The recording client** captures **LLM interactions** (the request/response pairs within LLM-backed processors). It wraps the `llm.Client` at construction time.

Together, they enable a complete e2e test: restore fixture data from a source snapshot, then replay recorded LLM interactions for quality/summary/run_summary stages. No live source or live LLM required.

### Key components

| Component | Location | Purpose |
|---|---|---|
| **Recording LLM Client** | `internal/llm/recording/` | Wraps any `llm.Client`; records all `ChatCompletion` calls to a JSON tape file (record mode) or replays them (replay mode). |
| **Eval Spec** | `testdata/eval/*.yml` | YAML files defining test cases: which curator document to use, which tape file, and what assertions to check. |
| **Eval Runner** | `internal/eval/` | Test harness that loads eval specs, builds the pipeline with the recording client injected, executes it, and checks assertions. |
| **Stage Observer** | `internal/eval/observer.go` | Thin wrapper processors (same pattern as `snapshot.Wrap*`) that capture `[]*PostBlock` after each stage for assertion. |

## 4. Component Details

### 4.1 Recording LLM Client

A new `llm.Client` implementation that supports two modes:

```go
package recording

type Mode string

const (
    ModeRecord Mode = "record"  // Proxy to real client, save interactions
    ModeReplay Mode = "replay"  // Return saved interactions (no real LLM)
)

// Interaction captures a single ChatCompletion call.
type Interaction struct {
    Key      string           `json:"key"`      // stable lookup key (hash of request messages)
    Request  llm.ChatRequest  `json:"request"`
    Response llm.ChatResponse `json:"response"`
    Error    string           `json:"error,omitempty"`
}

// Tape is the serialized collection of interactions, keyed for concurrent-safe lookup.
type Tape struct {
    Interactions []Interaction `json:"interactions"`
    RecordedAt   time.Time     `json:"recorded_at"`
    Model        string        `json:"model,omitempty"`
}

// Client wraps an llm.Client and records/replays interactions.
type Client struct {
    inner  llm.Client // nil in replay mode
    mode   Mode
    tape   *Tape
    mu     sync.Mutex
    index  map[string][]int // key -> indices into tape.Interactions
    used   map[int]bool     // tracks which interactions have been consumed
}

func NewClient(inner llm.Client, mode Mode, tape *Tape) *Client
func NewReplayClient(tape *Tape) *Client
func (c *Client) ChatCompletion(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error)
func (c *Client) Close() error  // writes tape to disk in record mode
func interactionKey(req llm.ChatRequest) string  // deterministic hash of request messages
```

**Record mode**: Proxies every `ChatCompletion` call to `inner`, computes a stable key from the request, appends the keyed interaction to `tape.Interactions`, and writes the tape to disk via `Close()`.

**Replay mode**: On each `ChatCompletion` call, computes the same key from the incoming request and looks up a matching interaction in the tape. Returns the recorded response. No real LLM call is made. If no match is found, the test fails with a descriptive error.

#### Concurrency-safe replay via key-based matching

The quality, summary, and image-captioning processors all support `MaxConcurrency > 1`, dispatching LLM calls from goroutines. This means the *order* of `ChatCompletion` calls is non-deterministic between runs. A simple FIFO index would break.

Instead, the recording client uses **key-based matching**:

1. **Key computation**: `interactionKey(req)` produces a deterministic hash from the full request content (model + all message roles and contents, concatenated and hashed with SHA-256). Since each block has a unique ID and content, the rendered prompts are naturally distinct, producing unique keys.

2. **Record mode**: Each interaction is stored with its computed key. If two requests happen to produce the same key (e.g., retries with identical content), they are stored as separate entries under the same key and replayed in FIFO order within that key.

3. **Replay mode**: On each `ChatCompletion` call, the client computes the key, looks up the next unconsumed interaction with that key, marks it consumed, and returns the recorded response. This is safe regardless of goroutine scheduling order.

```go
func interactionKey(req llm.ChatRequest) string {
    h := sha256.New()
    h.Write([]byte(req.Model))
    for _, msg := range req.Messages {
        h.Write([]byte(msg.Role))
        h.Write([]byte(msg.Content))
        for _, part := range msg.Parts {
            h.Write([]byte(part.Type))
            h.Write([]byte(part.Text))
            h.Write([]byte(part.ImageURL))
        }
    }
    return hex.EncodeToString(h.Sum(nil))
}
```

This approach means:
- Tests exercise the same concurrent code paths as production (no need to force `MaxConcurrency=1`).
- Replay is deterministic regardless of goroutine scheduling.
- Retries with identical prompts (e.g., JSON parse failures) are handled correctly via per-key FIFO ordering.

#### Configuration

The recording client is configured at two levels:

**1. Environment variables (for CLI / manual use)**

```bash
# Normal operation (default) -- real LLM calls, no recording
go run ./cmd/curator -config curator.yaml -run-once

# Record mode -- real LLM calls, writes tape to disk
CURATOR_LLM_RECORD=./tapes/my_run.json go run ./cmd/curator -config curator.yaml -run-once

# Replay mode -- no real LLM, reads tape from disk
CURATOR_LLM_REPLAY=./tapes/my_run.json go run ./cmd/curator -config curator.yaml -run-once
```

The env vars are added to `EnvConfig` in `internal/config/env.go` (following the project convention of centralized env access) and loaded via the existing `envString` helper in `LoadEnv()`. The factory receives these values through `EnvConfig` rather than calling `os.Getenv` directly:

```go
// In internal/config/env.go, add to EnvConfig:
type EnvConfig struct {
    // ... existing fields ...
    LLMRecordPath string // CURATOR_LLM_RECORD -- tape file path for record mode
    LLMReplayPath string // CURATOR_LLM_REPLAY -- tape file path for replay mode
}

// In LoadEnv():
LLMRecordPath: strings.TrimSpace(envString("CURATOR_LLM_RECORD", "")),
LLMReplayPath: strings.TrimSpace(envString("CURATOR_LLM_REPLAY", "")),
```

```go
// In factory.go, after creating the real LLM client:
if path := envCfg.LLMRecordPath; path != "" {
    client = recording.NewClient(client, recording.ModeRecord, recording.NewTape())
    defer client.Close()  // saves tape to path
}
if path := envCfg.LLMReplayPath; path != "" {
    tape, _ := recording.LoadTape(path)
    client = recording.NewReplayClient(tape)
}
```

**2. Programmatic (for eval tests)**

```go
// Record against real LLM
tape := recording.NewTape()
client := recording.NewClient(realLLMClient, recording.ModeRecord, tape)
// ... run pipeline ...
tape.SaveTo("testdata/eval/tapes/my_test.json")

// Replay for deterministic CI
tape, _ := recording.LoadTape("testdata/eval/tapes/my_test.json")
client := recording.NewReplayClient(tape)
// ... run pipeline -- no real LLM calls ...
```

#### Tape file format

```json
{
  "recorded_at": "2025-03-14T08:00:00Z",
  "model": "gpt-4o-mini",
  "interactions": [
    {
      "key": "a1b2c3d4e5f6..."  ,
      "request": {
        "model": "gpt-4o-mini",
        "messages": [
          {"role": "system", "content": "You are a quality evaluator..."},
          {"role": "user", "content": "Title: Example Post\nContent: ..."}
        ],
        "temperature": 0.0
      },
      "response": {
        "content": "{\"score\": 0.85, \"reason\": \"Highly relevant technical discussion\"}"
      }
    }
  ]
}
```

### 4.2 Eval Spec Format

Each eval spec is a YAML file defining a single test scenario:

```yaml
# testdata/eval/reddit_quality_summary.yml
name: "Reddit quality + summary full flow"
description: "Tests quality filtering and post summarization with Reddit-style fixture data"

# The curator document to use for this eval (file reference)
curator_document: testdata/eval/fixtures/reddit_flow.yml

# LLM tape file for deterministic replay.
# When EVAL_LLM_RECORD=1 is set, a new tape is recorded to this path.
# When absent and no env var is set, the test is skipped.
llm_tape: testdata/eval/tapes/reddit_quality_summary.json

# Assertions on the pipeline output
assertions:
  # Stage-level assertions
  after_source:
    block_count:
      min: 3
      max: 10

  after_quality:
    block_count:
      min: 1
    blocks:
      - match: { title_contains: "benchmark" }
        quality:
          result: "pass"
          score_min: 0.5

  after_post_summary:
    blocks:
      - match: { quality_result: "pass" }
        summary:
          not_empty: true
          max_length: 2000

  after_run_summary:
    run_summary:
      not_empty: true
      contains_any:
        - "benchmark"
        - "performance"

  after_output:
    email:
      subject_contains: "Curator"
      body_contains:
        - "benchmark"
```

### 4.3 Eval Runner

The eval runner is a Go test file that:

1. Discovers eval spec YAML files in `testdata/eval/`
2. For each spec, builds a `core.Flow` from the referenced curator document
3. Injects the recording LLM client (replay or record mode based on env vars)
4. Wraps processors with stage observers to capture intermediate state
5. Runs the pipeline via `runner.RunOnce()`
6. Checks assertions against captured state
7. Writes a JSON result report

```go
// internal/eval/eval_test.go
//go:build eval

func TestEval(t *testing.T) {
    specs, err := LoadSpecs("../../testdata/eval")
    require.NoError(t, err)

    for _, spec := range specs {
        t.Run(spec.Name, func(t *testing.T) {
            result := RunEval(t, spec)
            WriteReport(t, spec, result)
        })
    }
}
```

**Running modes**:

```bash
# CI: replay recorded tapes (deterministic, no LLM needed)
go test -tags=eval ./internal/eval/...

# Local dev: record new tapes against a real LLM
EVAL_LLM_RECORD=1 OPENAI_API_KEY=... go test -tags=eval ./internal/eval/... -run TestEval/reddit_quality

# Local dev: run against real LLM without recording (one-off)
EVAL_LLM_LIVE=1 OPENAI_API_KEY=... go test -tags=eval ./internal/eval/... -run TestEval/reddit_quality
```

### 4.4 Stage Observer

To capture intermediate pipeline state without modifying the runner, wrap each processor with a thin observer:

```go
package eval

// ObservedStage captures the state of blocks after a processor runs.
type ObservedStage struct {
    Name       string            `json:"name"`
    Type       string            `json:"type"`       // "source", "quality", "post_summary", "run_summary", "output"
    BlockCount int               `json:"block_count"`
    Blocks     []*core.PostBlock `json:"blocks"`
    RunSummary *core.RunSummary  `json:"run_summary,omitempty"`
    Duration   time.Duration     `json:"duration"`
}

// Observer collects stage snapshots during a pipeline run.
type Observer struct {
    Stages []ObservedStage
}
```

This uses the same wrapper pattern as the existing `snapshot.Wrap*` functions -- it wraps each processor type and captures blocks after the inner processor completes.

### 4.5 Eval Report

After each eval run, the runner writes a structured JSON report:

```json
{
  "spec_name": "reddit_quality_summary",
  "timestamp": "2025-03-14T08:01:00Z",
  "duration_ms": 1234,
  "llm_mode": "replay",
  "stages": [
    {
      "name": "reddit-source",
      "type": "source",
      "block_count": 5,
      "duration_ms": 100
    },
    {
      "name": "is_relevant",
      "type": "quality",
      "block_count": 3,
      "duration_ms": 450
    }
  ],
  "assertions": {
    "total": 8,
    "passed": 7,
    "failed": 1,
    "results": [
      {
        "path": "after_quality.block_count.min",
        "expected": ">=1",
        "actual": "3",
        "passed": true
      },
      {
        "path": "after_post_summary.blocks[0].summary.max_length",
        "expected": "<=2000",
        "actual": "2150",
        "passed": false
      }
    ]
  },
  "llm_interactions": 8
}
```

## 5. Fixture Data Strategy

### Fixture Curator Documents

Store purpose-built curator documents in `testdata/eval/fixtures/`. These use the `testfile` source (or inline RSS fixtures like the existing e2e test) to load controlled input data, and include full prompt templates for all LLM stages.

```yaml
# testdata/eval/fixtures/reddit_flow.yml
workflow:
  name: "Eval - Reddit Quality + Summary"
  trigger:
    - cron:
        schedule: "* * * * *"
  sources:
    - testfile:
        path: "testdata/eval/fixtures/reddit_posts.json"
  quality:
    - llm:
        name: "is_relevant"
        system_template: "You are a content quality evaluator..."
        prompt_template: |
          Evaluate the following post for relevance.
          Title: {{.Title}}
          Content: {{.Content}}
        evaluations:
          - "Technical depth"
          - "Novel insights"
        action_type: "pass_drop"
        threshold: 0.5
  post_summary:
    - llm:
        name: "post_sum"
        type: "llm"
        context: "post"
        system_template: "You are a technical summarizer..."
        prompt_template: |
          Summarize the following post concisely.
          Title: {{.Title}}
          Content: {{.Content}}
  run_summary:
    - llm:
        name: "full_sum"
        type: "llm"
        context: "flow"
        system_template: "You are a digest summarizer..."
        prompt_template: |
          Create a brief digest of these posts:
          {{range .Blocks}}- {{.Title}}: {{.Summary.Summary}}
          {{end}}
  output:
    - email:
        to: "test@example.com"
        from: "curator@example.com"
        subject: "Eval Test Digest"
        template: |
          <html><body>
          {{ range .Blocks }}<p>{{ .Title }}</p>{{ end }}
          </body></html>
```

### Fixture Post Data

Extend the `testfile` source to also accept JSON files containing pre-built `PostBlock` arrays (not just single markdown files). This allows controlling exactly what posts enter the pipeline:

```json
[
  {
    "id": "eval-post-1",
    "title": "New LLM benchmark shows 2x improvement",
    "content": "Researchers at MIT published a new benchmark...",
    "author": "researcher42",
    "url": "https://example.com/post-1",
    "created_at": "2025-03-10T10:00:00Z",
    "metadata": { "score": "150", "comment_count": "45" }
  },
  {
    "id": "eval-post-2",
    "title": "Check out my cat",
    "content": "Here is a picture of my cat sitting on a keyboard.",
    "author": "catperson99",
    "url": "https://example.com/post-2",
    "created_at": "2025-03-10T11:00:00Z",
    "metadata": { "score": "5", "comment_count": "2" }
  }
]
```

### Using snapshots as fixture data

Alternatively, use a real pipeline run to capture source output via the existing snapshot system, then reference that snapshot in the eval curator document with `restore: true`:

```yaml
sources:
  - reddit:
      subreddits: ["MachineLearning"]
      snapshot:
        restore: true
        path: "testdata/eval/fixtures/ml_posts_snapshot.json"
```

This is useful when you want to test with realistic data from a real source run, rather than hand-crafted fixture posts.

## 6. Workflow Examples

### Workflow 1: Setting up a new e2e test

```bash
# 1. Create fixture data and curator document
vim testdata/eval/fixtures/my_flow.yml
vim testdata/eval/fixtures/my_posts.json

# 2. Create eval spec with assertions
vim testdata/eval/my_test.yml

# 3. Record LLM interactions against a real LLM
EVAL_LLM_RECORD=1 OPENAI_API_KEY=... go test -tags=eval ./internal/eval/... -run TestEval/my_test -v

# 4. Inspect the report
cat testdata/eval/results/my_test.json

# 5. Commit the tape + spec (CI can now replay deterministically)
git add testdata/eval/
```

### Workflow 2: Iterating on a prompt

```bash
# 1. Edit the prompt template in the fixture curator document
vim testdata/eval/fixtures/reddit_flow.yml

# 2. Run against real LLM to see new results
EVAL_LLM_LIVE=1 OPENAI_API_KEY=... go test -tags=eval ./internal/eval/... -run TestEval/reddit_quality -v

# 3. If happy, record a new tape for CI
EVAL_LLM_RECORD=1 go test -tags=eval ./internal/eval/... -run TestEval/reddit_quality

# 4. Update assertions if needed, commit new tape
git add testdata/eval/
```

### Workflow 3: Recording LLM interactions via CLI (outside of eval tests)

```bash
# Record a full pipeline run with your real curator document
CURATOR_LLM_RECORD=./tapes/v1.json go run ./cmd/curator -config curator.yaml -run-once

# Later, replay deterministically (e.g., to test a code change)
CURATOR_LLM_REPLAY=./tapes/v1.json go run ./cmd/curator -config curator.yaml -run-once
```

### Workflow 4: CI regression check

```yaml
# .github/workflows/test.yml addition
- name: Run eval tests
  run: go test -tags=eval ./internal/eval/...
```

No environment variables needed -- replay mode is the default when tape files exist.

## 7. Implementation Plan

### Phase 1: Recording LLM Client (foundation)
- `internal/llm/recording/client.go` -- record/replay client
- `internal/llm/recording/tape.go` -- tape serialization (load/save)
- Unit tests for record, replay, and strict-matching modes
- Add `LLMRecordPath` / `LLMReplayPath` fields to `EnvConfig` in `internal/config/env.go`
- Wire into factory, reading from `EnvConfig` (not `os.Getenv` directly)

### Phase 2: Eval Runner Core
- `internal/eval/spec.go` -- eval spec YAML parsing
- `internal/eval/observer.go` -- stage observer wrappers
- `internal/eval/runner.go` -- pipeline construction and execution with recording client
- `internal/eval/assertions.go` -- assertion evaluation engine
- `internal/eval/report.go` -- JSON report generation
- `internal/eval/eval_test.go` -- test entrypoint (`go test -tags=eval`)

### Phase 3: Fixture Infrastructure
- Extend `testfile` source to accept JSON `PostBlock` arrays
- Create sample fixture data in `testdata/eval/fixtures/`
- Create 2-3 example eval specs with recorded tapes

### Phase 4: CI Integration
- Add eval test step to `.github/workflows/test.yml`
- Document the workflow in `docs/eval_testing.md`

## 8. Design Decisions

### Why a recording LLM client instead of extending snapshots?

Snapshots and the recording client solve different problems at different levels:

- **Snapshots** operate **between processors** -- they capture the pipeline state (`[]*PostBlock`) that flows from one stage to the next. They are a production debugging feature configured in the Curator Document.
- **The recording client** operates **within processors** -- it captures the LLM request/response pairs that a processor makes during execution. It is a test infrastructure feature configured via env vars or programmatically.

They compose naturally: use snapshot restore to inject fixture data at the source stage, then use the recording client to deterministically replay LLM interactions from that point forward. No duplication because they don't overlap.

### Why YAML eval specs instead of pure Go test code?

Eval specs are primarily data (fixture references, assertions) not logic. YAML keeps them accessible to non-Go-developers who might be writing prompts, and makes it easy to add new test cases without writing Go code. The actual test logic lives in Go.

### Why `go test -tags=eval` instead of a separate binary?

Using `go test` keeps the eval system integrated with the existing test infrastructure, gets test parallelism and caching for free, and avoids a separate build/run step. The `eval` build tag ensures these tests don't run during normal `go test ./...`.

### Temperature handling for determinism

When recording tapes, the recording client should force `temperature: 0` (or the lowest supported value) to maximize reproducibility of recorded responses. In replay mode, temperature is irrelevant since no real LLM call is made.

### Concurrency in record/replay

The quality, summary, and image-captioning processors all support `MaxConcurrency > 1`, dispatching LLM calls from concurrent goroutines. A naive FIFO replay (returning `tape[idx++]`) would be non-deterministic because goroutine scheduling varies between runs.

The recording client solves this with **key-based matching**: each interaction is keyed by a SHA-256 hash of the full request content (model + messages). Since each block produces a unique rendered prompt, keys are naturally distinct. In replay, the client looks up the matching interaction by key rather than by position, so the result is correct regardless of call order. See Section 4.1 for details.

## 9. Future Extensions

- **Prompt comparison tool**: A CLI that diffs two eval reports to show how prompt changes affected block counts, quality scores, summary lengths, etc.
- **LLM-as-judge evaluation**: Use a separate LLM call to score output quality (e.g., "rate this summary 1-5 for accuracy and conciseness").
- **Prompt versioning**: Tag tapes with prompt template hashes for tracking quality over time.
- **Visual diff in CI**: GitHub Actions comment with a table showing eval results on each PR.
- **Cost tracking**: Record token counts from LLM interactions for cost estimation.
