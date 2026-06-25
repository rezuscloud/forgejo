package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
	}
	cmd.AddCommand(newRepoViewCmd())
	cmd.AddCommand(newRepoCloneCmd())
	return cmd
}

func newRepoViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view [OWNER/NAME]",
		Short: "View a repository's info",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClientWithRepoArg(cmd, args)
			if err != nil {
				return err
			}
			r, _, err := c.Repo.RepoGet(context.Background(), owner, repo)
			if err != nil {
				return err
			}
			fmt.Printf("%s/%s\n", owner, repo)
			if r.Description != "" {
				fmt.Printf("> %s\n", r.Description)
			}
			fmt.Printf("\n%d stars - %d watching - %d forks\n",
				r.StarsCount, r.WatchersCount, r.ForksCount)
			fmt.Printf("%d open issues - %d releases\n",
				r.OpenIssuesCount, r.ReleaseCounter)
			if r.Language != "" {
				fmt.Printf("Primary language: %s\n", r.Language)
			}
			if r.DefaultBranch != "" {
				fmt.Printf("Default branch: %s\n", r.DefaultBranch)
			}
			if r.HtmlUrl != "" {
				fmt.Printf("\nView online at %s\n", r.HtmlUrl)
			}
			return nil
		},
	}
}

func newRepoCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <OWNER/NAME> [DIRECTORY]",
		Short: "Clone a repository",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClientWithRepoArg(cmd, args)
			if err != nil {
				return err
			}
			r, _, err := c.Repo.RepoGet(context.Background(), owner, repo)
			if err != nil {
				return err
			}
			cloneURL := r.SshUrl
			if cloneURL == "" {
				cloneURL = r.CloneUrl
			}
			if cloneURL == "" {
				return fmt.Errorf("no clone URL for %s/%s", owner, repo)
			}
			dir := ""
			if len(args) > 1 {
				dir = args[1]
			}
			fmt.Printf("Cloning %s\n", cloneURL)
			gitArgs := []string{"clone", cloneURL}
			if dir != "" {
				gitArgs = append(gitArgs, dir)
			}
			gitCmd := exec.Command("git", gitArgs...)
			gitCmd.Stdin = os.Stdin
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr
			return gitCmd.Run()
		},
	}
}

func resolveClientWithRepoArg(cmd *cobra.Command, args []string) (*forgejo.Client, string, string, error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	if len(args) > 0 && strings.Contains(args[0], "/") {
		repoFlag = args[0]
	}
	if repoFlag != "" {
		cmd.Flags().Set("repo", repoFlag)
	}
	return resolveClient(cmd)
}
