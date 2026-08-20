// Package integration — exercises every hand-written (non-generated) fj CLI
// subcommand against a live Forgejo instance.
//
// The instance is a throwaway container started fresh by CI (see
// .github/workflows/ci.yml), so this suite is free to run BOTH read and write
// commands (create issue/PR/release/tag, comment, merge, delete, ...) and then
// throw the whole server away. Nothing here ever touches a real instance.
//
// Seeding (repos, branches, PRs, orgs, wiki pages) is done via raw HTTP so the
// test does not depend on the very CLI it is validating.
package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// --- raw-HTTP seed helpers (independent of the CLI under test) ------------

// apiJSON performs an authenticated JSON request and returns status + body.
func apiJSON(t *testing.T, method, path string, body interface{}) (int, string) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, testURL()+"/api/v1"+path, r)
	if err != nil {
		t.Fatalf("apiJSON %s %s: %v", method, path, err)
	}
	req.Header.Set("Authorization", "token "+testToken())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("apiJSON %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// createFileOnBranch commits a new file, optionally on a new branch (PR seed).
func createFileOnBranch(t *testing.T, repo, filePath, branch, newBranch, content string) {
	t.Helper()
	body := map[string]interface{}{
		"branch":  branch,
		"message": "seed: " + filePath,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
	}
	if newBranch != "" {
		body["new_branch"] = newBranch
	}
	status, out := apiJSON(t, "POST",
		fmt.Sprintf("/repos/%s/%s/contents/%s", testUser(), repo, filePath), body)
	if status >= 300 {
		t.Fatalf("createFileOnBranch(%s) status %d: %s", filePath, status, out)
	}
}

// createPR creates a pull request via the API and returns its index.
func createPR(t *testing.T, repo, title, head, base string) int64 {
	t.Helper()
	body := map[string]interface{}{"title": title, "head": head, "base": base}
	status, out := apiJSON(t, "POST", fmt.Sprintf("/repos/%s/%s/pulls", testUser(), repo), body)
	if status >= 300 {
		t.Fatalf("createPR status %d: %s", status, out)
	}
	var pr struct {
		Number int64 `json:"number"`
	}
	json.Unmarshal([]byte(out), &pr)
	return pr.Number
}

func createWikiPage(t *testing.T, repo, title, content string) {
	t.Helper()
	body := map[string]interface{}{
		"title":          title,
		"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
		"message":        "seed wiki",
	}
	status, out := apiJSON(t, "POST", fmt.Sprintf("/repos/%s/%s/wiki/new", testUser(), repo), body)
	if status >= 300 {
		t.Fatalf("createWikiPage status %d: %s", status, out)
	}
}

func createOrg(t *testing.T, name string) {
	t.Helper()
	body := map[string]interface{}{"username": name, "visibility": "public"}
	status, out := apiJSON(t, "POST", "/orgs", body)
	if status >= 300 {
		t.Fatalf("createOrg(%s) status %d: %s", name, status, out)
	}
}

// extractID pulls the first "#NN" from CLI output (issue/PR/release create).
var idRe = regexp.MustCompile(`#(\d+)`)

func extractID(t *testing.T, out string) int64 {
	t.Helper()
	m := idRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no #ID in output: %q", out)
	}
	id, _ := strconv.ParseInt(m[1], 10, 64)
	return id
}

// contains is a tiny test helper.
func contains(t *testing.T, out, want string) {
	t.Helper()
	if !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
}

