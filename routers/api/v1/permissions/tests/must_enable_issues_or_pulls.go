// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

<<<<<<< HEAD
	unit_model "forgejo.org/models/unit"
=======
>>>>>>> upstream/v16.0/forgejo
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.MustEnableIssuesOrPulls, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass if a repository with pull requests and issues unit set is present
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository(),
			),
		},
		{
			// fail if a repository is present and both pull requests and issues unit are disabled
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository().
				SetRepositoryDisabledUnits([]unit_model.Type{unit_model.TypeIssues, unit_model.TypePullRequests}).
				SetRepositoryInit(true),
			),
=======
			data: newTestData(map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":            "doerregular",
				"repository":      "userowner/repositorypublic",
				"repository-init": "true",
				"disable-units":   "repo.pulls,repo.issues",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "Not Found",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
<<<<<<< HEAD
		data.shared.SetRepositoryInit(true)
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureDisableUnits(t, permissions, data.shared.RepositoryDisabledUnits())
=======
		data.Set("repository-init", "true")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureDisableUnits(t, permissions, data)
>>>>>>> upstream/v16.0/forgejo
	},
})
