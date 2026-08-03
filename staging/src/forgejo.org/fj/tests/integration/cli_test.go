// Package integration — CLI tests that build and run the fj binary against
// a live Forgejo instance, verifying the auto-generated commands work.
package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildFjBinary builds the fj CLI from the repo root and returns its path.
func buildFjBinary(t *testing.T) string {
	t.Helper()
	// Find the repo root (4 levels up from staging/src/forgejo.org/fj/tests/integration)
	repoRoot := os.Getenv("FORGEJO_REPO_ROOT")
	if repoRoot == "" {
		// Try to find it relative to the test file
		dir, _ := os.Getwd()
		for i := 0; i < 10; i++ {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				if _, err := os.Stat(filepath.Join(dir, "cmd", "fj")); err == nil {
					repoRoot = dir
					break
				}
			}
			dir = filepath.Dir(dir)
		}
	}
	if repoRoot == "" {
		t.Skip("FORGEJO_REPO_ROOT not set and repo root not found")
	}

	binaryPath := filepath.Join(t.TempDir(), "fj-test")
	cmd := exec.Command("go", "build", "-mod=mod", "-o", binaryPath, "./cmd/fj")
	cmd.Dir = repoRoot
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build fj: %v", err)
	}
	return binaryPath
}

// runFj executes the fj binary with the given args and returns stdout.
func runFj(t *testing.T, binaryPath string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"--host", testURL()}, args...)
	cmd := exec.Command(binaryPath, fullArgs...)
	cmd.Env = append(os.Environ(), "FORGEJO_TOKEN="+testToken())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Logf("fj %v\nstdout: %s\nstderr: %s", args, stdout.String(), stderr.String())
	}
	return stdout.String(), err
}

// TestCLIApiVersion tests `fj version` against the live server.
func TestCLIApiVersion(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)

	out, err := runFj(t, binary, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Server Version")) {
		t.Fatalf("version output missing Server Version:\n%s", out)
	}
	t.Logf("version output:\n%s", out)
}

// TestCLIApiRepoGet tests the auto-generated `fj api repo repo-get` command.
func TestCLIApiRepoGet(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("cli-repoget")
	createTestRepo(t, repo)

	out, err := runFj(t, binary, "api", "repo", "repo-get",
		"--owner", testUser(),
		"--repo", repo)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output from fj api:\n%s\nerr: %v", out, err)
	}
	if result["name"] != repo {
		t.Fatalf("repo name mismatch: expected %q, got %v", repo, result["name"])
	}
	t.Logf("repo-get returned: %s", result["full_name"])
}

// TestCLIApiRepoListTags tests the auto-generated `fj api repo repo-list-tags`.
func TestCLIApiRepoListTags(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("cli-tags")
	createTestRepo(t, repo)

	out, err := runFj(t, binary, "api", "repo", "repo-list-tags",
		"--owner", testUser(),
		"--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	// Should be a JSON array (possibly empty)
	out = string(bytes.TrimSpace([]byte(out)))
	if len(out) > 0 && out[0] != '[' {
		t.Fatalf("expected JSON array, got: %s", out[:min(100, len(out))])
	}
	t.Logf("repo-list-tags returned %d bytes", len(out))
}

// TestCLIApiMiscGetVersion tests `fj api misc get-version`.
func TestCLIApiMiscGetVersion(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)

	out, err := runFj(t, binary, "api", "misc", "get-version")
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON from fj api misc get-version:\n%s", out)
	}
	if result.Version == "" {
		t.Fatal("empty version")
	}
	t.Logf("server version via fj api: %s", result.Version)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
