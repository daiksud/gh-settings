---
type: Feature
title: Manage repository Actions event policies
description: Manage listed All workflows event policies by ID without adopting names or deleting unlisted policies.
---

## Contract

The [configuration reference](../settings.md#actions-event-policies) owns the
public schema, supported 36-event vocabulary, permissions and recovery semantics.
The [command reference](../commands.md#actions-event-policy-commands) owns export
selection; the [Action guide](../github-actions.md#actions-event-policies) owns
caller and published-package boundaries.

## Rule: Select complete repository-owned event objects

- `repository.settings.actions.policies` is an optional list with optional
  positive integer `id`, required `policy_name`, required `enforcement_status`
  (`disabled` or `active`) and required
  `restrict_events.allowed_events` (nonempty, unique supported strings).
- Omission and `[]` manage nothing and make no policy requests. Removing an entry
  stops ownership; unlisted policies are never updated or deleted.
- Nulls, wrong types, unknown/API-shaped keys, duplicate IDs/exact names, invalid
  enforcement and unsupported/duplicate events fail before remote work.
- No ID means create-only; exact remote-name collisions identify the existing ID
  and fail. An ID must exist in this repository and never falls back to create.
  ID-bound updates replace the supported whole event object, may rename to a
  noncolliding name, and compare event lists as sets. A planned rename cannot
  release a name for another operation.
- All workflows targeting is implicit. Remote omitted/null/empty conditions or
  `workflow_path` with empty exclusions and either empty includes or `[~ALL]`
  are equivalent. Other targeting, actors, Preview/unknown rules, fields, event
  or enforcement choices retain complete identity but cannot be selected for
  update or exported. Unlisted unsupported objects stay untouched; incomplete
  observations fail rather than becoming empty/default state.
- OIDC, actor restrictions, Preview lockfile controls, workflow path patterns and
  workflow/configuration-file authoring are excluded. There is no delete input.

## Rule: Observe the whole collection and guard each mutation

- Plan reads complete paginated repository-only identities and individual
  details, validates collection consistency, and reports typed creates/updates
  without writes. Missing, denied, malformed or unstable reads fail.
- Apply freshly reads and preflights the whole policy collection before any
  policy write. Exact matches are skipped. Each selected update also requires a
  fresh detail matching its baseline immediately before writing; drift or newly
  unsupported content stops it before mutation.
- POST creation and PUT update contain name, enforcement and the single event
  rule only. Conditions are omitted to create All workflows or preserve the
  preflight-verified All workflows targeting during update.
- Both a complete write acknowledgment and an independent final GET must match
  the requested identity/name, enforcement, event set and supported targeting.
  Acknowledgment alone is not success. Redirects and failed/mismatching reads
  fail honestly, with known created IDs retained for recovery where available.
- A failure stops later operations; earlier confirmed writes remain applied and
  uncertain effects remain uncertain. No retry, atomic rollback or concurrency
  lock is promised. An uncertain create must not be automatically repeated.

## Rule: Export independently and recover created identities

- `export --actions-policies` selects the full collection, including IDs, sorted
  by exact policy name with sorted events. It composes with other export flags;
  duplicate flags fail before target/API/file work. Ordinary export and
  `--actions` do not select it.
- All pages/details must be acquired and representable before saving. An
  unsupported object or failed read preserves the existing export destination;
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
Free, personally owned public UI and a disabled-policy storage probe. Local
fixtures cover active/disabled metadata transitions; actual enforcement and
workflow execution were not tested.
Neither successful storage readback nor fixture success proves actual event
blocking. Published packages retain their own contracts; this source addition
does not repin callers or change a live policy.

## Test mapping

| Boundary | Owning checks |
| --- | --- |
| Strict YAML, omission, aliases/merges and round trips | [Policy codec](../../internal/workspace/ui_actions_event_policy_test.go) |
| Supported values, identity, whole-object comparison, unsupported isolation and export ordering | [Policy model](../../internal/settings/actions_event_policies_test.go) |
| UI/native translation and export composition | [Policy projection](../../internal/github/ui_actions_event_policy_test.go) |
| Complete pagination/detail reads, ownership, targeting, exact writes, drift guard and independent verification | [API transport](../../internal/github/actions_policies_test.go) |
| Fresh collection preflight, sorted operations, no-op/omission, creation recovery and partial failure | [Application](../../internal/app/actions_event_policies_test.go) |
| Independent export option, CLI object presentation and ID recovery | [CLI](../../internal/cli/actions_event_policies_test.go) |
| Typed closed reports, escaped Summary and status-only privacy | [Reports and Summary](../../internal/actions/actions_event_policies_test.go) |
| Guarded creation-ID recording and snapshot preservation | [ID writeback](../../internal/workspace/created_actions_policy_ids_test.go) |
| Built CLI validate/plan/report/Summary, create/update/repeat apply, partial recovery, export isolation and file safety | [Process acceptance](../../tests/acceptance/actions_event_policies_test.go) |

Automated checks use local fixtures and dummy credentials, with no live writes.
