//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Tag lifecycle: create -> list -> delete -> verify gone.
func TestTag_Lifecycle(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("tag-life")
	createTestRepo(t, repo)

	out, err := runFj(t, binary, "--repo", testUser()+"/"+repo, "tag", "create", "v1.0.0", "--message", "first tag")
	if err != nil {
		t.Fatalf("tag create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created tag v1.0.0") {
		t.Fatalf("unexpected create output: %q", out)
	}
	if !tagExists(t, testUser(), repo, "v1.0.0") {
		t.Fatal("tag not visible via API after create")
	}

	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "tag", "list")
	if err != nil {
		t.Fatalf("tag list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "v1.0.0") {
		t.Fatalf("tag list missing v1.0.0: %q", out)
	}

	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "tag", "delete", "v1.0.0")
	if err != nil {
		t.Fatalf("tag delete: %v\n%s", err, out)
	}
	if tagExists(t, testUser(), repo, "v1.0.0") {
		t.Fatal("tag still visible via API after delete")
	}
}
