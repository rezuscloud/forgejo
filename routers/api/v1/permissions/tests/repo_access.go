// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.RepoAccess, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// The authenticated doer can access to a publicly
			// readable repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository(),
			),
		},
		{
			// An anonymous visitor can access to a publicly
			// readable repository
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true).
				SetRepository(),
			),
		},
		{
			// The admin user can access a private repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetDoerAdmin(true).
				SetRepository().
				SetRepositoryPrivate(true),
			),
		},
		{
			// The unprivileged authenticated user is denied
			// access to a private repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository().
				SetRepositoryPrivate(true),
			),
			error: "Not Found",
		},
		{
			// An anonymous visitor is denied
			// access to a private repository
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true).
				SetRepository().
				SetRepositoryPrivate(true),
			),
			error: "Not Found",
		},
		{
			// The Forgejo Actions user token can access the repository
			// because it is bound to it
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().SetDoerName(user_model.ActionsUserName).
				SetDoerActions(true).
				SetRepository(),
			),
		},
		{
			// The Forgejo Actions user token cannot access the repository
			// although it is publicly readable
			// because it is bound to a different repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().SetDoerName(user_model.ActionsUserName).
				SetDoerActions(true).
				SetDoerActionsRepoID(111111111111).
				SetRepository(),
			),
			error: "Not Found",
		},
		{
			// The Forgejo Actions user token can access the repository
			// because it is bound to it even when it was created from a
			// forked pull request event
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().SetDoerName(user_model.ActionsUserName).
				SetDoerActions(true).
				SetDoerActionsIsForkPullRequest(true).
				SetRepository(),
			),
=======
			data: newTestData(map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":       "anonymous",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":       "doeradmin",
				"repository": "userowner/repositoryprivate",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositoryprivate",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"doer":       "anonymous",
				"repository": "userowner/repositoryprivate",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"doer":       user_model.ActionsUserName,
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":        user_model.ActionsUserName,
				"repository":  "userowner/repositorypublic",
				"task.RepoID": "unrelated",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"doer":                   user_model.ActionsUserName,
				"repository":             "userowner/repositorypublic",
				"task.IsForkPullRequest": "true",
			}),
>>>>>>> upstream/v16.0/forgejo
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
<<<<<<< HEAD
		data.shared.SetRepositoryDefault()
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureSetRepository(t, permissions, data.shared.RepositoryName(), data.shared.RepositoryInit(), data.shared.RepositoryPrivate(), data.shared.RepositoryArchived())
=======
		data.SetDefault("repository", "userowner/repositorypublic")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureSetRepository(t, permissions, data)
>>>>>>> upstream/v16.0/forgejo
	},
})
