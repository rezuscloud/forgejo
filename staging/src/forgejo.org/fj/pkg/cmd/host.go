package cmd

import (
	"fmt"
	"os"

	forgejo "forgejo.org/client-go"
	"forgejo.org/fj/internal/client"
	"forgejo.org/fj/internal/gitremote"
	"forgejo.org/fj/internal/keys"
	"github.com/spf13/cobra"
"strings"
)

func resolveHostClient(cmd *cobra.Command, host string) (*forgejo.Client, error) {
	// Read --host from the cobra flags if not explicitly provided
	if host == "" {
		if hf := cmd.Flag("host"); hf != nil {
			host = hf.Value.String()
		}
	}
	if host == "" {
		if r, e := gitremote.Discover(""); e == nil {
			host = r.Host
		}
	}
	if host == "" {
		return nil, fmt.Errorf("--host is required")
	}
	// Normalize: if the host already has a scheme, use it as-is. Otherwise
	// default to https:// (the production case). This allows integration tests
	// to pass http://localhost:3000.
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	ki, _ := keys.Load()
	if ki != nil {
		// Try matching by hostname (strip scheme/port for keys lookup)
		lookupHost := host
		lookupHost = strings.TrimPrefix(strings.TrimPrefix(lookupHost, "https://"), "http://")
		if login := ki.GetLogin(lookupHost); login != nil {
			return client.FromKeysURL(host, ki, lookupHost)
		}
	}
	token := os.Getenv("FORGEJO_TOKEN")
	if token != "" {
		return client.FromTokenURL(host, token)
	}
	return nil, fmt.Errorf("not logged in to %s", host)
}
