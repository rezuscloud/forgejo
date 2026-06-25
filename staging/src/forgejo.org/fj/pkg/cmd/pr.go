package cmd

import (
	"context"
	"fmt"
	"strconv"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newPrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pr",
		Short:   "Manage pull requests",
		Aliases: []string{"prs"},
	}
	cmd.AddCommand(newPrListCmd())
	cmd.AddCommand(newPrViewCmd())
	cmd.AddCommand(newPrCreateCmd())
	cmd.AddCommand(newPrMergeCmd())
	return cmd
}

func newPrListCmd() *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			prs, _, err := c.Repo.RepoListPullRequests(context.Background(), owner, repo, state, "", 0, nil, "", "", "", 1, 20)
			if err != nil {
				return err
			}
			if len(prs) == 0 {
				fmt.Println("no pull requests")
				return nil
			}
			for _, pr := range prs {
				fmt.Printf("#%d %s [%s] %s\n", pr.Number, statusSymbol(stateStr(pr.State)), stateStr(pr.State), pr.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&state, "state", "s", "open", "PR state (open/closed/all)")
	return cmd
}

func newPrViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <INDEX>",
		Short: "View a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			pr, _, err := c.Repo.RepoGetPullRequest(context.Background(), owner, repo, index)
			if err != nil {
				return err
			}
			fmt.Printf("#%d %s\n", pr.Number, pr.Title)
			fmt.Printf("State: %s  Merged: %v\n", stateStr(pr.State), pr.Merged)
			if pr.Base != nil && pr.Head != nil {
				fmt.Printf("Base: %s  Head: %s\n", pr.Base.Ref, pr.Head.Ref)
			}
			if pr.Body != "" {
				fmt.Printf("\n%s\n", pr.Body)
			}
			if pr.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", pr.HtmlUrl)
			}
			return nil
		},
	}
}

func newPrCreateCmd() *cobra.Command {
	var title, body, head, base string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if head == "" {
				return fmt.Errorf("--head is required")
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			pr, _, err := c.Repo.RepoCreatePullRequest(context.Background(), owner, repo, &forgejo.CreatePullRequestOption{
				Title: title,
				Body:  body,
				Head:  head,
				Base:  base,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created #%d: %s\n", pr.Number, pr.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "PR title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "PR body")
	cmd.Flags().StringVar(&head, "head", "", "head branch (required)")
	cmd.Flags().StringVar(&base, "base", "main", "base branch")
	return cmd
}

func newPrMergeCmd() *cobra.Command {
	var style string
	cmd := &cobra.Command{
		Use:   "merge <INDEX>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			body := &forgejo.MergePullRequestOption{Do: style}
			_, err = c.Repo.RepoMergePullRequest(context.Background(), owner, repo, index, body)
			if err != nil {
				return err
			}
			fmt.Printf("Merged #%d (%s)\n", index, style)
			return nil
		},
	}
	cmd.Flags().StringVarP(&style, "style", "s", "merge", "merge style (merge/rebase/squash/rebase-merge/fast-forward-only)")
	return cmd
}

