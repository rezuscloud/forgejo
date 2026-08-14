# distributions/

Historically this directory held hand-edited `forgejo-<version>.env` files that
pinned `FORGEJO_RELEASE_VERSION` (e.g. `16.0.2-rezuscloud.3`) for the release
workflow. **Those are gone.**

The fork follows **upstream-identity versioning**:

- **Identity is the upstream version** — the binary reports `16.0.2`, nothing else.
- Release tags `v16.0.2-rezus.N` are cut **by the fork-maintenance sync engine**
  (machine-computed counter), never by hand.
- The release workflow (`.github/workflows/release.yml`) derives everything
  from the tag: docker tag `16.0.2-rezus.N` (hyphen-encoded `+rezus.N`
  semver build metadata), `VERSION=16.0.2+rezus.N`, chart `18.0.2-rezus.N`
  (upstream forgejo-helm N+2 convention), fingerprint `<version>-<shortsha>`.

There is nothing to edit here. If you need a release, tag per the sync
engine's convention or run the release workflow with `workflow_dispatch`
passing the tag string.
