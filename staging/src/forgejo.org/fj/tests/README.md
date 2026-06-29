# `fj` Integration & Parity Test Suite

End-to-end tests for the `fj` CLI (and the `client-go` SDK it is generated
against) exercised against a **containerized Forgejo instance**. This suite is
**additive** to upstream Forgejo — it lives entirely under the staging module
and never touches the Forgejo server's own test suite.

## Why a separate suite (conflict-free by construction)

Upstream Forgejo owns `tests/integration/` for the **server**. We must not
touch it — it cares about Forgejo only. Our suite cares about the **tools that
integrate with Forgejo**: the Go SDK and the `fj` CLI. Three properties keep us
conflict-free across every upstream merge:

| Property | Evidence | Consequence |
|----------|----------|-------------|
| Location is additive | Everything lives under `staging/src/forgejo.org/fj/tests/` | Upstream has no `staging/` — `git merge` never conflicts |
| Excluded from upstream's test scope | Upstream `Makefile`: `GO_DIRS := build cmd models modules routers services tests` | Upstream `make test` / lint never compiles or runs our tests |
| Separate CI surface | Workflow at `.github/workflows/cli-parity.yml` | Upstream `.github/` is empty (Forgejo uses `.forgejo/workflows`) — no collision, different trigger |

In short: **upstream's integration tests run unchanged and unaware; our suite
runs as an independent workflow that integrates Forgejo with its auxiliary
tools**, exactly mirroring how Kubernetes ships `staging/`-level tests for
`client-go`/`kubectl` separately from the API-server's own e2e suite.

## What this suite verifies

Three layers, one container:

1. **SDK** (`client-go`) — exercised through the CLI (the CLI calls the SDK).
2. **Raw API surface** (`fj api <service> <method>`) — the auto-generated
   command tree, ~506 commands, one test each (`zz_generated_integration_test.go`).
3. **Hand-written UX commands** (`fj issue/repo/pr/auth/release/tag/user/org/…`)
   — lifecycle tests that prove **feature parity with the Rust `forgejo-cli`**.

The parity contract is `spec/swagger.json`: every endpoint the CLI exposes maps
to an `operationId` in the spec. When upstream renames or removes an
`operationId`, the parity inventory test fails and names the affected `fj`
command — the monorepo advantage: spec, SDK, CLI, and tests drift together.

## Running

```bash
# Auto-starts a Forgejo container if FORGEJO_TEST_URL is unset.
go test -tags=integration -p 1 ./staging/src/forgejo.org/fj/tests/integration/

# Against an externally-provided instance (the k8s-config CronJob path):
FORGEJO_TEST_URL=http://localhost:3000 FORGEJO_TEST_TOKEN=xxx \
  go test -tags=integration ./staging/src/forgejo.org/fj/tests/integration/

# Against a specific Forgejo image (override the default):
FORGEJO_TEST_IMAGE=codeberg.org/forgejo/forgejo:15 \
  go test -tags=integration ./staging/src/forgejo.org/fj/tests/integration/

# Parity inventory only (no container, no live instance — fast codegen check):
go test ./staging/src/forgejo.org/fj/tests/integration/ -run TestParityInventory -v
```

## Test image

The default image is `codeberg.org/forgejo/forgejo:15` (validated live on
2026-06-29). Override with `FORGEJO_TEST_IMAGE`.

The `.github/workflows/cli-parity.yml` workflow has two integration modes:

- **default (`upstream-15`)** — fast, runs on every PR/push; uses the upstream
  image. This is what the suite was validated against.
- **optional (`fork-image`)** — `workflow_dispatch` only, heavy; builds the
  fork's own image from this repo's `Dockerfile` and tests against it. This is
  the only path that exercises the Actions log API (PR #12666) which is absent
  from upstream :15. Skipped on PR/push to keep CI fast.

`-p 1` (serial) is required: Forgejo has eventual-consistency surfaces (issue
indexing, PR creation after push) that race under parallel test packages — the
same reason the Rust `forgejo-api` suite runs `--jobs 1`.

## Layout

```
tests/integration/
├── main_test.go               TestMain: container lifecycle (build tag: integration)
├── container_test.go          docker exec helpers (build tag: integration)
├── lifecycle_helpers_test.go  env/dir/API helpers for UX lifecycle tests
├── integration_test.go        SDK smoke tests + shared API helpers   (existing)
├── cli_test.go                fj binary build/run helpers + smoke    (existing)
├── codegen_test.go            `hack/gen-staging.sh` drift check      (new, no tag)
├── parity_inventory_test.go   UX-command ↔ operationId drift report  (new, no tag)
├── issue_test.go              hand-written `issue` lifecycle suite
├── tag_test.go                hand-written `tag` lifecycle suite
├── release_test.go            hand-written `release` lifecycle suite
├── repo_test.go               hand-written `repo` view/clone/fork suite
├── auth_test.go               isolated auth-store suite
├── user_test.go               `user key` lifecycle suite
├── wiki_test.go               hand-written `wiki` list/view suite
├── actions_test.go            variables/secrets + empty-state suite
└── zz_generated_integration_test.go  506 raw `fj api` tests          (existing)
```

The design, issue tracker, and per-command parity matrices live in
[`docs/cli-parity/`](../../../../../../docs/cli-parity/) at the repo root.
