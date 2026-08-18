#!/usr/bin/env bash
# sync-validate.sh — invariant gates for the fork-maintenance mapping rows.
#
# Runs after every upstream merge, before the merge is pushed. The model:
#   ours ⊇ theirs must hold AND our delta must still build + be intact.
# Regen output (SDK, tidy) is EXPECTED here — sync.yml commits it on top of
# the merge before pushing. The image build (dev-build.yml) is the final gate.
#
# Usage: sync-validate.sh <merge-base-sha>   (run from the repo root)
set -euo pipefail

MERGE_BASE="${1:?usage: sync-validate.sh <merge-base-sha>}"
HEAD_SHA=$(git rev-parse HEAD)

SWAGGER_TEMPLATE="templates/swagger/v1_json.tmpl"
SDK_DIR="staging/src/forgejo.org/client-go"
FJ_DIR="staging/src/forgejo.org/fj"

# Delta signatures — upstream-touched files where our modifications must
# survive every merge. If a signature disappears, the merge dropped our
# delta: fail loudly instead of shipping an upstream-only tree.
SIGNATURES=(
  "go.mod|forgejo.org/client-go => ./staging/src/forgejo.org/client-go"
  "go.mod|forgejo.org/fj => ./staging/src/forgejo.org/fj"
)

echo "== [1/4] SDK regen (only if swagger changed) =="
if [ -f "$SWAGGER_TEMPLATE" ]; then
  if ! git diff --quiet "$MERGE_BASE" "$HEAD_SHA" -- "$SWAGGER_TEMPLATE" 2>/dev/null; then
    echo "swagger changed — regenerating SDK + CLI + tests"
    cat > /tmp/strip.go << 'EOF'
package main
import ("encoding/json";"fmt";"os";"regexp")
func main(){
 b,_:=os.ReadFile(os.Args[1])
 s:=regexp.MustCompile(`{{[^}]+}}`).ReplaceAllString(string(b),"")
 var v interface{}; json.Unmarshal([]byte(s),&v)
 out,_:=json.MarshalIndent(v,"","  ")
 os.WriteFile(os.Args[2],out,0644)
 fmt.Println("stripped")
}
EOF
    go run /tmp/strip.go "$SWAGGER_TEMPLATE" "$SDK_DIR/spec/swagger.json"
    (cd "$SDK_DIR" && go run ./gen -spec spec/swagger.json -out . \
      -cli-out "$FJ_DIR/pkg/cmd/" -test-out "$FJ_DIR/tests/integration/")
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
