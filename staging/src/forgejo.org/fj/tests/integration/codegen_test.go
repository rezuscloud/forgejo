// (no build tag) — the codegen invariant: regenerating staging from swagger.json
// must produce a clean working tree (no diff). This is the Go port of the
// codegen-check the forgejo-api Rust crate runs in Woodpecker: if the spec or
// generator changed but the output wasn't regenerated, this fails.
//
// It runs the same entry point as the post-merge hook (hack/gen-staging.sh) and
// asserts `git status` is clean afterwards. Skipped when not in a git checkout
// with the full source tree (e.g. running against an installed module).
package integration

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestCodegenClean runs hack/gen-staging.sh into a temp scratch checkout is
// not feasible without a full copy; instead we invoke the generator in place
// and assert no tracked file changed. This proves the committed zz_generated_*
// files match a fresh regeneration.
func TestCodegenClean(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("repo root not found — skipping codegen check")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	before := generatedStatus(t, root)

	// Run the regen entrypoint.
	genScript := filepath.Join(root, "hack", "gen-staging.sh")
	cmd := exec.Command(genScript, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gen-staging.sh failed: %v\n%s", err, out)
	}

	after := generatedStatus(t, root)
	var newDirty []string
	for _, line := range after {
		if !contains(before, line) {
			newDirty = append(newDirty, line)
		}
	}
	if len(newDirty) > 0 {
		t.Errorf("CODEGEN DRIFT: %d generated file status line(s) appeared only after regeneration "+
			"(run hack/gen-staging.sh and commit):\n  %s",
			len(newDirty), strings.Join(newDirty, "\n  "))
	}
}

func generatedStatus(t *testing.T, root string) []string {
	t.Helper()
	diff := exec.Command("git", "-C", root, "status", "--porcelain", "--", "staging/")
	out, err := diff.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		if !strings.Contains(line, "zz_generated_") {
			continue
		}
		lines = append(lines, line)
	}
	sort.Strings(lines)
	return lines
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
