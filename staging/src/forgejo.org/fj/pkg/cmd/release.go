package cmd

import (
	"context"
	"fmt"
	"strconv"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage releases",
	}
	cmd.AddCommand(newReleaseListCmd())
	cmd.AddCommand(newReleaseViewCmd())
	cmd.AddCommand(newReleaseCreateCmd())
	cmd.AddCommand(newReleaseDeleteCmd())
	return cmd
}

func newReleaseListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List releases on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			releases, _, err := c.Repo.RepoListReleases(context.Background(), owner, repo, false, false, "", 1, 20)
			if err != nil {
				return err
			}
			if len(releases) == 0 {
				fmt.Println("no releases")
				return nil
			}
			for _, rel := range releases {
				fmt.Printf("#%d %s (%s)\n", rel.Id, rel.Name, rel.TagName)
			}
			return nil
		},
	}
}

func newReleaseViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <ID>",
		Short: "View a release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid release id: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			rel, _, err := c.Repo.RepoGetRelease(context.Background(), owner, repo, id)
			if err != nil {
				return err
			}
			fmt.Printf("%s (%s)\n", rel.Name, rel.TagName)
			fmt.Printf("Draft: %v  Prerelease: %v\n", rel.Draft, rel.Prerelease)
			if rel.Body != "" {
				fmt.Printf("\n%s\n", rel.Body)
			}
			if rel.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", rel.HtmlUrl)
			}
			return nil
		},
	}
}

func newReleaseCreateCmd() *cobra.Command {
	var name, body, tag string
	var draft, prerelease bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			rel, _, err := c.Repo.RepoCreateRelease(context.Background(), owner, repo, &forgejo.CreateReleaseOption{
				Name:       name,
				Body:       body,
				TagName:    tag,
				Draft:      draft,
				Prerelease: prerelease,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created #%d: %s (%s)\n", rel.Id, rel.Name, rel.TagName)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "release name")
	cmd.Flags().StringVarP(&body, "body", "b", "", "release body")
	cmd.Flags().StringVar(&tag, "tag", "", "tag name (required)")
	cmd.Flags().BoolVar(&draft, "draft", false, "mark as draft")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "mark as prerelease")
	return cmd
}

func newReleaseDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <ID>",
		Short: "Delete a release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid release id: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.RepoDeleteRelease(context.Background(), owner, repo, id)
			if err != nil {
				return err
			}
			fmt.Printf("Deleted release #%d\n", id)
			return nil
		},
	}
}
