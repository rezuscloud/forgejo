// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	unit_model "forgejo.org/models/unit"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTestBuilder([]string{"ReqOwner ", "ReqOwner"}, func(t *testing.T, signatureString string, signature []any) {
	t.Helper()
	unitTypes := signature[1].([]unit_model.Type)
	fixtures := []*testCase{
		{
<<<<<<< HEAD
			// pass because the doer owns the repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().SetDoerName("userowner").
				SetDoerScope("read:user,write:repository").
				SetRepository().SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			// pass because the doer does not own the repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetDoerScope("read:user,write:repository").
				SetRepository(),
			),
			error: "user should be the owner of the repo",
		},
		{
			// pass because the doer is admin even if it does not own the repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer().
				SetDoerAdmin(true).
				SetRepository(),
			),
		},
	}
	for _, unitType := range unitTypes {
		fixtures = append(fixtures, &testCase{
			data: newTestData(map[string]string{}, newSharedData().
				SetRepositoryDisabledUnits([]unit_model.Type{unitType})),
=======
			data: newTestData(map[string]string{
				"doer":       "userowner",
				"repository": "userowner/repositorypublic",
				"scope":      "read:user,write:repository",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":       "regular",
				"repository": "userowner/repositorypublic",
				"scope":      "read:user,write:repository",
			}),
			error: "user should be the owner of the repo",
		},
	}
	for _, unitType := range unitTypes {
		unit := unitsTypeToString(unitType)
		fixtures = append(fixtures, &testCase{
			data: newTestData(map[string]string{
				"disable-units": unit,
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "Not Found",
		})
	}
	signatureStringToFunctionTest[signatureString] = functionTest{
		sequenceFilter: []string{
			"APIAuthorization",
			"RepoAccess",
			"ReqOwner",
		},
		interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
<<<<<<< HEAD
			fixtureDisableUnits(t, permissions, data.shared.RepositoryDisabledUnits())
		},
		fulfillNeeds: func(t *testing.T, data *testData) {
			t.Helper()
			data.shared.SetDoer()
			data.shared.SetDoerAdmin(true)
=======
			fixtureDisableUnits(t, permissions, data)
		},
		fulfillNeeds: func(t *testing.T, data *testData) {
			t.Helper()
			data.Set("doer", "doeradmin")
>>>>>>> upstream/v16.0/forgejo
		},
		testCases:  fixtures,
		staticArgs: 1,
		call: func(t *testing.T, ctx apiv1_permissions.Context, _ *testData, args []any) {
			unitTypes := args[0].([]unit_model.Type)
			t.Logf("calling ReqOwner(ctx, %+v)", unitTypes)
			apiv1_permissions.ReqOwner(ctx, unitTypes)
		},
	}
})
