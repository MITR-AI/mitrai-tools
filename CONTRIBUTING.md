# Contributing to mitrai-tools

Thanks for your interest! This repo is open-core: everything here is Apache-2.0 and community contributions
are welcome. This guide is also how **maintainers** handle incoming contributor PRs, so the process is the
same for everyone and the public history stays production-grade.

## How contributor PRs are handled (the workflow)

1. **Fork & branch.** External contributors work on a fork; maintainers use a topic branch. Never commit to
   `main` directly — it is protected.
2. **One focused PR.** Keep a PR scoped to a single change/tool. Open it against `main`.
3. **CI must pass.** Every PR runs build + tests for the affected tool(s). Red CI blocks merge.
4. **Tests required.** Bugfixes ship a failing-test-first; features ship negative + edge coverage, not just
   the happy path. A PR that changes logic without a test change will be asked to add one.
5. **Maintainer review.** At least one maintainer (see [CODEOWNERS](.github/CODEOWNERS)) must approve. Review
   conversations must be resolved before merge.
6. **Squash-merge only (see below).** The maintainer merges via **Squash and merge**; the branch is deleted.
7. **DCO.** By submitting a PR you certify the [Developer Certificate of Origin](https://developercertificate.org/)
   (i.e. you wrote it / have the right to submit it under Apache-2.0). Sign-off (`git commit -s`) is appreciated.

Contributors only ever touch the open-source cores in this repo. Commercial/premium features live in a
separate private repository and are **not** part of this project.

## Commit & PR standards (PRODUCTION-GRADE — enforced)

The git history and PR discussion of this repo are **public and permanent** — treat them as production.

- **Squash every PR into ONE commit.** The repo is configured to allow **only** "Squash and merge"
  (merge-commit and rebase-merge are disabled). No `main` history of "wip", "fix", "address review" noise —
  one PR becomes one clean commit.
- **Conventional Commits** for the squashed message: `type(scope): summary`
  (`feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `perf`), imperative mood, ≤ 72-char subject,
  a body explaining *why* when it isn't obvious. The squash title defaults to the **PR title**, so write the
  PR title to this standard.
- **PR description** follows the [pull request template](.github/PULL_REQUEST_TEMPLATE.md): what, why, how
  tested. No empty descriptions.
- **Comments & code** are professional and production-quality — clear names, no dead code, no debug prints,
  no throwaway language in commits/comments/reviews. Assume a customer will read it.

Maintainers: when squashing, **edit the squash commit title/body to meet this standard** before confirming —
do not accept GitHub's auto-generated bullet list of intermediate commits.

## Local checks before you push

CI runs these across every tool module (matrix Go 1.22 + 1.23): `gofmt`, `go build`, `go vet`,
**`go test -race`**, **golangci-lint**, a **coverage floor (≥65% per module — add tests, don't lower it)**,
and (advisory) govulncheck + CodeQL. Reproduce locally:

```sh
cd <tool>/          # e.g. agent-trace-viewer
gofmt -l .          # must print nothing
go build ./... && go vet ./...
go test -race -cover ./...   # keep module coverage ≥ 65%; failure-surface tests (neg + edge), not just happy path
golangci-lint run ./...      # defaults: errcheck / staticcheck / govet / unused / ineffassign / gosimple
```

## Adding a new tool

A tool is a self-contained directory with its own `go.mod` (zero or minimal dependencies), a `README.md`
(what it is, who it's for, the OSS-vs-premium split), tests covering the failure surface, and — if it exposes
an agent-invokable capability — a `skills/<name>/SKILL.md`. Add its CI to `.github/workflows/ci.yml`.
