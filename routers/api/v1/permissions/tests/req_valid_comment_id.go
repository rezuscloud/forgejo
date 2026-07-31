// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.ReqValidCommentID, functionTest{
	testCases: []*testCase{
		{
<<<<<<< HEAD
			// pass because the ID is for a comment that belongs to an issue that
			// belongs to the repository
			data: newTestData(map[string]string{
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",
			}, newSharedData().
				SetDoer().
				SetRepository(),
			),
=======
			data: newTestData(map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",
			}),
>>>>>>> upstream/v16.0/forgejo
		},
		// This fixture is unreachable because this permissions function is always used after
		// a RepoAccess that enforces the same restriction for non admin users
		// {
		// 	data: newTestData(map[string]string{
<<<<<<< HEAD
		// 		"issue":       "issueOne",
		// 		"issueAuthor": "issueAuthor",
		// 		"comment":     "comment for ReqValidCommentID",
		// 	}, newSharedData().
		// 		SetDoer().
		// 		SetRepository().
		// 		SetRepositoryPrivate(true),
		// 	),
		// 	error: "Not Found",
		// },
		{
			// fail because the comment pointer to the issue is nil, which
			// can happen when it fails to load
			data: newTestData(map[string]string{
=======
		// 		"doer":        "doerregular",
		// 		"repository":  "userowner/repositoryprivate",
		// 		"issue":       "issueOne",
		// 		"issueAuthor": "issueAuthor",
		// 		"comment":     "comment for ReqValidCommentID",
		// 	}),
		// 	error: "Not Found",
		// },
		{
			data: newTestData(map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
>>>>>>> upstream/v16.0/forgejo
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"NilIssue": "true",
<<<<<<< HEAD
			}, newSharedData().
				SetDoer().
				SetRepository(),
			),
			error: "Not Found",
		},
		{
			// fail because the issue associated with the comment belongs to
			// a repository that is different from the repository found in
			// the context
			data: newTestData(map[string]string{
=======
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
>>>>>>> upstream/v16.0/forgejo
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"InconsistentID": "true",
<<<<<<< HEAD
			}, newSharedData().
				SetDoer().
				SetRepository(),
			),
=======
			}),
>>>>>>> upstream/v16.0/forgejo
			error: "Not Found",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"RepoAccess",
		"ReqValidCommentID",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetDefault("issue", "issueOne")
		data.SetDefault("issueAuthor", "issueAuthor")
		data.SetDefault("comment", "comment for ReqValidCommentID")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
<<<<<<< HEAD
		issueAuthor := fixtureCreateUser(t, &user_model.User{Name: data.Get("issueAuthor")})
		issue := fixtureSetIssue(t, permissions, data.Get("issue"), issueAuthor.Name)
		fixtureCreateComment(t, permissions, issue, data.Get("comment"))
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		comment := fixtureGetComment(t, data.Get("comment"))
=======
		fixtureCreateUser(t, &user_model.User{Name: data.Get("issueAuthor")})
		fixtureSetIssue(t, permissions, data)
		fixtureCreateComment(t, permissions, data)
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		comment := fixtureGetComment(t, data)
>>>>>>> upstream/v16.0/forgejo
		if data.Has("NilIssue") {
			comment.Issue = nil
		}
		if data.Has("InconsistentID") {
			comment.Issue.RepoID = 123456
		}
		t.Logf("calling ReqValidCommentID(ctx, %+v)", comment)
		apiv1_permissions.ReqValidCommentID(ctx, comment)
	},
})
