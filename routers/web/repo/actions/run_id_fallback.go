// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/modules/util"
	app_context "forgejo.org/services/context"
)

// RunIDFallback accepts the global run ID where the /actions/runs/{run}
// route expects the per-repo run index. Run links and commit-status
// target URLs carry the index, but the API and the database expose the
// ID — numbers pasted from those surfaces 404 otherwise, which reads as
// "this run does not exist" for a run that plainly does (#132, #136).
//
// Mounted on the /runs/{run} route group, so every sub-route (view, logs,
// artifacts, rerun, cancel, ...) inherits the fallback from one place.
// The redirect is 307 (method- and body-preserving): POST routes rerun
// identically against the canonical URL. A number that is neither this
// repo's index nor a repo-scoped ID is left to the handler's 404.
func RunIDFallback(ctx *app_context.Context) {
	n := ctx.ParamsInt64("run")
	if n <= 0 {
		return
	}
	_, err := actions_model.GetRunByIndex(ctx, ctx.Repo.Repository.ID, n)
	if err == nil || !errors.Is(err, util.ErrNotExist) {
		return // canonical index (or a server error the handler will report)
	}
	run, err := actions_model.GetRunByID(ctx, n)
	if err != nil || run.RepoID != ctx.Repo.Repository.ID {
		return // not a repo-scoped ID either — the handler 404s as before
	}
	idPrefix := ctx.Repo.RepoLink + "/actions/runs/" + strconv.FormatInt(n, 10)
	canonical := ctx.Repo.RepoLink + "/actions/runs/" + strconv.FormatInt(run.Index, 10)
	if p := ctx.Req.URL.Path; strings.HasPrefix(p, idPrefix) {
		ctx.Redirect(strings.Replace(p, idPrefix, canonical, 1), http.StatusTemporaryRedirect)
	}
}
