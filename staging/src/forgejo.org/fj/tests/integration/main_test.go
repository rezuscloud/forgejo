//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"
)

// TestMain orchestrates the test instance lifecycle for the whole package.
//
// Two modes, chosen automatically:
//
//  1. External instance (the k8s-config CronJob path): if FORGEJO_TEST_URL is
//     already set, TestMain is a pass-through — existing behavior is unchanged.
//
//  2. Containerized (local dev + the cli-parity workflow): otherwise, start a
//     Forgejo container, bootstrap an admin user + token, export the standard
//     env vars, run the tests, and tear the container down on exit.
//
// Build-tagged `//go:build integration` so that without -tags=integration the
// package compiles WITHOUT a TestMain (upstream's `go test ./...` and the
// existing no-tag tests behave exactly as before). With the tag, this is the
// sole TestMain for the package.
//
// Conflict-free: this file is additive and lives under staging/, which
// upstream's Makefile (GO_DIRS) excludes from its test scope.
func TestMain(m *testing.M) {
	code := runWithInstance(m)
	os.Exit(code)
}

func runWithInstance(m *testing.M) int {
	// Mode 1: external instance provided.
	if os.Getenv("FORGEJO_TEST_URL") != "" {
		fmt.Fprintf(os.Stderr, "[integration] using external Forgejo instance: %s\n", os.Getenv("FORGEJO_TEST_URL"))
		return m.Run()
	}

	// Mode 2: start our own container. For an explicit integration run,
	// container setup failures must be LOUD, not silently skipped, otherwise CI
	// can pass while every live-server test is being skipped.
	if !dockerAvailable() {
		fmt.Fprintln(os.Stderr, "[integration] docker not available and FORGEJO_TEST_URL not set")
		return 1
	}

	fmt.Fprintf(os.Stderr, "[integration] starting Forgejo test container: %s\n", forgejoImage())
	id, hostPort, err := startForgejoContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[integration] start container failed: %v\n", err)
		return 1
	}
	containerID = id
	containerHost = "http://localhost:" + hostPort
	defer stopContainer(containerID)
	fmt.Fprintf(os.Stderr, "[integration] container %s mapped to %s\n", id[:min(12, len(id))], containerHost)

	if err := waitForForgejo(containerHost); err != nil {
		fmt.Fprintf(os.Stderr, "[integration] wait for Forgejo failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "[integration] Forgejo API is up")

	// Create the admin user via docker exec. Avoid testing.T here: we want
	// explicit stderr on failure, not a skipped suite.
	if err := bootstrapAdminUserNoTest(id); err != nil {
		fmt.Fprintf(os.Stderr, "[integration] bootstrap admin failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "[integration] admin user ready")

	token, err := createAdminToken(containerHost)
	if err != nil || token == "" {
		fmt.Fprintf(os.Stderr, "[integration] create admin token failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "[integration] admin token ready")

	// Export for the existing helpers (testURL/testToken/testUser).
	os.Setenv("FORGEJO_TEST_URL", containerHost)
	os.Setenv("FORGEJO_TEST_TOKEN", token)
	os.Setenv("FORGEJO_TEST_USER", adminUser)

	return m.Run()
}
