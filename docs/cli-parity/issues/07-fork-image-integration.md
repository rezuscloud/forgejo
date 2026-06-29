# 07 — Optional fork-image integration

**Status:** done
**Depends on:** 01, 05
**Blocks:** —

## Goal

By default the integration suite runs against the upstream Forgejo image
(`codeberg.org/forgejo/forgejo:15`, validated live on 2026-06-29). That path is
fast but does **not** exercise the fork's distinguishing server features — most
notably the Actions job/run logs REST API (PR forgejo/forgejo#12666) which only
ships in v16.

This issue adds an **optional, heavy** workflow mode that builds the fork's own
image from this repo's `Dockerfile` and runs the suite against it, proving the
SDK/CLI work against the exact distribution that ships.

## How it's opt-in

The `integration-fork-image` job is gated on:

```yaml
if: github.event_name == 'workflow_dispatch' && github.event.inputs.test_image == 'fork-image'
```

So it never runs on PR/push (keeping the default path fast). Trigger manually
via the Actions tab: **Run workflow → test_image = fork-image**.

## What it does

1. `actions/checkout` with `fetch-depth: 0` + `fetch-tags: true` (Forgejo
   computes its version from `git describe`; missing tags → malformed version
   → DB migrate rejects it).
2. Resolve `FORGEJO_RELEASE_VERSION` from the first `distributions/forgejo-*.env`
   and write it to `VERSION`.
3. `docker build` the rootful image via `Dockerfile` with
   `--build-arg RELEASE_VERSION=…`, tagged `forgejo-fork-test:local`.
4. Build `fj` (proves the `replace` graph compiles).
5. Run the full integration suite with `FORGEJO_TEST_IMAGE=forgejo-fork-test:local`
   and a generous 45m timeout (the image build alone takes ~10-15m).

## Acceptance criteria

- [x] `integration-fork-image` job present, gated to `workflow_dispatch`.
- [x] Builds the fork image from the in-repo Dockerfile.
- [x] Runs the same staged suite against it via `FORGEJO_TEST_IMAGE`.
- [x] Default `upstream-15` path untouched (fast, validated).
- [x] Documented in `tests/README.md`.
