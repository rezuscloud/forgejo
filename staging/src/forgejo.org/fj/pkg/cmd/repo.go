package cmd

import (
	"context"
	"fmt"
	"net/url"
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
	cmd.AddCommand(newRepoForkCmd())
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
			// Prefer HTTPS by default so cloning works against ephemeral test
			// instances without an SSH listener/host key setup. Fall back to SSH
			// when no HTTPS clone URL is advertised.
			cloneURL := r.CloneUrl
			if cloneURL == "" {
				cloneURL = r.SshUrl
			}
			if cloneURL == "" {
				return fmt.Errorf("no clone URL for %s/%s", owner, repo)
			}
			dir := ""
			if len(args) > 1 {
				dir = args[1]
			}
			cloneURL = rewriteCloneURLForHost(cmd, cloneURL)
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

// newRepoForkCmd implements `fj repo fork` (operationId createFork). Forks a
// repo into the authenticated user's namespace (or --org). Mirrors the Rust
// forgejo-cli's `repo fork`.
func newRepoForkCmd() *cobra.Command {
	var org string
	cmd := &cobra.Command{
		Use:   "fork [OWNER/NAME]",
		Short: "Fork a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClientWithRepoArg(cmd, args)
			if err != nil {
				return err
			}
			opt := &forgejo.CreateForkOption{}
			if org != "" {
				opt.Organization = org
			}
			if _, err := c.Repo.CreateFork(context.Background(), owner, repo, opt); err != nil {
				return err
			}
			fmt.Printf("Forked %s/%s", owner, repo)
			if org != "" {
				fmt.Printf(" into %s", org)
			}
			fmt.Println()
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", "", "organization to fork into (default: your user)")
	return cmd
}

// rewriteCloneURLForHost rewrites an advertised clone URL to the actual host
// the CLI is targeting. This is critical for ephemeral test containers: the
// server advertises localhost:3000 inside the container, but the test runner
// talks to the mapped host port (e.g. localhost:32776). Preserving the path
// while swapping scheme+host keeps clone working without requiring SSH.
func rewriteCloneURLForHost(cmd *cobra.Command, cloneURL string) string {
	hostFlag, _ := cmd.Flags().GetString("host")
	if hostFlag == "" {
		return cloneURL
	}
	target, err1 := url.Parse(hostFlag)
		advertised, err2 := url.Parse(cloneURL)
	if err1 != nil || err2 != nil || target.Host == "" {
		return cloneURL
	}
	advertised.Scheme = target.Scheme
	advertised.Host = target.Host
	return advertised.String()
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
