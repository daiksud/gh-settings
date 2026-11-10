---
type: Feature
title: Manage Copilot code review MCP tools
description: Declare the stored MCP tools toggle without configuring servers or enabling reviews.
---

## Feature: Manage Allow Copilot to use MCP tools when reviewing pull requests

### Rule: Select one stored boolean independently

- `repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests` is an optional boolean. Omission and empty page mappings leave it unmanaged; true and false are explicit intent. Null and non-booleans fail validation before target or API work.
- One declared toggle counts as one managed scalar, including when unchanged. A complete typed acquisition is required; unread, missing or malformed state must not become false.
- Custom instructions and review effort remain independently selectable. `export --copilot-mcp` selects only the MCP boolean, including false, and composes with other Copilot and export options. Duplicate flags fail before target, API or file work.
- Without an explicit Copilot selection, ordinary export and `--actions` make no code-review settings request. Failed selected acquisition preserves the existing file.

### Rule: Apply narrowly and verify preservation

- Plan never writes. Apply reads fresh complete settings and skips a matching toggle. A changed toggle PATCHes only `mcp_enabled` at `/repos/{owner}/{repo}/copilot/code-review/settings`.
- Independent GET readback must confirm the requested boolean and unchanged acquired siblings: custom instructions, configured/effective review effort, available effort choices and severity filtering. The mutation response alone is not verification.
- Missing/null/non-boolean MCP values, incomplete or malformed sibling settings, denied reads and HTTP 404 fail. Configured-null review effort is a valid sibling state and remains distinct from a string override. A preflight failure prevents the MCP PATCH; a write/readback failure reports possible effects without rollback or automatic retry. Earlier successful writes remain and later work stops.
- A sibling mismatch fails verification without asserting that this PATCH caused it. Fresh preflight and readback are not a concurrency lock or saved-plan application.
- Detailed CLI/Action output uses the fixed UI path and native booleans. Same-version report reconstruction rejects invalid paths/types, equal values and duplicate controls; status-only output hides the control and values.

### Rule: Store configuration without configuring or executing MCP

- The toggle controls whether future eligible reviews may use MCP tools. It does not read, create or edit MCP server configuration, instruction files or secrets; add grants or licenses; enable or request reviews; launch workflows; execute MCP tools; or manage billing.
- UI configurability and successful stored readback do not establish review entitlement, actual MCP use, tool permissions or consumption. Credentials remain caller-owned.

### UI and API evidence

The personally owned public repository under GitHub Free exposed a configurable **Allow Copilot to use MCP tools when reviewing pull requests** toggle, including with a No access banner. This establishes the eligible configuration UI, not review access, MCP execution or billing behavior.

The official [configuration guide](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review) and [OpenAPI contract](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json) document GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings` and the `mcp_enabled` boolean. Repository administrators and classic `repo` scope are documented; GitHub App support is declared, but exact fine-grained/App-token grants were not independently established. No grant or entitlement probe is added.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Optional declaration, acquired observations, closed comparisons and managed counts | [Domain tests](../../internal/settings/copilot_mcp_test.go) |
| Strict boolean YAML, omissions, independent selection and round trip | [Codec tests](../../internal/workspace/ui_copilot_mcp_test.go) |
| Complete acquisition, fresh preflight, MCP-only payload and independent preservation verification | [Transport tests](../../internal/github/copilot_mcp_test.go) |
| Selected plan/export, direct apply progress and omitted-resource isolation | [Application tests](../../internal/app/copilot_mcp_test.go) |
| Composable export flag, duplicate rejection and help | [CLI tests](../../internal/cli/copilot_mcp_test.go) |
| Export false while retaining existing family projections | [Projection tests](../../internal/github/ui_copilot_mcp_export_test.go) |
| Typed boolean reports, fixed failure path and status-only privacy | [Action tests](../../internal/actions/copilot_mcp_test.go) |
| Built executable plan/apply/export and file preservation | [Process acceptance](../../tests/acceptance/copilot_mcp_test.go) |

Tests use isolated files and local API fixtures with dummy credentials. They do not establish real App-token grants, review execution or MCP tool use.
