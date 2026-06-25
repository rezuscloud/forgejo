// Package keys manages the persistent authentication store for fj.
// It mirrors the Rust CLI's keys.json format (ADR-0004: token-only SDK,
// CLI owns auth).
package keys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// KeyInfo is the top-level auth store, persisted as keys.json.
type KeyInfo struct {
	Hosts    map[string]*LoginInfo `json:"hosts"`
	Aliases  map[string]string     `json:"aliases,omitempty"`
}

// LoginInfo holds credentials for one host.
// Application = a personal access token.
// OAuth = an OAuth2 token with refresh (from fj auth login browser flow).
type LoginInfo struct {
	Type         string `json:"type"`          // "Application" or "OAuth"
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"` // OAuth only
	// ExpiresAt is the OAuth token expiry. The Rust CLI writes this as a unix
	// timestamp integer OR the sentinel string "never" (for Application tokens
	// that do not expire). We accept either to stay compatible with the Rust
	// keys.json format.
	ExpiresAt ExpiresAt `json:"expires_at,omitempty"` // OAuth only
}

// ExpiresAt tolerates either an integer unix timestamp or the Rust CLI's
// "never" sentinel string (written for non-expiring Application tokens).
type ExpiresAt int64

// UnmarshalJSON accepts numbers and strings. "never" (case-insensitive) maps
// to 0 (no expiry). Numeric strings are parsed as a unix timestamp.
func (e *ExpiresAt) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		return nil
	}
	if strings.EqualFold(s, `"never"`) || s == `"never"` {
		*e = 0
		return nil
	}
	// try a plain integer
	var i int64
	if err := json.Unmarshal(data, &i); err == nil {
		*e = ExpiresAt(i)
		return nil
	}
	// fall back to a numeric string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if n, err := strconv.ParseInt(str, 10, 64); err == nil {
			*e = ExpiresAt(n)
			return nil
		}
		if strings.EqualFold(str, "never") {
			*e = 0
			return nil
		}
	}
	// unknown shape — treat as no expiry rather than failing the whole file
	*e = 0
	return nil
}

// keysPath returns the path to keys.json, mirroring the Rust CLI:
// ~/.local/share/forgejo-cli/keys.json (Linux XDG)
func keysPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// XDG_DATA_HOME override
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "forgejo-cli", "keys.json"), nil
	}
	return filepath.Join(home, ".local", "share", "forgejo-cli", "keys.json"), nil
}

// Load reads keys.json, returning an empty KeyInfo if it doesn't exist.
func Load() (*KeyInfo, error) {
	path, err := keysPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &KeyInfo{Hosts: map[string]*LoginInfo{}}, nil
		}
		return nil, fmt.Errorf("reading keys.json: %w", err)
	}
	var ki KeyInfo
	if err := json.Unmarshal(data, &ki); err != nil {
		return nil, fmt.Errorf("parsing keys.json: %w", err)
	}
	if ki.Hosts == nil {
		ki.Hosts = map[string]*LoginInfo{}
	}
	return &ki, nil
}

// Save writes keys.json atomically.
func (ki *KeyInfo) Save() error {
	path, err := keysPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating keys dir: %w", err)
	}
	data, err := json.MarshalIndent(ki, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling keys.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing keys.json: %w", err)
	}
	return nil
}

// GetLogin returns the LoginInfo for a host, or nil if not found.
func (ki *KeyInfo) GetLogin(host string) *LoginInfo {
	if ki == nil || ki.Hosts == nil {
		return nil
	}
	return ki.Hosts[host]
}

// SetLogin sets the LoginInfo for a host.
func (ki *KeyInfo) SetLogin(host string, li *LoginInfo) {
	ki.Hosts[host] = li
}

// RemoveLogin deletes a host from the store.
func (ki *KeyInfo) RemoveLogin(host string) bool {
	if _, ok := ki.Hosts[host]; ok {
		delete(ki.Hosts, host)
		return true
	}
	return false
}

// ListHosts returns the sorted list of authenticated hosts.
func (ki *KeyInfo) ListHosts() []string {
	hosts := make([]string, 0, len(ki.Hosts))
	for h := range ki.Hosts {
		hosts = append(hosts, h)
	}
	return hosts
}
