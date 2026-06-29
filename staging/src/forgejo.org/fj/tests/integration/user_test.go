//go:build integration

package integration

import (
	"strings"
	"testing"
)

// User key UX: add -> list. Delete isn't implemented yet in the hand-written
// UX layer, so this verifies the implemented parity surface.
func TestUser_KeyAddList(t *testing.T) {
	skipIfNoInstance(t)
	binary := buildFjBinary(t)
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJ7C0xP9y9uB2J4O0vZxK6l3v1Qq4z5m8n7p6s5r4t3u integration@test"

	out, err := runFj(t, binary, "user", "key", "add", key, "--title", "integration-key")
	if err != nil {
		t.Fatalf("user key add: %v\n%s", err, out)
	}
	if !strings.Contains(out, "integration-key") {
		t.Fatalf("user key add output missing title: %q", out)
	}

	out, err = runFj(t, binary, "user", "key", "list")
	if err != nil {
		t.Fatalf("user key list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "integration-key") {
		t.Fatalf("user key list missing added key: %q", out)
	}
}
