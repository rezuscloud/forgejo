package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

// cliVersion is the fj binary version (overridden by -ldflags at release time).
var cliVersion = "0.1.0-dev"

func newVersionCmd() *cobra.Command {
	var clientOnly bool
	var short bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the fj, client API, and server versions",
		Long: `Print version information, mirroring ` + "`kubectl version`" + `:

  Client Version:  the fj binary version
  API Version:     the Forgejo REST API level this CLI's SDK was generated from
                   (the swagger spec the SDK maps — like kubectl's client API version)
  Server Version:  the live Forgejo server version (queried from GET /version)

Use --client to skip the server query (e.g. when not authenticated).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if short {
				fmt.Printf("fj %s (api %s)\n", cliVersion, forgejo.SpecAPIVersion)
				return nil
			}

			w := os.Stdout
			fmt.Fprintf(w, "Client Version: fj %s\n", cliVersion)
			fmt.Fprintf(w, "API Version:    %s\n", forgejo.SpecAPIVersion)
			fmt.Fprintf(w, "               (SDK generated from %s %s)\n", forgejo.SpecTitle, forgejo.SpecVersion)

			if clientOnly {
				return nil
			}

			srvVersion, host, err := serverVersion(cmd)
			if err != nil {
				fmt.Fprintf(w, "Server Version: <unavailable: %v>\n", err)
				return nil
			}
			label := "Server Version:"
			if host != "" {
				label = fmt.Sprintf("Server Version (%s):", host)
			}
			fmt.Fprintf(w, "%s %s\n", label, srvVersion)

			if warn := apiCompatNote(forgejo.SpecAPIVersion, srvVersion); warn != "" {
				fmt.Fprintln(w, warn)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&clientOnly, "client", false, "client version only (skip the server query)")
	cmd.Flags().BoolVar(&short, "short", false, "print a single compact line")
	return cmd
}

// serverVersion resolves a client from flags/keys/remote and queries the live
// Forgejo /version endpoint. It never hard-fails — the CLI still reports client
// + API versions when no host/auth is configured.
func serverVersion(cmd *cobra.Command) (version, host string, err error) {
	host = cmd.Flag("host").Value.String()
	c, e := resolveHostClient(cmd, host)
	if e != nil {
		return "", host, e
	}
	sv, _, e := c.Misc.GetVersion(context.Background())
	if e != nil {
		return "", host, e
	}
	return sv.Version, host, nil
}

// apiCompatNote returns a one-line warning if the server's API level differs
// from the level the SDK was generated from, analogous to kubectl's
// "server is X versions ahead" notice. It parses the gitea API suffix from the
// server version; absent the suffix it falls back to a best-effort prefix.
func apiCompatNote(clientAPI, serverVersion string) string {
	srvAPI := forgejo.GiteaAPIVersion(serverVersion)
	if srvAPI == "" || srvAPI == clientAPI {
		return ""
	}
	switch strings.Compare(srvAPI, clientAPI) {
	case 1:
		return fmt.Sprintf("# note: server API (%s) is newer than client API (%s) — consider upgrading fj.", srvAPI, clientAPI)
	case -1:
		return fmt.Sprintf("# note: client API (%s) is newer than server API (%s) — the server may not support some endpoints.", clientAPI, srvAPI)
	}
	return ""
}