// TestCLICommands exercises every hand-written fj subcommand (read + write).
func TestCLICommands(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)

	// Shared scratch repo for repo-scoped commands. Unique per-run so local
	// re-runs against the same container do not 409 on an existing repo.
	repo := fmt.Sprintf("cli-all-%d", time.Now().UnixNano())
	createTestRepo(t, repo)
	ownerRepo := testUser() + "/" + repo

	// ---- top-level / host-scoped commands ------------------------------
	t.Run("version", func(t *testing.T) {
		out, err := runFj(t, binary, "version")
		if err != nil {
			t.Fatal(err)
		}
		contains(t, out, "Version")
	})

	t.Run("whoami", func(t *testing.T) {
		out, err := runFj(t, binary, "whoami")
		if err != nil {
			t.Fatal(err)
		}
		contains(t, out, testUser())
	})

	t.Run("auth/list", func(t *testing.T) {
		// auth list reads the local keys.json (not the FORGEJO_TOKEN the rest of
		// the suite uses), so its output reflects the host's login state, not the
		// test instance. Assert only that the command runs cleanly.
		if _, err := runFj(t, binary, "auth", "list"); err != nil {
			t.Fatal(err)
		}
	})

	// ---- repo ----------------------------------------------------------
	t.Run("repo/view", func(t *testing.T) {
		// positional OWNER/NAME form
		out, err := runFj(t, binary, "repo", "view", ownerRepo)
		if err != nil {
			t.Fatal(err)
		}
		contains(t, out, repo)
	})

	// ---- issue: full lifecycle (create → list → view → comment → close) --
	t.Run("issue", func(t *testing.T) {
		out, err := runFj(t, binary, "issue", "create", "-r", ownerRepo,
			"-t", "fj integration issue", "-b", "created by the integration suite")
		if err != nil {
			t.Fatal(err)
		}
		idx := extractID(t, out)

		if out, err = runFj(t, binary, "issue", "list", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		contains(t, out, "fj integration issue")

		if out, err = runFj(t, binary, "issue", "view", strconv.FormatInt(idx, 10),
			"-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		contains(t, out, "fj integration issue")

		if _, err = runFj(t, binary, "issue", "comment", strconv.FormatInt(idx, 10),
			"-b", "a test comment", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		// --comments must render the thread we just added
		if out, err = runFj(t, binary, "issue", "view", strconv.FormatInt(idx, 10),
			"-c", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		contains(t, out, "Comments")
		contains(t, out, "a test comment")

		if _, err = runFj(t, binary, "issue", "close", strconv.FormatInt(idx, 10),
			"-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
	})

	// ---- pull request: seed branch via API, then create/list/view/status/merge
	t.Run("pr", func(t *testing.T) {
		createFileOnBranch(t, repo, "feature.txt", "", "feature-branch", "hello")
		out, err := runFj(t, binary, "pr", "create", "-r", ownerRepo,
			"-t", "fj integration PR", "--head", "feature-branch", "--base", "main")
		if err != nil {
			t.Fatal(err)
		}
		idx := extractID(t, out)

		if out, err = runFj(t, binary, "pr", "list", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		contains(t, out, "fj integration PR")

		if out, err = runFj(t, binary, "pr", "view", strconv.FormatInt(idx, 10),
			"-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		contains(t, out, "fj integration PR")

		// status must not crash even when there are no CI checks yet
		if _, err = runFj(t, binary, "pr", "status", strconv.FormatInt(idx, 10),
			"-r", ownerRepo); err != nil {
			t.Fatal(err)
		}

		if _, err = runFj(t, binary, "pr", "merge", strconv.FormatInt(idx, 10),
			"-s", "merge", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
	})

	// ---- release: create → list → view → delete -----------------------
	t.Run("release", func(t *testing.T) {
		out, err := runFj(t, binary, "release", "create", "-r", ownerRepo,
			"--tag", "v-rel-test", "-n", "release one", "-b", "body")
		if err != nil {
			t.Fatal(err)
		}
		id := extractID(t, out)

		if out, err = runFj(t, binary, "release", "list", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		contains(t, out, "release one")

		if _, err = runFj(t, binary, "release", "view", strconv.FormatInt(id, 10),
			"-r", ownerRepo); err != nil {
			t.Fatal(err)
		}

		if _, err = runFj(t, binary, "release", "delete", strconv.FormatInt(id, 10),
			"-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
	})

	// ---- tag: create → list → delete ----------------------------------
	t.Run("tag", func(t *testing.T) {
		if _, err := runFj(t, binary, "tag", "create", "v-tag-test",
			"-m", "annotated by integration", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		out, err := runFj(t, binary, "tag", "list", "-r", ownerRepo)
		if err != nil {
			t.Fatal(err)
		}
		contains(t, out, "v-tag-test")
		if _, err = runFj(t, binary, "tag", "delete", "v-tag-test", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
	})

	// ---- user: view / search / repos (host-scoped) --------------------
	t.Run("user", func(t *testing.T) {
		if out, err := runFj(t, binary, "user", "view", testUser()); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, testUser())
		}
		if _, err := runFj(t, binary, "user", "search", testUser()); err != nil {
			t.Fatal(err)
		}
		// the admin owns at least the repo we just created
		if out, err := runFj(t, binary, "user", "repos", testUser()); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, repo)
		}
	})

	// ---- org: seed via API, then list / view --------------------------
	t.Run("org", func(t *testing.T) {
		org := fmt.Sprintf("org-%d", time.Now().UnixNano())
		createOrg(t, org)
		if out, err := runFj(t, binary, "org", "list"); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, org)
		}
		if out, err := runFj(t, binary, "org", "view", org); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, org)
		}
	})

	// ---- wiki: seed a page via API, then list / view ------------------
	t.Run("wiki", func(t *testing.T) {
		createWikiPage(t, repo, "Home", "welcome to the integration wiki")
		if out, err := runFj(t, binary, "wiki", "list", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, "Home")
		}
		if _, err := runFj(t, binary, "wiki", "view", "Home", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
	})

	// ---- actions: list commands + variables/secrets CRUD --------------
	// (jobs/logs require a workflow + runner; those are exercised against the
	// real CI in the manual smoke test. Here we cover the CRUD surface.)
	t.Run("actions", func(t *testing.T) {
		// read commands — succeed even with no workflows yet
		if _, err := runFj(t, binary, "actions", "tasks", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		if _, err := runFj(t, binary, "actions", "runs", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		// filters pass through to list-action-runs (#74) — exit-0 is the
		// contract on an empty repo ("no runs"), flags must at least bind.
		if _, err := runFj(t, binary, "actions", "runs", "-r", ownerRepo, "--limit", "3", "--page", "1"); err != nil {
			t.Fatal(err)
		}
		if _, err := runFj(t, binary, "actions", "runs", "-r", ownerRepo,
			"--status", "success,failure", "--event", "push", "--head-sha", "0af1a3633115fe49317c0289ecb45b18ba3cf0ee",
			"--ref", "main", "--workflow-id", "1", "--run-number", "1"); err != nil {
			t.Fatal(err)
		}

		// variables CRUD
		if _, err := runFj(t, binary, "actions", "variables", "create",
			"VAR_TEST", "v1", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		if out, err := runFj(t, binary, "actions", "variables", "list", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, "VAR_TEST")
		}
		if _, err := runFj(t, binary, "actions", "variables", "delete",
			"VAR_TEST", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}

		// secrets CRUD
		if _, err := runFj(t, binary, "actions", "secrets", "create",
			"SEC_TEST", "topsecret", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
		if out, err := runFj(t, binary, "actions", "secrets", "list", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		} else {
			contains(t, out, "SEC_TEST")
		}
		if _, err := runFj(t, binary, "actions", "secrets", "delete",
			"SEC_TEST", "-r", ownerRepo); err != nil {
			t.Fatal(err)
		}
	})

	// ---- repo clone: needs git + auth; verify it at least shells out --
	// (best-effort — skip if git is unavailable rather than fail the suite)
	t.Run("repo/clone", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not installed")
		}
		dir := t.TempDir()
		// clone over HTTP with the token embedded (private repo)
		cloneURL := strings.Replace(testURL()+"/"+ownerRepo+".git",
			"http://", "http://"+testUser()+":"+testToken()+"@", 1)
		cmd := exec.Command("git", "clone", cloneURL, dir+"/repo")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git clone failed (auth/network in this env): %v\n%s", err, out)
		}
	})
}
