# Agent Trace Viewer

`trace-view` is a small, standalone Go tool for AI-agent developers, students,
and SMB teams debugging what an agent run actually did: its LLM and tool calls,
retries, timing, tokens, cost, and errors. It reads only local files and uses
only the Go standard library.

## Trace format

A `.json` file is one object with `schema_version`, `trace_id`, RFC3339
`started_at`/`ended_at`, `meta` (`agent`, `model`, optional `tenant`), and
`spans`. Every span supplies `span_id`, optional `parent_span_id`, `name`, a
kind (`llm`, `tool`, `plan`, `verify`, `memory`, `other`), status (`ok` or
`error`), timestamps, and optional `attributes`, `tokens` (`in`, `out`),
`cost_usd`, and `error`. A `.jsonl` file contains one span object per line; the
viewer derives run bounds from those spans.

## Commands

```sh
go build -o trace-view .
./trace-view show trace/testdata/simple.json
./trace-view summary trace/testdata/multiagent.jsonl
./trace-view serve trace/testdata/simple.json --port 8099
```

Example `show` output:

```text
answer question [plan] ok 3s tokens=20/10 cost=$0.010000
  search docs [tool] ok 1s tokens=0/0 cost=$0.000000
```

`summary` reports wall-clock time, span counts by kind, token and cost totals,
errors, and the three slowest spans. `serve` prints a local URL for a
self-contained interactive tree/timeline; select a span to see attributes and
errors. A span whose parent is absent is promoted to an `[ORPHAN]` root.

## Open core

This OSS core covers local trace capture, viewing, and summarization. Hosted
storage, team collaboration, and managed operational features are premium.
