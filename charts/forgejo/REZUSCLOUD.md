# Rezuscloud Forgejo distribution

This chart is a vendored copy of the upstream [forgejo-helm](https://code.forgejo.org/forgejo-helm/forgejo-helm) chart, kept structurally identical so it is a **drop-in replacement**: every value the official chart accepts works here unchanged. The only differences are the default image repository and the `Chart.appVersion`, which track the Rezuscloud Forgejo builds published to GHCR.

It is published as an OCI chart, exactly like upstream:

```console
helm pull oci://ghcr.io/rezuscloud/forgejo-monorepo-charts/forgejo --version 18.0.0-rc.1
helm install forgejo oci://ghcr.io/rezuscloud/forgejo-monorepo-charts/forgejo \
  --version 18.0.0-rc.1 -n forgejo -f values.yaml
```

Everything in the upstream `README.md` below (parameters, persistence, ingress, OAuth2, commit signing, …) applies verbatim.

## Vendoring contract (upstream re-sync)

The chart is vendored from the upstream [forgejo-helm](https://code.forgejo.org/forgejo-helm/forgejo)
chart by `hack/sync-forgejo-chart.sh <upstream-version>` and maintained by the
fork-maintenance engine (`forks/forgejo-chart.yaml` in k8s-config, `mode:
subtree`). Three files form the contract:

| File | Role |
|------|------|
| `charts/forgejo.rezuscloud/UPSTREAM_CHART` | the upstream pin (one semver line, comments allowed) |
| `charts/forgejo.rezuscloud/PATCHES.yaml` | the allowed delta: `preserve:` (locally-added paths) + `patches:` (differing files, each with a `signature` that must be present) |
| `hack/check-chart-patches.sh` | the CI twin of the engine gate (`.github/workflows/vendored-guards.yml`, job `chart`) |

Rule: **every difference between this tree and the upstream chart at the pin
must be declared in PATCHES.yaml** — an undeclared diff, or a declared patch
whose signature vanished, is RED (unaccounted drift; re-vendoring is
mechanical). A declared patch that no longer differs is a WARN (stale entry).
The local chart version + appVersion are preserved across re-vendoring (chart
N+2 → app N, driven by `distributions/*.env` + `forgejo-release.yml`), so an
upstream bump never bumps the local release line. Bump classes follow the same
matrix as `runner/` (see `runners/README.md`): patch → auto PR + auto-merge
after CI; minor → prepared PR + manual merge; major → advisory only.

## Versioning model (stable vs RC)

The chart version mirrors the upstream `forgejo-helm` numbering scheme (chart `N+2` → app `N`):

| Chart version | Forgejo appVersion | Image source | Channel |
|---------------|--------------------|--------------|---------|
| `18.0.0-rc.1` | `16.0.0-rc.1`      | `ghcr.io/rezuscloud/forgejo-monorepo-forgejo` | **RC** (test) |
| `18.x.x` (future, no prerelease suffix) | `16.x.x` | GHCR custom build | stable |

Releases are produced by `.github/workflows/forgejo-release.yml`, driven by a metadata file in `distributions/forgejo-<version>.env`. Each release publishes:

- `ghcr.io/rezuscloud/forgejo-monorepo-forgejo:<version>` (rootful)
- `ghcr.io/rezuscloud/forgejo-monorepo-forgejo:<version>-rootless`
- `oci://ghcr.io/rezuscloud/forgejo-monorepo-charts/forgejo:<chart-version>`

RC chart versions carry a SemVer prerelease suffix (`-rc.N`); stable releases do not. Selecting which channel you deploy is just selecting the chart `--version`.

## Deploying an RC build for testing

Because the chart is a drop-in, testing an RC build is the standard Helm workflow: pick the RC chart version (and optionally pin the exact RC image). Two supported patterns:

### 1. Test a whole RC chart release (recommended)

Point your release at the RC chart version. This is identical to how you would test an upstream RC chart:

```console
helm install forgejo-test \
  oci://ghcr.io/rezuscloud/forgejo-monorepo-charts/forgejo \
  --version 18.0.0-rc.1 \
  -n forgejo-test --create-namespace \
  -f values-rc-testing.yaml
```

### 2. Pin a specific RC image against any chart

To test an immutable RC image build (e.g. `16.0.0-rc.1-cbafedda0ae0`) without changing the chart, override `image.tag`. Remember the chart appends the `-rootless` suffix when `image.rootless: true`, so pass a tag that already carries it or set `image.rootless: false`:

```yaml
image:
  rootless: true
  tag: "16.0.0-rc.1-cbafedda0ae0-rootless"
```

See `values-rc-testing.yaml` for a complete non-production test overlay (single replica, a distinct `ROOT_URL`/namespace, relaxed logging).

## Production

`k8s-config/apps/forgejo/` consumes this chart via a Flux `HelmRelease` pinned to the production channel. The chart and image advance together as one SemVer unit so Flux rollback operates on a versioned chart release, not an ad-hoc image tag.

## Re-vendoring

`hack/sync-forgejo-chart.sh` re-pulls the upstream chart and re-applies the Rezuscloud patches (image repository, appVersion, chart version). This file (`REZUSCLOUD.md`) and `values-rc-testing.yaml` are preserved across re-vendors.
