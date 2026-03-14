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

These primitives are useful but disconnected. There is no way to run the full pipeline with LLM stages included in a deterministic, repeatable way.

## 2. Design Goals

1. **Deterministic CI tests**: Full-pipeline tests that run in CI without a live LLM endpoint, using recorded LLM interactions.
2. **Prompt iteration workflow**: Run the full pipeline against a real LLM with fixture data, record the interactions, and replay them deterministically afterward.
3. **Minimal new abstractions**: Build on existing types (`core.PostBlock`, `core.RunSummary`, `llm.Client`, snapshot JSON) rather than introducing new frameworks.
4. **Simple**: No custom test spec format or assertion engine. Tests are written in Go using standard `go test` and existing assertion libraries.

## 3. Architecture Overview

```
                 +----------------------------+
                 |      Curator Pipeline      |
                 |  (sources, quality,        |
                 |   summary, run_summary,    |
                 |   output)                  |
                 +-------------+--------------+
                               |
                    uses llm.Client interface
                               |
                 +-------------v--------------+
                 |    Recording LLM Client    |
                 |                            |
                 |  record: proxy to real LLM |
                 |    + save tape to disk     |
                 |                            |
                 |  replay: return saved tape |
                 |    (no real LLM calls)     |
                 +----------------------------+
```

### How the recording client relates to snapshots

Snapshots and the recording client operate at **different levels** and compose together:

- **Snapshots** capture **pipeline state** (the `[]*PostBlock` and `RunSummary` between processor stages). They are configured per-processor in the Curator Document YAML. Use snapshot `restore` on a source to inject controlled fixture data into the pipeline without hitting a live source.

- **The recording client** captures **LLM interactions** (the request/response pairs within LLM-backed processors). It wraps the `llm.Client` at construction time.

Together, they enable a complete e2e test: restore fixture data from a source snapshot, then replay recorded LLM interactions for quality/summary/run_summary stages. No live source or live LLM required.

## 4. Recording LLM Client

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

### Concurrency-safe replay via key-based matching

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

### Configuration

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

**2. Programmatic (for Go tests)**

```go
// Record against real LLM
tape := recording.NewTape()
client := recording.NewClient(realLLMClient, recording.ModeRecord, tape)
// ... run pipeline ...
tape.SaveTo("testdata/tapes/my_test.json")

// Replay for deterministic CI
tape, _ := recording.LoadTape("testdata/tapes/my_test.json")
client := recording.NewReplayClient(tape)
// ... run pipeline -- no real LLM calls ...
```

### Tape file format

```json
{
  "recorded_at": "2025-03-14T08:00:00Z",
  "model": "gpt-4o-mini",
  "interactions": [
    {
      "key": "a1b2c3d4e5f6...",
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

## 5. Writing E2E Tests

With the recording client available, full-pipeline e2e tests are written as standard Go test functions. No special spec format or assertion engine is needed.

### Example: E2E test with recorded LLM interactions

```go
//go:build e2e

func TestFullPipelineWithLLM(t *testing.T) {
    if os.Getenv("CURATOR_E2E") == "" {
        t.Skip("set CURATOR_E2E=1 to enable e2e tests")
    }

    // Load a recorded tape for deterministic replay
    tape, err := recording.LoadTape("testdata/tapes/quality_summary.json")
    require.NoError(t, err)

    client := recording.NewReplayClient(tape)

    // Build pipeline from a curator document using the replay client
    doc, err := config.LoadCuratorDocument("testdata/fixtures/test_flow.yml")
    require.NoError(t, err)

    flow, err := factory.NewFromDocument(doc, client, envCfg)
    require.NoError(t, err)

    // Run the pipeline
    result, err := runner.RunOnce(context.Background(), flow)
    require.NoError(t, err)

    // Assert on the output using standard Go test assertions
    assert.GreaterOrEqual(t, len(result.Blocks), 1)
    for _, block := range result.Blocks {
        assert.NotNil(t, block.Quality)
        assert.NotNil(t, block.Summary)
        assert.NotEmpty(t, block.Summary.Summary)
    }
    assert.NotNil(t, result.RunSummary)
}
```

### Composing with snapshot restore

For tests that need controlled source data without a live source, use the existing snapshot `restore` feature in the fixture curator document:

```yaml
# testdata/fixtures/test_flow.yml
workflow:
  name: "E2E Test Flow"
  trigger:
    - cron:
        schedule: "* * * * *"
  sources:
    - testfile:
        path: "testdata/fixtures/test_posts.md"
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
        subject: "Test Digest"
```

## 6. Workflow Examples

### Workflow 1: Recording a tape for a new e2e test

```bash
# 1. Create a fixture curator document with testfile source
vim testdata/fixtures/test_flow.yml

# 2. Record LLM interactions against a real LLM
CURATOR_LLM_RECORD=./testdata/tapes/test_flow.json \
  go run ./cmd/curator -config testdata/fixtures/test_flow.yml -run-once

