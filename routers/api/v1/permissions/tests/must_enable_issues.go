// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

<<<<<<< HEAD
	"forgejo.org/models/unit"
=======
>>>>>>> upstream/v16.0/forgejo
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.MustEnableIssues, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass if a repository with issues unit set is present
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository(),
			),
		},
		{
			// fail if a repository is present but the issues unit is disabled
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository().
				SetRepositoryDisabledUnits([]unit.Type{unit.TypeIssues}),
			),
=======
			data: newTestData(map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":          "doerregular",
				"repository":    "userowner/repositorypublic",
				"disable-units": "repo.issues",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "Not Found",
		},
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
<<<<<<< HEAD
		fixtureDisableUnits(t, permissions, data.shared.RepositoryDisabledUnits())
=======
		fixtureDisableUnits(t, permissions, data)
>>>>>>> upstream/v16.0/forgejo
	},
})
