//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Actions UX (implemented surface that does not require a running workflow):
// variables create/list/delete, secrets create/list/delete, runs/tasks empty state.
func TestActions_VariablesSecretsAndEmptyRuns(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("actions-life")
	createTestRepo(t, repo)
	base := []string{"--repo", testUser() + "/" + repo, "actions"}

	out, err := runFj(t, binary, append(base, "variables", "create", "HELLO", "world")...)
	if err != nil {
		t.Fatalf("actions variables create: %v\n%s", err, out)
	}
	out, err = runFj(t, binary, append(base, "variables", "list")...)
	if err != nil {
		t.Fatalf("actions variables list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "HELLO") {
		t.Fatalf("actions variables list missing HELLO: %q", out)
	}
	if _, err = runFj(t, binary, append(base, "variables", "delete", "HELLO")...); err != nil {
		t.Fatalf("actions variables delete: %v", err)
	}

	out, err = runFj(t, binary, append(base, "secrets", "create", "SECRET_ONE", "shh")...)
	if err != nil {
		t.Fatalf("actions secrets create: %v\n%s", err, out)
	}
	out, err = runFj(t, binary, append(base, "secrets", "list")...)
	if err != nil {
		t.Fatalf("actions secrets list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "SECRET_ONE") {
		t.Fatalf("actions secrets list missing SECRET_ONE: %q", out)
	}
	if _, err = runFj(t, binary, append(base, "secrets", "delete", "SECRET_ONE")...); err != nil {
		t.Fatalf("actions secrets delete: %v", err)
	}

	out, err = runFj(t, binary, append(base, "runs")...)
	if err != nil {
		t.Fatalf("actions runs: %v\n%s", err, out)
	}
	if !strings.Contains(out, "no runs") {
		t.Fatalf("expected empty runs state, got: %q", out)
	}

	out, err = runFj(t, binary, append(base, "tasks")...)
	if err != nil {
		t.Fatalf("actions tasks: %v\n%s", err, out)
	}
	if !strings.Contains(out, "0 tasks") && !strings.Contains(out, "tasks") {
		t.Fatalf("expected tasks output, got: %q", out)
	}
}
