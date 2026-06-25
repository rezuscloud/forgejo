package cmd

import (
	"context"
	"fmt"

	"forgejo.org/fj/internal/client"
	"forgejo.org/fj/internal/keys"
	"github.com/spf13/cobra"
)

func newWhoAmICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			if host == "" {
				return fmt.Errorf("--host is required")
			}
			ki, _ := keys.Load()
			c, err := client.FromKeys(ki, host)
			if err != nil {
				return err
			}
			user, _, err := c.Misc.UserGetCurrent(context.Background())
			if err != nil {
				return err
			}
			login := user.Login
			email := user.Email
			fmt.Printf("currently signed in as %s", login)
			if email != "" {
				fmt.Printf(" <%s>", email)
			}
			fmt.Printf(" @ %s\n", host)
			return nil
		},
	}
}
