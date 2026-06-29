# 05 — Separate GitHub Actions workflow

**Status:** done
**Depends on:** 01
**Blocks:** —

## Goal

A workflow at `.github/workflows/cli-parity.yml` that builds `fj`, starts a
Forgejo container, and runs the integration + parity suite. It is
**independent of upstream Forgejo's CI** (Forgejo uses `.forgejo/workflows`;
upstream `.github/` is empty) and runs only when the staging tree or the
workflow itself changes.

## Triggers

- `pull_request` and `push` to `rezus/forgejo` touching:
  `staging/**`, `cmd/fj/**`, `docs/arch/**`, `.github/workflows/cli-parity.yml`
- `workflow_dispatch` (manual).

Deliberately **not** triggered by changes under `models/`, `routers/`,
`services/`, etc. — those are upstream Forgejo's concern and already covered by
the centralized `k8s-config` `validate-fork.sh`.

## Job

1. Checkout (full history not required).
2. `actions/setup-go` from the root `go.mod`.
3. `go build ./cmd/fj` (proves the `replace` graph compiles).
4. `go test ./staging/src/forgejo.org/fj/tests/integration/ -run TestParityInventory -v`
   (fast codegen/spec gate, no container).
5. `go test -tags=integration -p 1 ./staging/src/forgejo.org/fj/tests/integration/`
   (`TestMain` starts the container; `docker` is available on `ubuntu-latest`).

## Conflict-free guarantee

- Upstream `.github/` is empty → no merge conflict on the workflow file.
- Upstream `Makefile` `GO_DIRS` excludes `staging/` → upstream CI never runs
  these tests even if it walked the tree.
- The workflow does **not** invoke upstream `make test` or touch
  `tests/integration/` — upstream Forgejo integration tests are untouched.

## Acceptance criteria

- [x] `.github/workflows/cli-parity.yml` present.
- [x] Path-filtered to additive dirs only.
- [x] Runs parity inventory (no container) + full integration (container).
- [x] Serial (`-p 1`).
