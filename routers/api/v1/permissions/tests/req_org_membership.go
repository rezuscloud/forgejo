// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTest(apiv1_permissions.ReqOrgMembership, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass because the org owner is a member of the org
			data: newTestData(map[string]string{
				"org":    "ReqOrgMembershipOrg",
				"setOrg": "true",
			}, newSharedData().
				SetDoer(),
			),
		},
		{
			// pass because the doer admin although it is not a member of
			// the org
			data: newTestData(map[string]string{
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"setOrg":   "true",
			}, newSharedData().
				SetDoer().
				SetDoerAdmin(true),
			),
		},
		{
			// fail because the doer is not a member of the org
			data: newTestData(map[string]string{
				"org":      "ReqOrgMembershipOrg",
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"setOrg":   "true",
			}, newSharedData().
				SetDoer(),
			),
			error: "Must be an organization member",
		},
		{
			// pass because the doer a member of the context team
			data: newTestData(map[string]string{
				"org":     "ReqOrgMembershipOrg",
				"setTeam": "true",
			}, newSharedData().
				SetDoer(),
			),
		},
		{
			// fail because the doer is not a member of the context team
			data: newTestData(map[string]string{
				"org":      "ReqOrgMembershipOrg",
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"setTeam":  "true",
			}, newSharedData().
				SetDoer(),
			),
			error: "Not Found",
		},
		{
			// fail because org is not set in the context
			data: newTestData(map[string]string{
				"setOrg": "true",
			}, newSharedData()),
=======
			data: newTestData(map[string]string{
				"org":    "ReqOrgMembershipOrg",
				"setOrg": "true",
			}),
		},
		{
			data: newTestData(map[string]string{
				"doer":   "doeradmin",
				"setOrg": "true",
			}),
		},
		{
			data: newTestData(map[string]string{
				"org":    "ReqOrgMembershipOrg",
				"doer":   "regularuser",
				"setOrg": "true",
			}),
		},
		{
			data: newTestData(map[string]string{
				"org":      "ReqOrgMembershipOrg",
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"doer":     "regularuser",
				"setOrg":   "true",
			}),
			error: "Must be an organization member",
		},
		{
			data: newTestData(map[string]string{
				"org":     "ReqOrgMembershipOrg",
				"doer":    "regularuser",
				"setTeam": "true",
			}),
		},
		{
			data: newTestData(map[string]string{
				"org":      "ReqOrgMembershipOrg",
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"doer":     "regularuser",
				"setTeam":  "true",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"setOrg": "true",
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "unprepared context",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"TokenRequiresScopes",
		"ReqOrgMembership",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetDefault("org", "ReqOrgMembership")
		data.SetDefault("setOrg", "true")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
<<<<<<< HEAD
		orgOwner := data.shared.DoerName()
=======
		orgOwner := data.Get("doer")
>>>>>>> upstream/v16.0/forgejo
		if data.Has("orgOwner") {
			orgOwner = data.Get("orgOwner")
		}
		var org *org_model.Organization
		if data.Has("org") {
			fixtureCreateUser(t, &user_model.User{Name: orgOwner})
			org = fixtureCreateOrg(t, &org_model.Organization{Name: data.Get("org")}, &user_model.User{Name: orgOwner})
		}

		if data.Get("setOrg") == "true" {
			permissions.SetOrganization(org)
		}

		if data.Get("setTeam") == "true" {
			team, err := org_model.GetTeam(t.Context(), org.ID, org_model.OwnerTeamName)
			require.NoError(t, err)
			permissions.SetTeam(team)
		}
	},
})
