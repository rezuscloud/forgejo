# 06 — llm-wiki documentation

**Status:** planned
**Depends on:** 01, 02, 05
**Blocks:** —

## Goal

Reflect the parity suite in the `rezuscloud/llm-wiki` instance so the
architecture is documented from the deterministic graph, not from memory.

## Changes

- Update `wiki/entities/forgejo-cli.md`: note the parity contract
  (`spec/swagger.json` → `operationId`), the separate containerized workflow,
  and that parity is machine-checked (not asserted).
- Update `wiki/entities/forgejo.md`: reference the separate `.github/workflows/cli-parity.yml`
  (distinct from the tag-triggered `release.yml`).
- Update `wiki/concepts/fork-maintenance.md` `Post-merge hooks` table: the
  `forgejo.sh` hook now also regenerates the parity map (issue #03) and the
  centralized `validate-fork.sh` runs this suite in addition to the existing
  build/codegen/integration checks.
- `index.md` row updates (updated date, summary).

## Acceptance criteria

- [ ] `[[forgejo-cli]]` documents the parity contract + workflow.
- [ ] `[[forgejo]]` references the separate parity workflow.
- [ ] `[[fork-maintenance]]` hook table updated.
- [ ] `index.md` dates bumped.

This issue is tracked in the `llm-wiki` repo, not here; it is listed for
cross-reference only.
