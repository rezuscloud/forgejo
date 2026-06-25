package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(newUserViewCmd())
	cmd.AddCommand(newUserSearchCmd())
	cmd.AddCommand(newUserReposCmd())
	return cmd
}

func newUserViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <USERNAME>",
		Short: "View a user's info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			user, _, err := c.User.UserGet(context.Background(), args[0])
			if err != nil {
				return err
			}
			fmt.Println(user.Login)
			if user.FullName != "" {
				fmt.Printf("Full name: %s\n", user.FullName)
			}
			if user.Email != "" {
				fmt.Printf("Email: %s\n", user.Email)
			}
			fmt.Printf("Followers: %d  Following: %d\n", user.FollowersCount, user.FollowingCount)
			fmt.Printf("Repos: %d\n", user.StarredReposCount)
			if user.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", user.HtmlUrl)
			}
			return nil
		},
	}
}

func newUserSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <QUERY>",
		Short: "Search for users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			result, _, err := c.User.UserSearch(context.Background(), args[0], 0, "", 1, 20)
			if err != nil {
				return err
			}
			_ = result
			fmt.Println("(search results depend on the response type)")
			return nil
		},
	}
}

func newUserReposCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repos <USERNAME>",
		Short: "List a user's repositories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			repos, _, err := c.User.UserListRepos(context.Background(), args[0], 1, 50)
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				fmt.Println("no repositories")
				return nil
			}
			for _, r := range repos {
				fmt.Printf("%s/%s", r.Owner.Login, r.Name)
				if r.Description != "" {
					fmt.Printf(" — %s", r.Description)
				}
				fmt.Println()
			}
			return nil
		},
	}
}
