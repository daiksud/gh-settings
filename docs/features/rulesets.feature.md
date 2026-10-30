---
type: Feature
title: Declarative repository Rulesets
description: Select whole Ruleset objects by numeric ID without deleting unlisted objects.
---

## Status and goal

Rulesets use ID-selected whole-object ownership. The [configuration reference](../settings.md#rulesets) owns schema, fixed defaults and observation limits. Optional-absence export handling is current-source behavior requiring a release containing the fix. Historical [ADR 0025](../adr/0025-declarative-repository-rulesets.md) remains a record of the superseded ownership decision.

## Ownership and identity

- Given omitted Rulesets or an empty list, validate/plan/apply select no object and acquire no Rulesets state or target solely for that resource.
- Given a positive declared REST ID, update only that supported repository-owned branch/tag object. Missing or wrong repository/type IDs fail; never create as fallback.
- Given no ID, create only a new name. A collision fails with the existing ID and guidance to add it; never adopt by name.
- Given a valid ID and noncolliding new name, rename the same object. Unlisted names/IDs remain untouched and are never implicitly deleted.
- Reject duplicate declaration IDs or exact names. Preserve existing exact-name comparison; no heuristic case-folding or trimming is introduced.

## Schema and presence

Current UI entries live under `repository.settings.rulesets`. They require `ruleset_name`, `enforcement_status` (`active` or `disabled`) and exactly one `target_branches`/`target_tags` object. Branch selectors are `include_default_branch`, `include_all_branches`, `include_by_pattern` and `exclude_by_pattern`; tags use `include_all_tags` and pattern arrays. Patterns use UI spelling without internal ref prefixes. An explicit empty target object yields empty include/exclude lists; no target or protected scope is guessed.

Omitted `bypass_list` and unselected `rules` become empty whole-object policy. Each bypass entry selects one supported role (`repository_admin`, `maintain`, `write`), numeric team/app/user, or `deploy_keys: true`, and requires `bypass_mode` (`always`, `exempt`, `for_pull_requests_only`) with existing applicability checks. No actor/grant is created. Rules contain four basic boolean checkboxes and four optional parameter groups; an omitted group removes its rule rather than retaining remote membership.

PR-rule booleans default false, `required_approvals` defaults zero (UI range 0–10), reviewer/check lists default empty and allowed merge methods default all three. `require_review_from_specific_teams` entries require numeric `reviewer`, `approvals` and `file_patterns`. Checks use `check` plus optional `source: {any_source: true}` or `{app: positive-id}`; omitted source means any source, not retained App binding. The configuration reference owns the full exact field inventory.

UI `restrict_who_can_dismiss_pull_request_reviews` and `people_teams_or_apps_that_can_dismiss_reviews` translate to the temporary internal write-only dismissal restriction. Both omitted means no request member irrespective of observed values; explicit controls retain false/empty companion defaults. Nonempty actors require an explicitly true restriction. Comparable whole-object defaults still appear in plan.

Null, unknown keys and wrong types remain invalid locally. Explicit false, zero and empty are not omitted. Reviewer pattern order and repetitions remain meaningful; independent identity-based collections remain canonical sets. Any-source status checks retain their documented distinct identity from App-bound checks.

`rules.require_code_scanning_results` requires complete `code_scanning_tools` entries with `tool_name`, `alerts` and `security_alerts`. Identities are case-sensitive. `rules.require_code_quality_results` requires explicit UI `severity` (`errors`, `warnings_and_higher`, `notes_and_higher`, `all`). Neither rule supplies defaults or borrows missing parameters from a prior policy; cross-rule parameters are invalid. Tool collections are canonical sets, including explicit empty arrays. The configuration reference owns the exact enums. Complete API reads, export, plan and Action reports retain both rules alongside all other supported policy.

## Complete observation and export

Reads require complete repository ownership/ID, required supported policy, bypass visibility and every page/detail. Unknown writable fields, unsupported rules, missing required values, null values, denied reads and partial data fail rather than becoming defaults or success. Optional `dismissal_restriction` absence remains nil/unobserved; malformed present objects fail. Declaration defaults must never be applied to an API read.

Opted-in export includes numeric IDs plus observed comparable policy. It always excludes dismissal restriction, even if observed: export does not fully capture that option, and replacing YAML with export output loses its explicit write intent. Ordinary export adds no Rulesets request. Export → validate → plan against unchanged exact supported comparable state yields zero differences. A normalized enforcement, Required approvals or Quality severity fallback remains actionable, with ID/name-scoped export warnings in name order. Other whole-object defaults and unselected-object ownership remain unchanged. Failed export preserves the destination. Push/inherited resources remain outside mutation ownership. Server-generated metadata is explicitly classified and never replayed as writable policy.

## Plan, apply and idempotence

Plan is read-only and reports only selected objects, with their IDs and supported comparable before/after values. Explicit dismissal restriction fails plan before remote work, without a false no-difference or drift result. YAML/validate and apply support it; plan and export do not. Omitted intent does not display an observed dismissal value. Unlisted objects are not diff entries. Empty selection counts zero, nonempty selection counts once. Strict same-version Action reports reject incomplete objects, missing/mismatching IDs, duplicate identities and delete operations; Summary escapes values without credentials.

Apply reads fresh state after earlier resources and prepares every selected Ruleset operation before the first Rulesets write. POST/PUT never send YAML IDs as payload fields. Every response must confirm positive ID, target repository ownership and requested comparable policy; updates retain the selected ID. Normalized enforcement/Required approvals/Quality updates and writes adding, updating or removing pull-request, code-scanning or code-quality rules require independent post-write GET identity/policy verification, excluding dismissal-restriction values but still rejecting malformed present objects. Explicit dismissal intent is sent on every apply, including a repeat; success reports API acceptance and explicitly unverified value, not idempotent value verification. Changed comparable values or failed readback stop later work with uncertain-write semantics. No DELETE, hidden retry, rollback or saved-plan execution exists.

Confirmed created IDs are appended immediately to the original YAML through the shared guarded recorder. Repeat apply then selects that ID, avoiding duplicate creation. Explicit no-writeback and GitHub Actions leave configuration unchanged and display IDs for manual adoption. A successful remote create followed by local persistence failure is a partial failure with recovery ID; earlier confirmed label/Ruleset IDs remain available. Unverified response IDs are reported but never persisted as confirmed creation. Output failure retains ID diagnostics on stderr. No Git commit/push is made.

## API, access and availability

The adapter pins API version `2026-03-10`, uses bounded requests and refuses redirects. [Repository Rulesets REST](https://docs.github.com/en/rest/repos/rules) owns request permissions/shapes; complete bypass visibility requires Ruleset write access beyond ordinary public listing. API-only evaluate is outside desired UI choices; complete enforcement GET strings use the [active ID-bound normalization policy](../settings.md#ruleset-enforcement-observation-normalization). Quality severity follows its [active policy](../settings.md#ruleset-quality-severity-observation-normalization). [Required approvals](../settings.md#ruleset-required-approvals-observation-normalization) retains typed integer provenance. Other five Ruleset fallbacks remain pending #480. Account/plan eligibility and team-feature prerequisites remain GitHub's responsibility. Do not expand credentials or change live protection to make a test pass.

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
| Write-only dismissal | [Model](../../internal/settings/ruleset_optional_test.go), [API](../../internal/github/ruleset_optional_test.go), [plan boundary](../../internal/app/ruleset_writeonly_test.go), [built process](../../tests/acceptance/ruleset_omission_roundtrip_test.go): export omission even when observed, explicit plan rejection, exact payload intent, repeat apply, API-accepted/unverified notice, comparable readback failures and no retry. |

All cases use local HTTP/files/Git and dummy credentials. Hosted candidate/required CI and review results must be inspected separately; local tests are not evidence of live settings changes or publication.

The current YAML has no API-shaped Ruleset aliases or legacy parser. Independent UI maps in process tests are checked against exact HTTP expectations. CLI/Summary and serialized private object reports retain native policy objects/identifiers pending the #481 full UI reporting audit.

## Rule: Keep normalized enforcement distinct from raw state

- Given a complete typed out-of-UI enforcement string, including `evaluate`, unknown, empty or whitespace, retain exact raw provenance per positive ID and project Disabled. Missing/null/type and later nested/entry failures return no partial observation; pure readers stay silent.
- Export warns only after complete comparable acquisition, in deterministic name order with ID/name context. A later UI conversion failure preserves legitimate warnings but does not save.
- Plan only declared IDs. A fallback-equal selected object still requires one whole update; unlisted normalized objects are not printed or written. Ordinary equality, collision, create-only and dismissal contracts remain unchanged.
- Detailed output labels the before status as normalized and the object as effective policy, never raw evidence. Closed private metadata rejects forged/no-op/create/ID-conflicting entries; Summary escapes HTML and status-only suppresses object/raw context. Existing collection count stays one pending #481.
- Apply emits selected pre-write provenance and only complete fresh GET diagnostics after verified identity/ownership. Strict mutation response and independent GET must confirm desired whole policy; retained Evaluate/unknown cannot confirm a fallback. A canonical acknowledgment alone is insufficient. Prior REST effects remain reported on failure; malformed preflight makes no Ruleset write and export preserves the destination.

Test mapping: [ID-bound carrier](../../internal/settings/ruleset_enforcement_observation_test.go), [closed changes](../../internal/settings/ruleset_enforcement_change_test.go), [selected reconciliation](../../internal/settings/ruleset_enforcement_reconcile_test.go), [HTTP acquisition](../../internal/github/ruleset_enforcement_read_test.go), [strict fresh verification](../../internal/github/ruleset_enforcement_verification_test.go), [scoped diagnostics](../../internal/github/ruleset_enforcement_warning_test.go), [export scope](../../internal/app/normalization_ruleset_enforcement_export_test.go), [private report/HTML/privacy](../../internal/actions/normalization_ruleset_enforcement_test.go), [executable commands](../../tests/acceptance/normalization_ruleset_enforcement_test.go). Other five Ruleset controls, one Environment ref-type control and complete #481 reporting remain pending.

## Rule: Keep normalized Quality distinct from missing or removed rules

- Given a complete present code-quality rule, canonical API `errors`, `warnings`, `notes` and `all` map to the four UI choices. Other typed severity strings project Errors with exact ID-bound provenance; absent rules stay absent, and missing/null/type/duplicate/incomplete policy fails without partial acquisition.
- A fixed concrete aggregate retains simultaneous enforcement and Quality provenance. Plan only selected IDs and make one whole-object update even for fallback equality. Contextual diagnostics order enforcement, Required approvals, then Quality; effective-policy display is emitted once.
- Desired Quality removal shows inactive/rule-removed intent, never an invented after severity. Strict original mutation response and independent GET must confirm actual absence; retained canonical or unknown rules cannot confirm removal. Unknown readback cannot confirm Errors even with a canonical acknowledgment.
- Same-version private report metadata validates each concrete record, including present-zero rejection, and never supplies mutation values. Status-only hides provenance/object details; the existing collection count remains one pending #481. Canonical no-op, unlisted privacy, prior REST effects, export preservation and write-only dismissal limits remain unchanged.

Test mapping: [mapping](../../internal/settings/ruleset_quality_mapping_test.go), [carrier](../../internal/settings/ruleset_quality_observation_test.go), [closed changes](../../internal/settings/ruleset_normalizations_change_test.go), [reconciliation](../../internal/settings/ruleset_quality_reconcile_test.go), [HTTP acquisition](../../internal/github/ruleset_quality_read_test.go), [strict confirmation/removal](../../internal/github/ruleset_quality_verification_test.go), [diagnostics](../../internal/github/ruleset_quality_warning_test.go), [private metadata](../../internal/actions/normalization_ruleset_quality_test.go), [export](../../internal/app/normalization_ruleset_quality_export_test.go), [executable commands](../../tests/acceptance/normalization_ruleset_quality_test.go). Other five Ruleset controls, one Environment ref-type control and complete #481 reporting remain pending.

## Rule: Distinguish normalized Required approvals from acquired zero

- Acquire a complete integer from each present PR rule. Keep UI 0–10 exact; project other integers to 0 with ID-bound typed provenance. Missing/null/fractional/string/overflow fields fail without partial results; absent rules stay absent.
- Selected normalized equality requires one whole-object update, with independent enforcement and Quality records preserved. Plan/Summary show unquoted integer choices or inactive PR-rule removal and one effective policy. Warning order is enforcement, Required approvals, Quality.
- Verify exact original mutation response and fresh GET. A canonical acknowledgment followed by same-ID/name raw 11/-1 cannot prove desired 0; retaining a canonical PR rule cannot confirm removal. Do not retry or roll back uncertain effects.
- Same-version reports require explicit typed metadata members and reject forged, present-zero, create, ID-conflicting and unsupported desired counts. Status-only hides object/provenance. Count one, omitted/unlisted privacy, write-only dismissal and prior-resource failure semantics remain unchanged; five other Ruleset controls and one Environment ref-type control remain pending, along with #481.

Test mapping: [mapping](../../internal/settings/ruleset_required_approvals_mapping_test.go), [carrier](../../internal/settings/ruleset_required_approvals_observation_test.go), [closed changes](../../internal/settings/ruleset_required_approvals_change_test.go), [reconciliation](../../internal/settings/ruleset_required_approvals_reconcile_test.go), [integer JSON](../../internal/settings/ruleset_required_approvals_json_test.go), [acquisition](../../internal/github/ruleset_required_approvals_read_test.go), [strict verification/removal](../../internal/github/ruleset_required_approvals_verification_test.go), [diagnostics](../../internal/github/ruleset_required_approvals_warning_test.go), [private reports](../../internal/actions/normalization_ruleset_required_approvals_test.go), [export](../../internal/app/normalization_ruleset_required_approvals_export_test.go), [built executable](../../tests/acceptance/normalization_ruleset_required_approvals_test.go).
