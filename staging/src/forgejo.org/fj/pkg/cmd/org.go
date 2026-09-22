package cmd

import (
	"context"
	"fmt"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
	}
	cmd.AddCommand(newOrgViewCmd())
	cmd.AddCommand(newOrgListCmd())
	cmd.AddCommand(newOrgCreateCmd())
	return cmd
}

// newOrgCreateCmd is the friendly surface over POST /orgs (#105): the raw
// generated command names its body-blob flag after the spec's body
// parameter ("organization"), so `--organization <name>` posts a bare
// string and the forge answers 422 UserName-required. Here every option is
// its own flag, mapped into CreateOrgOption.
func newOrgCreateCmd() *cobra.Command {
	var username, fullName, description, website, location, visibility string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return fmt.Errorf("--username is required")
			}
			host, _ := cmd.Flags().GetString("host")
			c, err := resolveHostClient(cmd, host)
			if err != nil {
				return err
			}
			org, _, err := c.Org.OrgCreate(context.Background(), &forgejo.CreateOrgOption{
				Username:    username,
				FullName:    fullName,
				Description: description,
				Website:     website,
				Location:    location,
				Visibility:  visibility,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created %s", org.Username)
			if org.Visibility != "" {
				fmt.Printf(" (%s)", org.Visibility)
			}
			fmt.Println()
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "organization name (required, unique)")
	cmd.Flags().StringVar(&fullName, "full-name", "", "display name")
	cmd.Flags().StringVar(&description, "description", "", "description")
	cmd.Flags().StringVar(&website, "website", "", "website URL")
	cmd.Flags().StringVar(&location, "location", "", "location")
	cmd.Flags().StringVar(&visibility, "visibility", "", "public (default), limited or private")
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
