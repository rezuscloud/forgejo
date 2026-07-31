// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

<<<<<<< HEAD
=======
	user_model "forgejo.org/models/user"
>>>>>>> upstream/v16.0/forgejo
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.APIAuthorization, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(),
			),
=======
			data: newTestData(map[string]string{
				"doer": "anonymous",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer": "doerregular",
			}),
>>>>>>> upstream/v16.0/forgejo
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
<<<<<<< HEAD
		data.shared.SetDoerDefault()
		if data.shared.DoerActions() {
			data.shared.SetRepositoryDefault()
		}
		data.shared.SetDoerScopeDefault("read:repository")
		data.shared.SetTokenLevelDefault("read")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.shared.HasRepositoryName() && data.shared.DoerActions() {
			fixtureSetRepository(t, permissions, data.shared.RepositoryName(), data.shared.RepositoryInit(), data.shared.RepositoryPrivate(), data.shared.RepositoryArchived())
=======
		data.SetDefault("doer", "doerregular")
		if data.Get("doer") == user_model.ActionsUserName {
			data.SetDefault("repository", "userowner/repositorypublic")
		}
		data.SetDefault("scope", "read:repository")
		data.SetDefault("level", "read")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.Has("repository") && data.Get("doer") == user_model.ActionsUserName {
			fixtureSetRepository(t, permissions, data)
>>>>>>> upstream/v16.0/forgejo
		}
		fixtureSetDoer(t, permissions, data)
	},
})
