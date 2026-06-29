#!/usr/bin/env bash
# =============================================================================
# gen-staging.sh — Regenerate ALL staging artifacts from the swagger spec
# =============================================================================
# Single entry point for the post-merge hook (k8s-config forgejo.sh) and for
# the codegen-check step in the cli-parity workflow. Emits:
#
#   client-go/zz_generated_*.go      (types, services, spec version)
#   fj/pkg/cmd/zz_generated_commands.go   (raw `fj api` commands)
#   fj/pkg/cmd/zz_generated_parity.go     (parity contract — FAILS on spec drift)
#   fj/tests/integration/zz_generated_integration_test.go (raw `fj api` tests)
#
# The parity step exits non-zero if any mapped operationId in gen/parity.yaml
# no longer exists in swagger.json, so an upstream rename/removal breaks the
# sync PR's regen before it is pushed — the monorepo automation payoff.
#
# Usage: gen-staging.sh [REPO_ROOT]
# =============================================================================
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
CG_DIR="$ROOT/staging/src/forgejo.org/client-go"
FJ_CMD_DIR="$ROOT/staging/src/forgejo.org/fj/pkg/cmd"
FJ_TEST_DIR="$ROOT/staging/src/forgejo.org/fj/tests/integration"

mkdir -p "$FJ_CMD_DIR" "$FJ_TEST_DIR"

echo "[staging] regenerating from $CG_DIR/spec/swagger.json ..."
cd "$CG_DIR"
go run ./gen \
  -spec spec/swagger.json \
  -out . \
  -cli-out "$FJ_CMD_DIR" \
  -test-out "$FJ_TEST_DIR" \
  -parity gen/parity.yaml \
  -parity-out "$FJ_CMD_DIR"

echo "[staging] go mod tidy (staging modules) ..."
(cd "$CG_DIR"   && go mod tidy)
(cd "$FJ_CMD_DIR/.." && go mod tidy)

echo "[staging] done. Files changed:"
cd "$ROOT" && git status -s staging/
