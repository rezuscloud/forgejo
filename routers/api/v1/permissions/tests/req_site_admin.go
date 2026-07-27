// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqSiteAdmin, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass because the doer is the admin
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetDoerAdmin(true),
			),
		},
		{
			// fail because the doer is not the admin
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(),
			),
=======
			data: newTestData(map[string]string{
				"doer": "doeradmin",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer": "regularuser",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "user should be the site admin",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
<<<<<<< HEAD
		if !data.shared.HasDoerName() {
			data.shared.SetDoer()
			data.shared.SetDoerAdmin(true)
		}
=======
		data.SetDefault("doer", "doeradmin")
>>>>>>> upstream/v16.0/forgejo
	},
})
