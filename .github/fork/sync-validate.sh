#!/usr/bin/env bash
# sync-validate.sh — invariant gates for the fork-maintenance mapping rows.
#
# Runs after every upstream merge, before the merge is pushed. The model:
#   ours ⊇ theirs must hold AND our delta must still build + be intact.
# Regen output (SDK, tidy) is EXPECTED here — the sync commits it on top of
# the merge before pushing. The image build (dev-build.yml) is the final gate.
#
# Usage: sync-validate.sh <merge-base-sha>   (run from the repo root)
set -euo pipefail

MERGE_BASE="${1:?usage: sync-validate.sh <merge-base-sha>}"
HEAD_SHA=$(git rev-parse HEAD)

SWAGGER_TEMPLATE="templates/swagger/v1_json.tmpl"
# Absolute paths: the regen below runs from inside $SDK_DIR, so relative
# FJ_DIR paths used to resolve nowhere — and writeFile swallows errors, so
# the CLI/test outputs silently never regenerated in CI. Absolute fixes
# that and keeps -polish-out honest for the descriptor layer.
SDK_DIR="$(pwd)/staging/src/forgejo.org/client-go"
FJ_DIR="$(pwd)/staging/src/forgejo.org/fj"

# Delta signatures — upstream-touched files where our modifications must
# survive every merge. If a signature disappears, the merge dropped our
# delta: fail loudly instead of shipping an upstream-only tree.
SIGNATURES=(
  "go.mod|forgejo.org/client-go => ./staging/src/forgejo.org/client-go"
  "go.mod|forgejo.org/fj => ./staging/src/forgejo.org/fj"
  # review-thread API delta (#115/#118, #119): reply + resolve + in_reply_to
  "modules/structs/pull_review.go|CreatePullReviewCommentReplyOptions"
  "modules/structs/pull_review.go|InReplyTo int64"
  "routers/api/v1/api.go|m.Post(\"/replies\", reqToken(), mustNotBeArchived(), bind(api.CreatePullReviewCommentReplyOptions{})"
  "routers/api/v1/api.go|/resolutions"
  "routers/api/v1/repo/pull_review.go|createPullReviewCommentReply"
  "routers/api/v1/repo/pull_review.go|CreatePullReviewCommentResolution"
  "services/convert/pull_review.go|codeConversationHeadID"
  "services/convert/pull_review.go|LoadResolveDoers"
  "templates/swagger/v1_json.tmpl|\"in_reply_to\""
  "tests/integration/api_pull_review_test.go|TestAPIPullReviewCommentReplyResolve"
)

echo "== [1/4] SDK regen (only if swagger changed) =="
if [ -f "$SWAGGER_TEMPLATE" ]; then
  if ! git diff --quiet "$MERGE_BASE" "$HEAD_SHA" -- "$SWAGGER_TEMPLATE" 2>/dev/null; then
    echo "swagger changed — regenerating SDK + CLI + tests"
    # order-stable strip (.github/fork/specstrip): preserves the committed
    # spec's key order/escaping/version so an unchanged template re-strips
    # to a byte-identical spec (the old sorted-key remarshal churned every
    # line and dropped info.version)
    go run ./.github/fork/specstrip -out "$SDK_DIR/spec/swagger.json" "$SWAGGER_TEMPLATE"
    (cd "$SDK_DIR" && go run ./gen -spec spec/swagger.json -out . \
      -cli-out "$FJ_DIR/pkg/cmd/" -test-out "$FJ_DIR/tests/integration/" \
      -polish-out "$FJ_DIR/pkg/cmd/")
  else
    echo "swagger unchanged"
  fi
fi

echo "== [2/4] go mod tidy =="
go mod tidy

echo "== [3/4] build the delta (staging modules + root cmd) =="
(cd "$SDK_DIR" && go build ./...)
(cd "$FJ_DIR" && go build ./...)
go build -o /dev/null ./cmd/fj   # -o /dev/null: a root-level ./fj binary collides with the fj module path

echo "== [3.5/4] behavioral gate: feature unit tests (harmostes#564 contract) =="
# Signatures prove the delta's TEXT survived the merge; this proves the
# FEATURES still work. Hermetic unit tests only (no server) — the container
# integration suite runs post-push in ci.yml; this runs PRE-push, blocking
# the direct mapping-row push. Explicit -timeout: a hung test must fail the
# gate, not stall the sync. Add the package here when a feature gains tests.
(cd "$SDK_DIR" && go test -timeout=120s ./...)
(cd "$FJ_DIR" && go test -timeout=120s ./pkg/...)

echo "== [4/4] delta signatures intact =="
FAILED=0
for sig in "${SIGNATURES[@]}"; do
  FILE="${sig%%|*}"; NEEDLE="${sig#*|}"
  if grep -qF "$NEEDLE" "$FILE"; then
    echo "  ok: $FILE contains our delta"
  else
    echo "::error::signature lost: '$NEEDLE' no longer in $FILE — the merge dropped our delta"
    FAILED=1
  fi
done
[ "$FAILED" -eq 0 ] || exit 1
echo "validated — invariant holds (ours ⊇ theirs, delta builds and is intact)"
