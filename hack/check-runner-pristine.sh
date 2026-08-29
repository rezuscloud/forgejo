#!/usr/bin/env bash
set -euo pipefail

# Verifies runner/ is byte-identical to the upstream archive at the tag pinned
# in runners/RUNNER_UPSTREAM. Complements hack/sync-runner.sh (which writes the
# tree + pin) with a read-only check: RED here means unaccounted local edits
# crept into the vendored tree.
#
# Severity model (shared with check-runner-lag.sh):
#   RED  = automation should have already fixed it (re-sync is mechanical).
#   n/a  — this check has no warn tier: pristine is binary.
#
# Usage: hack/check-runner-pristine.sh   (exit 0 = pristine, 1 = drift)

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIN_FILE="$ROOT/runners/RUNNER_UPSTREAM"
TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

PIN="$(tail -n1 "$PIN_FILE" | tr -d '[:space:]')"
[[ "$PIN" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "ERROR: pin '$PIN' is not a vX.Y.Z tag" >&2; exit 1; }

echo "· fetching code.forgejo.org/forgejo/runner at $PIN"
curl -fsSL "https://code.forgejo.org/forgejo/runner/archive/${PIN}.tar.gz" -o "$TMP/src.tar.gz"
tar -xzf "$TMP/src.tar.gz" -C "$TMP"
SRC_DIR="$(echo "$TMP/runner"*)"

echo "· diffing runner/ against upstream $PIN"
# --strip-trailing-cr: this repo normalizes CRLF→LF at commit time (git
# text=auto), so ~a dozen upstream CRLF files differ by EOL policy only.
# The guard exists for CONTENT drift, not the repo's EOL policy.
if diff -r --strip-trailing-cr "$SRC_DIR" "$ROOT/runner"; then
  echo "✓ runner/ is byte-pristine at $PIN"
else
  echo "" >&2
  echo "✗ runner/ drifted from upstream $PIN — unaccounted local edits" >&2
  echo "  fix: hack/sync-runner.sh $PIN (re-sync), or move the logic into" >&2
  echo "  .github/workflows/ or runners/ — the vendored tree stays pristine." >&2
  exit 1
fi
