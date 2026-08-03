package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Short:   "Manage issues",
		Aliases: []string{"issues"},
	}
	cmd.AddCommand(newIssueListCmd())
	cmd.AddCommand(newIssueViewCmd())
	cmd.AddCommand(newIssueCreateCmd())
	cmd.AddCommand(newIssueCommentCmd())
	cmd.AddCommand(newIssueCloseCmd())
	return cmd
}

func newIssueListCmd() *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			issues, _, err := c.Repo.IssueListIssues(context.Background(), owner, repo, state, "", "", "", "", time.Time{}, time.Time{}, "", "", "", 1, 20, "")
			if err != nil {
				return err
			}
			if len(issues) == 0 {
				fmt.Println("no issues")
				return nil
			}
			for _, is := range issues {
				fmt.Printf("#%d %s [%s] %s\n", is.Number, statusSymbol(stateStr(is.State)), stateStr(is.State), is.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&state, "state", "s", "open", "issue state (open/closed/all)")
	return cmd
}

func newIssueViewCmd() *cobra.Command {
	var showComments bool
	cmd := &cobra.Command{
		Use:   "view <INDEX>",
		Short: "View an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			is, _, err := c.Repo.IssueGetIssue(context.Background(), owner, repo, index)
			if err != nil {
				return err
			}
			fmt.Printf("#%d %s\n", is.Number, is.Title)
			fmt.Printf("State: %s\n", stateStr(is.State))
			if is.Body != "" {
				fmt.Printf("\n%s\n", is.Body)
			}
			if showComments {
				comments, _, err := c.Repo.IssueGetComments(context.Background(), owner, repo, index, time.Time{}, time.Time{})
				if err != nil {
					return fmt.Errorf("fetch comments: %w", err)
				}
				fmt.Printf("\n— Comments (%d) —\n", len(comments))
				for _, cm := range comments {
					author := "unknown"
					if cm.User != nil && cm.User.Login != "" {
						author = cm.User.Login
					}
					if ts := timeStr(cm.CreatedAt); ts != "" {
						fmt.Printf("\n[%s] %s:\n", ts, author)
					} else {
						fmt.Printf("\n%s:\n", author)
					}
					if cm.Body != "" {
						fmt.Println(cm.Body)
					}
				}
			}
			if is.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", is.HtmlUrl)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&showComments, "comments", "c", false, "show the comment thread")
	return cmd
}

func newIssueCreateCmd() *cobra.Command {
	var title, body string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			is, _, err := c.Repo.IssueCreateIssue(context.Background(), owner, repo, &forgejo.CreateIssueOption{
				Title: title,
				Body:  body,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created #%d: %s\n", is.Number, is.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "issue title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "issue body")
	return cmd
}

func newIssueCommentCmd() *cobra.Command {
	var body string
	cmd := &cobra.Command{
		Use:   "comment <INDEX>",
		Short: "Comment on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if body == "" {
				return fmt.Errorf("--body is required")
			}
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, _, err = c.Repo.IssueCreateComment(context.Background(), owner, repo, index, &forgejo.CreateIssueCommentOption{
				Body: body,
			})
			if err != nil {
				return err
			}
			fmt.Println("Comment added")
			return nil
		},
	}
	cmd.Flags().StringVarP(&body, "body", "b", "", "comment body (required)")
	return cmd
}

func newIssueCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <INDEX>",
		Short: "Close an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, _, err = c.Repo.IssueEditIssue(context.Background(), owner, repo, index, &forgejo.EditIssueOption{
				State: "closed",
			})
			if err != nil {
				return err
			}
			fmt.Printf("Closed #%d\n", index)
			return nil
		},
	}
}
