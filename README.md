# mitrai-tools

**Open-source developer tools for building AI agents.** Part of the [MitrAI](https://github.com/MITR-AI)
platform, published open-core: the tools here are free and open (Apache-2.0); hosted, team, and enterprise
editions are commercial.

> **What "open-core" means here:** everything in this repository is genuinely free to use and self-host,
> forever. We monetize a separate, private layer *around* these cores — hosting, team collaboration, scale,
> and enterprise security/compliance. A solo developer or student should be able to do real work with only
> what's in this repo. See the individual tool READMEs for the OSS-vs-premium split.

## Tools

| Tool | What it does | Status |
|---|---|---|
| [`agent-trace-viewer`](agent-trace-viewer/) | Inspect, summarize, and debug an AI agent run — step-by-step tool/LLM calls, timing, tokens, cost, errors (CLI + local web UI). Reads an open, OpenTelemetry-compatible trace format. | alpha |
| [`conversation-replay`](conversation-replay/) | Time-travel through a recorded AI agent **conversation** — replay turn-by-turn, inspect full state at any turn, and diff two runs to find where they diverged (CLI + local web UI). Complements the trace viewer (turns vs spans). | alpha |

More tools land here one at a time (memory explorer, architecture-drift detector, …).

## Skills

Some tools ship a **Skill** (an [Anthropic Agent Skill](https://docs.claude.com) — `SKILL.md` + progressive
disclosure) so an AI agent can invoke the tool's capability directly. Skills in this repo are generic and
open. (Per-tenant and premium skills live in MitrAI's private hosted registry, never here.)

## Contributing

We welcome contributions — see **[CONTRIBUTING.md](CONTRIBUTING.md)**. In short: fork, branch, open a PR
against `main`. Every PR is **squash-merged** into a single, clean, Conventional-Commits commit; CI (build +
tests) must pass and a maintainer must approve.

## Layout

```
mitrai-tools/
  <tool-name>/            # one self-contained tool per directory (own go.mod, zero/minimal deps)
  .github/                # PR template, CODEOWNERS, CI
  CONTRIBUTING.md
  LICENSE                 # Apache-2.0
```

## License

[Apache License 2.0](LICENSE). © MitrAI.
