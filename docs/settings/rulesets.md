---
type: Reference
title: "Ruleset configuration"
description: "Ruleset configuration declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Rulesets

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

Deletion uses the command option `--prune rulesets`, not YAML tombstones. Without
prune, only listed objects are created/updated. With prune, an omitted Rulesets
list stays unmanaged, `[]` selects all acquired Rulesets for deletion, and a
nonempty list retains its declared IDs while pruning unlisted objects. Declare
all Rulesets to retain before pruning. The [prune contract](pruning.md#collection-pruning)
owns preflight, ordering and partial-failure behavior.

The `delete` member is removed: old `delete: true`/`false` declarations fail with
migration guidance. Remove the unwanted entry only after listing every other
Ruleset to keep; inspect `plan --prune rulesets`, then use `apply --prune rulesets`.
This is broader than the old single-object selector. Complete unsupported,
inherited/Push or normalized policy remains a deletion blocker. A complete supported
PR rule is eligible, including either additional-approval value.
Exact identity/policy preflight and HTTP 204 plus complete absence readback are
required; no retry, substitution or rollback is introduced.

Deletion uses [GitHub's repository Ruleset DELETE endpoint](https://docs.github.com/en/rest/repos/rules#delete-a-repository-ruleset) with Administration write permission. Complete bypass visibility is required before any deletion. Explicit deletion removes the whole selected Ruleset. Any acquired unsupported protection prevents deletion; an omitted API field does not establish that its underlying protection is disabled. Organization/Enterprise-level policy, legacy branch protection, grants, releases and bypass configuration are outside this operation. Implementation verification uses local fixtures and does not delete live protection.

Every present-policy entry requires `ruleset_name`, `enforcement_status` (`active` or `disabled`) and exactly one of `target_branches`/`target_tags`. Branch targets support `include_default_branch`, `include_all_branches`, `include_by_pattern` and `exclude_by_pattern`; tag targets support `include_all_tags` and the pattern arrays. Patterns use the UI spelling without internal ref prefixes. An explicit empty target object selects empty include/exclude lists; omitting all target selectors is invalid. No protected scope is guessed.

Omitted `bypass_list` and unselected `rules` become empty policy on an owned object. Basic booleans include `restrict_creations`, `restrict_deletions`, `block_force_pushes`, `require_linear_history` and `require_signed_commits`; false/omission leaves that rule unselected. Parameterized groups include `restrict_updates`, `require_a_pull_request_before_merging`, `require_status_checks_to_pass`, `require_code_scanning_results`, `require_code_quality_results`, `restrict_code_coverage`, `require_deployments_to_succeed`, `automatically_request_copilot_code_review`. Omitting a group removes it on the owned object; it does not preserve an existing group. Target-specific applicability remains required.

`restrict_creations: true` allows only bypass actors to create matching branches or tags. `restrict_updates: {allow_fetch_and_merge: false}` allows only bypass actors to update them; selecting an empty object supplies a declaration-only false default. Explicit true allows pulling upstream changes. Both rules support branch and tag targets. Export preserves the acquired upstream-sync flag; an absent, null or malformed flag is an acquisition error. Updates adding, changing or removing either rule require independent whole-policy GET verification after mutation acknowledgment. These repository rules use the standard Rulesets plan availability and Administration write permission; no organization-wide Ruleset is managed. See [GitHub's rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) and [REST parameter contract](https://docs.github.com/en/rest/repos/rules).

`automatically_request_copilot_code_review: {review_new_pushes: true, review_draft_pull_requests: false}` selects automatic Copilot reviews for a branch Ruleset. Selecting an empty object defaults both options false only in the declaration; omitting the group removes the rule. Export requires both acquired booleans with their actual values; missing/null/malformed options fail rather than inventing a policy. Updates and removals require independent whole-policy GET verification after the mutation response. Automatic review requires author access to Copilot code review and available AI credits, subject to GitHub's applicable organization policies. This repository rule neither grants licenses nor configures Copilot approval eligibility. See [GitHub's automatic review controls](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review) and [repository Rulesets API](https://docs.github.com/en/rest/repos/rules).

A bypass entry selects exactly one actor: `role` (`repository_admin`, `maintain`, `write`), positive `team`, `app` or `user`, or `deploy_keys: true`. `bypass_mode` is required and must be `always`, `exempt` or `for_pull_requests_only`; existing branch/actor applicability constraints apply. No actor/grant/credential is created. Organization-only actors and unadopted rules are unsupported.

### Ruleset enforcement observation normalization

A complete, correctly typed enforcement string outside Active/Disabled, except the unsupported `evaluate` value, including unknown, empty or whitespace strings, projects to the named `disabled` observation fallback. Preserve the exact raw token and its positive Ruleset ID outside desired policy. Acquired `evaluate` fails instead of projecting to Disabled; it cannot be exported or overwritten as a normalized policy. Export warns with the UI field, fallback, ID and name, in exported name order. Missing/null/wrong types, incomplete nested policy, ownership/ID failures and incomplete pagination remain acquisition errors without partial export.

A selected normalized ID requires one whole-object update even when desired Disabled equals the projection. Plan/detailed Summary show the contextual warning, a UI enforcement decision marked normalized/write-required and the complete effective supported policy. The effective object is not a raw observation. Unlisted IDs remain unmanaged, undisclosed and unwritten; desired whole-object defaults and name/ID collision guards are unchanged. Same-version private reports carry closed validation/presentation evidence and never supply apply input. A selected Rulesets collection counts once.

Apply uses fresh collection preflight and exact desired whole-object payloads. Normalized updates always require an independent post-write GET, including basic Rulesets without review/analysis rules. Both mutation response and GET must retain canonical desired policy, correct ID and repository ownership; neither verification normalizes unknown values into success. A lying canonical acknowledgment followed by retained unknown GET fails with uncertain effects. Visible fresh diagnostics never supply verification values. Canonical equal objects remain no-ops; earlier completed resource writes remain reported on later failure.

### Ruleset Quality severity observation normalization

For a present, complete code-quality rule, a correctly typed severity outside the four mapped UI choices projects to `errors`, with exact raw provenance bound to the positive Ruleset ID. Unknown, empty, whitespace and UI-spelling API strings follow this policy; missing/null/wrong-type required parameters remain acquisition errors. An absent rule stays absent. Neither observation fallback invents desired intent or makes an incomplete policy usable.

Enforcement and Quality keep independent records on one whole-object change. Export warnings follow Ruleset name order and retain independent field provenance within each object. Selected normalized equality still requires a write; unlisted IDs remain private. Plan and detailed Summary show normalized/write-required UI choices, or `inactive (rule removed; write required)` when desired policy removes the Quality group, followed by one effective policy. Same-version private reports validate each concrete record independently and retain status-only privacy; existing collection count stays one.

Apply sends only the complete desired policy and verifies the original mutation response plus an independent fresh GET. Unknown severity cannot prove an Errors match, and removing the rule requires actual absence: a retained canonical or unknown rule fails with uncertain effects. Warning projection never supplies verification values. Canonical equal policy remains a no-op; acquisition failures preserve export destinations and make no Ruleset write.

### Ruleset Required approvals observation normalization

A complete present PR rule acquires `required_approving_review_count` as an integer. UI choices are 0–10. Other integers, including -1 and API-only 11, project to the named 0 fallback while retaining exact integer provenance per positive Ruleset ID. No string, fraction, overflow, null or missing member supplies a count. An absent PR rule stays absent. Native transport validation remains separate from the UI range.

A selected normalized count requires one whole-object update, including desired 0 matching the projection. Plan and detailed Summary show unquoted integers marked normalized/write-required. Enforcement and Required approvals retain independent concrete records and warn in that order within name-sorted objects. Unlisted IDs stay unmanaged and private. Existing collection counts stay one; ordinary canonical equality stays a no-op.

Private `required_approvals_normalization` metadata requires a string path and integer raw/fallback members, including explicit zero; present-zero, forged and ID-conflicting records fail closed. Report values never supply mutation input. Apply verifies original acknowledgment and independent fresh GET against exact desired whole policy: retained 11/-1 cannot confirm 0, and a retained canonical PR rule cannot confirm removal. Incomplete acquisition makes no Ruleset write and preserves export destinations; later failure retains earlier resource effects.

### Ruleset Allowed merge methods observation normalization

A complete present pull-request rule requires a nonempty, unique list of strings. If any acquired `allowed_merge_methods` member is outside `merge`, `squash` and `rebase`, project the entire list to the named default: Merge, Squash, Rebase. Retain the exact original list and its order, including empty or whitespace strings, as owned ID-bound provenance. Missing/null lists, null or non-string members, empty lists, duplicates, incomplete policies and failed pagination remain acquisition errors. An absent PR rule stays absent.

Export warns and saves only effective choices. Plan and detailed Summary show the original list, the normalized default and the selected desired choices in named UI order; fallback-equal selection still requires one whole-object update. Existing declared-object defaults apply when the methods field is omitted within a selected PR rule. Removing the PR rule removes its owned controls, including additional approval, and requires verified absence. Unlisted IDs and omitted Ruleset collections remain undisclosed and unwritten.

Apply verifies the original mutation acknowledgment and an independent fresh GET against the desired whole policy. A retained unsupported list cannot confirm the full default, and a retained canonical PR rule cannot confirm removal. Fresh warnings are diagnostic evidence only. Incomplete acquisition preserves export destinations and makes no Ruleset write; later failures retain earlier or uncertain effects.

Private `merge_methods_normalization` metadata requires a string path and present typed raw/fallback lists, rejects empty, forged and ID-conflicting evidence even beside valid companion records, and never supplies apply input or proves acquisition completeness. Status-only Summary hides policy/provenance; the existing collection count stays one.

### Ruleset scanning Alerts observation normalization

Each tool in a complete code-scanning rule acquires `alerts_threshold` as a string. Exact `none`, `errors`, `errors_and_warnings` and `all` choices remain unchanged. Other strings, including empty and whitespace values, project to the named `errors` fallback while retaining the exact raw value under the positive Ruleset ID and exact tool name. Tool names are identities: no trimming, case folding, substitution or synthetic tool is introduced. Canonical tools and each tool's Security alerts choice remain intact.

Missing/null/non-string thresholds or names, invalid/duplicate names, malformed companion fields and incomplete acquisition remain errors. Security alerts retain their own independent [typed normalization policy](#ruleset-scanning-security-alerts-observation-normalization). An absent scanning rule stays absent; a present empty tool list stays empty. Each declared tool still requires its own Alerts and Security alerts choices; observed fallbacks do not fill omitted declarations.

Requested export saves effective policy and warns after complete acquisition. Scanning Alerts warnings precede Security alerts, ordered by exact tool name. Each includes Ruleset ID/name and `tool="name"` context. Selected fallback-equal or different choices require one whole-object update. Removing a tool or replacing its name retains the old tool's warning and `inactive (tool removed; write required)` decision; removing the scanning rule shows `inactive (rule removed; write required)`. Plan and detailed Summary show one effective policy. Unlisted Ruleset IDs stay unmanaged and undisclosed.

Original acknowledgment and an independent fresh GET must confirm the exact desired whole policy. A retained unknown Alerts string cannot confirm Errors; a retained canonical tool or scanning rule cannot confirm removal. Contextual fresh warnings never supply successful verification values. Failed acquisition makes no Ruleset write and preserves export destinations; later failure retains honest reporting of earlier or uncertain effects. Private `scanning_alerts_normalizations` records are presentation evidence, not apply input or acquisition proof; status-only output stays private and collection count remains one.

### Ruleset scanning Security alerts observation normalization

Each complete tool's `security_alerts_threshold` string maps exactly to `none`, `critical`, `high_or_higher`, `medium_or_higher` or `all`. Other strings, including empty and whitespace values, project to the named `high_or_higher` fallback. Keep the exact raw string under the positive Ruleset ID and exact tool name. No identity default, trimming or case folding applies. The [scanning Alerts contract](#ruleset-scanning-alerts-observation-normalization) also governs complete acquisition, tool/rule ownership, removal, replacement and honest failure; Security alerts retain a separate record and choice.

Both thresholds may need normalization on the same tool. Retain both independently, keep canonical companions exact and preserve absent rules and explicit empty tool lists. Missing/null/non-string thresholds, invalid or duplicate tool identities and incomplete acquisition remain errors rather than fallback values. Desired Alerts and Security alerts remain explicitly required on every declared tool.

Warnings and UI decisions order Alerts records first, then Security alerts records, each by exact tool name, before bypass modes. Selected fallback-equal intent stays write-required. Original acknowledgment and independent fresh GET must confirm the exact desired thresholds or actual removal: retained unknown strings cannot confirm High or higher, and retained canonical tools or rules cannot confirm absence. Plan and detailed Summary keep one effective policy; private `scanning_security_alerts_normalizations` records never supply apply input or prove acquisition. Status-only hides context and values, unlisted Rulesets stay unmanaged, and collection count remains one.

### Ruleset bypass-mode observation normalization

A complete acquired `bypass_mode` string outside the native `always`, `exempt` and `pull_request` choices projects to the UI default `always`. Keep its exact raw string, including empty/whitespace values, bound to the Ruleset ID and the actor's exact type and optional numeric ID. Canonical companions keep their own identities and choices. Missing/null modes, malformed identities, duplicate actors, incomplete acquisition and unsupported actor/target applicability remain errors; normalization never invents an actor, role, grant or ID.

Requested export warns after complete acquisition and saves only effective UI policy. Warnings follow Ruleset name order, then enforcement, Required approvals, Allowed merge methods, scanning Alerts and Security alerts (each in tool-name order), bypass actors ordered by type and numeric ID. Actor diagnostics include `actor_type` and include `actor_id` only when present. These are identity context, not public YAML aliases. Unlisted Ruleset IDs stay unmanaged and undisclosed in plan/apply.

Selected normalized equality still requires one whole-object update. Plan and detailed Summary show the raw warning and `always` marked as a normalized observation, followed by the desired UI mode marked write-required. Removing an actor, explicitly or through existing whole-object declaration defaults, shows `inactive (actor removed; write required)`. Each operation has one effective-policy line; canonical equal objects remain no-ops. Observation fallback does not populate omitted desired resources or change whole-object ownership.

Apply verifies the original mutation response and an independent fresh GET against the desired observable policy. A canonical acknowledgment followed by the same unknown raw mode cannot confirm `always`; a retained canonical actor cannot confirm removal. Fresh warnings require complete owned readback and never supply verification values. Incomplete acquisition makes no Ruleset write and preserves export destinations; a later failure reports possible and earlier completed effects without implying rollback.

Private same-version reports retain actor-bound `bypass_mode_normalizations`, validate each record independently beside scalar evidence, and omit nil actor IDs rather than accepting JSON null. They preserve status-only privacy and the existing collection count one; records never enter public YAML or apply input and cannot prove acquisition completeness.

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

The scope correction on 2026-11-09 follows the user's observed availability of **Require code quality results** in a personal repository. The [official Rulesets reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-code-quality-results) documents the rule and Free public Rulesets, and the [public REST API](https://docs.github.com/en/rest/repos/rules#update-a-repository-ruleset) documents its severity field. Product entitlement descriptions are not evidence that this individual control is unavailable. The rule requires configured Code Quality analysis; managing it does not enable the product, start analysis, create workflows or change billing. Coverage uses the separate [coverage threshold rule](#code-coverage-thresholds). See [Code Quality normalization](#ruleset-quality-severity-observation-normalization).

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
account acceptance or enforcement. See the [coverage evidence](../contributing/settings-coverage.md)
and [behavior specification](../features/rulesets.feature.md#code-coverage-thresholds).

### Unimplemented branch controls

Public YAML does not accept `require_a_pull_request_before_merging.require_review_from_specific_teams`, including empty collections and explicit false. The personal-public Ruleset editor observed on 2026-11-13 visibly offered the **Require review from specific teams** checkbox, despite official documentation describing the control as unavailable for user-owned repositories. The checkbox was not expanded, so complete usable team selection and the personal-repository API contract remain unconfirmed; this discrepancy does not establish a policy exclusion or implemented support. Organization team ownership and access management remain outside the target. The [official rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) and [coverage documentation](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/restrict-code-coverage) provide the control context. This policy change neither makes those keys valid nor disables live settings.

Acquired nonempty PR `required_reviewers` and unknown or malformed rules still fail complete acquisition without dropping protection. Ordinary code-scanning merge protection, Code Quality thresholds, PR approval counts, public fork approval and automatic Copilot code review remain separate controls. All existing Ruleset updates independently recheck ownership, identity and complete policy before PUT; changes to additional approval, PR-rule presence, Code Quality presence or raw severity, and Coverage presence or either threshold since acquisition stop the update. Preflight is not an atomic lock: concurrent changes between GET and PUT can still race, and post-write failures report that writes may have applied without rollback.

### Unsupported Push Rulesets

`target_pushes` and the `restrict_file_paths`, `restrict_file_path_length`, `restrict_file_extensions` and `restrict_file_size` controls are not accepted, including false or empty declarations. [GitHub limits Push Rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository) to paid-plan private/internal repositories and their forks, outside the [personal-public target](scope.md#feature-adoption-scope) because of repository visibility, not price. No compatibility alias or automatic migration is provided.

An acquired repository-owned Push Ruleset or file-restriction rule fails complete acquisition. Export preserves its destination; plan, updates and explicit deletion stop before a Rulesets write. Unsupported policy is never silently omitted or converted into a branch/tag policy. Inherited organization/enterprise policy remains outside ownership.

### Plan, apply, and export

Complete supported observations, including bypass visibility, are required. Unknown writable fields/rules, missing values, invalid types, unreadable details and failed pages stop mutation/export. Inherited organization/enterprise Rulesets remain outside mutation ownership. Repository-owned Push Rulesets are unsupported and stop complete acquisition before any Rulesets mutation or export. Other protection can still affect effective behavior.

Without prune, plan shows only listed creates/ID updates and default-derived reductions. Explicit prune additionally shows its acquired deletion candidates. A nonempty collection counts once as managed. Apply reads fresh state after earlier families, prepares selected operations before the first Rulesets write and writes in stable name order. Fresh comparable matches make no write.

POST/PUT responses must confirm identity, ownership and requested comparable policy. Every create/update, including basic and empty-rule policies, then requires an independent detail GET with `includes_parents=false` to confirm the same ID, repository ownership and complete requested policy. Missing, denied, redirected or mismatching readback returns nonzero and stops later operations, including prune deletions: writes may already have applied. No retry or rollback occurs. A reported positive creation ID remains in recovery diagnostics when verification fails, but is not automatically written to YAML. Only verified creations follow [guarded writeback](labels.md#apply-id-writeback).

Unsupported protection is never omitted to produce a usable policy. Acquired Evaluate enforcement, merge queue, metadata restrictions, license compliance, or a present review-dismissal restriction stop Ruleset export, comparison and mutation. Review-dismissal absence in an API response does not prove the stored restriction is disabled. These controls are not accepted configuration. See the [Rulesets specification](../features/rulesets.feature.md) and [UI mapping](../contributing/ui-ruleset-mapping.md).
