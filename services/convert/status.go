// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	git_model "forgejo.org/models/git"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
)

// ToCommitStatus converts git_model.CommitStatus to api.CommitStatus
func ToCommitStatus(ctx context.Context, status *git_model.CommitStatus) (*api.CommitStatus, error) {
	statusURL, err := status.APIURL(ctx)
	if err != nil {
		return nil, err
	}
	apiStatus := &api.CommitStatus{
		Created:     status.CreatedUnix.AsTime(),
		Updated:     status.CreatedUnix.AsTime(),
		State:       status.State,
		TargetURL:   status.TargetURL,
		Description: status.Description,
		ID:          status.Index,
		URL:         statusURL,
		Context:     status.Context,
	}

	// A missing creator degrades to an omitted field (ToUser(nil) is nil);
	// unlike the URL above, this cannot panic and carries no routing data.
	if status.CreatorID != 0 {
		creator, _ := user_model.GetUserByID(ctx, status.CreatorID)
		apiStatus.Creator = ToUser(ctx, creator, nil)
	}

	return apiStatus, nil
}

// ToCombinedStatus converts List of CommitStatus to a CombinedStatus
func ToCombinedStatus(ctx context.Context, statuses []*git_model.CommitStatus, repo *api.Repository) (*api.CombinedStatus, error) {
	if len(statuses) == 0 {
		return nil, nil
	}

	retStatus := &api.CombinedStatus{
		SHA:        statuses[0].SHA,
		TotalCount: len(statuses),
		Repository: repo,
		URL:        "",
	}

	retStatus.Statuses = make([]*api.CommitStatus, 0, len(statuses))
	for _, status := range statuses {
		apiStatus, err := ToCommitStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		retStatus.Statuses = append(retStatus.Statuses, apiStatus)
		if retStatus.State == "" || status.State.NoBetterThan(retStatus.State) {
			retStatus.State = status.State
		}
	}
	// According to https://docs.github.com/en/rest/commits/statuses?apiVersion=2022-11-28#get-the-combined-status-for-a-specific-reference
	// > Additionally, a combined state is returned. The state is one of:
	// > failure if any of the contexts report as error or failure
	// > pending if there are no statuses or a context is pending
	// > success if the latest status for all contexts is success
	if retStatus.State.IsError() {
		retStatus.State = api.CommitStatusFailure
	}

	return retStatus, nil
}
