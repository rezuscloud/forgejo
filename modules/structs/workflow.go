// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package structs

// DispatchWorkflowOption options when dispatching a workflow
// swagger:model
type DispatchWorkflowOption struct {
	// Git reference for the workflow
	//
	// required: true
	Ref string `json:"ref"`
	// Input keys and values configured in the workflow file.
	Inputs map[string]string `json:"inputs"`
	// Flag to return the run info
	// default: false
	ReturnRunInfo bool `json:"return_run_info"`
}

// DispatchWorkflowRun represents a workflow run
// swagger:model
type DispatchWorkflowRun struct {
	// the workflow run id
	ID int64 `json:"id"`
	// a unique number for each run of a repository
	RunNumber int64 `json:"run_number"`
	// the jobs name
	Jobs []string `json:"jobs"`
}
// ActionWorkflow represents a workflow file discoverable in a repository
// swagger:model
type ActionWorkflow struct {
	// the workflow file name — the key DispatchWorkflow takes
	Filename string `json:"filename"`
	// the workflow's `name:` field (falls back to the filename)
	Name string `json:"name"`
	// the workflow file's path in the repository
	Path string `json:"path"`
	// workflows are always active in Forgejo (GitHub-compat field)
	State string `json:"state"`
	// the Actions web URL filtered to this workflow
	HTMLURL string `json:"html_url"`
}

// ListActionWorkflowsResponse is a paginated-shaped list of a repository's workflows
// swagger:model
type ListActionWorkflowsResponse struct {
	TotalCount int64             `json:"total_count"`
	Workflows  []*ActionWorkflow `json:"workflows"`
}
