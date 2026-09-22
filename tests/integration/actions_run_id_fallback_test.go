// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	actions_model "forgejo.org/models/actions"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestActionsWebRouteRunIDFallback covers the /actions/runs/{run} route
// accepting the global run ID where the per-repo index is expected: run
// links carry the index, but the API and DB expose the ID, and numbers
// pasted from those surfaces 404'd before (#132, #136).
func TestActionsWebRouteRunIDFallback(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// two repos with one run each: distinct indexes AND distinct run IDs
		newRepo := func(name string) (*repo_model.Repository, func()) {
			repo, _, f := tests.CreateDeclarativeRepo(t, user2, name,
				[]unit_model.Type{unit_model.TypeActions}, nil,
				[]*files_service.ChangeRepoFile{
					{
						Operation:     "create",
						TreePath:      ".gitea/workflows/pr.yml",
						ContentReader: strings.NewReader("name: test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
					},
				},
			)
			assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))
			return repo, f
		}

		repo, f1 := newRepo("actionsRunIDFallback1")
		defer f1()
		otherRepo, f2 := newRepo("actionsRunIDFallback2")
		defer f2()

		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: repo.ID})
		require.NoError(t, run.LoadAttributes(t.Context()))
		assert.NotEqual(t, run.Index, run.ID, "test needs index and id to differ")

		// canonical index URL keeps its normal redirect to the latest attempt
		req := NewRequest(t, "GET", fmt.Sprintf("%s/actions/runs/%d", repo.HTMLURL(), run.Index))
		resp := MakeRequest(t, req, http.StatusTemporaryRedirect)
		assert.Contains(t, resp.Header().Get("Location"), fmt.Sprintf("/actions/runs/%d/jobs/", run.Index))

		// the run ID redirects to the canonical index URL (same sub-paths)
		req = NewRequest(t, "GET", fmt.Sprintf("%s/actions/runs/%d", repo.HTMLURL(), run.ID))
		resp = MakeRequest(t, req, http.StatusTemporaryRedirect)
		assert.Equal(t, fmt.Sprintf("%s/actions/runs/%d", repo.Link(), run.Index), resp.Header().Get("Location"))

		// a foreign repo's run ID must NOT redirect — the handler 404s
		otherRun := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: otherRepo.ID})
		req = NewRequest(t, "GET", fmt.Sprintf("%s/actions/runs/%d", repo.HTMLURL(), otherRun.ID))
		MakeRequest(t, req, http.StatusNotFound)

		// a number that is neither this repo's index nor its run ID 404s
		req = NewRequest(t, "GET", fmt.Sprintf("%s/actions/runs/%d", repo.HTMLURL(), otherRun.ID+run.ID+9999))
		MakeRequest(t, req, http.StatusNotFound)
	})
}
