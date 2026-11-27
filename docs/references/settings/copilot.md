---
type: Reference
title: "Copilot code review settings"
description: "Copilot code review settings declaration, ownership and API contracts."
---

# Copilot code review settings

The **Copilot → Code review → General** page controls how eligible reviews run. Under
`repository.settings.copilot.code_review.general`, declare any combination of these
four fields. Each is independent. Omitting a field leaves its value unmanaged,
including when the entire mapping is empty.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        general:
          review_effort_level: lite
          allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false
          search_public_repositories: false
          use_custom_instructions: false
```

| Control | Accepted values |
| --- | --- |
| [Review effort level](#copilot-code-review-effort-level) | `default`, or a repository-available `lite`, `balanced` or `max`. |
| [Allow Copilot to use MCP tools](#copilot-code-review-mcp-tools) | Boolean. |
| [Search public repositories](#search-public-repositories) | Boolean, when available for the repository. |
| [Use custom instructions](#copilot-code-review-custom-instructions) | Boolean. |

A saved setting can apply to other users with review access even if the person
configuring it sees a **No access** banner. Saving a preference does not grant a
license, enable or request reviews, start a workflow, or establish actual use of
instructions or MCP tools. See [GitHub's code review configuration guide](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review).

## Copilot code review effort level

`review_effort_level` selects the configured effort independently of the
currently effective effort. For example, an explicit `balanced` remains an
override even when the inherited value is also `balanced`. The desired string
must be nonempty. Null, unknown choices and the API spelling `inherit` are
invalid in YAML.

GitHub reports the configured choice, effective choice and available efforts.
An explicit `lite`, `balanced` or `max` must be available in the observed
repository. Incomplete data or an unknown configured or effective choice fails
the read. GitHub can also report a null configured value for a stored override
that has become unavailable. Export represents configured null as `default`.
A declaration of `default` then matches what GitHub exposes, and apply leaves
the setting alone.

When a change is needed, apply sends only `review_effort`. For `default` it
sends `inherit`. Fresh readback must confirm the requested observable state and
preservation of the other settings, including custom instructions, MCP and
severity filtering. See [GitHub's effort choices](https://docs.github.com/en/copilot/concepts/agent-capabilities/agent-powered-workflows-and-operations/code-review)
and the [review effort specification](../../development/features/copilot-review-effort.feature.md).

## Copilot code review custom instructions

`use_custom_instructions` selects whether eligible reviews use repository
instruction files. Organization instructions apply independently. Apply sends
only `custom_instructions_enabled` when the selected boolean differs and
confirms the result with an independent GET. The CLI leaves the instruction
files themselves untouched. See the [custom instructions specification](../../development/features/copilot-custom-instructions.feature.md).

## Copilot code review MCP tools

`allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests` controls whether
eligible reviews may use MCP tools. MCP server configuration, secrets and access
grants have separate owners. When the value differs, apply sends only
`mcp_enabled`. Readback must confirm the selected boolean and all acquired
companions, including custom instructions, configured and effective effort,
available choices, and severity filtering. A configured-null effort remains
valid. Missing, malformed or changed companions fail verification. See the
[MCP tools specification](../../development/features/copilot-mcp.feature.md).

## Search public repositories

`search_public_repositories` selects whether eligible reviews may use public
repository search for context. GitHub accepts changes to
`cross_repo_context_enabled` when the setting is available for the repository.
A returned boolean supplies the current value, including false. A missing value
supplies no evidence. Export reports an unavailable setting and retains any
existing declaration. Plan and apply require a readable value when selected.

Apply sends just the changed boolean and verifies it with a fresh GET, including
the other observable code review settings. A rejected PATCH or failed readback
is an error. See the [public search specification](../../development/features/copilot-public-search.feature.md).

## Export and permissions

Export independently records each readable preference, including false. An
unavailable observation produces an info diagnostic and retains compatible
prior intent on re-export. For declared values, denied reads, HTTP 404 and
missing or mistyped data are errors. Detailed plans show the field's path and
value. A status-only Action summary hides them. If verification fails after a
write, inspect current state before retrying.

GitHub documents Administration read for acquisition and Administration write
for updates, with repository administrator access, GitHub App support and the
classic `repo` scope. The recorded owner-authenticated storage checks did not
verify every App installation or review entitlement. See the [permission matrix](permissions.md),
[repository settings API](https://docs.github.com/en/rest/copilot/copilot-code-review#get-copilot-code-review-settings-for-a-repository)
and [official REST schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json).
