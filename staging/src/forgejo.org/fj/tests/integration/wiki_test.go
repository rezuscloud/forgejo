//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Wiki UX: create page via API fixture -> list -> view.
func TestWiki_ListView(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("wiki-life")
	createTestRepo(t, repo)
	createWikiPage(t, testUser(), repo, "Home", "# Hello wiki")

	out, err := runFj(t, binary, "--repo", testUser()+"/"+repo, "wiki", "list")
	if err != nil {
		t.Fatalf("wiki list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Home") {
		t.Fatalf("wiki list missing page title: %q", out)
	}

	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "wiki", "view", "Home")
	if err != nil {
		t.Fatalf("wiki view: %v\n%s", err, out)
	}
	if !strings.Contains(out, "# Home") {
		t.Fatalf("wiki view missing heading: %q", out)
	}
}
