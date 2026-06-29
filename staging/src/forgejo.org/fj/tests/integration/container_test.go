//go:build integration

// Package integration — container lifecycle helpers.
//
// These helpers start a Forgejo container when no live instance is provided
// via FORGEJO_TEST_URL, bootstrap an admin user + token, and tear the container
// down on exit. Everything here is additive: upstream Forgejo's own
// tests/integration/ suite is never touched, and this file lives under the
// staging module which upstream's Makefile (GO_DIRS) excludes from its scope.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Container lifecycle state, set by startForgejoContainer and cleaned up by
// stopForgejoContainer. Kept in package-level vars so TestMain can orchestrate
// them and the deferred teardown can run after m.Run().
var (
	containerID   string
	containerHost string // host:port for FORGEJO_TEST_URL
)

// forgejoImage returns the container image to start. Override with
// FORGEJO_TEST_IMAGE.
//
// Default: `codeberg.org/forgejo/forgejo:15`, which is a published and tested
// upstream image tag. The repo's custom 16.x distribution image can be used by
// setting FORGEJO_TEST_IMAGE explicitly in CI or locally.
func forgejoImage() string {
	if img := os.Getenv("FORGEJO_TEST_IMAGE"); img != "" {
		return img
	}
	return "codeberg.org/forgejo/forgejo:15"
}

// adminCreds are the credentials bootstrapAdminUser provisions inside the
// container. Fixed values are fine — the container is ephemeral and isolated.
const (
	adminUser    = "root"
	adminPass    = "integration123!"
	adminEmail   = "root@example.com"
	tokenName    = "fj-integration-test"
)

// dockerAvailable reports whether the docker CLI is usable on this host.
func dockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// freeHostPort asks docker to map container port 3000 to an ephemeral host
// port (-P style). We read the mapped port back via `docker port`.
func startForgejoContainer() (id, hostPort string, err error) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"--rm",
		"-p", "3000", // ephemeral host port → container 3000
		"-e", "FORGEJO__security__INSTALL_LOCK=true",
		"-e", "FORGEJO__server__LOCAL_ROOT_URL=http://localhost:3000/",
		forgejoImage(),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("docker run: %w\n%s", err, stderr.String())
	}
	id = strings.TrimSpace(string(out))

	// Resolve the mapped host port.
	portOut, err := exec.CommandContext(ctx, "docker", "port", id, "3000").Output()
	if err != nil {
		stopContainer(id)
		return "", "", fmt.Errorf("docker port: %w", err)
	}
	// Output looks like "0.0.0.0:32768\n" or "[::]:32768\n".
	line := strings.TrimSpace(string(portOut))
	if i := strings.LastIndex(line, ":"); i >= 0 {
		hostPort = line[i+1:]
	}
	if hostPort == "" {
		stopContainer(id)
		return "", "", fmt.Errorf("could not parse mapped port from %q", line)
	}
	return id, hostPort, nil
}

// stopContainer is best-effort; errors are swallowed (container is --rm).
func stopContainer(id string) {
	if id == "" {
		return
	}
	_ = exec.Command("docker", "rm", "-f", id).Run()
}

// waitForForgejo polls /api/v1/version until it returns 200 or the timeout
// elapses. Forgejo can take ~30-60s to be ready on a cold container.
func waitForForgejo(baseURL string) error {
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/v1/version")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("forgejo not ready at %s within 120s", baseURL)
}

// bootstrapAdminUser creates the admin user inside the container via the
// forgejo CLI (docker exec). Forgejo v16 ships a `forgejo` binary; older
// images expose `gitea` — we try both.
func bootstrapAdminUser(t *testing.T, id string) {
	t.Helper()
	if err := bootstrapAdminUserNoTest(id); err != nil {
		t.Fatal(err)
	}
}

// bootstrapAdminUserNoTest is the TestMain-friendly variant that returns an
// error instead of depending on *testing.T.
func bootstrapAdminUserNoTest(id string) error {
	config := "/data/gitea/conf/app.ini"
	workPath := "/data/gitea"
	var attempts []string
	for _, bin := range []string{"/usr/local/bin/forgejo", "/usr/local/bin/gitea", "forgejo", "gitea"} {
		cmd := exec.Command("docker", "exec", "--user", "git", id,
			bin,
			"--config", config,
			"-w", workPath,
			"admin", "user", "create",
			"--admin",
			"--username", adminUser,
			"--password", adminPass,
			"--email", adminEmail,
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if runErr := cmd.Run(); runErr == nil {
			return nil // created
		} else if strings.Contains(strings.ToLower(stderr.String()), "already exist") {
			return nil // user already exists (container reuse)
		} else {
			attempts = append(attempts, fmt.Sprintf("%s: %v: %s", bin, runErr, strings.TrimSpace(stderr.String())))
		}
	}
	return fmt.Errorf("could not create admin user in container %s (%s)", id, strings.Join(attempts, " | "))
}

// createAdminToken provisions an admin-scoped API token for the bootstrap user
// via the API using HTTP basic auth. Returns the token string.
func createAdminToken(baseURL string) (string, error) {
	body := map[string]interface{}{
		"name":   tokenName,
		"scopes": []string{"all"},
	}
	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", baseURL+"/api/v1/users/"+adminUser+"/tokens", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(adminUser, adminPass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("create token: status %d: %s", resp.StatusCode, string(respBytes))
	}
	var result struct {
		Sha1 string `json:"sha1"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", err
	}
	if result.Sha1 != "" {
		return result.Sha1, nil
	}
	return result.Token, nil
}
