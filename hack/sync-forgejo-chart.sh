#!/usr/bin/env bash
set -euo pipefail

# Vendor the upstream Forgejo Helm chart and re-apply the Rezuscloud patch
# contract. Delegate of the fork-maintenance engine (k8s-config
# forks/forgejo-chart.yaml, mode: subtree) — invoked as
#   hack/sync-forgejo-chart.sh <upstream-chart-version>
# and usable manually. The pin lives in charts/forgejo.rezuscloud/UPSTREAM_CHART;
# the allowed delta is declared in charts/forgejo.rezuscloud/PATCHES.yaml and
# verified by hack/check-chart-patches.sh (CI) and the engine's gate.
#
# The LOCAL chart version + appVersion are preserved across the swap (they are
# independent of the upstream pin: chart N+2 → app N, driven by
# distributions/*.env + forgejo-release.yml). Override with
# CHART_VERSION_OVERRIDE / APP_VERSION_OVERRIDE when cutting a release line.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="$ROOT/charts/forgejo"
SIDECAR="$ROOT/charts/forgejo.rezuscloud"
PIN_FILE="$SIDECAR/UPSTREAM_CHART"
TMP="$(mktemp -d)"

UPSTREAM_REF="${1:-}"
[[ -n "$UPSTREAM_REF" ]] || { echo "usage: $0 <upstream-chart-version>   e.g. 17.1.5" >&2; exit 1; }
[[ "$UPSTREAM_REF" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "ERROR: '$UPSTREAM_REF' is not a chart semver (X.Y.Z)" >&2; exit 1; }

UPSTREAM_OCI="${UPSTREAM_OCI:-oci://code.forgejo.org/forgejo-helm/forgejo}"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

# Preserve the local release-line identity before the swap.
CHART_VERSION="$(yq -r '.version' "$CHART_DIR/Chart.yaml" | tr -d '"')"
APP_VERSION="$(yq -r '.appVersion' "$CHART_DIR/Chart.yaml" | tr -d '"')"
CHART_VERSION="${CHART_VERSION_OVERRIDE:-$CHART_VERSION}"
APP_VERSION="${APP_VERSION_OVERRIDE:-$APP_VERSION}"

# Back up the vendored dependency tarballs (preserve-list: charts/) across the swap.
DEP_TGZ=()
[ -d "$CHART_DIR/charts" ] && DEP_TGZ=("$CHART_DIR"/charts/*.tgz)

echo "· vendoring $UPSTREAM_OCI at $UPSTREAM_REF (local chart $CHART_VERSION / app $APP_VERSION)"
helm pull "$UPSTREAM_OCI" --version "$UPSTREAM_REF" --untar --untardir "$TMP" \
  || helm pull "$UPSTREAM_OCI" --version "$UPSTREAM_REF" --untar --untardir "$TMP" --plain-http

rm -rf "$CHART_DIR"
cp -a "$TMP/forgejo" "$CHART_DIR"

# Re-apply the Rezuscloud overlays (preserve-list, from the sidecar dir).
for f in REZUSCLOUD.md values-rc-testing.yaml; do
  [ -f "$SIDECAR/$f" ] && cp "$SIDECAR/$f" "$CHART_DIR/$f"
done
mkdir -p "$CHART_DIR/charts"
for t in "${DEP_TGZ[@]}"; do [ -e "$t" ] && cp "$t" "$CHART_DIR/charts/"; done

python3 <<PYEOF
import re
from pathlib import Path

chart = Path("$CHART_DIR/Chart.yaml")
values = Path("$CHART_DIR/values.yaml")

c = chart.read_text()
c = re.sub(r"^appVersion: .*$", "appVersion: $APP_VERSION", c, count=1, flags=re.M)
c = re.sub(r"^version: .*$", "version: $CHART_VERSION", c, count=1, flags=re.M)
c = c.replace(
    "- https://codeberg.org/forgejo/forgejo\ntype: application",
    "- https://codeberg.org/forgejo/forgejo\n"
    "- https://github.com/rezuscloud/forgejo-monorepo\n"
    "- https://github.com/rezuscloud/forgejo\ntype: application",
)
chart.write_text(c)

v = values.read_text()
v = v.replace(
    "# Default values for gitea.\n",
    "# Default values for Forgejo.\n"
    "# Vendored from the upstream forgejo-helm chart (pin: charts/forgejo.rezuscloud/UPSTREAM_CHART)\n"
    "# and patched per charts/forgejo.rezuscloud/PATCHES.yaml — keep local changes minimal\n"
    "# and declared there.\n",
)
v = v.replace("registry: code.forgejo.org", "registry: ghcr.io")
v = v.replace("repository: forgejo/forgejo", "repository: rezuscloud/forgejo")
v = v.replace(
    "## @param image.tag Visit: [Image tag](https://code.forgejo.org/forgejo/-/packages/container/forgejo/versions). Defaults to `appVersion` within Chart.yaml.\n",
    "## @param image.tag Custom Rezuscloud Forgejo image tag. Defaults to `appVersion` within Chart.yaml.\n",
)
values.write_text(v)
PYEOF

printf '# Upstream forgejo-helm chart version vendored at charts/forgejo/\n# (hack/sync-forgejo-chart.sh bumps it; PATCHES.yaml declares the allowed delta).\n%s\n' \
  "$UPSTREAM_REF" > "$PIN_FILE"

echo "· helm lint (sanity — patches must leave a working chart)"
helm lint "$CHART_DIR" >/dev/null

echo "✓ charts/forgejo vendored at upstream $UPSTREAM_REF (local chart $CHART_VERSION / app $APP_VERSION)"
git -C "$ROOT" add -A -f charts/forgejo "$PIN_FILE"
echo "  staged (git add -A -f). next: commit, then check-chart-patches.sh must be green"
