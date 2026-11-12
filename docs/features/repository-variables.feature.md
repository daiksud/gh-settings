---
type: Feature
title: Manage selected repository variables
description: Upsert named non-sensitive variables and explicitly prune present collections.
---

## Feature: Manage Actions and Agents repository variables

### Rule: Select named non-sensitive values

- `repository.settings.actions.variables` and `repository.settings.agents.variables` independently select optional arrays of required `name` and nonempty string `value` pairs. The same canonical name can have a different value in each family.
- Without prune, omission and [] select nothing and unlisted variables remain unchanged. Explicit variable prune owns retention for present lists; omission always stays unmanaged.
- Names contain ASCII letters, digits and underscores, start with a letter or underscore, and cannot start with `GITHUB_` (case insensitive). Identity is case insensitive; canonical names are uppercase.
- Missing/null/empty values fail locally. The Actions live probe rejected an empty value with HTTP 422; no whitespace-trimming restriction is inferred. Values are plaintext and must contain only non-sensitive information. No secret detection or masking is promised.
- This feature covers repository Actions and Agents variables. [Environment variables](environments.feature.md#selected-environment-variables) have separate scoped ownership; secrets and organization variables are excluded. There is no delete or rename declaration.

### Rule: Read selected names and verify writes

- Without prune, plan reads only declared names; explicit prune also acquires the complete selected inventory. Only HTTP 404 means absent; denied, malformed or incomplete reads fail.
- Apply reads fresh selected names, creates absent variables, updates changed values and skips equal values. It verifies each successful write with an individual GET before continuing.
- Failures stop later writes; earlier changes remain applied. A failed mutation or verification can have an unknown remote effect. There is no rollback, retry or concurrency lock.
- Complete acquired GET string values remain observations, including an empty string if returned. Export rejects an observed empty value before saving because it cannot produce a supported nonempty declaration.
- Ordinary export makes no variable requests. Repeatable `--variable NAME` selects Actions values and `--agent-variable NAME` selects Agents values. Both opt into individual reads of known non-sensitive variables. Selecting one family does not read the other. A missing selected export name fails without replacing output.
- Configuration, export and plan reports can contain the plaintext values; never use these interfaces for credentials or other secrets.

### Scenario: Keep the same name independent across families

- Given Actions variable `DISPLAY_NAME` is `workflow` and Agents variable `DISPLAY_NAME` is `agent`
- When only the Agents value is declared as `assistant`
- Then the plan identifies an Agents value change
- And apply changes only the Agents value and preserves the Actions value

### API and permissions

The [Actions Variables REST API](https://docs.github.com/en/rest/actions/variables) and [Agents Variables REST API](https://docs.github.com/en/rest/agents/variables) provide individual GET, POST creation and PATCH update at separate family paths. Fine-grained tokens and GitHub App tokens require repository **Variables** read/write for Actions or **Agent variables** read/write for Agents. Credentials are supplied by the caller; implementation does not grant access.

The Actions control was observed on GitHub Free, personally owned public `daiksud/test-gh-settings-public` on 2026-11-09. UI creation was read through the API, an API update appeared in the Variables tab, and the disposable variable was removed. This is adoption evidence, not a runtime entitlement check or proof of every credential's permissions.

The separate [Agents Variables REST API](https://docs.github.com/en/rest/agents/variables) supports the same selected-name lifecycle. On 2026-11-10 JST, a Free personally owned public test repository exposed the Agents Variables UI. API creation and update were independently read back and appeared in the UI; deletion restored its empty baseline. UI saving stopped at a reauthentication prompt, so successful UI saving is not claimed. This evidence does not establish actual agent consumption or permissions for an untested credential.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Required value, strict YAML, omission, empty-value rejection, ASCII identity and round trip | [Actions codec](../../internal/workspace/ui_variables_test.go), [Agents codec](../../internal/workspace/ui_agents_variables_test.go) |
| Selected GET, complete values, absence, exact write payloads and verification failures | [Transport tests](../../internal/github/variables_test.go) |
| Preserve Actions General and independent same-name family values during export projection | [Projection test](../../internal/github/ui_variables_export_test.go) |
| Read-only plan, selected export, sorted upsert progress and partial failure | [Actions application](../../internal/app/variables_test.go), [Agents application](../../internal/app/agents_variables_test.go) |
| Explicit named export options and invalid selections | [Actions CLI](../../internal/cli/variables_test.go), [Agents CLI](../../internal/cli/agents_variables_test.go) |
| Typed plaintext reports, escaped Summary and status-only privacy | [Actions reports](../../internal/actions/variables_test.go), [Agents reports](../../internal/actions/agents_variables_test.go) |
| Actual binary, local validation, family-specific Action report, exact API version, no redirects, create/update/skip, unlisted preservation and failed-export file preservation | [Process acceptance](../../tests/acceptance/variables_test.go) |

Automated tests use local fixtures and dummy credentials, without live repository changes. Live adoption evidence does not establish GitHub App credential permissions or workflow or agent consumption of a variable.

### Explicit variable pruning

Current source supports `actions-variables` and `agents-variables` as independent
[prune scopes](prune.feature.md). Present [] selects the complete acquired
collection for deletion; nonempty lists retain canonical declared names.
Pruning does not limit other declared work. Complete 30-item pagination, exact
stable total counts and all-candidate validation precede any remote write.
Fresh family reads must retain the prepared candidates. Upserts precede fixed
deletions; every DELETE checks acquired name/value and requires 204 plus complete
inventory absence. Drift, denied/incomplete reads and uncertain writes stop later
work while retaining earlier progress. Detailed reports show `delete` to `absent`;
status-only still validates without exposing names/values. No live operation is
part of these tests.

Additional checks: `internal/settings/variable_prune_test.go`,
`internal/github/variable_prune_test.go`, `internal/app/variable_prune_test.go`,
`internal/actions/variable_prune_test.go`, and the built-CLI lifecycle
`tests/acceptance/prune_variables_test.go` cover full acquisition, family/scope
isolation, preflight, identity/value races, partial failure, restart and reports.
