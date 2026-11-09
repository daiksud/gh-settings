---
type: Feature
title: Declarative repository Rulesets
description: Select whole Ruleset objects by numeric ID without deleting unlisted objects.
---

## Status and goal

Rulesets use ID-selected whole-object ownership. The [configuration reference](../settings.md#rulesets) owns schema, fixed defaults and observation limits. Historical [ADR 0025](../adr/0025-declarative-repository-rulesets.md) remains a record of the superseded ownership decision.

## Ownership and identity

- Given omitted Rulesets or an empty list, validate/plan/apply select no object and acquire no Rulesets state or target solely for that resource.
- Given a positive declared REST ID, update only that supported repository-owned branch/tag object. Missing or wrong repository/type IDs fail; never create as fallback.
- Given no ID, create only a new name. A collision fails with the existing ID and guidance to add it; never adopt by name.
- Given a valid ID and noncolliding new name, rename the same object. Unlisted names/IDs remain untouched and are never implicitly deleted.
- Reject duplicate declaration IDs or exact names. Preserve existing exact-name comparison; no heuristic case-folding or trimming is introduced.

## Schema and presence

Current UI entries live under `repository.settings.rulesets`. They require `ruleset_name`, `enforcement_status` (`active` or `disabled`) and exactly one `target_branches`/`target_tags` object. Branch selectors are `include_default_branch`, `include_all_branches`, `include_by_pattern` and `exclude_by_pattern`; tags use `include_all_tags` and pattern arrays. Patterns use UI spelling without internal ref prefixes. An explicit empty target object yields empty include/exclude lists; no target or protected scope is guessed.

Omitted `bypass_list` and unselected `rules` become empty whole-object policy. Each bypass entry selects one supported role (`repository_admin`, `maintain`, `write`), numeric team/app/user, or `deploy_keys: true`, and requires `bypass_mode` (`always`, `exempt`, `for_pull_requests_only`) with existing applicability checks. No actor/grant is created. Rules contain five basic boolean checkboxes and optional parameter groups; an omitted group removes its rule rather than retaining remote membership.

PR-rule booleans default false, `required_approvals` defaults zero (UI range 0–10), reviewer/check lists default empty and allowed merge methods default all three. `require_review_from_specific_teams` entries require numeric `reviewer`, `approvals` and `file_patterns`. Checks use `check` plus optional `source: {any_source: true}` or `{app: positive-id}`; omitted source means any source, not retained App binding. The configuration reference owns the full exact field inventory.

Null, unknown keys and wrong types remain invalid locally. Explicit false, zero and empty are not omitted. Reviewer pattern order and repetitions remain meaningful; independent identity-based collections remain canonical sets. Any-source status checks retain their documented distinct identity from App-bound checks.

`rules.require_code_scanning_results` requires complete `code_scanning_tools` entries with `tool_name`, `alerts` and `security_alerts`. Identities are case-sensitive. `rules.require_code_quality_results` requires explicit UI `severity` (`errors`, `warnings_and_higher`, `notes_and_higher`, `all`). Neither rule supplies defaults or borrows missing parameters from a prior policy; cross-rule parameters are invalid. Tool collections are canonical sets, including explicit empty arrays. The configuration reference owns the exact enums. Complete API reads, export, plan and Action reports retain both rules alongside all other supported policy.

## Complete observation and export

Reads require complete repository ownership/ID, required supported policy, bypass visibility and every page/detail. Unknown writable fields, unsupported rules, missing required values, null values, denied reads and partial data fail rather than becoming defaults or success. A present `dismissal_restriction` and acquired `evaluate` enforcement fail as unsupported protection; neither is stripped or normalized into an editable policy. Absence of the restriction in an API response does not prove it is disabled. Declaration defaults must never be applied to an API read.

Merge queue, metadata-pattern, license-compliance and secret-scanning rules are unsupported. A returned unsupported rule or parameter, including a review-dismissal restriction, fails the complete collection before export, comparison, update or explicit deletion. Validation rejects the corresponding unsupported YAML controls before remote work. The supported PR rule and its remaining review settings are retained.

Opted-in export includes numeric IDs plus observed comparable policy. Ordinary export adds no Rulesets request. Export → validate → plan against unchanged exact supported comparable state yields zero differences. A normalized enforcement, Required approvals or Quality severity fallback remains actionable, with ID/name-scoped export warnings in name order. Other whole-object defaults and unselected-object ownership remain unchanged. Failed export preserves the destination. Inherited resources remain outside mutation ownership; repository-owned Push policy fails as unsupported. Server-generated metadata is explicitly classified and never replayed as writable policy.

## Plan, apply and idempotence

Plan is read-only and reports only selected objects, with their IDs and supported comparable before/after values. Unlisted objects are not diff entries. Empty selection counts zero, nonempty selection counts once. Strict same-version Action reports reject incomplete objects, missing/mismatching IDs, duplicate identities and malformed deletion operations; Summary escapes values without credentials.

Apply reads fresh state after earlier resources and prepares every selected Ruleset operation before the first Rulesets write. POST/PUT never send YAML IDs as payload fields. Every response must confirm positive ID, target repository ownership and requested comparable policy; updates retain the selected ID. Normalized enforcement/Required approvals/Quality updates and writes adding, updating or removing creation, update, pull-request, deployment, code-scanning, code-quality, Copilot-review, coverage rules, require independent post-write GET identity/policy verification. Changed comparable values or failed readback stop later work with uncertain-write semantics. Only explicit positive-ID/name `delete: true` entries permit DELETE; no hidden retry, rollback or saved-plan execution exists.

Confirmed created IDs are appended immediately to the original YAML through the shared guarded recorder. Repeat apply then selects that ID, avoiding duplicate creation. Explicit no-writeback and GitHub Actions leave configuration unchanged and display IDs for manual adoption. A successful remote create followed by local persistence failure is a partial failure with recovery ID; earlier confirmed label/Ruleset IDs remain available. Unverified response IDs are reported but never persisted as confirmed creation. Output failure retains ID diagnostics on stderr. No Git commit/push is made.

## API, access and availability

The adapter pins API version `2026-03-10`, uses bounded requests and refuses redirects. [Repository Rulesets REST](https://docs.github.com/en/rest/repos/rules) owns request permissions/shapes; complete bypass visibility requires Ruleset write access beyond ordinary public listing. Supported enforcement GET strings and other complete unknown values follow the [active ID-bound normalization policy](../settings.md#ruleset-enforcement-observation-normalization). Quality severity follows its [active policy](../settings.md#ruleset-quality-severity-observation-normalization). [Required approvals](../settings.md#ruleset-required-approvals-observation-normalization) retains typed integer provenance; [Allowed merge methods](../settings.md#ruleset-allowed-merge-methods-observation-normalization) retains typed ordered list provenance; [Team reviewer Approvals](../settings.md#ruleset-team-reviewer-approvals-observation-normalization) retains exact Team-bound counts; [bypass modes](../settings.md#ruleset-bypass-mode-observation-normalization) retain exact actor-bound strings. Scanning [Alerts](../settings.md#ruleset-scanning-alerts-observation-normalization) and [Security alerts](../settings.md#ruleset-scanning-security-alerts-observation-normalization) retain independent tool-bound strings. Account/plan eligibility and team-feature prerequisites remain GitHub's responsibility. Do not expand credentials or change live protection to make a test pass.

Fresh reads and response verification do not lock concurrent edits. Whole-object management can overwrite intervening edits to supported selected settings; unknown writable fields and other unsupported policy values cause failure rather than silent erasure. Fixtures establish client behavior, not live entitlement, server mutation success or atomicity.

## Implementation stages and acceptance plan

One focused breaking Rulesets change builds on labels' ID recorder. Environment uses a separate name-identity/upsert contract and is not implemented by this feature. Topics and unrelated scalar repository fields retain their own ownership. No release or live policy mutation is part of fixture acceptance.

## Existing test mapping and evidence limits

| Boundary | Tests and behavior |
| --- | --- |
| Model | [ID selection/defaults](../../internal/settings/ruleset_ids_test.go), [validation/copies](../../internal/settings/rulesets_test.go), [parameters](../../internal/settings/ruleset_parameters_test.go): collision guidance, wrong IDs, no implicit deletion, whole supported values, create/update defaults, ordered patterns and strict read state. |
| YAML | [Codec](../../internal/workspace/ui_rulesets_test.go): positive numeric ID round-trip, strict types/null/unknown rejection and aliases. |
| API | [Inventory/payload](../../internal/github/rulesets_test.go), [identity](../../internal/github/ruleset_identity_test.go), [parameters](../../internal/github/ruleset_parameters_test.go): complete pages/details, owner/type isolation, ID-free complete writes, response mismatch/redirect/refusal. |
| Application | [Sequencing](../../internal/app/rulesets_test.go), [ID recording](../../internal/app/ruleset_ids_test.go): empty no-op, fresh preparation, partial failure, confirmed recorder ordering and earlier-label recovery. |
| Action | [Whole-object report](../../internal/actions/rulesets_test.go), [parameters](../../internal/actions/ruleset_parameters_test.go): complete selected policy/IDs, native false/zero/order, no unlisted values and escaped Summary. |
| Built process | [Lifecycle](../../tests/acceptance/rulesets_test.go), [ID modes/recovery](../../tests/acceptance/ruleset_ids_test.go), [parameters](../../tests/acceptance/ruleset_parameters_test.go): export/zero plan, rename, unmanaged siblings, writeback/repeat, opt-out/Actions, partial failures and response loss. |
| Code analysis rules | [Model](../../internal/settings/ruleset_code_analysis_test.go), [YAML](../../internal/workspace/ui_rulesets_test.go), [API](../../internal/github/ruleset_code_analysis_test.go), [Action](../../internal/actions/ruleset_code_analysis_test.go), [built process](../../tests/acceptance/ruleset_code_analysis_test.go): exact thresholds, closed complete reads, complete existing-policy export, zero plan, exact writes, independent readback failures, escaped reports and repeat no-op. The shared [fixture](../../tests/fixtures/ruleset-code-analysis.json) retains all seven rules from a read-only public snapshot with repository identity normalized. |
| Unsupported controls and acquired protection | [Built-process rejection](../../tests/acceptance/ruleset_unsupported_controls_test.go): `TestRemovedRulesetDeclarationsFailBeforeRemoteWork` rejects unsupported declarations before remote requests; `TestUnsupportedRulesetReadsBlockExportUpdateAndDeletion` rejects unsupported observed enforcement/rules/parameters without export replacement or update/delete writes. |

All cases use local HTTP/files/Git and dummy credentials. Hosted candidate/required CI and review results must be inspected separately; local tests are not evidence of live settings changes or publication.

The current YAML has no API-shaped Ruleset aliases or legacy parser. Independent UI maps in process tests are checked against exact HTTP expectations. CLI and detailed Summary project selected objects to UI names and choices; serialized private reports retain native policy identities.

## Rule: Keep normalized enforcement distinct from raw state

- Given a complete typed out-of-UI enforcement string, including unknown, empty or whitespace but excluding unsupported `evaluate`, retain exact raw provenance per positive ID and project Disabled. Missing/null/type and later nested/entry failures return no partial observation; pure readers stay silent.
- Export warns only after complete comparable acquisition, in deterministic name order with ID/name context. A later UI conversion failure preserves legitimate warnings but does not save.
- Plan only declared IDs. A fallback-equal selected object still requires one whole update; unlisted normalized objects are not printed or written. Ordinary equality, collision and create-only contracts remain unchanged.
- Detailed output labels the before status as normalized and the object as effective policy, never raw evidence. Closed private metadata rejects forged/no-op/create/ID-conflicting entries; Summary escapes HTML and status-only suppresses object/raw context. Existing collection count stays one.
- Apply emits selected pre-write provenance and only complete fresh GET diagnostics after verified identity/ownership. Strict mutation response and independent GET must confirm desired whole policy; retained unknown values cannot confirm a fallback. A canonical acknowledgment alone is insufficient. Prior REST effects remain reported on failure; malformed preflight makes no Ruleset write and export preserves the destination.

Test mapping: [ID-bound carrier](../../internal/settings/ruleset_enforcement_observation_test.go), [closed changes](../../internal/settings/ruleset_enforcement_change_test.go), [selected reconciliation](../../internal/settings/ruleset_enforcement_reconcile_test.go), [HTTP acquisition](../../internal/github/ruleset_enforcement_read_test.go), [strict fresh verification](../../internal/github/ruleset_enforcement_verification_test.go), [scoped diagnostics](../../internal/github/ruleset_enforcement_warning_test.go), [export scope](../../internal/app/normalization_ruleset_enforcement_export_test.go), [private report/HTML/privacy](../../internal/actions/normalization_ruleset_enforcement_test.go), [executable commands](../../tests/acceptance/normalization_ruleset_enforcement_test.go).

## Rule: Keep normalized Quality distinct from missing or removed rules

- Given a complete present code-quality rule, canonical API `errors`, `warnings`, `notes` and `all` map to the four UI choices. Other typed severity strings project Errors with exact ID-bound provenance; absent rules stay absent, and missing/null/type/duplicate/incomplete policy fails without partial acquisition.
- A fixed concrete aggregate retains simultaneous enforcement and Quality provenance. Plan only selected IDs and make one whole-object update even for fallback equality. Contextual diagnostics order enforcement, Required approvals, then Quality; effective-policy display is emitted once.
- Desired Quality removal shows inactive/rule-removed intent, never an invented after severity. Strict original mutation response and independent GET must confirm actual absence; retained canonical or unknown rules cannot confirm removal. Unknown readback cannot confirm Errors even with a canonical acknowledgment.
- Same-version private report metadata validates each concrete record, including present-zero rejection, and never supplies mutation values. Status-only hides provenance/object details; the existing collection count remains one. Canonical no-op, unlisted privacy, prior REST effects, export preservation remain unchanged.

Test mapping: [mapping](../../internal/settings/ruleset_quality_mapping_test.go), [carrier](../../internal/settings/ruleset_quality_observation_test.go), [closed changes](../../internal/settings/ruleset_normalizations_change_test.go), [reconciliation](../../internal/settings/ruleset_quality_reconcile_test.go), [HTTP acquisition](../../internal/github/ruleset_quality_read_test.go), [strict confirmation/removal](../../internal/github/ruleset_quality_verification_test.go), [diagnostics](../../internal/github/ruleset_quality_warning_test.go), [private metadata](../../internal/actions/normalization_ruleset_quality_test.go), [export](../../internal/app/normalization_ruleset_quality_export_test.go), [executable commands](../../tests/acceptance/normalization_ruleset_quality_test.go).

## Rule: Distinguish normalized Required approvals from acquired zero

- Acquire a complete integer from each present PR rule. Keep UI 0–10 exact; project other integers to 0 with ID-bound typed provenance. Missing/null/fractional/string/overflow fields fail without partial results; absent rules stay absent.
- Selected normalized equality requires one whole-object update, with independent enforcement and Quality records preserved. Plan/Summary show unquoted integer choices or inactive PR-rule removal and one effective policy. Warning order is enforcement, Required approvals, Quality.
- Verify exact original mutation response and fresh GET. A canonical acknowledgment followed by same-ID/name raw 11/-1 cannot prove desired 0; retaining a canonical PR rule cannot confirm removal. Do not retry or roll back uncertain effects.
- Same-version reports require explicit typed metadata members and reject forged, present-zero, create, ID-conflicting and unsupported desired counts. Status-only hides object/provenance. Count one, omitted/unlisted privacy and prior-resource failure semantics remain unchanged.

Test mapping: [mapping](../../internal/settings/ruleset_required_approvals_mapping_test.go), [carrier](../../internal/settings/ruleset_required_approvals_observation_test.go), [closed changes](../../internal/settings/ruleset_required_approvals_change_test.go), [reconciliation](../../internal/settings/ruleset_required_approvals_reconcile_test.go), [integer JSON](../../internal/settings/ruleset_required_approvals_json_test.go), [acquisition](../../internal/github/ruleset_required_approvals_read_test.go), [strict verification/removal](../../internal/github/ruleset_required_approvals_verification_test.go), [diagnostics](../../internal/github/ruleset_required_approvals_warning_test.go), [private reports](../../internal/actions/normalization_ruleset_required_approvals_test.go), [export](../../internal/app/normalization_ruleset_required_approvals_export_test.go), [built executable](../../tests/acceptance/normalization_ruleset_required_approvals_test.go).

## Rule: Retain actor-bound bypass-mode provenance

- A complete unsupported mode string projects to `always` with exact Ruleset/actor identity and raw evidence. Canonical actors stay unchanged; nil IDs are not numeric IDs. Missing/null modes, malformed identities, duplicate actors, unsupported applicability and later failed acquisition produce no partial observation.
- Requested export warns in name order; selected plan/apply disclose only managed IDs. Within an object, scalar warnings precede bypass records sorted by actor type and numeric ID. Pure readers remain silent.
- Fallback-equal intent still requires one whole-object update. Actor removal retains its original warning and an inactive/actor-removed decision. Canonical equal policy remains unchanged and does not write.
- A canonical PUT acknowledgment does not establish a successful update: strict fresh GET must confirm raw desired policy and removed actor absence. Fresh complete diagnostics cannot turn unchanged unknown modes or retained canonical actors into success.
- Private report round-trip preserves closed actor records and owned IDs with scalar companions, detailed escaping and status-only privacy. It never becomes apply input or proof of complete acquisition; collection count remains one.

Test mapping: [owned carrier](../../internal/settings/ruleset_bypass_observation_test.go), [reconciliation](../../internal/settings/ruleset_bypass_reconcile_test.go), [reader](../../internal/github/ruleset_bypass_read_test.go), [strict confirmation](../../internal/github/ruleset_bypass_verification_test.go), [fresh diagnostics](../../internal/github/ruleset_bypass_readback_warning_test.go), [private reports](../../internal/actions/normalization_ruleset_bypass_test.go), [presentation](../../internal/presentation/ruleset_bypass_test.go) and [built executable](../../tests/acceptance/normalization_ruleset_bypass_test.go).

## Rule: Retain raw Allowed merge methods lists

- Given a complete PR rule with a nonempty unique typed list, any unsupported string projects the whole list to Merge, Squash, Rebase. Keep exact raw order; empty/whitespace strings are acquired values. Missing/null/non-string members, empty lists, duplicates and incomplete policies fail without partial evidence.
- Absent PR rules stay absent. Selected fallback-equal, subset and actual PR-rule removal intents retain provenance and require one whole-object update; declaration defaults remain distinct from observed fallback. Unlisted IDs and omitted collections are silent no-ops.
- Export saves effective UI choices and emits scoped warnings. Plan/detailed Summary preserve raw order and render desired choices in named UI order. Closed private reports retain typed arrays and reject unbound/forged evidence; status-only output hides details and collection count remains one.
- Original PUT acknowledgment and independent raw GET must confirm the desired policy. Retained unknown methods cannot confirm the default; retained canonical PR rules cannot confirm removal. Failure preserves honest earlier/uncertain effects and caller YAML.

Test mapping: [owned state](../../internal/settings/ruleset_merge_methods_observation_test.go), [reconciliation](../../internal/settings/ruleset_merge_methods_reconcile_test.go), [pure projector](../../internal/github/normalization_ruleset_merge_methods_test.go), [complete acquisition](../../internal/github/ruleset_merge_methods_read_test.go), [strict confirmation](../../internal/github/ruleset_merge_methods_verification_test.go), [fresh warnings](../../internal/github/ruleset_merge_methods_readback_warning_test.go), [private reports](../../internal/actions/normalization_ruleset_merge_methods_test.go), [presentation](../../internal/presentation/ruleset_merge_methods_test.go) and [built executable](../../tests/acceptance/normalization_ruleset_merge_methods_test.go).

## Rule: Retain Team-bound reviewer approval counts

- A complete typed count outside 0–10 projects to 0 with exact Ruleset/Team ID provenance. Keep canonical companions and ordered, repeated, negated or explicitly empty file patterns intact. Missing/null/fractional/overflow values, malformed identities or patterns, duplicate reviewers, more than 15 reviewers and incomplete acquisition fail without partial state.
- Absent PR rules and empty reviewer lists invent no reviewer. Desired team entries still require explicit approvals and file patterns. Unlisted Ruleset IDs remain private and unmanaged.
- Selected fallback-equal or differing counts require one whole-object update. Reviewer removal, replacement by another ID and PR-rule removal retain the original warning and accurate inactive decisions. Numeric reviewer-ID order follows Allowed merge methods and precedes Quality, with one effective-policy display.
- Original acknowledgment and independent fresh GET must confirm raw desired policy. Unknown counts cannot prove 0; retained canonical reviewers or PR rules cannot prove removal. Fresh contextual warnings do not turn failed confirmation into success.
- Typed private records preserve explicit zero, reject malformed or forged metadata independently of valid companions, own policy/record data and keep status-only output private. Reports never supply apply input; collection count remains one.

Test mapping: [owned carrier](../../internal/settings/ruleset_team_approvals_observation_test.go), [closed changes](../../internal/settings/ruleset_team_approvals_change_test.go), [reconciliation](../../internal/settings/ruleset_team_approvals_reconcile_test.go), [pure projector](../../internal/github/normalization_ruleset_team_approvals_test.go), [complete reader and fresh diagnostics](../../internal/github/ruleset_team_approvals_read_test.go), [strict confirmation](../../internal/github/ruleset_team_approvals_verification_test.go), [pre-write warnings](../../internal/github/ruleset_team_approvals_warning_test.go), [private reports](../../internal/actions/normalization_ruleset_team_approvals_test.go), [presentation](../../internal/presentation/ruleset_team_approvals_test.go) and [built executable](../../tests/acceptance/normalization_ruleset_team_approvals_test.go).

## Rule: Retain exact tool-bound scanning Alerts strings

- Complete canonical Alerts strings remain exact; unsupported strings, including empty/whitespace, project Errors with raw provenance under the exact tool name and Ruleset ID. Preserve canonical tools and Security alerts companions. Never repair identity, missing fields, malformed types, duplicates or incomplete acquisition.
- An absent scanning rule and an explicit empty tool list remain distinct. Declared tool choices stay required. Selected normalized equality, different intent, tool removal/name replacement and scanning-rule removal each retain one whole-object update; unlisted IDs remain private.
- Tool-name-ordered warning/decision lines follow Team reviewer Approvals and precede Quality. Removed tools and removed rules have distinct inactive decisions. Canonical acknowledgment plus retained unknown raw GET cannot confirm Errors; retained canonical objects cannot confirm removal. Failed confirmation reports uncertain effects.
- Private reports preserve explicit empty raw strings, independently validate every concrete record, own data and keep status-only output private. They never supply mutation input or prove acquisition completeness. Collection count remains one.

Test mapping: [mapping/state/closed changes/reconciliation](../../internal/settings/ruleset_scanning_alerts_observation_test.go), [projector and complete acquisition](../../internal/github/ruleset_scanning_alerts_read_test.go), [strict confirmation and scoped diagnostics](../../internal/github/ruleset_scanning_alerts_verification_test.go), [private reports and presentation](../../internal/actions/normalization_ruleset_scanning_alerts_test.go), [export](../../internal/app/normalization_ruleset_scanning_alerts_export_test.go) and [built executable](../../tests/acceptance/normalization_ruleset_scanning_alerts_test.go).

## Rule: Retain independent tool-bound Security alerts strings

- Complete supported Security alerts map exactly to None, Critical, High or higher, Medium or higher or All. Other strings, including empty/whitespace, project High or higher while retaining exact tool-name-bound raw evidence. Missing/null/type/identity/duplicate/pagination failures remain acquisition errors.
- Alerts and Security alerts may both be unknown on one tool. Preserve both concrete records and canonical companions independently; neither changes tool identity or fills an omitted desired choice. Absent rules and empty lists remain distinct.
- Selected equality, differing choices, tool removal/name replacement and rule removal require one whole-object update with independently ordered contextual warnings. Strict acknowledgment and fresh GET confirm exact raw desired policy or removal, never projected equality.
- Private metadata keeps explicit empty raw strings, validates both record families independently and preserves owned policy, escaping and status-only privacy. Collection count remains one.

Test mapping: [mapping/state/closed changes/reconciliation](../../internal/settings/ruleset_scanning_security_alerts_observation_test.go), [projector and complete acquisition](../../internal/github/ruleset_scanning_security_alerts_read_test.go), [strict confirmation and scoped warnings](../../internal/github/ruleset_scanning_security_alerts_verification_test.go), [private reports/presentation](../../internal/actions/normalization_ruleset_scanning_security_alerts_test.go), [export](../../internal/app/normalization_ruleset_scanning_security_alerts_export_test.go) and [executable commands including simultaneous thresholds](../../tests/acceptance/normalization_ruleset_scanning_security_alerts_test.go).

## Restrict creations and updates

- Branch and tag Rulesets map `restrict_creations: true` to parameterless `creation` and `restrict_updates: {allow_fetch_and_merge: false}` to `update` with its explicit API boolean. Empty update objects default false only in declarations.
- Complete export preserves true and false upstream-sync values. Missing/null/wrong-type parameters and unknown companion fields fail without partial observation or writes. Parameters from another rule are rejected.
- Executable export, validate, plan, apply and repeat no-difference behavior use independent literal payload expectations in `TestRulesetCreationUpdatesExecutableLifecycle`. A mutation acknowledgment that claims removal while fresh GET retains either restriction fails. Readback covers both previous and desired policies.
- `TestRulesetUpdateAcquisitionRequiresBoolean` covers incomplete acquisition, wrong-rule parameters and owned conversion. These local API fixtures neither change live protections nor establish account eligibility or effective enforcement.

## Required deployments

- `require_deployments_to_succeed.required_deployment_environments` is a required explicit list on branch and tag Rulesets. Empty lists remain selected; omission removes the rule. Invalid, blank or duplicate names and mixed-rule parameters fail before remote work.
- Complete acquired environment names export, validate and plan without loss; canonical ordering does not change membership. Unknown fields and incomplete reads fail without a partial policy.
- Apply checks the exact declared whole-object payload, mutation acknowledgment and independent GET, including deployment-rule removal. A stale GET fails with uncertain-write reporting; a matching repeat produces no differences or writes.
- [Domain](../../internal/settings/ruleset_deployments_test.go), [transport](../../internal/github/ruleset_deployments_test.go) and [built-product](../../tests/acceptance/ruleset_deployments_test.go) tests use local fixtures and dummy credentials; they do not establish live GitHub enforcement or entitlement.

## Automatically request Copilot code review

- The branch-only `automatically_request_copilot_code_review` group selects `copilot_code_review`. Review new pushes and Review draft pull requests map to independently acquired booleans; an empty desired group defaults both false, while omission removes the rule.
- Missing, null, malformed or unknown acquired options fail without partial output. Copilot parameters on unrelated rules are rejected, and returned checkbox pointers are owned.
- Updates and removals require strict mutation acknowledgment and a separate GET confirming the complete desired policy, selected ID and repository ownership. A retained old policy, incomplete GET or denied read fails with uncertain-write reporting.
- `TestCopilotRulesetUIAndExport`, `TestCopilotRulesetAcquisitionFailsClosed` and `TestCopilotRulesetStrictIndependentReadback` cover mapping, export roundtrip, validation, ownership and failure cases. `TestRulesetCopilotExportValidatePlanApplyReadback` exercises the built CLI through export, validate, plan, apply, independent readback and repeat no writes with exact independently authored payloads.
- Local fixtures use dummy tokens. They do not establish actual Copilot review execution, licensing, AI credits or live protection.

## Code coverage

- Branch Rulesets can select `rules.restrict_code_coverage` with explicit finite 0–100 values for `minimum_line_coverage_percentage` and `maximum_line_coverage_drop`; fractional values and zero are retained exactly. A tag target or cross-rule parameters fail validation.
- Export preserves both acquired thresholds and revalidates to no differences. Missing, null, wrong-type or unknown parameters fail the entire observation rather than becoming zero or disappearing from whole-object updates.
- Add/update/remove checks the original acknowledgment and an independent ID- and repository-bound GET. Acknowledged but absent, stale or malformed readback reports uncertain effects. Repeated successful apply does not write.
- [Domain tests](../../internal/settings/ruleset_coverage_test.go) own threshold bounds and copying; [codec tests](../../internal/workspace/ui_ruleset_coverage_test.go) own numeric YAML types; [transport tests](../../internal/github/ruleset_coverage_test.go) own incomplete acquisition and fresh verification, including removal. [Executable coverage tests](../../tests/acceptance/ruleset_coverage_test.go) check export, validate, plan, apply, readback failure and no-difference repetition against a local API with dummy credentials. No live protection, coverage uploads or plan entitlement is changed or verified.

The [public contract](../settings.md#code-coverage-thresholds) owns Code Quality, coverage-upload and Team/Enterprise Cloud prerequisites. Organization/Enterprise policy is excluded even when a repository rule requires those subscriptions.

## Explicit deletion

- `{id: 7, ruleset_name: basic, delete: true}` selects one tombstone. False, null, absent/nonpositive ID, absent name and selected policy controls fail before remote work. Empty/omitted membership never deletes. The name must match when the ID exists; name collisions never redirect deletion.
- Complete acquisition retains excluded inherited IDs to distinguish unmanaged identity from genuine absence. Unsupported, malformed, unknown or normalized protection prevents deletion. Every selected operation is prepared before the first Ruleset write.
- A delete change has one complete selected before policy and no after policy. CLI plan, private report and detailed Summary show `delete` to `absent`; export writes only present policies.
- Before DELETE, exact detail GET must retain repository ownership, selected positive ID and unchanged complete policy. Only HTTP 204 followed by fresh complete inventory absence succeeds. An ambiguous 404, retained ID, denied/partial/null inventory or failed pagination reports uncertainty; no automatic retry follows.
- Repeating plan/apply after confirmed inventory absence produces no write. Unlisted objects survive. A failed deletion after earlier repository changes remains a partial failure with the deletion's remote effect unknown.
- `TestRulesetExplicitDeletionExecutableLifecycle` verifies branch/tag export, invalid token-free validation, read-only plan/report/Summary, strict deletion/readback, destination intent preservation and repeat no-diff using a local API. `TestRulesetDeleteRequiresStrictPreflightAndConfirmedInventoryAbsence` owns identity/drift, response status and partial acquisition cases. `TestRulesetDeletionSelectorsAndReconciliation` owns tombstone validation and excluded-ID/no-fallback semantics; `TestRulesetExplicitDeletionAppOperationAndPartialFailure` owns sequencing/status. None changes live protection.

## Unsupported Push policy

Push targets and file restriction controls are excluded by the Free/personal/public adoption scope. Removed YAML controls fail local validation, including false and empty values. Acquired Push policy or a file-restriction rule fails complete acquisition before export, update or explicit deletion; no partial policy or protection-dropping rewrite is produced. The [unsupported-control executable cases](../../tests/acceptance/ruleset_unsupported_controls_test.go) cover rejection without API calls, destination preservation and no writes for acquired unsupported state. Existing branch/tag lifecycle and scanning controls remain supported.
