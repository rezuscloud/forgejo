// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqAnyRepoReader, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass if the repository is public
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetRepository(),
			),
		},
		{
			// pass if the repository is private and the doer is admin
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetDoerAdmin(true).
				SetRepository().
				SetRepositoryPrivate(true),
			),
=======
			data: newTestData(map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":       "doeradmin",
				"repository": "userowner/repositoryprivate",
			}),
>>>>>>> upstream/v16.0/forgejo
		},
		// This fixture is unreachable because this permissions function is always used after
		// a RepoAccess that enforces the same restriction for non admin users
		// {
<<<<<<< HEAD
		// 	data: newTestData(map[string]string{}, newSharedData().
		// 		SetDoer().
		// 		SetRepository().
		// 		SetRepositoryPrivate(true),
		// 	),
=======
		// 	data: newTestData(map[string]string{
		// 		"doer":       "doerregular",
		// 		"repository": "userowner/repositoryprivate",
		// 	}),
>>>>>>> upstream/v16.0/forgejo
		// 	error: "Denied",
		// },
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"RepoAccess",
		"ReqAnyRepoReader",
	},
},
)
