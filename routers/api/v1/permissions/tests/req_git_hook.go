// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	"forgejo.org/modules/setting"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqGitHook, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass because the admin user can edit git hooks
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetDoerAdmin(true),
			),
		},
		{
			// fail because even the admin user is denied the right to edit
			// git hooks when it is disabled in the settings
			data: newTestData(map[string]string{
				"DisableGitHooks": "true",
			}, newSharedData().
				SetDoer().
				SetDoerAdmin(true),
			),
=======
			data: newTestData(map[string]string{
				"doer": "doeradmin",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":            "doeradmin",
				"DisableGitHooks": "true",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "must be allowed to edit Git hooks",
		},
	},
	protectSettingsBool: []*bool{
		&setting.DisableGitHooks,
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
<<<<<<< HEAD
		data.shared.SetDoerDefault()
=======
		data.SetDefault("doer", "doeradmin")
>>>>>>> upstream/v16.0/forgejo
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		setting.DisableGitHooks = data.Get("DisableGitHooks") == "true"
	},
})
