---
type: Reference
title: "YAML configuration"
description: "YAML configuration declaration, ownership and API contracts."
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Supported fields

This section describes the accepted current v0 schema. Existing classic branch
protection keys remain valid; acceptance does not include that family in
[supported v1 scope](scope.md#feature-adoption-scope).

`repository.about`, `repository.labels` and `repository.settings` are siblings. Settings contains `general`, `rulesets`, `branch_protection_rules`, `actions.general`, `actions.policies`, `actions.variables`, `agents.variables`, `copilot.code_review`, `moderation.interaction_limits.pull_request_limits`, `advanced_security`, `pages` and `environments`. Unknown keys, including otherwise writable API fields, are rejected.

The basic structure is `repository.settings.<page>[.<subpage>][.<section>].<control>`. Navigation-only sidebar categories are generally omitted; Moderation is an explicit configuration group containing Interaction limits. Do not invent a section solely to equalize depth: page-level controls remain direct, and compound values or resource entries retain their own nesting. The Actions sections use `actions_permissions.policy` and `workflow_permissions.default_permissions`; Secret Protection uses `secret_protection.enabled` with `push_protection` beside it. There is no generic `selection` or `value` wrapper for controls (variable entries have their own required `value`), no second Actions enable flag, and no universal `enabled` wrapper around simple booleans. These are public declaration names, not claims about HTML form attributes.

Under `repository.about`, `description` and `website` are strings, including `''`; `topics` is a direct string array. These translate to repository REST fields and the Topics endpoint. Nullable observed repository strings become empty strings through the existing reader; YAML null does not request that behavior.[^github-update-repository][^github-repository-openapi]

The following paths are relative to `repository.settings.general`:

| Path | Type |
| --- | --- |
| `template_repository` | boolean |
| `default_branch` | literal branch-name string |
| `features.wikis`, `features.issues`, `features.sponsorships`, `features.discussions`, `features.projects`, `features.pull_requests` | boolean |
| `features.issue_permissions.creation_allowed_by`, `features.pull_request_permissions.creation_allowed_by` | `all_users` or `collaborators_only` |
| `pull_requests.allow_merge_commits`, `pull_requests.allow_squash_merging`, `pull_requests.allow_rebase_merging` | boolean |
| `pull_requests.always_suggest_updating_pull_request_branches`, `pull_requests.allow_auto_merge`, `pull_requests.automatically_delete_head_branches` | boolean |
| `pull_requests.merge_commit.default_commit_message`, `pull_requests.squash_merge.default_commit_message` | UI menu choice below |
| `commits.require_contributors_to_sign_off_on_web_based_commits` | boolean |
| `danger_zone.archive_this_repository` | boolean; automatic export |
| `releases.enable_release_immutability` | boolean; automatic export |

Omitting a scalar leaves it unmanaged. Explicit false and empty strings remain intent. REST-backed About/General controls use [fresh reconciliation](../commands.md#apply-behavior): skip exact matches and independently verify declared state after changed writes. An explicitly disabled Issues/Pull requests feature cannot accompany its creation-permission choice. Likewise, an explicitly disabled merge method cannot accompany its commit-message menu. Omitting the enabling checkbox never implicitly enables it. Export hides choices behind explicitly disabled controls after validating the acquired raw values.

## Configuration semantics

The version marker is independent of the CLI release. Only one YAML mapping document is accepted. `repository` is optional; version-only, `repository: {}` and empty settings mappings are local no-ops without origin/authentication/API requirements. A present collection entry must still satisfy its complete object rules. Empty ordinary section objects add no scalar intent; they do not enable a parent or fill a policy. A parameterized rule object within a selected Ruleset still selects that rule, and a name-only Environment selects creation only when missing.

Unknown/duplicate/non-string keys, unsupported tags, malformed/multiple documents, wrong shapes, invalid choices and wrong scalar types fail before remote work. Aliases/merges are validated after resolution. Strings must remain strings, booleans must be true/false and integers must fit their declared range. Explicit null is invalid except the nullable Pages custom-domain scalar. Empty strings, zero, false and empty arrays retain their distinct contracts. No desired defaults are populated from observation metadata or planned #480 fallbacks.

Scalar omission is unmanaged. Labels/Rulesets/Actions policies own only listed whole objects with their stated field rules. Existing Environment built-in omissions are preserved from fresh state; creation omissions remain API-controlled. Topics and explicitly declared inner lists have their authoritative collection semantics. These distinctions remain the same through CLI and public Action use.

## Exported values

Export completes acquisition and conversion before writing. All independent families follow the [automatic export omission/retention contract](../features/readable-settings-export.feature.md); checkout/origin resolution, invalid prior files, family-info output and destination failures remain fatal. About/General observation groups are independent too; unavailable settings retain valid prior declarations only where compatible with positively acquired settings, with bounded info. No incomplete family becomes plausible empty/false YAML. Incomplete data remains an acquisition failure. Correctly typed out-of-UI Security statuses follow their [Security policy](security.md#secret-scanning-and-push-protection), and public-fork approval strings follow their [fork policy](actions.md#fork-pr-workflow-policies). Complete workflow permission strings, merge/squash message pairs and creation-policy strings follow their [workflow](actions.md#default-workflow-permissions), [menu](general.md#commit-message-observation-normalization) and [creation-policy](general.md#creation-policy-observation-normalization) fallback policies. Complete Pages modes and legacy folders follow their [raw-source policy](pages.md#pages-configuration); Actions execution follows its [visible/latent policy](general.md#execution-policy-observation-normalization). Ruleset enforcement follows its [ID-bound policy](rulesets.md#ruleset-enforcement-observation-normalization); Environment deployment modes follow their [raw-pair policy](environments.md#environment-deployment-mode-observation-normalization), and pattern ref types follow their [raw-identity policy](pages.md#environment-ref-type-observation-normalization). Only the explicitly documented typed fallback policies normalize observations; unsupported identities and incomplete values remain errors. Ordinary export attempts every supported family without selectors; excluded controls and secrets remain out of scope. Hidden conditional controls are omitted without replaying latent API values as selected intent. IDs, false/zero/empty values and nullable Pages domains retain their declared meanings.

An export followed by validate/plan roundtrips exact supported acquired values, subject to the resource-specific ownership and normalization limits below. A normalized fallback does not establish raw equality. Export does not claim live entitlement, effective protection or transactional apply. It overwrites existing declaration content; review before adoption.

[^github-update-repository]: Repository update request fields; original inspection 2026-10-11. This does not expand the supported UI inventory.

[^github-repository-openapi]: Fixed GitHub OpenAPI evidence for nullable observed description/homepage; original inspection 2026-10-12.
