<!--
  Your PR will be SQUASH-MERGED into a single commit. Write the PR **title** as a Conventional Commit,
  e.g. `feat(agent-trace-viewer): add span filtering` — it becomes the commit subject.
-->

## What
<!-- One or two sentences: what does this change do? -->

## Why
<!-- The motivation / problem being solved. Link an issue if there is one (Closes #123). -->

## How it was tested
<!-- Commands run and what you observed. New tests added? Cover the failure surface, not just happy path. -->

## Checklist
- [ ] `go build ./...` and `go test ./...` pass for the affected tool(s)
- [ ] `gofmt -l .` prints nothing
- [ ] Tests added/updated (negative + edge, not just happy path)
- [ ] PR title is a Conventional Commit (it becomes the squash commit subject)
- [ ] Docs/README updated if behavior or usage changed
