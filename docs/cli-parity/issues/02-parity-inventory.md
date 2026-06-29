# 02 — Parity inventory (gap report)

**Status:** done
**Depends on:** 01 (only for the workflow; the inventory itself needs no instance)
**Blocks:** 03, 04

## Goal

A machine-checked definition of "`fj` parity with the Rust `forgejo-cli`" that,
when run, prints a **gap report**: which expected UX commands are missing from
`fj`, and which declared `operationId`s no longer exist in the swagger spec
(upstream drift). This is the single artifact that answers "do we have parity?"
with a concrete, failing-until-green list.

## Definition of parity

A mapping from every expected top-level `fj` UX command to the swagger
`operationId` it must call. Two checks:

1. **Spec drift** — every mapped `operationId` must exist in
   `client-go/spec/swagger.json`. If upstream renames/removes it, the test
   fails and names the affected `fj` command.
2. **CLI coverage** — every mapped command must appear in `fj`'s command tree
   (discovered by parsing `fj --help` recursively). Missing commands are listed
   as gaps to implement.

## Acceptance criteria

- [x] `parity_inventory_test.go` loads `operationId`s from `swagger.json`.
- [x] Builds the `fj` binary and discovers implemented commands via `--help`.
- [x] Reports spec drift (FAIL) and missing commands (FAIL with the gap list).
- [x] Runs without a live instance (pure spec + binary analysis) so it is a
      fast codegen gate usable in any CI context.
- [x] Consumes the GENERATED parity contract (`cmd.CommandToOperation` from
      issue #03) instead of a hand-maintained map.

## Result

`TestParityInventory` reports `PARITY OK: 17 expected commands mapped and
implemented, 506 operationIds in spec` — green after the three initial gaps
(`issue search`, `repo fork`, `user key add`) were implemented.

## Seed coverage (to expand per issue #04)

| Command | operationId |
|---------|-------------|
| `issue create` | `issueCreateIssue` |
| `issue view` | `issueGetIssue` |
| `issue list` | `issueListIssues` |
| `issue comment` | `issueCreateComment` |
| `issue close` | `issueEditIssue` |
| `issue search` | `issueSearchIssues` |
| `pr create` | `repoCreatePullRequest` |
| `pr view` | `repoGetPullRequest` |
| `pr list` | `repoListPullRequests` |
| `pr merge` | `repoMergePullRequest` |
| `release create` | `repoCreateRelease` |
| `release list` | `repoListReleases` |
| `tag create` | `repoCreateTag` |
| `tag list` | (repo git refs) |
| `repo create` | `createCurrentUserRepo` / `createOrgRepo` |
| `repo fork` | `createFork` |
| `user key add` | `userCurrentPostKey` |
| `whoami` | `userGetCurrent` |
| `version` | `getVersion` |

## Files

- `staging/src/forgejo.org/fj/tests/integration/parity_inventory_test.go`
