---
type: ADR
title: Separate Environment protection ownership from lifecycle
description: Define named existing-environment intent without implicit deletion, guessed protections or undocumented bypass controls.
---

## Status

Historical design decision for #351. Current-source [Environment management](../settings.md#environments) supersedes this record's existing-only, field-level omission and numeric-incarnation binding proposals with named whole-object upsert and API-controlled omissions. The original rationale below is retained as history, not the current accepted YAML contract. Current source also supports typed custom patterns and administrator bypass using the evidence and limits in the current contract; neither this record nor source implementation authorizes live protection changes.

## Context

Environments carry deployment protections and may contain secrets. Treating their names as an authoritative collection like labels would let an omitted name delete protection and secrets. The create-or-update endpoint also cannot guarantee that an existing environment remains the same resource between a read and a name-addressed write. Visibility, subscription and authorization can hide protection data without proving that it is disabled.

The [Environment specification](../features/environments.feature.md) records the API evidence, ownership table, identity rules, failure semantics and acceptance gates. Existing declared-only intent, read-only plan and nontransactional direct apply remain the starting point.

## Decision

Reserve a named sequence under `settings.environments` for existing-environment protection intent. Membership is additive selection, never authoritative repository membership: omitted names and an empty sequence remove nothing. Reject lifecycle markers, rename and deletion. Names select a resource case-insensitively; positive repository and environment identities bind each operation. Require complete observations for owned values and any companions a write can affect. Unknown is not zero, false, empty or absent.

Choose explicit typed reviewer references with a numeric identity pin and a human-readable user login or organization/team slug. Resolve and verify the pair before mutation; a renamed/recreated principal must not silently acquire approval authority. Reviewer collection ownership is separate from Environment membership. Clearing reviewers, removing a timer and allowing all deployment refs are explicit protection reductions, not omission.

Do not adopt `can_admins_bypass` from an issue, UI label or returned metadata. The reviewed public REST request schema and GraphQL update input do not document that setter. Its read, write and preservation behavior need separate evidence. Unknown companion preservation also blocks an otherwise valid protection write when the chosen endpoint can affect that companion.

Keep creation/deletion/rename deferred. A 404 does not authorize upsert or establish successful deletion. A later lifecycle decision must cover exposure, secrets/protection loss, pending jobs, workflow recreation and verifiable absence. Model branch/tag patterns separately; #353 must not activate custom mode until its complete pattern observation and transition order are adopted. Neither design authorizes changes to this repository's live Environment or approval controls.

## Consequences

The contract supports independent model, codec and report design without claiming API readiness. #352 must establish the bounded protection subset it can read, preserve and verify before accepting any reserved key; unsupported portions remain visible. #353 additionally needs typed pattern observations and transition evidence. These gates do not block other independent issues.

Implementation stays in the existing settings/workspace/API/app/presentation layers, with no ownership database or transaction abstraction. Export is opt-in and must not turn unavailable protection into an apparently safe declaration. Failures retain earlier writes and never trigger automatic rollback, permission expansion or deployment approval.
