package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestPolishedReviewSurface pins the code-anchored review surface (issue #109):
// the polished review group exposes typed flags for the fields the raw
// `fj api` layer only offers as an opaque --body JSON blob — path and the
// diff-position pair on comment, event/commit-id on create and submit.
func TestPolishedReviewSurface(t *testing.T) {
	var review *cobra.Command
	for _, g := range NewPolishedCmds() {
		if g.Name() == "review" {
			review = g
			break
		}
	}
	if review == nil {
		t.Fatal("polished `review` group not registered by NewPolishedCmds")
	}

	cases := []struct {
		name     string
		wantFlgs []string
		required []string
	}{
		{"create", []string{"body", "event", "commit-id"}, []string{"event"}},
		{"comment", []string{"body", "path", "old-position", "new-position"}, []string{"body", "path"}},
		{"submit", []string{"body", "event"}, []string{"event"}},
	}
	for _, tc := range cases {
		sub := findSubCmd(review, tc.name)
		if sub == nil {
			t.Fatalf("review %s: command missing", tc.name)
		}
		got := map[string]bool{}
		sub.Flags().VisitAll(func(f *pflag.Flag) { got[f.Name] = true })
		if len(got) != len(tc.wantFlgs) {
			t.Errorf("review %s: flag set %v, want exactly %v", tc.name, keysOf(got), tc.wantFlgs)
		}
		for _, f := range tc.wantFlgs {
			if !got[f] {
				t.Errorf("review %s: flag --%s missing (have: %v)", tc.name, f, keysOf(got))
			}
		}
		for _, f := range tc.required {
			if err := runExpectRequired(sub, tc.wantFlgs, f); err != "" {
				t.Errorf("review %s: --%s required-guard not enforced: %s", tc.name, f, err)
			}
		}
	}
}

// runExpectRequired leaves every flag empty and runs RunE with a bare command:
// the generated required-guards fire before any client resolution, so the call
// is hermetic. Empty string = the guard for `flag` fired correctly.
func runExpectRequired(c *cobra.Command, flags []string, missing string) string {
	for _, f := range flags {
		if f != missing {
			_ = c.Flags().Set(f, "x")
		} else {
			_ = c.Flags().Set(f, "")
		}
	}
	err := c.RunE(c, []string{"o", "r", "1"})
	if err == nil {
		return "RunE succeeded with no host/flags"
	}
	if strings.Contains(err.Error(), "--"+missing+" is required") {
		return ""
	}
	return "unexpected error: " + err.Error()
}

func findSubCmd(g *cobra.Command, name string) *cobra.Command {
	for _, c := range g.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func keysOf(m map[string]bool) string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return strings.Join(out, ",")
}
