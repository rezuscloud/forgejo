package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newWikiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wiki",
		Short: "Manage wiki pages",
	}
	cmd.AddCommand(newWikiListCmd())
	cmd.AddCommand(newWikiViewCmd())
	return cmd
}

func newWikiListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List wiki pages on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			pages, _, err := c.Repo.RepoGetWikiPages(context.Background(), owner, repo, 1, 50)
			if err != nil {
				return err
			}
			if len(pages) == 0 {
				fmt.Println("no wiki pages")
				return nil
			}
			for _, p := range pages {
				fmt.Println(p.Title)
			}
			return nil
		},
	}
}

func newWikiViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <PAGE>",
		Short: "View a wiki page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			page, _, err := c.Repo.RepoGetWikiPage(context.Background(), owner, repo, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("# %s\n", page.Title)
			fmt.Printf("Commits: %d\n\n", page.CommitCount)
			if page.ContentBase64 != "" {
				// decode base64 content and print
				fmt.Println("(wiki content is base64-encoded in the API)")
			}
			if page.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", page.HtmlUrl)
			}
			return nil
		},
	}
}
