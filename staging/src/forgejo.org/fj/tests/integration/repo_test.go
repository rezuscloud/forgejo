//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Repo UX tests: view + clone + fork.
func TestRepo_ViewCloneFork(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)

	// view on a private repo
	privateRepo := uniqueRepoName("repo-view")
	createTestRepo(t, privateRepo)
	out, err := runFj(t, binary, "--repo", testUser()+"/"+privateRepo, "repo", "view")
	if err != nil {
		t.Fatalf("repo view: %v\n%s", err, out)
	}
	if !strings.Contains(out, testUser()+"/"+privateRepo) {
		t.Fatalf("repo view missing full name: %q", out)
	}

	// clone on a public repo
	publicRepo := uniqueRepoName("repo-clone")
	createPublicTestRepo(t, publicRepo)
	cloneDir := filepath.Join(t.TempDir(), "clone")
	out, err = runFjWithOptions(t, binary, t.TempDir(), nil, "", "repo", "clone", testUser()+"/"+publicRepo, cloneDir)
	if err != nil {
		t.Fatalf("repo clone: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); err != nil {
		t.Fatalf("clone dir missing .git: %v", err)
	}

	// fork into a dedicated org to avoid same-owner name collision.
	forkSrc := uniqueRepoName("repo-fork")
	createPublicTestRepo(t, forkSrc)
	org := uniqueRepoName("org")
	createTestOrg(t, org)
	out, err = runFj(t, binary, "repo", "fork", testUser()+"/"+forkSrc, "--org", org)
	if err != nil {
		t.Fatalf("repo fork: %v\n%s", err, out)
	}
	if !repoExists(t, org, forkSrc) {
		t.Fatalf("forked repo %s/%s not found via API", org, forkSrc)
	}
}
