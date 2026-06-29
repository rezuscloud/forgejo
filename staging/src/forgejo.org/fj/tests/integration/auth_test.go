//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Auth UX tests use an isolated XDG_DATA_HOME so they don't touch the real
// host auth store. This covers the implemented non-interactive commands:
// add-key -> list -> logout.
func TestAuth_AddListLogout(t *testing.T) {
	binary := buildFjBinary(t)
	xdg := tempXDG(t)
	env := map[string]string{"XDG_DATA_HOME": xdg}
	host := testURL()

	out, err := runFjWithOptions(t, binary, "", env, "", "auth", "add-key", "dummy-token")
	if err != nil {
		t.Fatalf("auth add-key: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Added token") {
		t.Fatalf("unexpected auth add-key output: %q", out)
	}

	out, err = runFjWithOptions(t, binary, "", env, "", "auth", "list")
	if err != nil {
		t.Fatalf("auth list: %v\n%s", err, out)
	}
	if !strings.Contains(out, host) {
		t.Fatalf("auth list missing host %q: %q", host, out)
	}

	out, err = runFjWithOptions(t, binary, "", env, "", "auth", "logout", host)
	if err != nil {
		t.Fatalf("auth logout: %v\n%s", err, out)
	}
	out, err = runFjWithOptions(t, binary, "", env, "", "auth", "list")
	if err != nil {
		t.Fatalf("auth list after logout: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No authenticated hosts") {
		t.Fatalf("expected empty auth store after logout, got: %q", out)
	}
}
