//go:build integration

// Package integration — hand-written UX command lifecycle tests.
//
// These are the per-command parity tests (issue #04). Each test exercises the
// full user-facing flow (create -> read -> mutate -> verify -> cleanup) of a
// hand-written `fj` UX command against the containerized instance, asserting
// human-readable output AND round-tripping via the API to confirm state. The
// command set and matrix are derived from the Rust forgejo-cli enum surface.
//
// This file covers the `issue` group: create, view, list, comment, close,
// search. It is the template for the remaining groups (tag, release, repo,
// pr, auth, user, org, wiki, actions).
package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestIssue_Lifecycle is the canonical CRUD+search flow for `fj issue`.
// Serial by design (package runs with -p 1); uses a unique repo per test.
func TestIssue_Lifecycle(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("issue-life")
	createTestRepo(t, repo)

	// --- create ---
	out, err := runFj(t, binary, "--repo", testUser()+"/"+repo,
		"issue", "create", "--title", "lifecycle issue", "--body", "created by test")
	if err != nil {
		t.Fatalf("issue create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "lifecycle issue") {
		t.Fatalf("issue create output missing title: %q", out)
	}
	index := extractIssueIndex(t, out) // "#42" or "Created #42"

	// --- view ---
	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "issue", "view", index)
	if err != nil {
		t.Fatalf("issue view: %v\n%s", err, out)
	}
	if !strings.Contains(out, "lifecycle issue") {
		t.Fatalf("issue view missing title: %q", out)
	}
	if !strings.Contains(out, "created by test") {
		t.Fatalf("issue view missing body: %q", out)
	}

	// --- list (open) ---
	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "issue", "list", "--state", "open")
	if err != nil {
		t.Fatalf("issue list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "lifecycle issue") {
		t.Fatalf("issue list (open) missing the issue: %q", out)
	}

	// --- comment ---
	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "issue", "comment", index, "--body", "a comment")
	if err != nil {
		t.Fatalf("issue comment: %v\n%s", err, out)
	}
	// Verify the comment landed via the API (round-trip check).
	if !apiHasComment(t, repo, index, "a comment") {
		t.Fatalf("comment not found on issue %s via API", index)
	}

	// --- close ---
	out, err = runFj(t, binary, "--repo", testUser()+"/"+repo, "issue", "close", index)
	if err != nil {
		t.Fatalf("issue close: %v\n%s", err, out)
	}

	// --- list (open) should now exclude it; list (closed) should include it ---
	out, _ = runFj(t, binary, "--repo", testUser()+"/"+repo, "issue", "list", "--state", "open")
	if strings.Contains(out, "lifecycle issue") {
		t.Fatalf("closed issue still listed under --state open: %q", out)
	}
	out, _ = runFj(t, binary, "--repo", testUser()+"/"+repo, "issue", "list", "--state", "closed")
	if !strings.Contains(out, "lifecycle issue") {
		t.Fatalf("closed issue missing from --state closed list: %q", out)
	}
}

// TestIssue_Search verifies `fj issue search` finds issues across repos.
func TestIssue_Search(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	repo := uniqueRepoName("issue-search")
	createTestRepo(t, repo)

	_, err := runFj(t, binary, "--repo", testUser()+"/"+repo,
		"issue", "create", "--title", "needle-in-haystack")
	if err != nil {
		t.Fatalf("issue create: %v", err)
	}

	// Search can lag slightly behind issue creation on a fresh instance. Retry
	// for a short window before failing.
	var out string
	for i := 0; i < 6; i++ {
		out, err = runFj(t, binary, "issue", "search", "--query", "needle-in-haystack", "--limit", "50")
		if err == nil && strings.Contains(out, "needle-in-haystack") {
			return
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		t.Fatalf("issue search: %v\n%s", err, out)
	}
	t.Fatalf("issue search did not find the issue after retries: %q", out)
}

// extractIssueIndex pulls the bare issue number ("42") from `fj issue create`
// output that looks like "Created #42: title".
func extractIssueIndex(t *testing.T, out string) string {
	t.Helper()
	// Find "#<digits>" and return the digits.
	start := strings.Index(out, "#")
	if start < 0 {
		t.Fatalf("no '#' in issue create output: %q", out)
	}
	rest := out[start+1:]
	end := strings.IndexAny(rest, " \n\t:")
	if end < 0 {
		end = len(rest)
	}
	idx := strings.TrimSpace(rest[:end])
	if idx == "" {
		t.Fatalf("could not parse issue index from %q", out)
	}
	return idx
}

// apiHasComment round-trips via the API to confirm a comment body is present
// on the issue, proving the CLI's human output isn't masking a failed write.
func apiHasComment(t *testing.T, repo, index, want string) bool {
	t.Helper()
	u := testURL() + "/api/v1/repos/" + testUser() + "/" + repo + "/issues/" + index + "/comments"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "token "+testToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("api comments: %v", err)
	}
	defer resp.Body.Close()
	var comments []struct {
		Body string `json:"body"`
	}
	json.NewDecoder(resp.Body).Decode(&comments)
	for _, c := range comments {
		if strings.Contains(c.Body, want) {
			return true
		}
	}
	return false
}
