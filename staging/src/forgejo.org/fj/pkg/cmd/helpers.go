package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	forgejo "forgejo.org/client-go"
	"forgejo.org/fj/internal/client"
	"forgejo.org/fj/internal/gitremote"
	"forgejo.org/fj/internal/keys"
	"github.com/spf13/cobra"
)

// resolveClient resolves the SDK client + owner/repo from:
// 1. --host/--repo flags
// 2. git remote in the current directory
// 3. FORGEJO_TOKEN env fallback
func resolveClient(cmd *cobra.Command) (*forgejo.Client, string, string, error) {
	host, _ := cmd.Flags().GetString("host")
	repoFlag, _ := cmd.Flags().GetString("repo")
	remoteFlag, _ := cmd.Flags().GetString("remote")

	if repoFlag == "" || host == "" {
		if r, e := gitremote.Discover(remoteFlag); e == nil {
			if host == "" {
				host = r.Host
			}
			if repoFlag == "" {
				repoFlag = r.Owner + "/" + r.Repo
			}
		}
	}

	if host == "" {
		return nil, "", "", fmt.Errorf("could not determine host (use --host, or run from a git repo)")
	}
	if repoFlag == "" {
		return nil, "", "", fmt.Errorf("could not determine repo (use --repo owner/name)")
	}

	parts := strings.SplitN(repoFlag, "/", 2)
	if len(parts) != 2 {
		return nil, "", "", fmt.Errorf("invalid repo %q (expected owner/name)", repoFlag)
	}
	owner, repo := parts[0], parts[1]

	ki, _ := keys.Load()
	if ki != nil {
		if login := ki.GetLogin(host); login != nil {
			c, err := client.FromKeys(ki, host)
			if err == nil {
				return c, owner, repo, nil
			}
		}
	}

	token := os.Getenv("FORGEJO_TOKEN")
	if token != "" {
		c, err := client.FromToken(host, token)
		if err != nil {
			return nil, "", "", err
		}
		return c, owner, repo, nil
	}

	return nil, "", "", fmt.Errorf("not logged in to %s (run: fj --host %s auth login)", host, host)
}

func valI64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// stateStr safely converts a *StateType to string.
func stateStr(s *forgejo.StateType) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// timeStr formats an optional timestamp for display ("" if absent/zero).
// Accepts the generated SDK's optional *time.Time fields directly.
func timeStr(t any) string {
	switch v := t.(type) {
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Format(time.RFC3339)
	case *time.Time:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.Format(time.RFC3339)
	}
	return ""
}

// commitStateStr safely stringifies a *CommitStatusState, treating nil or
// empty-string as "unknown". Forgejo can return an empty status for checks
// attached to a superseded SHA (e.g. after a force-push); this must never
// crash or be passed raw to the renderer.
func commitStateStr(s *forgejo.CommitStatusState) string {
	if s == nil || string(*s) == "" {
		return "unknown"
	}
	return string(*s)
}
