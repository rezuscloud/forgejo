// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.IndividualPermsChecker, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass if a public context user exists
			data: newTestData(map[string]string{
				"user": "IndividualPermsChecker",
			}, newSharedData()),
		},
		{
			// fail if a private context user exists
			data: newTestData(map[string]string{
				"user":           "IndividualPermsCheckerOne",
				"userVisibility": "private",
			}, newSharedData()),
			error: "Visit Project",
		},
		{
			// fail if a limited context user exists
			data: newTestData(map[string]string{
				"user":           "IndividualPermsCheckerTwo",
				"userVisibility": "limited",
			}, newSharedData().SetAnonymous(true)),
=======
			data: newTestData(map[string]string{
				"user": "IndividualPermsChecker",
			}),
		},
		{
			data: newTestData(map[string]string{
				"user": "IndividualPermsCheckerprivate",
			}),
			error: "Visit Project",
		},
		{
			data: newTestData(map[string]string{
				"doer": "anonymous",
				"user": "IndividualPermsCheckerlimited",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "Visit Project",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
<<<<<<< HEAD
		data.SetDefault("user", data.shared.DoerName())
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.Has("user") {
			name := data.Get("user")
			visibility := data.Get("userVisibility")
			fixtureCreateUser(t, &user_model.User{Name: name, Visibility: stringToVisibility(visibility)})
=======
		data.SetDefault("user", data.Get("doer"))
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.Has("user") && data.Get("user") != "anonymous" {
			name := data.Get("user")
			fixtureCreateUser(t, &user_model.User{Name: name})
>>>>>>> upstream/v16.0/forgejo
			permissions.SetUser(fixtureGetUser(t, name))
		}
	},
})
