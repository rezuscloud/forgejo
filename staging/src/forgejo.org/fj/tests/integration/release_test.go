//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Release lifecycle: create -> view -> list -> delete -> verify gone.
func TestRelease_Lifecycle(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("release-life")
	createTestRepo(t, repo)

	if _, err := runFj(t, binary, "--repo", testUser()+"/"+repo, "tag", "create", "v1.2.3"); err != nil {
		t.Fatal(err)
	}

	out, err := runFj(t, binary, "--repo", testUser()+"/"+repo,
		"release", "create", "--tag", "v1.2.3", "--name", "Release 1.2.3", "--body", "release body")
	if err != nil {
		t.Fatalf("release create: %v\n%s", err, out)
	}
	id := extractIssueIndex(t, out) // same #<id> parsing shape as issue create
	if !releaseExists(t, testUser(), repo, "v1.2.3") {
		t.Fatal("release not visible via API after create")
	}

	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "release", "view", id)
	if err != nil {
		t.Fatalf("release view: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Release 1.2.3") || !strings.Contains(out, "v1.2.3") {
		t.Fatalf("release view missing name/tag: %q", out)
	}

	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "release", "list")
	if err != nil {
		t.Fatalf("release list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Release 1.2.3") {
		t.Fatalf("release list missing release: %q", out)
	}

	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "release", "delete", id)
	if err != nil {
		t.Fatalf("release delete: %v\n%s", err, out)
	}
	if releaseExists(t, testUser(), repo, "v1.2.3") {
		t.Fatal("release still visible via API after delete")
	}
}
