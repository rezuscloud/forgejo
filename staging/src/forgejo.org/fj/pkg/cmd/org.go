package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
	}
	cmd.AddCommand(newOrgViewCmd())
	cmd.AddCommand(newOrgListCmd())
	return cmd
}

func newOrgViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <ORG>",
		Short: "View an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			org, _, err := c.Org.OrgGet(context.Background(), args[0])
			if err != nil {
				return err
			}
			fmt.Println(org.Username)
			if org.FullName != "" {
				fmt.Printf("Full name: %s\n", org.FullName)
			}
			if org.Description != "" {
				fmt.Printf("Description: %s\n", org.Description)
			}
			if org.Website != "" {
				fmt.Printf("Website: %s\n", org.Website)
			}
			return nil
		},
	}
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			orgs, _, err := c.Org.OrgGetAll(context.Background(), 1, 50)
			if err != nil {
				return err
			}
			if len(orgs) == 0 {
				fmt.Println("no organizations")
				return nil
			}
			for _, org := range orgs {
				fmt.Println(org.Username)
			}
			return nil
		},
	}
}
