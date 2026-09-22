// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	forgejo "forgejo.org/client-go"
	"github.com/spf13/cobra"
)

// newIssueLabelCmd manages labels on an issue — or a pull request, which the
// API addresses through the same issue endpoints. Label NAMES are accepted
// everywhere: the add/replace body and the remove identifier resolve names
// server-side, so no local name→id mapping is needed (#104). Mounted both in
// the descriptor-driven issue group (polish.json "extra") and the hand-written
// pr group.
func newIssueLabelCmd() *cobra.Command {
	var add, remove, set string
	cmd := &cobra.Command{
		Use:   "label <INDEX>",
		Short: "List or modify the labels on an issue or PR",
		Long: `List or modify the labels on an issue or PR.

With no flags, prints the current labels. --add appends, --remove deletes
(single or comma-separated names), --set replaces the whole label set.
Label names resolve server-side; the forge silently drops unknown names
on add/set — the trailing label list this command prints is the
verification surface. --set cannot be combined with --add/--remove.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid index %q: %w", args[0], err)
			}
			c, owner, repo, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			if set != "" && (add != "" || remove != "") {
				return fmt.Errorf("--set replaces all labels; do not combine it with --add or --remove")
			}
			if add != "" {
				names := splitLabelNames(add)
				if _, _, err := c.Repo.IssueReplaceLabels(context.Background(), owner, repo, index,
					&forgejo.IssueLabelsOption{Labels: labelRefs(names)}); err != nil {
					return fmt.Errorf("add labels: %w", err)
				}
			}
			for _, name := range splitLabelNames(remove) {
				if _, err := c.Repo.IssueRemoveLabel(context.Background(), owner, repo, index,
					name, &forgejo.DeleteLabelsOption{}); err != nil {
					return fmt.Errorf("remove label %q: %w", name, err)
				}
			}
			if set != "" {
				names := splitLabelNames(set)
				if _, _, err := c.Repo.IssueReplaceLabels(context.Background(), owner, repo, index,
					&forgejo.IssueLabelsOption{Labels: labelRefs(names)}); err != nil {
					return fmt.Errorf("set labels: %w", err)
				}
			}
			labels, _, err := c.Repo.IssueGetLabels(context.Background(), owner, repo, index)
			if err != nil {
				return err
			}
			if len(labels) == 0 {
				fmt.Println("no labels")
				return nil
			}
			for _, l := range labels {
				fmt.Printf("#%d %s %s\n", l.Id, l.Name, l.Color)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&add, "add", "", "labels to append (comma-separated names)")
	cmd.Flags().StringVar(&remove, "remove", "", "labels to delete (comma-separated names)")
	cmd.Flags().StringVar(&set, "set", "", "replace the label set with exactly these names")
	return cmd
}

func splitLabelNames(csv string) []string {
	if csv == "" {
		return nil
	}
	var names []string
	for _, n := range strings.Split(csv, ",") {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, n)
		}
	}
	return names
}

func labelRefs(names []string) []any {
	refs := make([]any, len(names))
	for i, n := range names {
		refs[i] = n
	}
	return refs
}
