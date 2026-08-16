// Package client provides helper functions to create a forgejo SDK client
// from the auth store (keys.json).
package client

import (
	"fmt"
	"net/http"
	"strings"

	api "forgejo.org/client-go"
	"forgejo.org/fj/internal/keys"
)

// TokenTransport injects the Forgejo token into every request.
type TokenTransport struct {
	Token string
	Base  http.RoundTripper
}

func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "token "+t.Token)
	if t.Base == nil {
		t.Base = http.DefaultTransport
	}
	return t.Base.RoundTrip(req)
}

// FromKeys creates a forgejo SDK client authenticated for the given host
// using credentials from keys.json. Returns an error if the host is not
// authenticated.
func FromKeys(ki *keys.KeyInfo, host string) (*api.Client, error) {
	login := ki.GetLogin(host)
	if login == nil {
		return nil, fmt.Errorf("not logged in to %s (run: fj auth login)", host)
	}
	httpClient := &http.Client{
		Transport: &TokenTransport{Token: login.Token},
	}
	// normalizeURL: hosts may carry a scheme (http://localhost:3000 in
	// integration tests) — a hardcoded https:// prefix mangles them.
	return api.NewClient(normalizeURL(host), httpClient)
}

// FromToken creates a forgejo SDK client with a plain token (no keys.json).
func FromToken(host, token string) (*api.Client, error) {
	return FromTokenURL(normalizeURL(host), token)
}

// FromTokenURL creates a client with a full server URL (may include http://).
func FromTokenURL(serverURL, token string) (*api.Client, error) {
	httpClient := &http.Client{
		Transport: &TokenTransport{Token: token},
	}
	return api.NewClient(serverURL, httpClient)
}

// FromKeysURL is like FromKeys but takes a full server URL (for integration tests).
func FromKeysURL(serverURL string, ki *keys.KeyInfo, host string) (*api.Client, error) {
	login := ki.GetLogin(host)
	if login == nil {
		return nil, fmt.Errorf("not logged in to %s (run: fj auth login)", host)
	}
	httpClient := &http.Client{
		Transport: &TokenTransport{Token: login.Token},
	}
	return api.NewClient(serverURL, httpClient)
}

// normalizeURL prepends https:// if no scheme is present.
func normalizeURL(host string) string {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		return "https://" + host
	}
	return host
}
