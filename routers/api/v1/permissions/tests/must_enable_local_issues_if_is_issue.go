// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

<<<<<<< HEAD
	unit_model "forgejo.org/models/unit"
=======
>>>>>>> upstream/v16.0/forgejo
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.MustEnableLocalIssuesIfIsIssue, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass if a repository with issues unit is present
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository(),
			),
		},
		{
			// fail if an issue exists in a repository with issues unit disabled
			data: newTestData(map[string]string{
				"issue":       "issue5000",
				"issueAuthor": "issueAuthor",
			}, newSharedData().
				SetDoer().
				SetRepository().
				SetRepositoryDisabledUnits([]unit_model.Type{unit_model.TypeIssues}),
			),
			error: "Not Found",
		},
		{
			// pass if a pull request exists in a repository with issues unit disabled
			data: newTestData(map[string]string{
=======
			data: newTestData(map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
				"issue":       "issue5000",
				"issueAuthor": "issueAuthor",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":          "doerregular",
				"repository":    "userowner/repositorypublic",
				"issue":         "issue5000",
				"issueAuthor":   "issueAuthor",
				"disable-units": "repo.issues",
			}),
			error: "Not Found",
		},
		{ // does not fail because it is an issue instead of a pull request
			data: newTestData(map[string]string{
				"doer":              "userowner",
				"repository":        "userowner/repositorypublic",
				"repository-init":   "true",
>>>>>>> upstream/v16.0/forgejo
				"pullRequestAuthor": "userowner",
				"pullRequestBranch": "MustEnableLocalIssuesIfIsIssue",
				"pullRequest":       "MustEnableLocalIssuesIfIsIssue",
				"issue":             "MustEnableLocalIssuesIfIsIssue",
<<<<<<< HEAD
			}, newSharedData().
				SetDoer().
				SetRepository().SetRepositoryName("userowner/repositorypublic").
				SetRepositoryDisabledUnits([]unit_model.Type{unit_model.TypeIssues}).
				SetRepositoryInit(true),
			),
=======
				"disable-units":     "repo.issues",
			}),
>>>>>>> upstream/v16.0/forgejo
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetDefault("issue", "issueOne")
		data.SetDefault("issueAuthor", "issueAuthor")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
<<<<<<< HEAD
		fixtureDisableUnits(t, permissions, data.shared.RepositoryDisabledUnits())
=======
		fixtureDisableUnits(t, permissions, data)
>>>>>>> upstream/v16.0/forgejo
		if data.Has("pullRequest") {
			require.True(t, data.Has("pullRequestBranch"))
			fixtureCreateBranch(t, permissions, data.Get("pullRequestBranch"))
			require.True(t, data.Has("pullRequestAuthor"))
			require.True(t, data.Has("pullRequest"))
<<<<<<< HEAD
			fixtureCreatePullRequest(t, permissions, data.Get("pullRequest"), data.Get("pullRequestAuthor"), data.Get("pullRequestBranch"))
			require.Equal(t, data.Get("issue"), data.Get("pullRequest"))
		} else if data.Has("issue") {
			issueAuthor := fixtureCreateUser(t, &user_model.User{Name: data.Get("issueAuthor")})
			fixtureSetIssue(t, permissions, data.Get("issue"), issueAuthor.Name)
=======
			fixtureCreatePullRequest(t, permissions, data)
			require.Equal(t, data.Get("issue"), data.Get("pullRequest"))
		} else {
			fixtureCreateUser(t, &user_model.User{Name: data.Get("issueAuthor")})
			fixtureSetIssue(t, permissions, data)
>>>>>>> upstream/v16.0/forgejo
		}
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
<<<<<<< HEAD
		var index int64
		if data.Has("issue") {
			index = fixtureGetIssue(t, data.Get("issue")).Index
		}
=======
		index := fixtureGetIssue(t, data).Index
>>>>>>> upstream/v16.0/forgejo
		t.Logf("calling MustEnableLocalIssuesIfIsIssue(ctx, %d)", index)
		apiv1_permissions.MustEnableLocalIssuesIfIsIssue(ctx, index)
	},
})
