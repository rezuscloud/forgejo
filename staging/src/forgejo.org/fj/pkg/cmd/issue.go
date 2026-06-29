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
		Use:   "issue",
		Short: "Manage issues",
		Aliases: []string{"issues"},
	}
	cmd.AddCommand(newIssueListCmd())
	cmd.AddCommand(newIssueViewCmd())
	cmd.AddCommand(newIssueCreateCmd())
	cmd.AddCommand(newIssueSearchCmd())
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
				fmt.Printf("#%d %s [%s] %s\n", issueNumber(is), statusSymbol(stateStr(is.State)), stateStr(is.State), is.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&state, "state", "s", "open", "issue state (open/closed/all)")
	return cmd
}

func newIssueViewCmd() *cobra.Command {
	return &cobra.Command{
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
			fmt.Printf("#%d %s\n", issueNumber(*is), is.Title)
			fmt.Printf("State: %s\n", stateStr(is.State))
			if is.Body != "" {
				fmt.Printf("\n%s\n", is.Body)
			}
			if is.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", is.HtmlUrl)
			}
			return nil
		},
	}
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
			fmt.Printf("Created #%d: %s\n", issueNumber(*is), is.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "issue title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "issue body")
	return cmd
}

// newIssueSearchCmd implements `fj issue search` (operationId issueSearchIssues).
// Searches issues across all repos the authenticated user can see, mirroring
// the Rust forgejo-cli's `issue search`.
func newIssueSearchCmd() *cobra.Command {
	var state, q string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search issues across repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			// issueSearchIssues(state, labels, milestones, q, priorityRepoId,
			//   type_, since, before, assigned, created, mentioned,
			//   reviewRequested, reviewed, owner, team, page, limit, sort)
			issues, _, err := c.Repo.IssueSearchIssues(
				context.Background(), state, "", "", q, 0, "",
				time.Time{}, time.Time{},
				false, false, false, false, false,
				"", "", 1, limit, "",
			)
			if err != nil {
				return err
			}
			if len(issues) == 0 {
				fmt.Println("no issues")
				return nil
			}
			for _, is := range issues {
				repo := ""
				if is.Repository != nil {
					repo = is.Repository.FullName + " "
				}
				fmt.Printf("%s#%d %s [%s]\n", repo, is.Id, is.Title, stateStr(is.State))
			}
			return nil
			},
		}
		cmd.Flags().StringVarP(&state, "state", "s", "open", "issue state (open/closed/all)")
		cmd.Flags().StringVarP(&q, "query", "q", "", "search query")
		cmd.Flags().IntVarP(&limit, "limit", "l", 20, "max results")
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
				Body:      body,
				UpdatedAt: time.Now(),
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

func issueNumber(is forgejo.Issue) int64 {
	if is.Number != 0 {
		return is.Number
	}
	return is.Id
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
				State:     "closed",
				UpdatedAt: time.Now(),
			})
			if err != nil {
				return err
			}
			fmt.Printf("Closed #%d\n", index)
			return nil
		},
	}
}

