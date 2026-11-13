---
type: Feature
title: Manage repository Actions restriction policies
description: Manage listed event/actor policies and workflow targets by ID without adopting names; explicit prune adds unlisted deletion.
---

## Contract

Ordinary `export` now follows the [automatic readable-settings contract](readable-settings-export.feature.md): observe all supported families without selectors, retain unavailable declarations with info, and keep plan/apply strict.

The [configuration reference](../settings.md#actions-event-policies) owns the
public schema, supported 36-event vocabulary, permissions and recovery semantics.
The [command reference](../commands.md#actions-event-policy-commands) owns export
selection; the [Action guide](../github-actions.md#actions-event-policies) owns
caller and published-package boundaries.

## Rule: Select complete repository-owned event objects

- `repository.settings.actions.policies` is an optional list with optional
  positive integer `id`, required `policy_name`, required `enforcement_status`
  (`disabled` or `active`), optional `restrict_events.allowed_events` (nonempty,
  unique supported strings), optional `restrict_actors.allowed_actors` (unique
  User or verified repository-role identities, including checked-empty), plus
  optional `target_workflows` with both `include` and `exclude` string arrays.
- Without prune, omission and `[]` manage nothing and make no policy requests.
  Removing an entry stops ownership. `--prune actions-policies` preserves omission,
  but `[]` selects all policies for deletion and a nonempty list retains declared
  IDs. ID-less entries remain create-only. Unlisted policies are never updated.
- Nulls, wrong types, unknown/API-shaped keys, duplicate IDs/exact names, invalid
  enforcement and unsupported/duplicate events fail before remote work.
- No ID means create-only; exact remote-name collisions identify the existing ID
  and fail. An ID must exist in this repository and never falls back to create.
  ID-bound updates may rename to a noncolliding name and compare event and
  actor lists as sets. Omitted restriction rules are preserved on updates and
  off on creates; `enabled: false` explicitly removes only that restriction,
  while `enabled: true` or omitted enabled requires its allowed list. False
  with a list is invalid. At least one effective rule is required. Metadata-only
  selected-ID updates may omit both rules. Checked-empty actors stay a present
  rule; they never mean rule removal. A planned rename cannot
  release a name for another operation.
- `target_workflows` owns both lists when supplied. At least one pattern is
  required; empty includes mean all except exclusions. `~ALL` is valid only as
  the sole include entry and never in exclusions. All workflows normalize to
  `[~ALL]` with empty exclusions; exclude-only targeting uses empty includes.
  Comparisons and exports sort each set. Invalid/blank/control-character
  patterns and duplicates fail locally. Matching remains GitHub's responsibility.
- Omitted targeting creates All workflows and preserves the acquired selection
  on update. Explicit `[~ALL]` with empty exclusions resets to All workflows.
  Remote omitted/null/empty conditions and empty or `[~ALL]` includes with empty
  exclusions are equivalent all-target states. A present `workflow_path` requires
  both complete typed arrays; unknown/malformed targeting must not become all.
- Unsupported actor identities, Preview/unknown rules, fields, event or enforcement choices
  retain complete identity but cannot be selected for update or exported.
  Without prune, unlisted unsupported objects stay untouched. With prune, any
  unsupported deletion candidate aborts before all writes; incomplete observations
  always fail.
- The [OIDC active template](oidc-subject.feature.md) has independent support.
  Writable immutable mode, unadopted actor kinds and Preview lockfile controls
  remain unimplemented; configuration-file authoring stays outside scope.
  There is no policy delete input.

## Rule: Observe the whole collection and guard each mutation

- Plan reads complete paginated repository-only identities and individual
  details, validates collection consistency, and reports typed creates/updates
  and explicit prune deletions without writes. Missing, denied, malformed or
  unstable reads fail.
- Apply freshly reads and preflights the whole policy collection before any
  policy write. Exact matches are skipped. Each selected update also requires a
  fresh detail matching its baseline immediately before writing; drift or newly
  unsupported content stops it before mutation.
- POST creation and PUT update contain name, enforcement and the complete
  resolved event/actor rule set, with changed explicit targeting in `conditions.workflow_path`. Omitted
  targeting creates All workflows or preserves the preflight-verified selection
  during update. Unchanged targeting need not be resent.
- Both a complete write acknowledgment and an independent final GET must match
  the requested identity/name, enforcement, rule presence, event/actor sets and targeting.
  Acknowledgment alone is not success. Redirects and failed/mismatching reads
  fail honestly, with known created IDs retained for recovery where available.
- A failure stops later operations; earlier confirmed writes remain applied and
  uncertain effects remain uncertain. No retry, atomic rollback or concurrency
  lock is promised. An uncertain create must not be automatically repeated.

## Rule: Export independently and recover created identities

- `export` selects the full collection, including IDs, sorted
  by exact policy name with sorted events, actors and patterns. Absent restriction
  rules export explicitly as `enabled: false`. Export always emits an
  explicit canonical `target_workflows`, including `[~ALL]`/`[]` for all. Ordinary export attempts it independently of other families.
- All pages/details must be acquired and representable before saving. An
  unsupported object or failed read emits info and retains the prior collection;
  unsupported protections are never silently erased.
- Confirmed new IDs use the guarded original-byte snapshot recorder with
  collection identity `actions_policies` and `policy_name` selector. Existing
  IDs/unrelated content stay intact. Snapshot changes, alias/merge ambiguity and
  unsafe paths fail without overwriting the source.
- `--no-writeback` and `GITHUB_ACTIONS=true` suppress local updates. Plan never
  writes the file. Known IDs survive later failures in recovery diagnostics;
  verify uncertain remote outcomes and record the correct ID before retrying.
  No automatic commit, push, adoption or create retry occurs.
- Detailed CLI/report/Summary show complete selected objects and escaped names;
  existing IDs remain distinct identity. Status-only hides names and values;
  Summary uses its protected report without API access. This is not apply input.

## API and evidence boundary

The [official Actions policies API](https://docs.github.com/en/rest/actions/policies)
requires repository **Administration write for fine-grained/App tokens even for
list and GET**, including export and plan. The client never grants access or
retries with another identity. Permission documentation does not certify a
particular installation or token's grants.

[Adoption evidence](https://github.com/daiksud/gh-settings/issues/647) covers the
Free, personally owned public UI and a disabled-policy storage probe. A further
[disabled-only targeting probe](https://github.com/daiksud/gh-settings/issues/649)
verified include/exclude patterns, omitted-update preservation
and an explicit All workflows reset, then restored the empty collection. Local
fixtures cover active/disabled metadata transitions; actual enforcement and
workflow execution were not tested.
On 2026-11-10, a further Disabled-only personal/public/Free probe verified
Repository admin=RepositoryRole5, Write=4, Maintain=2, owner User identity,
actor-only and combined event/actor storage, checked-empty actor lists and
unchecked actor-rule removal. The created policy alone was deleted; DELETE204,
list0, detail404 and the empty real UI restored the baseline. No Bot/App/team
mapping or actor execution effect was tested.
Neither successful storage readback nor fixture success proves actual event
or actor blocking. Published packages retain their own contracts; this source addition
does not repin callers or change a live policy.

## Test mapping

| Boundary | Owning checks |
| --- | --- |
| Strict YAML, omission, aliases/merges and round trips | [Policy codec](../../internal/workspace/ui_actions_event_policy_test.go) |
| Supported values, identity, whole-object comparison, unsupported isolation and export ordering | [Policy model](../../internal/settings/actions_event_policies_test.go) |
| UI/native translation and export composition | [Policy projection](../../internal/github/ui_actions_event_policy_test.go) |
| Complete pagination/detail reads, ownership, targeting, exact writes, drift guard and independent verification | [API transport](../../internal/github/actions_policies_test.go) and [actor transport](../../internal/github/actions_actor_policies_test.go) |
| Fresh collection preflight, sorted operations, no-op/omission, creation recovery and partial failure | [Application](../../internal/app/actions_event_policies_test.go) |
| Independent export option, CLI object presentation and ID recovery | [CLI](../../internal/cli/actions_event_policies_test.go) |
| Typed closed reports, escaped Summary and status-only privacy | [Reports and Summary](../../internal/actions/actions_event_policies_test.go) |
| Guarded creation-ID recording and snapshot preservation | [ID writeback](../../internal/workspace/created_actions_policy_ids_test.go) |
| Built CLI validate/plan/report/Summary, create/update/repeat apply, partial recovery, export isolation and file safety | [Process acceptance](../../tests/acceptance/actions_event_policies_test.go) |

Automated checks use local fixtures and dummy credentials, with no live writes.

## Rule: Explicit prune removes only prepared supported policies

- Validate all candidates before unrelated writes too. Fresh reconciliation must
  retain exactly the preflight deletion identities and states.
- Perform creates/updates before fixed deletions. New IDs cannot enter the
  deletion set. Recheck ownership, ID and complete policy before DELETE, require
  HTTP 204, and independently read every remaining page/detail to prove absence.
- A deleted restriction can permit future workflow execution. Unsupported or
  inherited protection must never be discarded. A partial failure stops later
  writes and reports earlier completed changes and uncertain effects honestly.

Domain, API, app and actual CLI fixtures cover empty-list presence, missing and
unsupported inventories, unchanged retained IDs, fixed creation/deletion order,
state drift, deletion acknowledgment and failed readback, repeat apply, partial
failure and both Action Summary modes. No live policy deletion is required.
