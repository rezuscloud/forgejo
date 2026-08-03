#!/usr/bin/env bash
set -euo pipefail

# Vendor the upstream Forgejo chart, then re-apply the local Rezuscloud patches
# needed to run a Forgejo 16 custom image before upstream ships an appVersion 16
# chart.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="$ROOT/charts/forgejo"
TMP="$(mktemp -d)"
UPSTREAM_VERSION="${UPSTREAM_VERSION:-17.1.1}"
APP_VERSION="${APP_VERSION:-16.0.0-rc.1}"
CHART_VERSION="${CHART_VERSION:-18.0.0-rc.1}"

cleanup() {
  rm -rf "$TMP"
}
trap cleanup EXIT

helm pull "oci://code.forgejo.org/forgejo-helm/forgejo" --version "$UPSTREAM_VERSION" --untar --untardir "$TMP"
rm -rf "$CHART_DIR"
cp -a "$TMP/forgejo" "$CHART_DIR"

# Preserve Rezuscloud-specific files that live alongside the vendored chart.
for f in REZUSCLOUD.md values-rc-testing.yaml; do
  if [ -f "$ROOT/charts/forgejo.rezuscloud/$f" ]; then
    cp "$ROOT/charts/forgejo.rezuscloud/$f" "$CHART_DIR/$f"
  fi
done

python3 <<PYEOF
from pathlib import Path
chart = Path(r"$CHART_DIR/Chart.yaml")
values = Path(r"$CHART_DIR/values.yaml")

c = chart.read_text()
c = c.replace("appVersion: 15.0.3", "appVersion: $APP_VERSION")
c = c.replace("version: 17.1.1", "version: $CHART_VERSION")
c = c.replace(
    "- https://codeberg.org/forgejo/forgejo\ntype: application",
    "- https://codeberg.org/forgejo/forgejo\n- https://github.com/rezuscloud/forgejo-monorepo\n- https://github.com/rezuscloud/forgejo\ntype: application",
)
chart.write_text(c)

v = values.read_text()
v = v.replace("# Default values for gitea.\n# This is a YAML-formatted file.\n# Declare variables to be passed into your templates.\n",
              "# Default values for Forgejo.\n# This chart is vendored from upstream forgejo-helm 17.1.1 and patched by\n# rezuscloud for a Forgejo 16.0.0-rc.1 deployment while the upstream chart\n# still targets Forgejo 15.x. Keep local changes minimal and rebase onto the\n# upstream chart once an official appVersion 16 chart is released.\n# This is a YAML-formatted file.\n# Declare variables to be passed into your templates.\n")
v = v.replace("registry: code.forgejo.org", "registry: ghcr.io")
v = v.replace("repository: forgejo/forgejo", "repository: rezuscloud/forgejo-monorepo-forgejo")
v = v.replace("## @param image.tag Visit: [Image tag](https://code.forgejo.org/forgejo/-/packages/container/forgejo/versions). Defaults to `appVersion` within Chart.yaml.\n",
              "## @param image.tag Custom Rezuscloud Forgejo image tag. Defaults to `appVersion` within Chart.yaml.\n")
values.write_text(v)
PYEOF

echo "Vendored charts/forgejo from upstream ${UPSTREAM_VERSION} -> appVersion ${APP_VERSION}, chart ${CHART_VERSION}"
echo "Re-apply Rezuscloud overlay files from charts/forgejo.rezuscloud/ if present."
