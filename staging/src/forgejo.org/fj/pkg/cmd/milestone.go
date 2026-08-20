package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newMilestoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "milestone",
		Short:   "Manage milestones",
		Aliases: []string{"milestones"},
	}
	cmd.AddCommand(newMilestoneListCmd())
	cmd.AddCommand(newMilestoneViewCmd())
	cmd.AddCommand(newMilestoneCreateCmd())
	cmd.AddCommand(newMilestoneEditCmd())
	cmd.AddCommand(newMilestoneCloseCmd())
	cmd.AddCommand(newMilestoneDeleteCmd())
	return cmd
}

func newMilestoneListCmd() *cobra.Command {
	var state, name string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List milestones on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			ms, _, err := c.Repo.IssueGetMilestonesList(context.Background(), owner, repo, state, name, 1, 20)
			if err != nil {
				return err
			}
			if len(ms) == 0 {
				fmt.Println("no milestones")
				return nil
			}
			for _, m := range ms {
				fmt.Printf("#%d %s [%s] %s\n", m.Id, statusSymbol(stateStr(m.State)), stateStr(m.State), m.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&state, "state", "s", "open", "milestone state (open/closed/all)")
	cmd.Flags().StringVar(&name, "name", "", "filter by milestone title")
	return cmd
}

func newMilestoneViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <ID>",
		Short: "View a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone id: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			m, _, err := c.Repo.IssueGetMilestone(context.Background(), owner, repo, id)
			if err != nil {
				return err
			}
			fmt.Printf("#%d %s\n", m.Id, m.Title)
			fmt.Printf("State: %s\n", stateStr(m.State))
			if ts := timeStr(m.DueOn); ts != "" {
				fmt.Printf("Due: %s\n", ts)
			}
			fmt.Printf("Issues: %d open / %d closed\n", m.OpenIssues, m.ClosedIssues)
			if m.Description != "" {
				fmt.Printf("\n%s\n", m.Description)
			}
			return nil
		},
	}
	return cmd
}

// dueFlag parses the RFC3339 --due flag into the SDK's optional *time.Time.
// Returns nil for an absent flag (parameter omitted server-side). Unlike the
// generated parseTime, invalid values are an error rather than a zero time —
// a non-nil zero DueOn would marshal to "0001-01-01T00:00:00Z" and clobber
// the milestone's due date.
func dueFlag(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("invalid due date (RFC3339): %s", s)
	}
	return &t, nil
}

func newMilestoneCreateCmd() *cobra.Command {
	var title, description, due string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new milestone",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			dueOn, err := dueFlag(due)
			if err != nil {
				return err
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			m, _, err := c.Repo.IssueCreateMilestone(context.Background(), owner, repo, &forgejo.CreateMilestoneOption{
				Title:       title,
				Description: description,
				DueOn:       dueOn,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created #%d: %s\n", m.Id, m.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "milestone title (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "milestone description")
	cmd.Flags().StringVar(&due, "due", "", "due date (RFC3339)")
	return cmd
}

func newMilestoneEditCmd() *cobra.Command {
	var title, description, state, due string
	cmd := &cobra.Command{
		Use:   "edit <ID>",
		Short: "Edit a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone id: %s", args[0])
			}
			dueOn, err := dueFlag(due)
			if err != nil {
				return err
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			// Unset flags produce empty strings, which the option's omitempty
			// tags drop from the PATCH body — only provided fields change.
			m, _, err := c.Repo.IssueEditMilestone(context.Background(), owner, repo, id, &forgejo.EditMilestoneOption{
				Title:       title,
				Description: description,
				State:       state,
				DueOn:       dueOn,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Updated #%d: %s\n", m.Id, m.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "new milestone title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "new milestone description")
	cmd.Flags().StringVar(&state, "state", "", "milestone state (open/closed)")
	cmd.Flags().StringVar(&due, "due", "", "due date (RFC3339)")
	return cmd
}

func newMilestoneCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <ID>",
		Short: "Close a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone id: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, _, err = c.Repo.IssueEditMilestone(context.Background(), owner, repo, id, &forgejo.EditMilestoneOption{
				State: "closed",
			})
			if err != nil {
				return err
			}
			fmt.Printf("Closed #%d\n", id)
			return nil
		},
	}
}

func newMilestoneDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <ID>",
		Short: "Delete a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone id: %s", args[0])
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.IssueDeleteMilestone(context.Background(), owner, repo, id)
			if err != nil {
				return err
			}
			fmt.Printf("Deleted milestone #%d\n", id)
			return nil
		},
	}
}
