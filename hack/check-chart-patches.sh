#!/usr/bin/env bash
set -euo pipefail

# Patch-accounting guard for the vendored forgejo-helm chart (CI-side twin of
# the fork-maintenance engine's gate). Verifies that the diff between
# charts/forgejo and the upstream chart at the pinned version is EXACTLY the
# contract in charts/forgejo.rezuscloud/PATCHES.yaml:
#
#   preserve  locally-added paths must be declared; anything else = RED
#   patches   differing files must be declared AND carry their signature;
#             missing signature or undeclared diff = RED; a declared patch
#             that no longer differs = WARN (stale entry)
#   missing   an upstream path absent locally = RED (deletions are not modeled)
#
# Then sanity-renders the chart: helm lint + helm template (default values).
# Tool failure is never masked — a broken guard fails red.
#
# Env overrides (drills): UPSTREAM_OCI, HELM_PLAIN_HTTP=1.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="$ROOT/charts/forgejo"
SIDECAR="$ROOT/charts/forgejo.rezuscloud"
PIN_FILE="$SIDECAR/UPSTREAM_CHART"
CONTRACT="$SIDECAR/PATCHES.yaml"

UPSTREAM_OCI="${UPSTREAM_OCI:-oci://code.forgejo.org/forgejo-helm/forgejo}"
PIN="$(awk 'NF && $0 !~ /^[[:space:]]*#/ {v=$0} END {print v}' "$PIN_FILE" | tr -d '[:space:]')"
[[ -n "$PIN" ]] || { echo "::error::no pin in $PIN_FILE"; exit 1; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "· pulling $UPSTREAM_OCI at pin $PIN"
HELM_FLAGS=()
[ "${HELM_PLAIN_HTTP:-0}" = "1" ] && HELM_FLAGS+=(--plain-http)
helm pull "$UPSTREAM_OCI" --version "$PIN" --untar --untardir "$TMP" "${HELM_FLAGS[@]}" \
  || { echo "::error::helm pull $UPSTREAM_OCI:$PIN failed (registry down? pin pulled?)"; exit 1; }
SRC_DIR="$(find "$TMP" -mindepth 1 -maxdepth 1 -type d | head -1)"

red=0
warn=0
while IFS= read -r d; do
  case "$d" in
    "Files $SRC_DIR"*)
      rel="$(echo "$d" | sed -E "s#^Files $SRC_DIR/(.*) and .*#\1#")"
      sig="$(yq -r ".patches[] | select(.path == \"$rel\") | .signature" "$CONTRACT" 2>/dev/null)"
      if [ -z "$sig" ] || [ "$sig" = "null" ]; then
        echo "::error::$rel differs from upstream $PIN and is not in PATCHES.yaml — re-vendor via hack/sync-forgejo-chart.sh $PIN or declare the patch"
        red=1
      elif ! grep -qF -- "$sig" "$CHART_DIR/$rel" 2>/dev/null; then
        echo "::error::$rel differs but signature '$sig' not found — patch unaccounted (rebased away?)"
        red=1
      else
        echo "✓ patch: $rel (signature present)"
      fi
      ;;
    "Only in $CHART_DIR"*)
      rel="$(echo "$d" | sed -E "s#^Only in $CHART_DIR/?(.*): (.*)#\1/\2#" | sed 's#^/##')"
      if yq -r '.preserve[]' "$CONTRACT" 2>/dev/null | grep -qxF "${rel%%/*}/" \
         || yq -r '.preserve[]' "$CONTRACT" 2>/dev/null | grep -qxF "$rel"; then
        echo "✓ preserve: $rel (declared)"
      else
        echo "::error::$rel exists only locally and is not in PATCHES.yaml preserve — declare it or drop it"
        red=1
      fi
      ;;
    "Only in $SRC_DIR"*)
      rel="$(echo "$d" | sed -E "s#^Only in $SRC_DIR/?(.*): (.*)#\1/\2#" | sed 's#^/##')"
      echo "::error::$rel exists upstream $PIN but is missing locally — deletions are not modeled; re-vendor"
      red=1
      ;;
  esac
done < <(diff -rq --strip-trailing-cr "$SRC_DIR" "$CHART_DIR" 2>/dev/null || true)

while IFS= read -r e; do
  [ -z "$e" ] && continue
  [ -e "$CHART_DIR/$e" ] || { echo "::error::preserve entry $e missing from charts/forgejo — a vanished preserved file is invisible to the diff"; red=1; }
done < <(yq -r '.preserve[]' "$CONTRACT" 2>/dev/null)

while IFS= read -r p; do
  [ -z "$p" ] && continue
  if [ ! -e "$CHART_DIR/$p" ]; then
    echo "::error::patch entry $p missing from charts/forgejo"
    red=1
  elif diff -q --strip-trailing-cr "$SRC_DIR/$p" "$CHART_DIR/$p" >/dev/null 2>&1; then
    echo "::warning::$p is byte-identical to upstream $PIN — stale PATCHES.yaml entry (upstream absorbed the patch?)"
    warn=1
  fi
done < <(yq -r '.patches[].path' "$CONTRACT" 2>/dev/null)

[ "$red" = "1" ] && { echo "::error::patch-accounting RED — unaccounted differences in charts/forgejo vs upstream $PIN"; exit 1; }

echo "· helm lint + template (the patched chart must render)"
helm lint "$CHART_DIR" >/dev/null || { echo "::error::helm lint failed"; exit 1; }
helm template forgejo "$CHART_DIR" "${HELM_FLAGS[@]}" >/dev/null || { echo "::error::helm template failed"; exit 1; }

echo "✓ chart patch-accounting green vs upstream $PIN (${warn} stale warnings)"
exit 0
