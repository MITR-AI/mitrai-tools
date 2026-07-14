# Conversation Replay

Conversation Replay is a small, model-free inspector for AI-agent developers debugging recorded conversations. It replays the dialogue—user, assistant, tool, and system turns—rather than execution spans. Use it beside a trace viewer: the trace viewer answers *where time went*; Conversation Replay answers *what the agent said, knew, and called at each turn*.

It is deterministic and local-only: it never calls an LLM or re-executes a tool.

## Format

A recording is JSON with this shape (timestamps are RFC3339):

```json
{
  "schema_version": "1.0", "conversation_id": "run-42",
  "started_at": "2026-01-01T09:00:00Z",
  "meta": {"agent": "support", "model": "recorded-model", "tenant": "optional"},
  "turns": [{
    "index": 0, "role": "user", "content": "Help me",
    "tool_calls": [{"name": "lookup", "args": {"id": "7"}, "result": "..."}],
    "model": "optional", "tokens": {"in": 4, "out": 8},
    "state": {"case": "7"}, "at": "2026-01-01T09:00:01Z"
  }]
}
```

Roles are `user`, `assistant`, `tool`, or `system`. Turn indices must be unique and ascending. JSONL is accepted too, with one turn object per line; it may repeat `conversation_id` as an extra field.

## Commands

```sh
go run . show testdata/simple.json
# Conversation demo-001 (3 turns)
# [0] user: Find the capital of France.
# [1] assistant: I will look that up.  tools: search  tokens: in=12 out=8

go run . at testdata/simple.json --turn 2
# Transcript through turn 2 ... State: {"answer":"Paris", "stage":"looked-up"}

go run . diff testdata/simple.json testdata/diverging.jsonl
# turn 1 content differs
# turn 1 tool calls differ
# turn 3 only in B

go run . serve testdata/simple.json --port 8110
# Conversation Replay listening at http://127.0.0.1:8110
```

The `serve` command provides one self-contained local HTML page with a scrubber, turn content, tool calls, and the state snapshot currently in effect.

## Open core

This OSS core covers replay, point-in-time inspection, and diffs of recorded runs. Hosted team features and live fork/what-if re-execution are premium capabilities and intentionally outside this repository.
