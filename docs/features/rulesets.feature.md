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

Required protection intent is name, enforcement and ref include. Target defaults to branch; ref exclude, bypass and rule lists default empty. Declared bypass actors require their mode. Every selected supported nested setting belongs to that object; there is no retained-member omission merge.

Parameterized-rule booleans default false, approval count defaults zero, reviewer/dismissal/check lists default empty and allowed merge methods default all three supported methods. Individual required reviewers require explicit minimum approvals and file patterns. The reference owns the exact table and constraints. Apply the same defaults for create and update, and show their protection consequences in plan.

Null, unknown keys and wrong types remain invalid locally. Explicit false, zero and empty are not omitted. Reviewer pattern order and repetitions remain meaningful; independent identity-based collections remain canonical sets. Any-source status checks retain their documented distinct identity from App-bound checks.

## Complete observation and export

Reads require complete repository ownership/ID, required supported policy, bypass visibility and every page/detail. Unknown writable fields, unsupported rules, missing required values, null values, denied reads and partial data fail rather than becoming defaults or success. Optional `dismissal_restriction` absence remains nil/unobserved; malformed present objects fail. Declaration defaults must never be applied to an API read.

Opted-in export includes numeric IDs plus observed supported policy without normalizing absent dismissal restrictions. Ordinary export adds no Rulesets request. For complete observations, export → validate → plan against unchanged fixture state yields zero differences. An absent restriction remains absent in exported YAML and passes local validation, but selecting that ID for plan/apply errors before any Rulesets write instead of guessing a disabled value or reporting a no-op. Unselected optional absence does not manage that object or block independent selections. Whole-object defaults and ownership remain unchanged. Failed export preserves the destination. Push/inherited resources remain outside mutation ownership. Server-generated metadata is explicitly classified and never replayed as writable policy.

## Plan, apply and idempotence

Plan is read-only and reports only selected objects, with their IDs and complete supported before/after values. Unlisted objects are not diff entries. Empty selection counts zero, nonempty selection counts once. Strict same-version Action reports reject incomplete objects, missing/mismatching IDs, duplicate identities and delete operations; Summary escapes values without credentials.

Apply reads fresh state after earlier resources and prepares every selected Ruleset operation before the first Rulesets write. POST/PUT never send YAML IDs as payload fields. Every response must confirm positive ID, target repository ownership and the complete desired supported policy; updates retain the selected ID. Missing identity, ignored fields or mismatching response stop later work with uncertain-write semantics. No DELETE, hidden retry, rollback or saved-plan execution exists.

Confirmed created IDs are appended immediately to the original YAML through the shared guarded recorder. Repeat apply then selects that ID, avoiding duplicate creation. Explicit no-writeback and GitHub Actions leave configuration unchanged and display IDs for manual adoption. A successful remote create followed by local persistence failure is a partial failure with recovery ID; earlier confirmed label/Ruleset IDs remain available. Unverified response IDs are reported but never persisted as confirmed creation. Output failure retains ID diagnostics on stderr. No Git commit/push is made.

## API, access and availability

The adapter pins API version `2026-03-10`, uses bounded requests and refuses redirects. [Repository Rulesets REST](https://docs.github.com/en/rest/repos/rules) owns request permissions/shapes; complete bypass visibility requires Ruleset write access beyond ordinary public listing. Account/plan eligibility, including evaluate and team features, remains GitHub's responsibility. Do not expand credentials or change live protection to make a test pass.

Fresh reads and response verification do not lock concurrent edits. Whole-object management can overwrite intervening edits to supported selected settings; unknown observed writable settings cause failure rather than silent erasure. Fixtures establish client behavior, not live entitlement, server mutation success or atomicity.

## Implementation stages and acceptance plan

One focused breaking Rulesets change builds on labels' ID recorder. Environment uses a separate name-identity/upsert contract and is not implemented by this feature. Topics and unrelated scalar repository fields retain their own ownership. No release or live policy mutation is part of fixture acceptance.

## Existing test mapping and evidence limits

| Boundary | Tests and behavior |
| --- | --- |
| Model | [ID selection/defaults](../../internal/settings/ruleset_ids_test.go), [validation/copies](../../internal/settings/rulesets_test.go), [parameters](../../internal/settings/ruleset_parameters_test.go): collision guidance, wrong IDs, no implicit deletion, whole supported values, create/update defaults, ordered patterns and strict read state. |
| YAML | [Codec](../../internal/workspace/rulesets_test.go): positive numeric ID round-trip, strict types/null/unknown rejection and aliases. |
| API | [Inventory/payload](../../internal/github/rulesets_test.go), [identity](../../internal/github/ruleset_identity_test.go), [parameters](../../internal/github/ruleset_parameters_test.go): complete pages/details, owner/type isolation, ID-free complete writes, response mismatch/redirect/refusal. |
| Application | [Sequencing](../../internal/app/rulesets_test.go), [ID recording](../../internal/app/ruleset_ids_test.go): empty no-op, fresh preparation, partial failure, confirmed recorder ordering and earlier-label recovery. |
| Action | [Whole-object report](../../internal/actions/rulesets_test.go), [parameters](../../internal/actions/ruleset_parameters_test.go): complete selected policy/IDs, native false/zero/order, no unlisted values and escaped Summary. |
| Built process | [Lifecycle](../../tests/acceptance/rulesets_test.go), [ID modes/recovery](../../tests/acceptance/ruleset_ids_test.go), [parameters](../../tests/acceptance/ruleset_parameters_test.go): export/zero plan, rename, unmanaged siblings, writeback/repeat, opt-out/Actions, partial failures and response loss. |
| Optional observation | [Model](../../internal/settings/ruleset_optional_test.go), [API](../../internal/github/ruleset_optional_test.go), [built process](../../tests/acceptance/ruleset_optional_test.go): absent versus malformed/explicit restriction, omission-preserving export, selected-ID safety guard, no false diff/success, no mutation and closed Action failure report. |

All cases use local HTTP/files/Git and dummy credentials. Hosted candidate/required CI and review results must be inspected separately; local tests are not evidence of live settings changes or publication.
