# Forgejo distributions

Each `.env` file here pins the metadata for one Forgejo distribution release
built and published by `.github/workflows/forgejo-release.yml`.

```sh
FORGEJO_SOURCE_REPOSITORY=rezuscloud/forgejo   # upstream-of-us fork
FORGEJO_SOURCE_REF=forgejo                      # branch/ref to build from
FORGEJO_RELEASE_VERSION=16.0.0-rc.1            # injected as RELEASE_VERSION + Chart.appVersion
FORGEJO_CHART_VERSION=18.0.0-rc.1              # published chart version
FORGEJO_IMAGE_REPOSITORY=ghcr.io/rezuscloud/forgejo-monorepo-forgejo
FORGEJO_CHART_OCI_REPOSITORY=oci://ghcr.io/rezuscloud/forgejo-monorepo-charts
```

## Channel model

- **RC channel** — filename `forgejo-<version>-rc<N>.env`, `FORGEJO_CHART_VERSION`
  carries a SemVer prerelease suffix (`-rc.N`). Use for testing ahead of an
  upstream Forgejo release (see `charts/forgejo/values-rc-testing.yaml`).
- **Stable channel** — filename `forgejo-<version>.env` with no prerelease
  suffix. Used once upstream Forgejo <version> is released and we promote the
  custom build.

To cut a new release: add a metadata file, push tag `forgejo-v<version>`, and the
workflow publishes images + chart + **the matching `fj` CLI binary** + GitHub
release asset. The Forgejo server image, the Helm chart appVersion, and the `fj`
Client Version all advance together as one SemVer unit (the CLI is generated
from this exact Forgejo swagger spec, so Client Version == Server Version for a
paired release).
