---
type: Reference
title: "Ruleset configuration"
description: "Ruleset configuration declaration, ownership and API contracts."
---

# Branch and tag rulesets

Declare the complete policy for each branch or tag ruleset you select in
`repository.settings.rulesets`. Omitting a rule or bypass actor from a selected
ruleset can remove that protection. Use [identity and ownership](#identity-and-ownership)
to decide which objects to include and [plan, apply, and export](#plan-apply-and-export)
to interpret changes.

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Identity and ownership

Rulesets are the only supported branch-protection mechanism in v1; classic
branch protection is [outside v1 scope](legacy-classic-protection.md#classic-branch-protection).

`repository.settings.rulesets` selects whole repository-owned branch or tag Rulesets by optional positive numeric ID. No ID means create-only; a collision requires explicit ID adoption. Unknown/wrong-target/type IDs fail without create fallback. Without prune, unlisted objects are not deleted and omission/`[]` selects nothing. With `--prune rulesets`, present lists own retention as described below. Names compare exactly without trimming/case folding; duplicate names/IDs and colliding renames fail.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 123
        ruleset_name: release-tags
        enforcement_status: active
        target_tags: {include_by_pattern: ['v*']}
        rules: {restrict_deletions: true}
```

### Deleting rulesets

`--prune rulesets` deletes whole unlisted rulesets. An omitted list stays
unmanaged. A present empty list selects all acquired rulesets for deletion.
List every ID you want to keep and preview with `gh settings plan --prune rulesets`.
The [pruning contract](pruning.md#collection-pruning) defines scope and preflight.

Deletion requires Administration write, complete bypass visibility and an exact
identity and policy check. Unsupported, inherited, Push or normalized policies
block deletion. A complete supported PR rule is eligible with either value of
the additional-approval checkbox. After an empty HTTP 204, apply checks the
complete inventory for absence. Other repository or organization protection
can remain in effect. [Repository Ruleset deletion API](https://docs.github.com/en/rest/repos/rules#delete-a-repository-ruleset)

## Targets, rules and bypass actors

Every present-policy entry requires `ruleset_name`, `enforcement_status` (`active` or `disabled`) and exactly one of `target_branches`/`target_tags`. Branch targets support `include_default_branch`, `include_all_branches`, `include_by_pattern` and `exclude_by_pattern`; tag targets support `include_all_tags` and the pattern arrays. Patterns use the UI spelling without internal ref prefixes. An explicit empty target object selects empty include/exclude lists; omitting all target selectors is invalid. No protected scope is guessed.

Omitted `bypass_list` and unselected `rules` become empty policy on an owned object. Basic booleans include `restrict_creations`, `restrict_deletions`, `block_force_pushes`, `require_linear_history` and `require_signed_commits`; false/omission leaves that rule unselected. Parameterized groups include `restrict_updates`, `require_a_pull_request_before_merging`, `require_status_checks_to_pass`, `require_code_scanning_results`, `require_code_quality_results`, `restrict_code_coverage`, `require_deployments_to_succeed`, `automatically_request_copilot_code_review`. Omitting a group removes it on the owned object; it does not preserve an existing group. Target-specific applicability remains required.

`restrict_creations: true` allows only bypass actors to create matching branches or tags. `restrict_updates: {allow_fetch_and_merge: false}` allows only bypass actors to update them; selecting an empty object supplies a declaration-only false default. Explicit true allows pulling upstream changes. Both rules support branch and tag targets. Export preserves the acquired upstream-sync flag; an absent, null or malformed flag is an acquisition error. Updates adding, changing or removing either rule require independent whole-policy GET verification after mutation acknowledgment. These repository rules use the standard Rulesets plan availability and Administration write permission; no organization-wide Ruleset is managed. See [GitHub's rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) and [REST parameter contract](https://docs.github.com/en/rest/repos/rules).

`automatically_request_copilot_code_review: {review_new_pushes: true, review_draft_pull_requests: false}` selects automatic Copilot reviews for a branch Ruleset. Selecting an empty object defaults both options false only in the declaration; omitting the group removes the rule. Export requires both acquired booleans with their actual values; missing/null/malformed options fail rather than inventing a policy. Updates and removals require independent whole-policy GET verification after the mutation response. Automatic review requires author access to Copilot code review and available AI credits, subject to GitHub's applicable organization policies. This repository rule neither grants licenses nor configures Copilot approval eligibility. See [GitHub's automatic review controls](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review) and [repository Rulesets API](https://docs.github.com/en/rest/repos/rules).

A bypass entry selects exactly one actor: `role` (`repository_admin`, `maintain`, `write`), positive `team`, `app` or `user`, or `deploy_keys: true`. `bypass_mode` is required and must be `always`, `exempt` or `for_pull_requests_only`; existing branch/actor applicability constraints apply. No actor/grant/credential is created. Organization-only actors and unadopted rules are unsupported.

### PR review and required check parameters

`rules.require_a_pull_request_before_merging` is a branch-only group:

| Field | Default / accepted value |
| --- | --- |
| `required_approvals` | 0; UI integers 0–10 |
| `require_an_additional_approval_for_unattributed_copilot_pull_requests` | true; explicit false disables the additional approval |
| `allowed_merge_methods` | all three: `merge`, `squash`, `rebase`; nonempty unique selected choices |
| `dismiss_stale_pull_request_approvals_when_new_commits_are_pushed`, `require_review_from_code_owners`, `require_approval_of_the_most_recent_reviewable_push`, `require_conversation_resolution_before_merging` | false |

The additional-approval checkbox maps to the REST PR-rule parameter
`require_extra_approval_for_unattributed_changes`. It adds one approval for
unattributed Copilot pull requests when `required_approvals` is nonzero; it has no
effect when the count is 0. GitHub documents this Preview control as enabled by
default for new and existing Rulesets. [Official Ruleset behavior](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#additional-approval-for-unattributed-copilot-pull-requests).

Selecting the PR group owns this boolean: omission within the group declares
true, while explicit false remains false. Export writes the actual acquired
boolean. Missing, null or incorrectly typed observations fail complete acquisition;
declaration defaults never fill unread values. Omitting the PR group removes the
whole rule, including additional approval. Existing explicit Ruleset prune can
remove its whole containing object after the normal complete-candidate checks.

Every update compares the fresh additional-approval value and PR-rule presence
against the original acquisition before PUT; drift stops the write. An intended
true/false change is compared against desired state only in the mutation response
and independent GET verification. Fresh checks do not lock concurrent edits;
post-write mismatches remain uncertain and no rollback occurs. Creation and
deletion retain complete-policy and identity checks.

Independent supported collections canonicalize by identity.

`rules.require_status_checks_to_pass` contains `required_status_checks` (`[{check: name, source: {any_source: true}}]` or `{source: {app: positive-id}}`), `require_branches_to_be_up_to_date_before_merging` and `do_not_require_status_checks_on_creation`. Defaults are empty checks and false flags. Omitted source means any source, not a retained App. Explicit source must select true Any source or App, not both. YAML null is invalid.

### Required deployments

`rules.require_deployments_to_succeed` selects a branch or tag rule and requires an explicit `required_deployment_environments` string list, for example `{required_deployment_environments: [staging, production]}`. Names retain exact case and whitespace; blank names, control characters and duplicate names are invalid. Lists compare as sets and export in stable order. Explicit `[]` retains an empty selected rule; omission removes the rule. No environment is created, renamed or deployed by this setting. GitHub accepts the rule on tag Rulesets and displays the selected environments in its UI. Successful storage and readback confirm the saved policy, not that deployment protection has been exercised; enforcement depends on GitHub and the Ruleset enforcement status.

[Repository Rulesets REST](https://docs.github.com/en/rest/repos/rules#create-a-repository-ruleset) defines the `required_deployments` transport and requires repository Administration write permission for mutations. [Ruleset availability](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) depends on repository visibility and plan: public repositories support Free; private repositories require Pro, Team or Enterprise Cloud. Deployment environments have their own [availability requirements](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments). This is a repository-level rule; organization and Enterprise rulesets remain outside ownership. Mutations adding, changing or removing it require independent post-write GET confirmation in addition to the mutation response. Unknown or incomplete acquired policy stops export and updates rather than dropping protection.

### Code scanning parameters

`rules.require_code_scanning_results` requires `code_scanning_tools`, each with `tool_name`, `alerts` (`none`, `errors`, `errors_and_warnings`, `all`) and `security_alerts` (`none`, `critical`, `high_or_higher`, `medium_or_higher`, `all`). Tool names are case-sensitive identities; duplicates fail and collection order canonicalizes by name. Explicit `[]` remains empty; no threshold default is inferred.

### Code quality parameters

`rules.require_code_quality_results` requires `severity`: `errors`, `warnings_and_higher`, `notes_and_higher` or `all`. The UI choices map to the public API values `errors`, `warnings`, `notes` and `all`; API tokens are not YAML aliases. This is a managed rule on the selected whole Ruleset: omitting the group removes it, so include every supported protection that must remain. Exact canonical equality is a no-op. Changes and removals require strict mutation-response confirmation plus an independent GET.

The [official Rulesets reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-code-quality-results) documents the rule and Free public Rulesets, and the [public REST API](https://docs.github.com/en/rest/repos/rules#update-a-repository-ruleset) documents its severity field. Product entitlement descriptions are not evidence that this individual control is unavailable. The rule requires configured Code Quality analysis; managing it does not enable the product, start analysis, create workflows or change billing. Coverage uses the separate [coverage threshold rule](#code-coverage-thresholds). See [Code Quality normalization](#values-outside-the-ui-choices).

### Code coverage thresholds

`rules.restrict_code_coverage` selects a branch rule with two numeric controls:

```yaml
rules:
  restrict_code_coverage:
    minimum_line_coverage_percentage: 80.5
    maximum_line_coverage_drop: 2.25
```

Both accept finite numbers from 0 through 100, including fractions. The minimum
is a line coverage percentage; the maximum drop is measured in percentage
points relative to the default branch. **Zero disables that threshold.**
A selected group defaults each omitted declaration member to 0 and sends both
numbers. An empty mapping selects a rule with both thresholds disabled; it is
distinct from omitting the group, which removes the rule on an owned Ruleset.
These are declaration defaults, never defaults for unread GitHub state.

The group must be a mapping. Booleans, null, quoted numbers, nonfinite or
out-of-range values, API field aliases and tag targets are rejected. Include
every supported protection that must remain on the selected whole object.
Unlisted Rulesets remain untouched by ordinary apply; explicit Ruleset prune
can delete a complete supported coverage Ruleset under its normal safety checks.

Export requires both acquired numeric thresholds and writes their actual values,
including zero. The official response schema makes them optional without
specifying omission defaults; missing/null members therefore make acquisition
unavailable rather than becoming disabled thresholds. Initial export omits the
unavailable Rulesets family, and re-export retains its valid prior intent. No
partial protection is exported. Exact matching objects cause no write. Before
PUT, a fresh coverage presence/value snapshot must still match the original
acquisition, alongside the Code Quality and additional-approval drift checks.
The mutation response and independent GET must confirm the complete requested
policy. Failed confirmation retains uncertain effects and creation-ID recovery;
only verified IDs are written back to local YAML.

The [official coverage guide](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/restrict-code-coverage)
and [Rules REST API](https://docs.github.com/en/rest/repos/rules) define the
control. It requires Code Quality and uploaded coverage data for evaluation;
it does not wait for uploads. Configure the associated required status checks
when merging must wait for coverage results. Managing these thresholds does not
enable Code Quality, upload data, author workflows, add checks, expand grants or
change billing. Configuration acceptance, execution entitlement and effective
merge enforcement are separate: fixture verification does not prove live
account acceptance or enforcement. See the [coverage evidence](../../development/settings-coverage.md)
and [behavior specification](../../development/features/rulesets.feature.md#code-coverage-thresholds).

## Values outside the UI choices

When GitHub returns a complete policy with an out-of-range value in one of the
following fields, export writes the documented fallback and warns with the
original value and ruleset identity. A field's fallback applies only to that
observation. Missing required fields, wrong types, malformed identities, duplicate
entries and incomplete pagination still make the ruleset family unavailable.

| Observed control | Value used in exported YAML | Conditions |
| --- | --- | --- |
| Enforcement | `disabled` | Unknown strings, including empty or whitespace. GitHub's `evaluate` mode is unsupported and blocks acquisition. |
| Required approvals | `0` | An integer outside the UI's 0–10 range. Missing, fractional or overflowing counts fail acquisition. |
| Allowed merge methods | `[merge, squash, rebase]` | A complete, nonempty list of unique strings containing an unknown method. The warning preserves the original list order. Empty lists and duplicates fail acquisition. |
| Scanning Alerts | `errors` | An unsupported string for an otherwise valid tool. |
| Scanning Security alerts | `high_or_higher` | An unsupported string for an otherwise valid tool. Each tool retains its own two thresholds. |
| Code Quality severity | `errors` | An unsupported string in a complete Code Quality rule. |
| Bypass mode | `always` | A string outside GitHub's `always`, `exempt` and `pull_request` values for an otherwise valid actor. |

Warnings follow ruleset name order. Scanning warnings identify the exact tool
name and show Alerts before Security alerts. Bypass warnings identify each
actor by type and ID when present. Each affected control keeps its own warning
even if several controls in one ruleset use fallbacks.

For a selected ruleset, plan and detailed Action summaries show the effective
policy and mark the intended update as write-required, even when the desired
choice equals the fallback. Removing a rule, tool or bypass actor requires
confirmed absence. Apply checks the complete desired policy, the same ID and
repository ownership in both the mutation response and an independent read.
A response that still contains the out-of-range value fails verification.
Unlisted IDs stay outside ordinary plan and apply. Status-only reports hide
policy values and identities.

## Unimplemented branch controls

Public YAML does not accept `require_a_pull_request_before_merging.require_review_from_specific_teams`, including empty collections and explicit false. The personal-public Ruleset editor observed on 2026-11-13 visibly offered the **Require review from specific teams** checkbox, despite official documentation describing the control as unavailable for user-owned repositories. A follow-up inspection of its expanded fields found no teams for the owner-name search. Usable personal-repository team selection remains unconfirmed. See the [UI evidence and coverage limits](../../development/settings-coverage.md). Organization team ownership and access management remain outside the target. The [official rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) and [coverage documentation](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/restrict-code-coverage) provide the control context.

Acquired nonempty PR `required_reviewers` and unknown or malformed rules still fail complete acquisition without dropping protection. Ordinary code-scanning merge protection, Code Quality thresholds, PR approval counts, public fork approval and automatic Copilot code review remain separate controls. All existing Ruleset updates independently recheck ownership, identity and complete policy before PUT; changes to additional approval, PR-rule presence, Code Quality presence or raw severity, and Coverage presence or either threshold since acquisition stop the update. Preflight is not an atomic lock: concurrent changes between GET and PUT can still race, and post-write failures report that writes may have applied without rollback.

## Unsupported Push Rulesets

`target_pushes` and the `restrict_file_paths`, `restrict_file_path_length`, `restrict_file_extensions` and `restrict_file_size` controls are not accepted, including false or empty declarations. [GitHub limits Push Rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository) to paid-plan private/internal repositories and their forks, outside the [personal-public target](scope.md#feature-adoption-scope) because of repository visibility, not price.

An acquired repository-owned Push Ruleset or file-restriction rule makes the Rulesets family unavailable. Export reports the failure and retains valid existing ruleset declarations or omits them on first export. Managed plan, updates and explicit deletion stop before a Rulesets write. Unsupported policy is never silently omitted or converted into a branch/tag policy. Inherited organization/enterprise policy remains outside ownership.

## Plan, apply, and export

Complete supported observations, including bypass visibility, are required. Unknown writable fields or rules, missing values, invalid types, unreadable details and incomplete pagination make the family unavailable for export and prevent updates. Inherited organization/enterprise Rulesets remain outside mutation ownership. Other protection can still affect effective behavior.

Without prune, plan shows only listed creates/ID updates and default-derived reductions. Explicit prune additionally shows its acquired deletion candidates. A nonempty collection counts once as managed. Apply reads fresh state after earlier families, prepares selected operations before the first Rulesets write and writes in stable name order. Fresh comparable matches make no write.

POST/PUT responses must confirm identity, ownership and requested comparable policy. Every create/update, including basic and empty-rule policies, then requires an independent detail GET with `includes_parents=false` to confirm the same ID, repository ownership and complete requested policy. Missing, denied, redirected or mismatching readback returns nonzero and stops later operations, including prune deletions: writes may already have applied. No retry or rollback occurs. A reported positive creation ID remains in recovery diagnostics when verification fails, but is not automatically written to YAML. Only verified creations follow [guarded writeback](labels.md#apply-id-writeback).

Unsupported protection is never omitted to produce a usable policy. Acquired Evaluate enforcement, merge queue, metadata restrictions, license compliance or a present review-dismissal restriction prevent the CLI from adopting a fresh Rulesets inventory, comparing it or updating it. Review-dismissal absence in an API response does not prove the stored restriction is disabled. These controls are not accepted configuration. See the [Rulesets specification](../../development/features/rulesets.feature.md) and [UI mapping](../../development/ui-ruleset-mapping.md).
