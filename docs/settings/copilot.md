---
type: Reference
title: "Copilot code review settings"
description: "Copilot code review settings declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Copilot code review effort level

`repository.settings.copilot.code_review.review_effort_level` corresponds to **Copilot → Code review → Review effort level**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        review_effort_level: lite
```

The optional string accepts `default`, `lite`, `balanced` and `max`. Omission leaves effort unmanaged, independently of `use_custom_instructions`. Null, empty strings, API spelling `inherit` and unknown choices are invalid declarations. Max is available only when the repository reports it as an available effort. This control does not enable reviews, request a review, grant access, change licensing or edit instruction files. Stored configuration does not establish review entitlement or actual review execution.

GitHub reports configured effort, effective effort and available efforts separately. A configured `lite`, `balanced` or `max` remains an explicit override, even when its value equals the inherited effective effort. `default` compares against configured null, not against effective `balanced`. GitHub can also return configured null for a stored override that is no longer available. Therefore export and comparison use `default` only as the **observable** default state: it does not prove that an underlying override was erased. A matching configured-null state produces no difference and apply skips it; applying this declaration is not a forced reset of a hidden override.

Plan is read-only. A selected read requires complete, correctly typed configured/effective effort and availability. Unknown configured or effective choices fail rather than being converted to a supported choice. An explicit `lite`, `balanced` or `max` must be available in the acquired state; it is not silently replaced by another choice.

Apply reads fresh state and availability, skips an observable match, and PATCHes only `review_effort` when a change is needed. `default` sends `inherit`; the other choices send their literal API strings. An independent GET must confirm the requested observable state. Sibling controls, including custom instructions, MCP and severity filtering, are not included in this effort PATCH. Their complete typed values must remain unchanged in readback; a mismatch fails verification without claiming whether the cause was this write or a concurrent change. Mutation or readback failures can leave unknown effects; earlier successful writes remain applied and later work stops. There is no rollback, automatic retry, concurrency lock or saved-plan application.

Ordinary `export` includes the observable effort choice. Unavailable observations retain prior declarations with info. Detailed CLI/Action reports show the fixed UI path and string choices; status-only output hides them.

The [repository settings API](https://docs.github.com/en/rest/copilot/copilot-code-review#get-copilot-code-review-settings-for-a-repository) documents GitHub App and fine-grained token support with Administration read for acquisition and Administration write for updates. These documented grants do not certify a particular installation or review entitlement. The other [credential boundaries](#copilot-code-review-custom-instructions) apply. GitHub documents the [configuration control](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review), [effort choices and Preview status](https://docs.github.com/en/copilot/concepts/agent-capabilities/agent-powered-workflows-and-operations/code-review) and [GET/PATCH schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json). The [behavior specification](../features/copilot-review-effort.feature.md) records test ownership and the limits of the live storage evidence.

## Copilot code review custom instructions

`repository.settings.copilot.code_review.use_custom_instructions` corresponds to **Copilot → Code review → Use custom instructions**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        use_custom_instructions: false
```

This is one optional boolean. Omission, `copilot: {}` and `code_review: {}` are unmanaged; explicit false is managed. Null and non-booleans are invalid. The [review effort level](#copilot-code-review-effort-level) is independently selectable. The [MCP toggle](#copilot-code-review-mcp-tools), approvals, licensing, access and automatic-review configuration remain separate. No instruction file is read, created or edited.

The toggle changes whether future eligible Copilot reviews use repository instruction files. It does **not** enable Copilot code review, grant access, request a review or launch a workflow. Organization instructions apply independently. In the verified Free/personal/public UI, the No access banner expressly allowed storing settings for people with review access, and the toggle was enabled. Storing configuration is distinct from review entitlement and instruction consumption.

Plan requires a complete acquired boolean and never writes. Apply reads fresh state, skips matches, PATCHes only `custom_instructions_enabled`, then requires matching independent GET readback. HTTP 404 is an error, not false; denied, incomplete, null or mistyped reads fail. Mutation/readback failures may have changed stored configuration; earlier successful writes remain and later operations stop. No rollback, automatic retry, concurrency lock or saved-plan application is provided.

Ordinary `export` independently includes the stored custom-instructions boolean, including false. Unavailable observations retain prior declarations with info. Detailed CLI/Action reports show the UI path and native booleans; status-only output omits them.

The official [Copilot code review API schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json) documents repository administrators, classic `repo` scope and GitHub App support for GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings`. Exact fine-grained/App-token grants were not independently established by the owner-authenticated probe; callers must authorize this endpoint with their chosen credential. No grant is added automatically. The [behavior specification](../features/copilot-custom-instructions.feature.md) records the verified storage/restoration and its limits.

## Copilot code review MCP tools

`repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests` corresponds to **Copilot → Code review → Allow Copilot to use MCP tools when reviewing pull requests**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false
```

This is one optional boolean. Omission, `copilot: {}` and `code_review: {}` leave it unmanaged; explicit false is managed. Null and non-booleans are invalid. Custom instructions and review effort remain independently selectable.

The toggle stores whether future eligible Copilot reviews may use MCP tools. It does not configure MCP servers, create or edit MCP configuration or secrets, grant permissions, enable or request reviews, launch workflows, manage billing or establish entitlement. The personally owned public repository under GitHub Free exposed a configurable toggle even with a No access banner. Configurable storage does not establish actual MCP use or review access.

Plan requires a complete typed settings read and never writes. Apply reads fresh settings, skips a matching boolean and PATCHes only `mcp_enabled` when needed. An independent GET must confirm the requested boolean and preservation of all acquired sibling settings, including custom instructions, configured/effective review effort, available effort choices and severity filtering. Missing, null or non-boolean MCP values, incomplete or malformed sibling settings, denied reads and HTTP 404 are errors rather than false. A configured-null review effort remains a valid sibling value and is preserved as observed. A preservation mismatch fails verification without attributing a concurrent change to this write. Preflight is not a concurrency lock. Mutation/readback failures may leave unknown effects; earlier successful writes remain and later operations stop. There is no automatic retry, rollback or saved-plan application.

Ordinary `export` independently includes the stored MCP boolean, including false. Unavailable observations retain prior declarations with info. Detailed CLI/Action reports use the fixed UI path and native booleans; status-only output hides the control and values.

The official [Copilot code review API schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json) defines GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings` and `mcp_enabled`. Repository administrators and classic `repo` scope are documented, and GitHub App tokens are supported; exact fine-grained/App-token grants were not independently established. The credential must already authorize this endpoint; gh-settings does not expand grants. See GitHub's [code-review configuration guide](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review) and the [behavior specification](../features/copilot-mcp.feature.md).
