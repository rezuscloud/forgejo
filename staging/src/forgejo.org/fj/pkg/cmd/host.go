package cmd

import (
	"fmt"
	"os"

	forgejo "forgejo.org/client-go"
	"forgejo.org/fj/internal/client"
	"forgejo.org/fj/internal/gitremote"
	"forgejo.org/fj/internal/keys"
	"github.com/spf13/cobra"
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
	ki, _ := keys.Load()
	if ki != nil {
		if login := ki.GetLogin(host); login != nil {
			return client.FromKeys(ki, host)
		}
	}
	token := os.Getenv("FORGEJO_TOKEN")
	if token != "" {
		return client.FromToken(host, token)
	}
	return nil, fmt.Errorf("not logged in to %s", host)
}
