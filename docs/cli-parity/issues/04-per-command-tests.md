# 04 — Per-command lifecycle tests

**Status:** in progress — `issue` group done as the template
**Depends on:** 01, 02
**Blocks:** —

## Goal

One test file per command group, exercising the full user-facing flow
(create → read → mutate → verify → cleanup) against the containerized
instance. Each group's test matrix is derived from the Rust `forgejo-cli`
`enum` surface so parity is proven, not asserted.

## Methodology (mirrors `forgejo-api` Rust suite)

- **Lifecycle**, not one-shot smoke.
- **Serial** (`-p 1`); Forgejo has eventual-consistency surfaces (the Rust
  `repo.rs` test sleeps 3s before creating a PR).
- A **unique repo per test** (`uniqueRepoName`, already in the helpers).
- Local git repos for clone/PR/checkout flows (port `forgejo-api`'s `Git`
  helper to Go).
- Assert **human-readable output** (the UX layer's value-add) AND round-trip
  via the API to confirm state.

## Rollout order (smallest lifecycle first)

1. `parity/issue_test.go` — template (create/view/list/comment/close/search).
2. `parity/tag_test.go` — create/list/view/delete.
3. `parity/release_test.go` — create/list/view/asset.
4. `parity/repo_test.go` — create/fork/view/clone.
5. `parity/pr_test.go` — create/view/list/merge/checkout (needs local git).
6. `parity/auth_test.go` — login/logout/add-key/list.
7. `parity/user_test.go`, `parity/org_test.go`, `parity/wiki_test.go`.
8. `parity/actions_test.go` — the fork's flagship feature (jobs/logs).

## Rust surface to cover (per group)

- **repo**: create, fork, migrate, view, readme, clone, star/unstar, browse,
  labels{view,create,edit,delete}, units, issues, prs, actions, packages,
  projects, releases
- **issue**: create, edit, comment, assign/unassign, close, search, view,
  templates, browse
- **pr**: search, create, view, status, checkout, comment, assign/unassign,
  edit, close, merge, browse, review, labels, comments, files, commits
- **user**: search, view, browse, follow/unfollow, following/followers,
  block/unblock, repos, orgs, activity, edit, key{…}, gpg{…}
- **org**: list, view, create, edit, activity, members, visibility,
  label{…}, repo{…}, team{…}
- **release**: create, edit, delete, list, view, browse,
  asset{list,create,delete,get,download}
- **auth**: login, logout, add-key, use-ssh, list
- **actions**: tasks, variables, secrets, dispatch, jobs, logs
- **tag**: create, delete, list, view
- **wiki**: contents, view, clone, browse

## Acceptance criteria

- [x] One file per group (`issue_test.go` complete); all green against a
      fresh container.
- [x] Every group's matrix tracked in its file header.
- [ ] Commands absent from the Go `fj` are listed in the parity inventory's
      gap report (they fail until implemented). The 3 initial gaps
      (`issue search`, `repo fork`, `user key add`) are now implemented →
      `TestParityInventory` is green.

## Done

- `issue_test.go` — full lifecycle (create/view/list/comment/close + search), with API round-trip verification of comments. Template for the rest.
- `tag_test.go` — create/list/delete.
- `release_test.go` — create/view/list/delete.
- `repo_test.go` — view/clone/fork.
- `auth_test.go` — isolated `auth add-key`/`list`/`logout` against a temp XDG data dir.
- `user_test.go` — `user key add`/`list`.
- `wiki_test.go` — fixture-create page via API, then `wiki list`/`view`.
- `actions_test.go` — variables/secrets create/list/delete + empty runs/tasks states.
