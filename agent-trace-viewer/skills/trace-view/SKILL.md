---
name: trace-view
description: Use when the user wants to inspect, summarize, or debug an AI agent run/trace file.
---

# Trace View

Use `trace-view` with a local JSON trace or JSONL span stream.

## Schema

A JSON trace contains `schema_version`, `trace_id`, RFC3339 `started_at` and
`ended_at`, optional `meta`, and `spans`. Each span has `span_id`, optional
`parent_span_id`, `name`, `kind` (`llm`, `tool`, `plan`, `verify`, `memory`,
`other`), `status` (`ok` or `error`), timestamps, optional attributes, tokens,
cost, and error text. JSONL contains one span object per line.

## Commands

```sh
trace-view show run.json
trace-view summary run.jsonl
trace-view serve run.json --port 8099
```

Use `show` for nesting and failures, `summary` for totals and slow spans, and
`serve` for an interactive local tree/timeline at the printed URL.
