package cmd

import (
	"context"
	"fmt"

	forgejo "forgejo.org/client-go"
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
	cmd.AddCommand(newUserKeyCmd())
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

// newUserKeyCmd implements `fj user key` (subcommands add/list).
// `user key add` maps to operationId userCurrentPostKey (the Misc service,
// since /user/keys classifies as misc). `user key list` maps to userListKeys
// for a given user. Mirrors the Rust forgejo-cli's `user key` group.
func newUserKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage SSH keys",
	}
	cmd.AddCommand(newUserKeyAddCmd())
	cmd.AddCommand(newUserKeyListCmd())
	return cmd
}

func newUserKeyAddCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "add <KEY>",
		Short: "Add an SSH public key to your account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			key, _, err := c.Misc.UserCurrentPostKey(context.Background(), &forgejo.CreateKeyOption{
				Title: title,
				Key:   args[0],
			})
			if err != nil {
				return err
			}
			fmt.Printf("Added key %d: %s\n", key.Id, key.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "key title")
	return cmd
}

func newUserKeyListCmd() *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List a user's SSH keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			if username == "" {
				me, _, err := c.Misc.UserGetCurrent(context.Background())
				if err != nil {
					return err
				}
				username = me.Login
			}
			keys, _, err := c.User.UserListKeys(context.Background(), username, "", 1, 50)
			if err != nil {
				return err
			}
			if len(keys) == 0 {
				fmt.Println("no keys")
				return nil
			}
			for _, k := range keys {
				fmt.Printf("%d %s %s\n", k.Id, k.Title, truncate(k.Key, 40))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "user", "u", "", "username (default: current user)")
	return cmd
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
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
