# 01 — Containerized `TestMain` harness

**Status:** done
**Depends on:** —
**Blocks:** 02, 04, 05

## Goal

A `TestMain` that, when no live Forgejo is provided via `FORGEJO_TEST_URL`,
starts a Forgejo container, bootstraps an admin user + token, exports the
standard test env vars, runs the tests, and tears the container down. When an
external instance **is** provided (the `k8s-config` CronJob path), it is a
no-op pass-through so existing behavior is unchanged.

## Non-goals

- Touching upstream Forgejo's `tests/integration/` suite.
- Adding a `go.mod` dependency to the `fj` module (use `os/exec` + `docker`,
  not testcontainers — keep the module lean, per `staging/README.md` rule #1).

## Acceptance criteria

- [x] `main_test.go` with `//go:build integration` providing the only
      `TestMain` for the package when the tag is set.
- [x] Detects `FORGEJO_TEST_URL` → external instance → no container started.
- [x] Otherwise starts `docker run` with the image from
      `$FORGEJO_TEST_IMAGE` (default `codeberg.org/forgejo/forgejo:16`),
      mapped to an ephemeral host port.
- [x] Polls `GET /api/v1/version` until 200 (timeout 120s).
- [x] Creates the admin user via `docker exec` (`forgejo`/`gitea` CLI).
- [x] Creates an admin API token via the API with `scopes: ["all"]`.
- [x] Exports `FORGEJO_TEST_URL` / `FORGEJO_TEST_TOKEN` / `FORGEJO_TEST_USER`
      into the process env so existing helpers (`testURL()`, `testToken()`)
      work unchanged.
- [x] Tears down the container on exit (best-effort, even on failure).
- [x] Without `-tags=integration` the package behaves exactly as before
      (no `TestMain` compiled; tests skip if no instance).

## Files

- `staging/src/forgejo.org/fj/tests/integration/main_test.go`
- `staging/src/forgejo.org/fj/tests/integration/container_test.go`
