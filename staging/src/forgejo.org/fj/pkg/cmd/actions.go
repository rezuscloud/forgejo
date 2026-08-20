package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

func newActionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "Manage repository actions",
	}
	cmd.AddCommand(newActionsJobsCmd())
	cmd.AddCommand(newActionsLogsCmd())
	cmd.AddCommand(newActionsTasksCmd())
	cmd.AddCommand(newActionsDispatchCmd())
	cmd.AddCommand(newActionsRunsCmd())
	cmd.AddCommand(newActionsVariablesCmd())
	cmd.AddCommand(newActionsSecretsCmd())
	return cmd
}

func newActionsJobsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jobs <RUN>",
		Short: "List the jobs in an action run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run id %q: %w", args[0], err)
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			jobs, _, err := c.Repo.ListActionRunJobs(context.Background(), owner, repo, runID)
			if err != nil {
				return err
			}
			if len(jobs) == 0 {
				fmt.Println("no jobs")
				return nil
			}
			for _, j := range jobs {
				runsOn := strings.Join(j.RunsOn, ",")
				fmt.Printf("#%d %s [%s] runs_on:%s\n", j.Id, j.Name, j.Status, runsOn)
			}
			return nil
		},
	}
}

func newActionsLogsCmd() *cobra.Command {
	var jobID int64
	var runID int64
	var outFile string
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View the logs of an action run or job",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			if jobID != 0 {
				logs, _, err := c.Repo.RepoGetActionJobLogs(context.Background(), owner, repo, jobID, 0)
				if err != nil {
					return err
				}
				fmt.Print(logs)
				return nil
			}
			if runID != 0 {
				logs, _, err := c.Repo.RepoGetActionRunLogs(context.Background(), owner, repo, runID)
				if err != nil {
					return err
				}
				path := outFile
				if path == "" {
					path = fmt.Sprintf("run-%d-logs.zip", runID)
				}
				if err := os.WriteFile(path, []byte(logs), 0644); err != nil {
					return err
				}
				fmt.Printf("wrote %d bytes to %s\n", len(logs), path)
				return nil
			}
			return fmt.Errorf("must specify --job <ID> or --run <ID>")
		},
	}
	cmd.Flags().Int64Var(&jobID, "job", 0, "print a single job's logs (plain text)")
	cmd.Flags().Int64Var(&runID, "run", 0, "download all jobs' logs for a run (zip)")
	cmd.Flags().StringVar(&outFile, "out", "", "output file for --run (default: run-<id>-logs.zip)")
	return cmd
}

func newActionsTasksCmd() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List the action tasks on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			res, _, err := c.Repo.ListActionTasks(context.Background(), owner, repo, page, 20, nil)
			if err != nil {
				return err
			}
			count := res.TotalCount
			if count == 1 {
				fmt.Println("1 task")
			} else {
				fmt.Printf("%d tasks\n", count)
			}
			for _, t := range res.WorkflowRuns {
				sym := statusSymbol(t.Status)
				sha := ""
				if t.HeadSha != "" && len(t.HeadSha) > 10 {
					sha = t.HeadSha[:10]
				}
				fmt.Printf("#%d (%s) %s %s (%s): %s\n",
					t.RunNumber, sha, sym, t.Name, t.Event, t.DisplayTitle)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&page, "page", "p", 1, "page number")
	return cmd
}

func newActionsDispatchCmd() *cobra.Command {
	var inputs []string
	cmd := &cobra.Command{
		Use:   "dispatch <WORKFLOW> <REF>",
		Short: "Dispatch a workflow",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName := args[0]
			ref := args[1]
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			inputMap := map[string]string{}
			for _, kv := range inputs {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid input %q (expected key=value)", kv)
				}
				inputMap[parts[0]] = parts[1]
			}
			body := &forgejo.DispatchWorkflowOption{
				Inputs:        inputMap,
				Ref:           ref,
				ReturnRunInfo: false,
			}
			_, _, err = c.Repo.DispatchWorkflow(context.Background(), owner, repo, workflowName, body)
			if err != nil {
				return err
			}
			fmt.Printf("Dispatched %s on %s with %d inputs\n", workflowName, ref, len(inputMap))
			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&inputs, "input", "I", nil, "workflow input (key=value, repeatable)")
	return cmd
}

func newActionsRunsCmd() *cobra.Command {
	var page, limit int
	var status, event, headSha, ref, workflowID string
	var runNumber int64
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "List action runs on a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			res, _, err := c.Repo.ListActionRuns(context.Background(), owner, repo,
				page, limit, parseStringSlice(event), parseStringSlice(status), runNumber, headSha, ref, workflowID)
			if err != nil {
				return err
			}
			runs := res.WorkflowRuns
			if len(runs) == 0 {
				fmt.Println("no runs")
				return nil
			}
			for _, r := range runs {
				sym := statusSymbol(r.Status)
				sha := ""
				if r.CommitSha != "" && len(r.CommitSha) > 10 {
					sha = r.CommitSha[:10]
				}
				fmt.Printf("#%d (%s) %s (%s): %s\n",
					r.IndexInRepo, sha, sym, r.Event, r.Title)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "runs per page")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (comma-separated: success,failure,running,...)")
	cmd.Flags().StringVar(&event, "event", "", "filter by event (comma-separated: push,pull_request,...)")
	cmd.Flags().StringVar(&headSha, "head-sha", "", "filter by head commit sha")
	cmd.Flags().StringVar(&ref, "ref", "", "filter by ref (branch)")
	cmd.Flags().StringVar(&workflowID, "workflow-id", "", "filter by workflow id")
	cmd.Flags().Int64Var(&runNumber, "run-number", 0, "filter by run number")
	return cmd
}

func newActionsVariablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variables",
		Short: "Manage action variables",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			vars, _, err := c.Repo.GetRepoVariablesList(context.Background(), owner, repo, 1, 50)
			if err != nil {
				return err
			}
			for _, v := range vars {
				fmt.Printf("%s", v.Name)
				if v.Data != "" {
					fmt.Printf(" = %s", v.Data)
				}
				fmt.Println()
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create <NAME> <VALUE>",
		Short: "Create a new variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.CreateRepoVariable(context.Background(), owner, repo, args[0], &forgejo.CreateVariableOption{Value: args[1]})
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <NAME>",
		Short: "Delete a variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.DeleteRepoVariable(context.Background(), owner, repo, args[0])
			return err
		},
	})
	return cmd
}

func newActionsSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage action secrets",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			secrets, _, err := c.Repo.RepoListActionsSecrets(context.Background(), owner, repo, 1, 50)
			if err != nil {
				return err
			}
			for _, s := range secrets {
				fmt.Println(s.Name)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create <NAME> <VALUE>",
		Short: "Create or update a secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.UpdateRepoSecret(context.Background(), owner, repo, args[0], &forgejo.CreateOrUpdateSecretOption{Data: args[1]})
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <NAME>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.Repo.DeleteRepoSecret(context.Background(), owner, repo, args[0])
			return err
		},
	})
	return cmd
}


