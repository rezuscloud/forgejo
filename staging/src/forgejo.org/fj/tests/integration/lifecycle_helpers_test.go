//go:build integration

package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runFjWithOptions is the lifecycle-test helper for commands that need an
// isolated HOME/XDG store (auth tests), a specific working directory (repo
// clone), or stdin input. It preserves the standard --host injection and
// FORGEJO_TOKEN env used by runFj.
func runFjWithOptions(t *testing.T, binaryPath string, dir string, extraEnv map[string]string, stdin string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"--host", testURL()}, args...)
	cmd := exec.Command(binaryPath, fullArgs...)
	if dir != "" {
		cmd.Dir = dir
	}
	env := append(os.Environ(), "FORGEJO_TOKEN="+testToken())
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Logf("fj %v\nstdout: %s\nstderr: %s", args, stdout.String(), stderr.String())
		combined := stdout.String()
		if stderr.Len() > 0 {
			if combined != "" {
				combined += "\n"
			}
			combined += stderr.String()
		}
		return combined, err
	}
	return stdout.String(), nil
}

func createTestOrg(t *testing.T, name string) string {
	t.Helper()
	body := map[string]interface{}{
		"username":  name,
		"full_name": "Integration Test Org",
	}
	bodyBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", testURL()+"/api/v1/orgs", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "token "+testToken())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode >= 300 {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("create org %s: %v (status %d)", name, err, status)
	}
	return name
}

func createPublicTestRepo(t *testing.T, name string) string {
	t.Helper()
	body := map[string]interface{}{
		"name":        name,
		"description": "integration test repo",
		"private":     false,
		"auto_init":   true,
	}
	bodyBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", testURL()+"/api/v1/user/repos", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "token "+testToken())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode >= 300 {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("create public repo %s: %v (status %d)", name, err, status)
	}
	return name
}

func repoExists(t *testing.T, owner, repo string) bool {
	t.Helper()
	status, err := apiGet("/repos/" + owner + "/" + repo)
	if err != nil {
		t.Fatalf("repo exists check: %v", err)
	}
	return status == 200
}

func tagExists(t *testing.T, owner, repo, tag string) bool {
	t.Helper()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/repos/%s/%s/tags", testURL(), owner, repo), nil)
	req.Header.Set("Authorization", "token "+testToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("tag exists check: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return strings.Contains(string(b), `"name":"`+tag+`"`)
}

func releaseExists(t *testing.T, owner, repo, tag string) bool {
	t.Helper()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/repos/%s/%s/releases/tags/%s", testURL(), owner, repo, tag), nil)
	req.Header.Set("Authorization", "token "+testToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("release exists check: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func createWikiPage(t *testing.T, owner, repo, title, content string) {
	t.Helper()
	body := fmt.Sprintf(`{"title":"%s","content_base64":"%s"}`,
		title, base64.StdEncoding.EncodeToString([]byte(content)))
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/new", testURL(), owner, repo), strings.NewReader(body))
	req.Header.Set("Authorization", "token "+testToken())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode >= 300 {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("create wiki page: %v (status %d)", err, status)
	}
}
func tempXDG(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	return filepath.Join(d, "xdg")
}
