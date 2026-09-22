// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"

	runnerv1 "code.forgejo.org/forgejo/actions-proto/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestAPIGetActionRunByIndex covers GET /repos/{o}/{r}/actions/runs/index/{index}
// — the index_in_repo → run resolution that lets clients use the number the
// web UI shows instead of the run's global database id (#108).
func TestAPIGetActionRunByIndex(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip()
	}
	now := time.Now()
	outcome := &mockTaskOutcome{
		result: runnerv1.Result_RESULT_SUCCESS,
		logRows: []*runnerv1.LogRow{
			{Time: timestamppb.New(now.Add(1 * time.Second)), Content: "hello from run index test"},
		},
		stepStates: []*runnerv1.StepState{
			{
				Id:        0,
				Result:    runnerv1.Result_RESULT_SUCCESS,
				LogIndex:  0,
				LogLength: 1,
				StartedAt: timestamppb.New(now),
				StoppedAt: timestamppb.New(now.Add(2 * time.Second)),
			},
		},
	}
	workflow := `name: api-run-index
on: push
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello from run index test
`
	treePath := ".forgejo/workflows/api-run-index.yml"

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session,
			auth_model.AccessTokenScopeWriteRepository,
			auth_model.AccessTokenScopeWriteUser,
		)

		apiRepoA := createActionsTestRepo(t, token, "actions-run-index-api", false)
		repoA := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: apiRepoA.ID})

		apiRepoB := createActionsTestRepo(t, token, "actions-run-index-api-other", false)

		runner := newMockRunner()
		runner.registerAsRepoRunner(t, user2.Name, repoA.Name, "mock-runner", []string{"ubuntu-latest"})

		opts := getWorkflowCreateFileOptions(user2, repoA.DefaultBranch,
			fmt.Sprintf("create %s", treePath), workflow)
		createWorkflowFile(t, token, user2.Name, repoA.Name, treePath, opts)

		task := runner.fetchTask(t)
		runner.execTask(t, task, outcome)

		actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.Id})
		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: actionTask.JobID})
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: job.RunID})
		// The fresh test repo's first run: its index is the number the UI
		// would show — and, crucially, it is NOT how the {run_id} routes
		// address it (they want the global DB id above).
		assert.Equal(t, int64(1), run.Index, "first run in a fresh repo must have index 1")

		t.Run("happy path: 200 and resolves index to the run", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/actions/runs/index/%d",
				repoA.FullName(), run.Index,
			)
			req.AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)

			var got api.ActionRun
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))

			assert.Equal(t, run.ID, got.ID, "by-index must resolve to the same run as by-id")
			assert.Equal(t, run.Index, got.Index, "index_in_repo round-trips")
			assert.Equal(t, run.Title, got.Title)
		})

		t.Run("by-id logs route unaffected (regression: the greedy flip broke it too)", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/actions/runs/%d/logs",
				repoA.FullName(), run.ID,
			)
			req.AddTokenAuth(token)
			// The mock runner produced log rows; the route must answer 200
			// (zip), never the "run with id 0" 404 of the param regression.
			MakeRequest(t, req, http.StatusOK)
		})

		t.Run("cross-repo: 404 — index belongs to another repo's sequence", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/actions/runs/index/%d",
				apiRepoB.FullName, run.Index,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("not found: 404 for unknown index", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/actions/runs/index/%d",
				repoA.FullName(), run.Index+999999,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("wrong scope: 403 without read:repository", func(t *testing.T) {
			weakToken := getTokenForLoggedInUser(t, session,
				auth_model.AccessTokenScopeReadUser,
			)
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/actions/runs/index/%d",
				repoA.FullName(), run.Index,
			)
			req.AddTokenAuth(weakToken)
			MakeRequest(t, req, http.StatusForbidden)
		})

		httpContextA := NewAPITestContext(t, user2.Name, repoA.Name, auth_model.AccessTokenScopeWriteUser)
		doAPIDeleteRepository(httpContextA)(t)
		httpContextB := NewAPITestContext(t, user2.Name, apiRepoB.Name, auth_model.AccessTokenScopeWriteUser)
		doAPIDeleteRepository(httpContextB)(t)
	})
}
