# Chunk Summary Process

This document describes how chunk-based summarization is expected to work in Curator when `SummaryPlan` is required on every `PostBlock` and when chunk-aware summary modes are enabled.

## Goals

- Provide a clear, deterministic summary flow for long content.
- Keep source processors responsible for chunk creation.
- Keep summary processors responsible for LLM summarization.
- Allow downstream workflows to choose between full summaries, chunk-only summaries, or map-reduce summaries.

## Core Data Model Expectations

- `PostBlock.Content` contains the primary content for the post.
- `PostBlock.Chunks` contains ordered content chunks produced by the source, each with raw content and an optional summary.
- `PostBlock.SummaryPlan` is required and dictates how summaries are produced.

### SummaryPlan Modes

- `full`: summarize the full `PostBlock.Content`.
- `per_chunk`: summarize each chunk and stop (no full summary synthesis step).
- `map_reduce`: summarize each chunk first, then synthesize a final summary using the chunk summaries.

## Source Responsibilities

Sources MUST:

1. Populate `PostBlock.SummaryPlan` for every post.
2. Populate `PostBlock.Chunks` when using `per_chunk` or `map_reduce` with chunk content (chunk summaries remain empty until summary processing).
3. Ensure chunk order is stable (slice order is treated as the canonical order).
4. Keep chunks as raw content (no LLM processing in sources).

If a source selects `per_chunk` or `map_reduce` but does not populate chunks, the summary processor should treat that as an error.

## Summary Processor Responsibilities

Summary processors MUST:

1. Require `PostBlock.SummaryPlan` on every post.
2. Branch on `SummaryPlan.Mode` to decide the execution path.
3. Produce errors when required inputs are missing (no backward compatibility).

### Mode: `full`

1. Render the summary prompt template using `PostBlock.Content`.
2. Call the LLM to produce a single summary.
3. Store the result in `PostBlock.Summary`.

### Mode: `per_chunk`

1. Iterate over `PostBlock.Chunks` in order.
2. For each chunk, render the *chunk summary template* and call the LLM.
3. Store per-chunk summaries on the chunk records.
4. Do not run a final synthesis step.
5. Continue pipeline execution with chunk summaries available for downstream consumption.

### Mode: `map_reduce`

1. Iterate over `PostBlock.Chunks` in order.
2. For each chunk, render the *chunk summary template* and call the LLM.
3. Store per-chunk summaries on the chunk records.
4. Render the final summary template using the aggregated chunk summaries.
5. Call the LLM to produce a final summary and store in `PostBlock.Summary`.

## Template Requirements

Chunk-capable summarization requires two templates:

- **Chunk summary template**: used to summarize each chunk independently.
- **Final summary template**: used to produce the full summary (for `map_reduce` and `full`).

For `per_chunk`, only the chunk summary template is used.

## Error Handling

- Missing `SummaryPlan`: error.
- `per_chunk` or `map_reduce` with zero chunks: error.
- Chunk summary failures should be surfaced as processor errors; policy on fail/drop should be explicit in the summary processor.

## Example Execution (Map-Reduce)

1. Source produces `PostBlock.Content`, `PostBlock.Chunks`, `SummaryPlan.Mode=map_reduce`.
2. Summary processor produces chunk summaries for each chunk.
3. Summary processor concatenates the chunk summaries into the final summary prompt.
4. Summary processor stores the final summary in `PostBlock.Summary`.

## Example Execution (Per-Chunk)

1. Source produces `PostBlock.Content`, `PostBlock.Chunks`, `SummaryPlan.Mode=per_chunk`.
2. Summary processor produces chunk summaries for each chunk.
3. No final summary is produced; downstream can read per-chunk summaries.