# 3. Write a Go test that replays the tape and asserts on output
vim internal/e2e/full_pipeline_test.go

# 4. Verify the test passes in replay mode
CURATOR_E2E=1 go test -tags=e2e ./internal/e2e -run TestFullPipelineWithLLM

# 5. Commit the tape + test
git add testdata/tapes/ internal/e2e/
```

### Workflow 2: Iterating on a prompt

```bash
# 1. Edit the prompt template in the fixture curator document
vim testdata/fixtures/test_flow.yml

# 2. Run against real LLM to see new results
CURATOR_LLM_RECORD=./testdata/tapes/test_flow.json \
  go run ./cmd/curator -config testdata/fixtures/test_flow.yml -run-once

# 3. Inspect the output, re-record if happy
# 4. Update test assertions if needed, commit new tape
```

### Workflow 3: Recording from a production-like run

```bash
# Record a full pipeline run with your real curator document
CURATOR_LLM_RECORD=./tapes/prod_run.json \
  go run ./cmd/curator -config curator.yaml -run-once

# Later, replay deterministically (e.g., to test a code refactor)
CURATOR_LLM_REPLAY=./tapes/prod_run.json \
  go run ./cmd/curator -config curator.yaml -run-once
```

### Workflow 4: CI regression check

The existing e2e test pattern works -- tests check for `CURATOR_E2E=1` and replay tapes:

```yaml
# .github/workflows/test.yml addition
- name: Run e2e tests
  env:
    CURATOR_E2E: "1"
  run: go test -tags=e2e ./internal/e2e/...
```

No `OPENAI_API_KEY` needed in CI -- replay mode requires no live LLM.

## 7. Implementation Plan

### Phase 1: Recording LLM Client
- `internal/llm/recording/client.go` -- record/replay client with key-based matching
- `internal/llm/recording/tape.go` -- tape serialization (load/save)
- Unit tests for record, replay, key matching, and concurrent access
- Add `LLMRecordPath` / `LLMReplayPath` fields to `EnvConfig` in `internal/config/env.go`
- Wire into factory, reading from `EnvConfig` (not `os.Getenv` directly)

### Phase 2: E2E Test with LLM Stages
- Create fixture curator document with testfile source + all LLM stages
- Record an initial tape against a real LLM
- Write a Go e2e test (`internal/e2e/`) that replays the tape and asserts on pipeline output
- Verify it passes in CI without `OPENAI_API_KEY`

### Phase 3: CI Integration
- Add e2e test step to `.github/workflows/test.yml`
- Document the recording/replay workflow

## 8. Design Decisions

### Why a recording LLM client instead of extending snapshots?

Snapshots and the recording client solve different problems at different levels:

- **Snapshots** operate **between processors** -- they capture the pipeline state (`[]*PostBlock`) that flows from one stage to the next. They are a production debugging feature configured in the Curator Document.
- **The recording client** operates **within processors** -- it captures the LLM request/response pairs that a processor makes during execution. It is a test infrastructure feature configured via env vars or programmatically.

They compose naturally: use snapshot restore to inject fixture data at the source stage, then use the recording client to deterministically replay LLM interactions from that point forward. No duplication because they don't overlap.

### Why standard Go tests instead of a custom spec format?

For the current number of pipeline configurations, standard Go tests with the recording client provide sufficient coverage without the complexity of a custom YAML spec/assertion format. Go tests are familiar, debuggable, and integrate naturally with CI. A spec-driven framework can be added later if the number of test configurations grows.

### Temperature handling for determinism

When recording tapes, the recording client should force `temperature: 0` (or the lowest supported value) to maximize reproducibility of recorded responses. In replay mode, temperature is irrelevant since no real LLM call is made.

### Concurrency in record/replay

The quality, summary, and image-captioning processors all support `MaxConcurrency > 1`, dispatching LLM calls from concurrent goroutines. A naive FIFO replay (returning `tape[idx++]`) would be non-deterministic because goroutine scheduling varies between runs.

The recording client solves this with **key-based matching**: each interaction is keyed by a SHA-256 hash of the full request content (model + messages). Since each block produces a unique rendered prompt, keys are naturally distinct. In replay, the client looks up the matching interaction by key rather than by position, so the result is correct regardless of call order. See Section 4 for details.

## 9. Future Extensions

- **YAML eval spec framework**: If the number of pipeline/prompt configurations grows, add a spec-driven test runner with YAML-defined test cases, stage observers, and structured assertion evaluation.
- **Prompt comparison tool**: A CLI that diffs two tape files or test outputs to show how prompt changes affected quality scores, summary lengths, etc.
- **LLM-as-judge evaluation**: Use a separate LLM call to score output quality (e.g., "rate this summary 1-5 for accuracy and conciseness").
- **Prompt versioning**: Tag tapes with prompt template hashes for tracking quality over time.
- **Visual diff in CI**: GitHub Actions comment with a table showing eval results on each PR.
- **Cost tracking**: Record token counts from LLM interactions for cost estimation.
