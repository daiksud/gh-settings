---
type: Feature
title: Manage Copilot code review effort level
description: Declare the observable repository review effort choice without enabling or requesting reviews.
---

## Feature: Manage Review effort level

Ordinary `export` now follows the [automatic readable-settings contract](readable-settings-export.feature.md): observe all supported families without selectors, retain unavailable declarations with info, and keep plan/apply strict.

### Rule: Select effort independently

- `repository.settings.copilot.code_review.review_effort_level` accepts only `default`, `lite`, `balanced` and `max`. Null, non-string, empty, API spelling `inherit` and unknown values fail local validation.
- Omission and empty page mappings leave effort unmanaged. The custom-instructions and [MCP](copilot-mcp.feature.md) toggles are independently selectable. One declared effort counts as one managed scalar, even when matching.
- Ordinary `export` independently observes the effort choice; unavailable observations retain prior intent with info.

### Rule: Compare observable configuration honestly

- A selected read acquires configured effort, effective effort and available choices with complete type validation. Configured and effective efforts accept Lite, Balanced and Max. Unknown configured/effective efforts fail; an availability list can include future choices without adopting them.
- Configured `lite`, `balanced` or `max` means an explicit override. It is distinct from Default even when effective effort is the same.
- Configured null maps to observable `default` for comparison and export. The API also uses null for an unavailable stored override, so neither comparison nor export asserts the underlying saved override was erased.
- A desired explicit effort must be currently available. Invalid or incomplete observations never become a default or a partial successful report. An unavailable effort observation during automatic export retains valid prior effort intent with info.

### Rule: Apply only declared effort

- Plan never writes. Apply reads fresh state and availability, skips observable matches, and PATCHes only `review_effort` when needed. Default sends `inherit`; Lite, Balanced and Max send their literal strings.
- A desired Default with configured null is a no-op, not a forced reset of a potentially hidden saved override. A successful update requires an independent GET confirming the requested observable choice.
- The effort PATCH does not include custom instructions, MCP or severity filtering. Their values are type-checked and must remain unchanged in readback; a mismatch reports uncertain effects without attributing concurrent changes to the write. Omitted effort sends no effort update even when custom instructions is declared.
- Read failures stop before the effort write. Mutation refusals and readback failures retain honest unknown-effect reporting; earlier successful writes remain and later operations stop. No retry, rollback, concurrency lock or saved-plan application is provided.
- Detailed CLI/Action output uses the fixed UI path and closed string choices. Same-version Action report reconstruction rejects invalid paths/types/choices, duplicates and no-op changes; status-only output hides the control and values.

### UI and API evidence

On November 10, 2026, the personally owned public test repository under GitHub Free exposed Default (Balanced), Lite, Balanced and Max Preview. Its No access banner expressly permitted storing configuration for people with review access. Changing Default to Lite in the UI produced configured/effective Lite in an independent GET. An effort-only `inherit` PATCH followed by independent GET restored every captured baseline field, and reloading the UI showed Default (Balanced). No review or workflow was requested; the test repository had no open PRs or workflows.

This demonstrates configurable storage and restoration, not review entitlement, actual review execution, instruction consumption, billing use or effective GitHub App permission grants. The feature does not enable reviews, grant access, change licensing, request a review, launch workflows or edit instruction files.

The official [configuration guide](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review), [effort-level documentation](https://docs.github.com/en/copilot/concepts/agent-capabilities/agent-powered-workflows-and-operations/code-review) and [OpenAPI contract](https://github.com/github/rest-api-description/blob/fa8d7bfb041cc9ef90ac51dcb8ed8cc8a0007166/descriptions/api.github.com/api.github.com.json) establish the control and GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings` path. Current source supports Max; Preview status does not exclude it. The [repository API](https://docs.github.com/en/rest/copilot/copilot-code-review) and [API announcement](https://github.blog/changelog/2026-11-06-copilot-code-review-api-management-and-review-overview-updates/), rechecked on November 13, 2026, document accepting currently available efforts and GitHub App/fine-grained Administration read/write grants. The schema explicitly gives both meanings of configured null. Repository administrator authority and classic `repo` scope are documented. Actual token grants remain caller-owned; the owner-authenticated storage probe did not test an App installation or Max execution.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Observable Default versus explicit override, supported choices, availability and managed counts | [Domain tests](../../internal/settings/copilot_review_effort_test.go) |
| Strict desired YAML, omissions, independent selection and round trip | [Codec tests](../../internal/workspace/ui_copilot_review_effort_test.go) |
| Complete acquisition, fresh availability, exact payload and independent verification | [Transport tests](../../internal/github/copilot_review_effort_test.go) |
| Selected plan/export, apply progress and omitted-resource isolation | [Application tests](../../internal/app/copilot_review_effort_test.go) |
| Automatic export, retired-selector rejection and help | [CLI tests](../../internal/cli/export_test.go) |
| Closed string reports, failure path and status-only privacy | [Action tests](../../internal/actions/copilot_review_effort_test.go) |
| Built executable plan/apply/export, Action reporting and file preservation | [Process acceptance](../../tests/acceptance/copilot_review_effort_test.go) |

Tests use isolated files and local API fixtures with dummy credentials. They do not establish live review execution or App-token grants.
