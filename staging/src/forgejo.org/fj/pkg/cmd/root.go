package cmd

import (
	"github.com/spf13/cobra"
)

// global flags
var (
	flagHost   string
	flagRepo   string
	flagRemote string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fj",
		Short: "Forgejo CLI",
		Long:  "fj is a command-line client for Forgejo (https://forgejo.org).",
	}

	root.PersistentFlags().StringVarP(&flagHost, "host", "H", "", "the forgejo instance to use")
	root.PersistentFlags().StringVarP(&flagRepo, "repo", "r", "", "repo to operate on (owner/name)")
	root.PersistentFlags().StringVarP(&flagRemote, "remote", "R", "", "git remote to use")

	root.AddCommand(newActionsCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newRepoCmd())
	root.AddCommand(newIssueCmd())
	// Descriptor-driven polished groups (gen/polish.json → zz_generated_polished.go).
	// Wired once; adding a group is a descriptor edit + regen — no root change.
	root.AddCommand(NewPolishedCmds()...)
	root.AddCommand(newPrCmd())
	root.AddCommand(newReleaseCmd())
	root.AddCommand(newTagCmd())
	root.AddCommand(newUserCmd())
	root.AddCommand(newOrgCmd())
	root.AddCommand(newWikiCmd())
	root.AddCommand(newWhoAmICmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(NewGeneratedAPICmd()) // auto-generated from swagger spec

	return root
}
