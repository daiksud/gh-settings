---
type: Reference
title: Prepared Ruleset UI mappings
description: Ruleset UI controls, internal API translation, observation defaults and complete unsupported-policy rejection.
sources:
  - id: ruleset-ui
    resource: https://github.com/daiksud/gh-settings/settings/rules
  - id: existing-bypass-ui
    resource: https://github.com/daiksud/agents/settings/rules/22615823
  - id: role-identities
    resource: https://raw.githubusercontent.com/integrations/terraform-provider-github/main/docs/resources/repository_ruleset.md
---

## Internal preparation

This Ruleset mapping was prepared for [#483](https://github.com/daiksud/gh-settings/issues/483) and is consumed by [#479](https://github.com/daiksud/gh-settings/issues/479) in current-source UI YAML. The [public reference](../settings.md#rulesets) defines the accepted contract. [#480](https://github.com/daiksud/gh-settings/issues/480) normalization and [#481](https://github.com/daiksud/gh-settings/issues/481) complete reporting remain separate; native internal object reports retain their existing protocol.

## UI evidence

English labels and choices were inspected on 2026-10-28. Existing gh-settings and agents Rulesets supplied root, bypass and active-rule controls. Conditional reviewer and code-analysis controls were revealed only in an unnamed, unsaved new Ruleset form; no Create or Save action was submitted, and that form was discarded. This verifies UI wording and options, not live mutation effects or effective protection.[^ruleset-ui][^existing-bypass-ui]

The visible bypass options' DOM identities map Repository admin to `RepositoryRole:5`, Maintain to `RepositoryRole:2`, and Write to `RepositoryRole:4`. The independent provider reference documents the same role IDs; the existing agents API/UI pair also identifies Repository admin 5. App, team and user identities remain dynamic; no installation-specific ID is hardcoded.[^role-identities]

## Prepared control paths

In these tables `R` means `repository.settings.rulesets[]`. The [model](../../internal/settings/uimodel/rulesets.go) has no serialization tags; these paths are the preparation handoff. The `ui_ruleset_*.go` files in [the adapter](../../internal/github/) contain the transport identifiers and conversions.

| UI control | UI path | Internal API representation |
| --- | --- | --- |
| Delete Ruleset | `R.delete: true` with exact `R.id` and `R.ruleset_name` | ID-addressed DELETE, no policy payload; all policy controls unselected |
| Selected Ruleset identity | `R.id` | Numeric ID selects an existing resource; no ID remains create-only; never a writable payload field |
| Ruleset Name | `R.ruleset_name` | `name` |
| Enforcement status | `R.enforcement_status` | `enforcement`; Active / Disabled become `active` / `disabled` |
| Target branches / Include default branch | `R.target_branches.include_default_branch` | `target: branch`, include sentinel `~DEFAULT_BRANCH` |
| Target branches / Include all branches | `R.target_branches.include_all_branches` | Include sentinel `~ALL` |
| Target branches / Include by pattern | `R.target_branches.include_by_pattern` | Include strings prefixed with `refs/heads/` |
| Target branches / Exclude by pattern | `R.target_branches.exclude_by_pattern` | Exclude strings prefixed with `refs/heads/` |
| Target tags / Include all tags | `R.target_tags.include_all_tags` | `target: tag`, include sentinel `~ALL` |
| Target tags / Include by pattern | `R.target_tags.include_by_pattern` | Include strings prefixed with `refs/tags/` |
| Target tags / Exclude by pattern | `R.target_tags.exclude_by_pattern` | Exclude strings prefixed with `refs/tags/` |
| Bypass list / Roles | `R.bypass_list[].role` | `RepositoryRole`; `repository_admin` → 5, `maintain` → 2, `write` → 4 |
| Bypass list / Teams, Apps, Users | `R.bypass_list[].team`, `.app`, `.user` | Dynamic numeric IDs; `Team`, `Integration`, `User` respectively |
| Bypass list / Deploy keys | `R.bypass_list[].deploy_keys` | Selected `true` marker → `DeployKey`, API `actor_id: null` |
| Select bypass mode | `R.bypass_list[].bypass_mode` | Always → `always`; Exempt → `exempt`; For pull requests only → `for_pull_requests_only` → API `pull_request` |
| Rules / Restrict creations | `R.rules.restrict_creations` | Rule type `creation` |
| Rules / Restrict updates | `R.rules.restrict_updates` | Rule type `update`; `.allow_fetch_and_merge` maps to required `update_allows_fetch_and_merge` |
| Rules / Restrict deletions | `R.rules.restrict_deletions` | Rule type `deletion` |
| Rules / Block force pushes | `R.rules.block_force_pushes` | `non_fast_forward` |
| Rules / Require linear history | `R.rules.require_linear_history` | `required_linear_history` |
| Rules / Require signed commits | `R.rules.require_signed_commits` | `required_signatures` |
| Require a pull request before merging | `R.rules.require_a_pull_request_before_merging` | Selects `pull_request` rule |
| Required approvals | PR rule `.required_approvals` | `required_approving_review_count`; UI integers 0–10 |
| Allowed merge methods | PR rule `.allowed_merge_methods` | Merge / Squash / Rebase → `merge` / `squash` / `rebase` |
| Dismiss stale pull request approvals when new commits are pushed | PR rule `.dismiss_stale_pull_request_approvals_when_new_commits_are_pushed` | `dismiss_stale_reviews_on_push` |
| Require review from Code Owners | PR rule `.require_review_from_code_owners` | `require_code_owner_review` |
| Require approval of the most recent reviewable push | PR rule `.require_approval_of_the_most_recent_reviewable_push` | `require_last_push_approval` |
| Require conversation resolution before merging | PR rule `.require_conversation_resolution_before_merging` | `required_review_thread_resolution` |
| Require an additional approval for unattributed Copilot pull requests | PR rule `.require_an_additional_approval_for_unattributed_copilot_pull_requests` | `require_extra_approval_for_unattributed_changes` |
| Require review from specific teams | PR rule `.require_review_from_specific_teams[]` | `required_reviewers` |
| Add reviewer / Reviewer | Team entry `.reviewer` | Numeric team ID with API type `Team` |
| Add reviewer / Approvals | Team entry `.approvals` | `minimum_approvals`; UI integers 0–10, explicitly required |
| Add reviewer / File patterns | Team entry `.file_patterns` | Required ordered list, preserving negations and repeated patterns |
| Require status checks to pass | `R.rules.require_status_checks_to_pass` | Selects `required_status_checks` rule |
| Require branches to be up to date before merging | Status rule `.require_branches_to_be_up_to_date_before_merging` | `strict_required_status_checks_policy` |
| Do not require status checks on creation | Status rule `.do_not_require_status_checks_on_creation` | `do_not_enforce_on_create` |
| Required status checks | Status rule `.required_status_checks[]` | API `required_status_checks` collection |
| Search for checks / selected check | Check entry `.check` | API `context`, preserved exactly |
| Select source / Any source | Check entry `.source.any_source` | Selected `true` → absent/null `integration_id`; UI option ID 0 is not an API identity |
| Select source / selected App | Check entry `.source.app` | Positive dynamic `integration_id`; do not infer it from the provider's display name |
| Require code scanning results | `R.rules.require_code_scanning_results` | Selects `code_scanning` rule |
| Code scanning tools | Scanning rule `.code_scanning_tools[]` | `code_scanning_tools` |
| Tool name | Scanning tool `.tool_name` | Case-sensitive API `tool`, not a fixed product enum |
| Security alerts | Scanning tool `.security_alerts` | `security_alerts_threshold` |
| Alerts | Scanning tool `.alerts` | `alerts_threshold` |
| Automatically request Copilot code review | `R.rules.automatically_request_copilot_code_review` | Selects `copilot_code_review`; `.review_new_pushes` → `review_on_push`, `.review_draft_pull_requests` → same API name |
| Restrict code coverage | `R.rules.restrict_code_coverage` | `code_coverage`; branch target only |
| Minimum line coverage percentage | Coverage `.minimum_line_coverage_percentage` | `minimum_coverage`; explicit finite number 0–100 |
| Maximum line coverage drop | Coverage `.maximum_line_coverage_drop` | `max_coverage_drop`; explicit finite number 0–100 |
| Require code quality results | `R.rules.require_code_quality_results` | Selects `code_quality` rule |
| Severity | Quality rule `.severity` | API `severity` with the conversion below |

Branch rules and Tag rules are subheadings under Rules, selected by the target object rather than a second independent discriminator. Require exactly one target object and an explicit enforcement choice. A present empty target object selects an empty inclusion. Empty or whitespace-only naming patterns are invalid before prefixing and after reverse prefix removal; nonblank contents are otherwise retained exactly. Tag criteria cannot use the default-branch sentinel.

Each bypass entry selects exactly one actor and explicitly supplies its mode. `deploy_keys: false`, multiple selectors, nonpositive dynamic IDs and unsupported role tokens are invalid. Existing target dependencies remain enforced, including no pull-request bypass for tag Rulesets or deploy keys.

## Analysis choices and defaults

| Control | Verified UI choices | Prepared choice → API value when different |
| --- | --- | --- |
| Alerts | None; Errors; Errors and Warnings; All | `none`, `errors`, `errors_and_warnings`, `all` |
| Security alerts | None; Critical; High or higher; Medium or higher; All | `none`, `critical`, `high_or_higher`, `medium_or_higher`, `all` |
| Severity | Errors; Warnings and higher; Notes and higher; All | `errors`; `warnings_and_higher` → `warnings`; `notes_and_higher` → `notes`; `all` |

The model supplies separate observation defaults for #480. They match the inspected new-form controls. Enforcement, Required approvals, Allowed merge methods, Team reviewer Approvals, scanning Alerts, scanning Security alerts, Quality severity and bypass-mode normalization are active:

| UI field | Explicit observation fallback |
| --- | --- |
| Enforcement status | `disabled` (`DefaultRulesetEnforcementStatus`) |
| Bypass mode | `always` (`DefaultBypassMode`) |
| Required approvals | 0 (`DefaultRequiredApprovals`) |
| Allowed merge methods | Merge, Squash, Rebase (`DefaultRulesetMergeMethods`, returning owned values) |
| Team reviewer Approvals | 0 (`DefaultTeamReviewApprovals`) |
| Scanning Alerts | `errors` (`DefaultCodeScanningAlerts`) |
| Scanning Security alerts | `high_or_higher` (`DefaultCodeScanningSecurityAlerts`) |
| Quality Severity | `errors` (`DefaultCodeQualitySeverity`) |

Actor/reviewer/source/tool identities and naming/file patterns are not closed enums with a synthetic identity default. The UI starts with no bypass actor selected; preparation never substitutes a repository administrator, another ID or another pattern for an unrepresentable value. Personal-repository UI has no OrganizationAdmin actor, and Evaluate was absent from the inspected enforcement menu. OrganizationAdmin remains unsupported in UI projection without an invented identity. Acquired Evaluate is rejected as unsupported, rather than normalized into a writable Disabled policy. Complete unknown enforcement projects to Disabled with exact ID-bound provenance and contextual warnings; the native validator still distinguishes API transport values from UI choices. #480 must retain raw identity/provenance and warning information for any adopted normalization, rather than quietly removing protection or granting bypass.

Bypass-mode normalization changes only a complete acquired mode string to `always`. Its record preserves exact actor type, ID presence and numeric value; canonical companions remain intact. Nil DeployKey identity is not interchangeable with a numeric actor. Invalid identity or actor/target applicability is never repaired by mode projection. Selected actor removal retains its original evidence and requires verified absence.

Allowed merge methods uses the shared UI/native membership mapping and an owned named full-list default. One unsupported member projects the entire acquired list; duplicate members, empty lists and malformed elements fail before projection. Exact raw order remains separate from canonical effective-policy ordering. Selecting a PR rule and removing it remain distinct decisions.

Correctly typed booleans have no out-of-UI value. Missing fields, invalid types, failed reads and incomplete pagination remain acquisition errors. These observation fallback values never fill omitted desired declarations. Existing whole-object declaration defaults are a separate contract: selected Rulesets own their rule lists, nil/false simple checkboxes leave a rule unselected, and selecting a parameterized rule uses the existing declared-object normalization. Required tool thresholds, quality severity and team entry approvals/patterns still require explicit values.

## Ownership, acquisition and activation handoff

`RulesetsDeclaration` preserves omitted versus empty collections and uses the existing constructor only for desired objects. IDs remain selectors; entries without IDs are create-only. Reconciliation updates only listed resources and does not delete unlisted Rulesets. Explicit positive-ID/name `delete: true` entries select deletion; they cannot select policy controls. Excluded inherited or other-source IDs remain separate inventory evidence, preventing an excluded resource from being interpreted as already absent.

`RulesetsObservation` requires a complete acquisition, uses `RulesetState.ID` rather than a possibly absent policy ID, and returns stable name order. It never applies declaration defaults to observations or strips unsupported protection to manufacture success. Failure in any entry returns no partial UI collection. Existing transport includes complete repository-owned push Rulesets, excludes inherited organization/Enterprise policy and performs its complete pagination and ownership checks. The observed decoder projects copied, typed enforcement, present Required approvals, present Allowed merge methods, complete Team reviewer counts, complete tool-bound Alerts and Security alerts strings, present Quality severity and complete bypass-mode members before one strict complete validation; mutation verification keeps the original strict decoder. Selected normalized IDs force whole-object updates even at effective equality, and require fresh GET in addition to strict mutation-response checks. Scoped export/apply diagnostics preserve unlisted-ID privacy. The [public enforcement policy](../settings.md#ruleset-enforcement-observation-normalization), [Required approvals policy](../settings.md#ruleset-required-approvals-observation-normalization), [Allowed merge methods policy](../settings.md#ruleset-allowed-merge-methods-observation-normalization), [Team reviewer Approvals policy](../settings.md#ruleset-team-reviewer-approvals-observation-normalization), [scanning Alerts policy](../settings.md#ruleset-scanning-alerts-observation-normalization), [scanning Security alerts policy](../settings.md#ruleset-scanning-security-alerts-observation-normalization), [Quality severity policy](../settings.md#ruleset-quality-severity-observation-normalization) and [bypass-mode policy](../settings.md#ruleset-bypass-mode-observation-normalization) own these active limits. Quality absence stays absent, desired rule removal requires verified absence, and simultaneous provenance remains independent on one whole-object update.

Inherited Rulesets, legacy branch protection and organization-wide policy remain outside this mapping. A present review-dismissal restriction is unsupported and stops acquisition; API omission does not establish disabled protection.

See [the test specification](testing.md#internal-ui-ruleset-preparation) for independent expectations and the limits of local evidence.

[^ruleset-ui]: Existing Ruleset and unsaved new branch/tag forms inspected read-only or with local unsaved control reveals, then discarded, 2026-10-28.
[^existing-bypass-ui]: Existing Repository admin / For pull requests only bypass inspected without editing its policy, 2026-10-28.
[^role-identities]: Provider-maintained reference explicitly lists associated base repository role IDs: maintain 2, write 4, admin 5; retrieved 2026-10-28.

Required approvals observes an actual integer separately from the UI range 0–10. Typed out-of-range counts project 0 with exact ID-bound integer evidence, while missing/null/fractional/overflow values remain acquisition errors. PR-rule absence stays absent and desired removal requires verified absence. The same observed decoder supplies export/plan/apply evidence, but mutation verification reads original raw responses. Native-valid 11 remains useful evidence of an API value outside the UI range; native validation has not been narrowed to a UI validator.

Team reviewer Approvals uses the same explicit UI range 0–10 and named zero default. Typed out-of-range integers retain provenance under the exact positive Team ID; canonical reviewers, ordered/repeated/negated file patterns and empty lists retain their acquired meaning. Missing or malformed members, duplicate IDs, non-Team identities and more than 15 reviewers fail without partial evidence. An absent PR rule or empty reviewer list creates no reviewer. Declared approvals and file patterns remain required. ID replacement is removal plus a new reviewer, and strict raw confirmation checks the complete desired policy.

Scanning Alerts uses one shared UI/native membership mapping. Complete unsupported strings project Errors and retain exact tool-name-bound evidence; names remain case-sensitive and are not trimmed. Canonical tools and Security alerts companions remain intact. Invalid identities, duplicate tools, missing/null/wrong-type fields and incomplete reads fail. Tool removal or replacement retains the original evidence and requires strict raw absence confirmation; Security alerts use an independent tool-bound record and High or higher fallback.

Scanning Security alerts shares its five-choice mapping between UI conversion, native validation and observation projection. Complete unsupported strings retain exact raw evidence and project High or higher. Both threshold records can coexist for the same tool without replacing each other; both must pass independent closed validation. Missing or malformed values and identities remain errors, and original raw acknowledgment/readback must confirm each desired threshold or removal.

`R.rules.require_deployments_to_succeed.required_deployment_environments` maps the selected deployment environment names to `required_deployments.parameters.required_deployment_environments`. The list is explicit, supports branch and tag Rulesets and is acquired completely; no environment or omitted value is synthesized. See the [configuration contract](../settings.md#required-deployments).

## Push target mapping

`R.target_pushes: true` selects REST `target: push` without `conditions`. `restrict_file_paths` maps `restricted_file_paths` plus `allowed_exceptions` to `file_path_restriction` and its `ignored_file_paths`; `restrict_file_path_length.maximum_file_path_length` maps `max_file_path_length`; `restrict_file_extensions.restricted_file_extensions` maps `file_extension_restriction`; `restrict_file_size.maximum_file_size` plus `allowed_exceptions` maps `max_file_size` and its `ignored_file_paths`. Both path/size exception lists must be explicit and completely acquired. Push rules are never projected onto branch/tag target controls. See [Push Rulesets](../settings.md#push-rulesets) for public limits and eligibility.
