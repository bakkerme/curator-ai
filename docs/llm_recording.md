# LLM Recording & Replay

The LLM recording client lets you capture every LLM interaction that happens
during a real curator run and save it to a **tape file**. That tape can then be
replayed in subsequent runs — including automated tests — without making any
real API calls.

This is the foundation of deterministic end-to-end testing: record once against
the live API; replay as many times as needed.

## Concepts

| Term | Meaning |
|------|---------|
| **Tape** | A JSON file containing an ordered list of recorded LLM interactions. |
| **Interaction** | One request/response pair: the exact messages sent to the model and the response returned. |
| **Key** | A SHA-256 hash derived from the model name and message contents, used to match replay requests to recorded responses. |
| **Record mode** | Curator proxies every LLM call to the real API and writes the interaction to the tape on exit. |
| **Replay mode** | Curator serves responses from the tape without making any real API calls. |

---

## Generating a Tape (Record Mode)

Set `CURATOR_LLM_RECORD` to the path where the tape file should be written,
then run curator normally. On exit, all captured interactions are flushed to
that file.

```bash
CURATOR_LLM_RECORD=./tape.json \
  go run ./cmd/curator -config curator.yaml -run-once
```

After the run, `tape.json` will exist and contain every LLM interaction that
occurred. The file is written with mode `0600` (owner-readable only) because
tapes contain full prompt and response text.

You can run with `-run-once` (single execution) or let curator run continuously
with a cron trigger; the tape is written when the process exits either way.

> **Note:** `CURATOR_LLM_RECORD` and `CURATOR_LLM_REPLAY` are mutually
> exclusive. Setting both will cause curator to exit with an error.

---

## Replaying a Tape (Replay Mode)

Set `CURATOR_LLM_REPLAY` to the path of a previously recorded tape file:

```bash
CURATOR_LLM_REPLAY=./tape.json \
  go run ./cmd/curator -config curator.yaml -run-once
```

In replay mode no real LLM calls are made. Instead, every `ChatCompletion`
request is matched against the tape by computing the same SHA-256 key and
returning the saved response. If no matching interaction is found, curator
returns an error for that request.

---

## Tape File Format

The tape is a pretty-printed JSON file:

```json
{
  "interactions": [
    {
      "key": "<sha256-hex>",
      "request": {
        "model": "gpt-4o-mini",
        "messages": [
          { "role": "system", "content": "You are a summarisation assistant..." },
          { "role": "user",   "content": "Summarise the following..." }
        ],
        "temperature": 0.7
      },
      "response": {
        "content": "This article discusses..."
      }
    }
  ],
  "recorded_at": "2025-01-15T10:30:00Z"
}
```

Key fields:

- **`key`** — deterministic hash of `model` + message roles/contents (see
  [Key Matching](#key-matching) below).
- **`request`** — the full request sent to the LLM, including multipart
  message parts for image-capable models.
- **`response`** — the response returned by the LLM.
- **`error`** — if the real API call failed, the error string is stored here
  and re-raised during replay.
- **`recorded_at`** — UTC timestamp of when the tape was created.

---

## Key Matching

The replay key is a SHA-256 hash of:

1. The model name
2. Each message's role and text content (in order)
3. Each message part's type, text, and image URL (for multipart messages)

Fields are separated by null bytes (`\x00`) to prevent hash collisions between
structurally different requests.

**Temperature and `max_tokens` are intentionally excluded** from the key. Two
requests with identical messages but different temperature settings will match
the same tape entry. This is by design: the key identifies the *content* of the
request, not the parameter configuration.

If a flow calls the LLM multiple times with exactly the same messages (e.g.
`MaxConcurrency > 1` processing identical items), each call is matched in the
order it was recorded, consuming tape entries sequentially.

---

## Using the Tape in E2E Tests

The existing e2e test (`internal/e2e/recording_test.go`) demonstrates the full
record/replay lifecycle using a mock OpenAI HTTP server. To run it:

```bash
CURATOR_E2E=1 go test ./internal/e2e/... -tags=e2e -run TestRecordingLLMClient -v
```

The test:

1. Starts a mock OpenAI-compatible HTTP server and a mock SMTP server.
2. Runs the curator binary in **record mode**, asserting that a tape file is
   created with at least one interaction.
3. Runs the curator binary again in **replay mode**, asserting that zero real
   LLM calls hit the mock server.

For your own e2e tests, the typical pattern is:

1. Run curator once in record mode against a staging/sandbox LLM endpoint to
   capture real interactions.
2. Commit the tape file alongside your test fixtures.
3. In CI, run curator in replay mode — no API key or network access required.

> The `CURATOR_E2E=1` guard and `//go:build e2e` build tag keep these tests
> out of the standard `go test ./...` run. See `internal/e2e/mailpit_test.go`
> for the established pattern.

---

## Environment Variables Reference

| Variable | Description |
|----------|-------------|
| `CURATOR_LLM_RECORD` | Path to write the tape file when recording. Enables record mode. |
| `CURATOR_LLM_REPLAY` | Path to read the tape file for replay. Enables replay mode. |

Both are optional and mutually exclusive. If neither is set, curator uses the
real LLM client normally.

---

## Implementation Notes

- **Package:** `internal/llm/recording`
  - `tape.go` — `Tape`, `Interaction`, and JSON (de)serialisation helpers.
  - `client.go` — `Client` implementing `llm.Client`; `NewRecordClient` and
    `NewReplayClient` constructors.
- **Wiring:** `internal/runner/factory/factory.go` — `NewFromEnvConfig`
  conditionally wraps the OpenAI client based on env vars; the `Close` method
  flushes the tape to disk on exit.
- **Concurrency:** replay is safe under concurrent access (`MaxConcurrency > 1`)
  because key lookups and index updates are protected by a mutex and entries are
  consumed in order.
- **Tape permissions:** files are written with `0600` (owner read/write only)
  because they may contain sensitive prompt text and API responses.
