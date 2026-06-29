# `fj` CLI Parity — Design & Issue Tracker

Goal: prove the Go `fj` CLI in this monorepo has **full feature parity** with
the Rust [`forgejo-cli`](https://github.com/rezuscloud/forgejo-cli) by testing
each command against a containerized Forgejo instance — **without touching
upstream Forgejo's own integration tests**.

## The constraint and how we satisfy it

> Upstream Forgejo owns `tests/integration/` for the server. That suite cares
> about Forgejo only. We add a **separate workflow** that integrates Forgejo
> with its auxiliary tools (the Go SDK + `fj` CLI).

Conflict-free by construction — verified against `origin/forgejo`:

1. **Location**: all new code under `staging/src/forgejo.org/fj/tests/`.
   Upstream has no `staging/` → `git merge` never conflicts.
2. **Test scope**: upstream `Makefile` defines
   `GO_DIRS := build cmd models modules routers services tests` — `staging/`
   is excluded, so upstream `make test` never compiles or runs our tests.
3. **CI surface**: new workflow at `.github/workflows/cli-parity.yml`.
   Upstream `.github/` is empty (Forgejo uses `.forgejo/workflows/`) → no
   collision, independent trigger.

## Architecture

```
        ┌──────────────────────────────────────────────────────┐
        │ .github/workflows/cli-parity.yml (separate workflow) │
        └───────────────────────┬──────────────────────────────┘
                                │
                   ┌────────────▼────────────┐
                   │  TestMain (main_test.go)│
                   │  start Forgejo container│
                   │  bootstrap admin + token│
                   └────────────┬────────────┘
                                │ m.Run() (serial, -p 1)
            ┌───────────────────┼───────────────────┐
            ▼                   ▼                   ▼
      SDK + raw `fj api`   hand-written UX    parity inventory
      (existing 506)       lifecycle tests    (spec drift check)
```

## Monorepo automation (Google-style)

Because the swagger spec, the SDK, the CLI, the generator, and the tests share
one tree, a feature change on the main product (Forgejo) reflects on the
auxiliary tools automatically:

- `spec/swagger.json` is the single contract.
- `client-go/gen/main.go` regenerates the SDK + raw CLI commands from it.
- `parity_inventory_test.go` cross-checks every hand-written UX command's
  declared `operationId` against the spec → **upstream rename/removal fails
  the build and names the broken `fj` command**.
- Issue #03 moves the parity map into the generator (`--parity-out`), so the
  drift check is itself generated, not hand-maintained.

## Issue tracker

| # | Issue | Status |
|---|-------|--------|
| 01 | [Containerized TestMain harness](issues/01-container-harness.md) | done |
| 02 | [Parity inventory (gap report)](issues/02-parity-inventory.md) | done |
| 03 | [Generator `--parity-out` (automation)](issues/03-generator-parity-map.md) | done |
| 04 | [Per-command lifecycle tests](issues/04-per-command-tests.md) | in progress (issue, tag, release, repo, auth, user key, wiki, actions done) |
| 05 | [Separate GitHub Actions workflow](issues/05-ci-workflow.md) | done |
| 07 | [Optional fork-image integration](issues/07-fork-image-integration.md) | done |
| 06 | [llm-wiki documentation](issues/06-wiki-doc.md) | planned |

## Reference patterns

- `forgejo-api` (Rust): CI service container `forgejo-testing:3000`,
  `TestingAdmin`/`password`, serial `--jobs 1`, per-test local git repos for
  clone/PR flows. ~24 `#[tokio::test]`.
- `forgejo-cli` (Rust): **has no integration tests** — parity was never
  defined. This suite defines it.

## See also

- [`staging/src/forgejo.org/fj/tests/README.md`](../../staging/src/forgejo.org/fj/tests/README.md)
- `llm-wiki` `[[fork-maintenance]]` concept page
