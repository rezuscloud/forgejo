package cmd

import (
	"context"
	"fmt"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage tags",
	}
	cmd.AddCommand(newTagListCmd())
	cmd.AddCommand(newTagCreateCmd())
	cmd.AddCommand(newTagDeleteCmd())
	return cmd
}

func newTagListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tags on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			tags, _, err := c.Repo.RepoListTags(context.Background(), owner, repo, 1, 50)
			if err != nil {
				return err
			}
			if len(tags) == 0 {
				fmt.Println("no tags")
				return nil
			}
			for _, t := range tags {
				msg := ""
				if t.Message != "" {
					msg = " " + t.Message
				}
				fmt.Printf("%s%s\n", t.Name, msg)
			}
			return nil
		},
	}
}

func newTagCreateCmd() *cobra.Command {
	var message, target string
	cmd := &cobra.Command{
		Use:   "create <TAG>",
		Short: "Create a tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, _, err = c.Repo.RepoCreateTag(context.Background(), owner, repo, &forgejo.CreateTagOption{
				TagName: args[0],
				Message: message,
				Target:  target,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created tag %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "tag message")
	cmd.Flags().StringVarP(&target, "target", "t", "", "target commit SHA (default: repo default branch HEAD)")
	return cmd
}

func newTagDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <TAG>",
		Short: "Delete a tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.RepoDeleteTag(context.Background(), owner, repo, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Deleted tag %s\n", args[0])
			return nil
		},
	}
}
