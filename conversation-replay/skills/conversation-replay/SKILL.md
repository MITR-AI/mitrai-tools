---
name: conversation-replay
description: Use when the user wants to replay, inspect at a point, or diff recorded AI agent conversations.
---

# Conversation Replay

Recorded conversations are JSON objects with `schema_version`, `conversation_id`,
`started_at`, `meta`, and ordered `turns`. Each turn has `index`, `role`
(`user`, `assistant`, `tool`, or `system`), `content`, optional `tool_calls`,
optional `{in,out}` `tokens`, optional `state`, and optional timestamp `at`.
JSONL is also accepted: one turn object per line.

```sh
# Compact chronological rendering
convreplay show run.json

# Transcript and effective state at a recorded turn
convreplay at run.json --turn 12

# Compare two recordings by turn index (non-zero when different)
convreplay diff baseline.json candidate.jsonl

# Open the self-contained local timeline viewer
convreplay serve run.json --port 8110
```

This tool only reads recordings; it never invokes a model or re-executes tools.
