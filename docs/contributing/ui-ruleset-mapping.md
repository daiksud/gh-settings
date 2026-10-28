---
type: Reference
title: Prepared Ruleset UI mappings
description: Ruleset UI controls, internal API translation, observation defaults and the unresolved dismissal mapping.
sources:
  - id: ruleset-ui
    resource: https://github.com/daiksud/gh-settings/settings/rules
  - id: existing-bypass-ui
    resource: https://github.com/daiksud/agents/settings/rules/22615823
  - id: role-identities
    resource: https://raw.githubusercontent.com/integrations/terraform-provider-github/main/docs/resources/repository_ruleset.md
---

## Internal preparation and remaining decision

This is the internal Ruleset mapping for [Issue #483](https://github.com/daiksud/gh-settings/issues/483), not accepted new YAML. The current parser, public configuration, API adapters and reports retain their existing behavior. [Issue #479](https://github.com/daiksud/gh-settings/issues/479) owns activation; [#480](https://github.com/daiksud/gh-settings/issues/480) owns normalization/warnings, and [#481](https://github.com/daiksud/gh-settings/issues/481) owns reporting.

**The mapping is incomplete for the current write-only `dismissal_restriction` setting.** No corresponding control was found in the inspected existing or new-form PR rule details. Do not invent a UI name, silently discard acquired values, or claim full #483 completion. The prepared read converter returns an explicit unresolved-mapping error if that object is present, even when its values are false/empty. The existing public implementation is untouched. Activation needs verified UI evidence or an authorized disposition of this setting; a new-format unsupported decision has not been made.

## UI evidence

English labels and choices were inspected on 2026-10-28. Existing gh-settings and agents Rulesets supplied root, bypass and active-rule controls. Conditional reviewer and code-analysis controls were revealed only in an unnamed, unsaved new Ruleset form; no Create or Save action was submitted, and that form was discarded. This verifies UI wording and options, not live mutation effects or effective protection.[^ruleset-ui][^existing-bypass-ui]

The visible bypass options' DOM identities map Repository admin to `RepositoryRole:5`, Maintain to `RepositoryRole:2`, and Write to `RepositoryRole:4`. The independent provider reference documents the same role IDs; the existing agents API/UI pair also identifies Repository admin 5. App, team and user identities remain dynamic; no installation-specific ID is hardcoded.[^role-identities]

## Prepared control paths

In these tables `R` means `repository.settings.rulesets[]`. The [model](../../internal/settings/uimodel/rulesets.go) has no serialization tags; these paths are the preparation handoff. The `ui_ruleset_*.go` files in [the adapter](../../internal/github/) contain the transport identifiers and conversions.

| UI control | Prepared path | Internal API representation |
| --- | --- | --- |
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

The model supplies separate observation defaults for #480. They match the inspected new-form controls, and are not applied by these strict preparation converters:

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

Actor/reviewer/source/tool identities and naming/file patterns are not closed enums with a synthetic identity default. The UI starts with no bypass actor selected; preparation never substitutes a repository administrator, another ID or another pattern for an unrepresentable value. Personal-repository UI has no OrganizationAdmin actor, and Evaluate was absent from the inspected enforcement menu. Their existing API support is preserved in the current public path; the prepared UI converter fails explicitly. #480 must retain raw identity/provenance and warning information for any adopted normalization, rather than quietly removing protection or granting bypass.

Correctly typed booleans have no out-of-UI value. Missing fields, invalid types, failed reads and incomplete pagination remain acquisition errors. These observation fallback values never fill omitted desired declarations. Existing whole-object declaration defaults are a separate contract: selected Rulesets own their rule lists, nil/false simple checkboxes leave a rule unselected, and selecting a parameterized rule uses the existing declared-object normalization. Required tool thresholds, quality severity and team entry approvals/patterns still require explicit values.

## Ownership, acquisition and activation handoff

`RulesetsDeclaration` preserves omitted versus empty collections and uses the existing constructor only for desired objects. IDs remain selectors; entries without IDs are create-only. Reconciliation updates only listed resources and does not delete unlisted Rulesets.

`RulesetsObservation` requires a complete acquisition, uses `RulesetState.ID` rather than a possibly absent policy ID, and returns stable name order. It never applies declaration defaults to observations or strips dismissal state to manufacture success. Failure in any entry returns no partial UI collection. `PreparedRulesets.ExcludedPush` retains the existing excluded-push diagnostic; activation/reporting must carry it forward. Existing transport still excludes inherited/push Rulesets and performs its complete pagination, ownership, mutation-response and independent readback checks.

The observed UI also contains Restrict creations, Restrict updates, Require deployments to succeed, Restrict code coverage and Automatically request Copilot code review. This preparation does not adopt those unsupported controls, inherited/push rules, legacy branch protection or organization-wide policy. Unresolved dismissal mapping must remain visible in the draft PR and activation checklist.

See [the test specification](testing.md#internal-ui-ruleset-preparation) for independent expectations and the limits of local evidence.

[^ruleset-ui]: Existing Ruleset and unsaved new branch/tag forms inspected read-only or with local unsaved control reveals, then discarded, 2026-10-28.
[^existing-bypass-ui]: Existing Repository admin / For pull requests only bypass inspected without editing its policy, 2026-10-28.
[^role-identities]: Provider-maintained reference explicitly lists associated base repository role IDs: maintain 2, write 4, admin 5; retrieved 2026-10-28.
