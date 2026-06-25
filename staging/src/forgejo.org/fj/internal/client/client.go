// Package client provides helper functions to create a forgejo SDK client
// from the auth store (keys.json).
package client

import (
	"fmt"
	"net/http"

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
	return api.NewClient("https://"+host, httpClient)
}

// FromToken creates a forgejo SDK client with a plain token (no keys.json).
func FromToken(host, token string) (*api.Client, error) {
	httpClient := &http.Client{
		Transport: &TokenTransport{Token: token},
	}
	return api.NewClient("https://"+host, httpClient)
}
