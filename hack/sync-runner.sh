#!/usr/bin/env bash
set -euo pipefail

# Bump the vendored forgejo/runner tree (runner/) to an upstream tag, mirroring
# hack/sync-forgejo-chart.sh. The tree stays byte-pristine upstream: all local
# logic (darwin builds, stamping, packaging) lives in .github/workflows/, so a
# bump is a clean tree replacement plus the pin file.
#
# Usage: hack/sync-runner.sh v12.7.4
#   UPSTREAM_REF must be a tag on code.forgejo.org/forgejo/runner.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNNER_DIR="$ROOT/runner"
PIN_FILE="$ROOT/runners/RUNNER_UPSTREAM"
TMP="$(mktemp -d)"
UPSTREAM_REF="${1:-}"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

[[ -n "$UPSTREAM_REF" ]] || { echo "usage: $0 <upstream-tag>   e.g. v12.7.4" >&2; exit 1; }
[[ "$UPSTREAM_REF" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "ERROR: '$UPSTREAM_REF' is not a vX.Y.Z tag" >&2; exit 1; }

echo "· fetching code.forgejo.org/forgejo/runner at $UPSTREAM_REF"
curl -fsSL "https://code.forgejo.org/forgejo/runner/archive/${UPSTREAM_REF}.tar.gz" -o "$TMP/src.tar.gz"
tar -xzf "$TMP/src.tar.gz" -C "$TMP"
SRC_DIR="$(echo "$TMP/runner"*)"

# sanity: it builds before we vendor it
( cd "$SRC_DIR" && CGO_ENABLED=0 go build -o "$TMP/forgejo-runner" . )
"$TMP/forgejo-runner" --version

rm -rf "$RUNNER_DIR"
mkdir -p "$RUNNER_DIR"
rsync -a --exclude '.git' "$SRC_DIR/" "$RUNNER_DIR/"

# keep the pin authoritative and in lockstep
cat > "$PIN_FILE" <<EOF
# Upstream forgejo/runner tag vendored at runner/ (hack/sync-runner.sh bumps it).
# The release stamps binaries <this>-rezus.<N>; keep in lockstep with the k8s fleet.
$UPSTREAM_REF
EOF

echo "✓ runner/ vendored at $UPSTREAM_REF — pin updated, build + --version verified"
echo "  next: commit runner/ + $PIN_FILE, then cut the next v*-rezus.* release"
