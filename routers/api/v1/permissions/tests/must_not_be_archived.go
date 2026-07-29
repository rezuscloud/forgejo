// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.MustNotBeArchived, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass if a repository is present
			data: newTestData(map[string]string{}, newSharedData().
				SetRepository(),
			),
		},
		{
			// fail if a repository is present but is archived
			data: newTestData(map[string]string{}, newSharedData().
				SetRepository().
				SetRepositoryArchived(true),
			),
=======
			data: newTestData(map[string]string{
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"repository": "userowner/repositoryarchived",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "is archived",
		},
	},
})
