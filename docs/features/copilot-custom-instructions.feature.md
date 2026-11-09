---
type: Feature
title: Manage Copilot code review custom instructions
description: Declare the repository Use custom instructions toggle without enabling reviews or editing instruction files.
---

## Feature: Manage the Use custom instructions setting

### Rule: Manage only an explicitly declared boolean

- `repository.settings.copilot.code_review.use_custom_instructions` is an optional boolean. Omission and empty page mappings are unmanaged; explicit true and false remain distinct. Null, non-booleans and unadopted sibling controls are invalid.
- A complete acquired boolean observation is required for comparison of a declared setting. An unread setting must not become false. One declared toggle counts as one managed value, whether matching or different.
- Only the Use custom instructions toggle is supported. Instruction files are neither read nor created/edited; review effort, MCP, approvals, licensing, access and automatic-review rules remain unmanaged.

### Rule: Preserve configuration without activating reviews

- Plan reads only when declared and never writes. Apply reads fresh state, skips a matching value, PATCHes only `custom_instructions_enabled`, and requires matching independent GET readback.
- Denied, missing, malformed and non-boolean reads fail. Mutation/verification failures can have remote effects; earlier successful writes remain applied and later writes stop. There is no rollback, automatic retry or saved-plan apply.
- Ordinary export makes no Copilot code-review settings request. `--copilot-custom-instructions` selects only the supported boolean and saves only after successful acquisition.
- This toggle controls future eligible reviewers' use of repository instruction files. It does not enable Copilot code review, grant access, change licensing, request reviews or launch workflows. Organization instructions apply independently.

### UI and API evidence

The personally owned public test repository showed an enabled toggle under GitHub Free. Its No access banner explicitly said settings still apply to people with review access. UI true-to-false was confirmed by GET200; PATCH200 restored true; independent GET matched every baseline field and reloaded UI returned to On with the same access banner. PR, workflow and Ruleset inventories remained unchanged. No instruction file or review was created.

The official [Copilot code review OpenAPI contract](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json) defines GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings`, permits dormant configuration and preserves omitted settings. GitHub's [custom instructions guide](https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/add-custom-instructions/add-repository-instructions#enabling-or-disabling-custom-instructions-for-copilot-code-review) describes the toggle separately from file authoring. Availability of the review feature is not inferred from configuration persistence.

Repository administrators and classic tokens with `repo` scope are documented. OpenAPI marks GitHub App support; exact App-token grants and review consumption were not independently verified. Credentials remain caller-owned; the implementation does not expand grants or enable prerequisites.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Omission, acquired true/false, missing observation, closed changes and managed counts | [Domain tests](../../internal/settings/copilot_instructions_test.go) |
| Strict boolean YAML, null/unsupported controls, empty pages and round trip | [Codec tests](../../internal/workspace/ui_copilot_instructions_test.go) |
| Complete GET, exact PATCH, matching skip, errors and independent verification | [Transport tests](../../internal/github/copilot_instructions_test.go) |
| Plan/apply progress, archive accounting, selected export and omitted-resource isolation | [Application tests](../../internal/app/copilot_instructions_test.go) |
| Explicit composable export flag and local duplicate rejection | [CLI tests](../../internal/cli/copilot_instructions_test.go) |
| Export false and retain existing Actions projection | [Projection tests](../../internal/github/ui_copilot_instructions_export_test.go) |
| Typed reports, fixed failure path, malformed-report rejection and status-only privacy | [Action tests](../../internal/actions/copilot_instructions_test.go) |
| Built executable, constructor version/redirect policy, both writes, failures and export preservation | [Process acceptance](../../tests/acceptance/copilot_instructions_test.go) |

Tests use dummy credentials, isolated Git/files and local API fixtures. They do not enable reviews, exercise instruction files or establish real App-token permissions. Live UI/API evidence confirms configuration only.
