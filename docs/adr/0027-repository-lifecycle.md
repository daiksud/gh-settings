---
type: ADR
title: Isolate repository lifecycle changes from ordinary settings
description: Adopt explicit archival sequencing and defer visibility and rename transitions.
---

## Status

Accepted in #331; archive/unarchive was implemented by #332 and published in v0.18.0. Visibility (#333) and rename (#334) remain deferred, not silently adopted as ordinary PATCH fields. Published v0.17.0 and older packages reject `archived` and the deferred keys.

## Context

The tool derives its one target from a checkout's origin, applies declared settings directly, rejects write redirects, and reports partial failure without rollback. Archive, visibility, and name changes can alter that target's writability, accessibility, or identity. Adding them to the existing bulk repository PATCH would make later operations and exported configurations misleading.

GitHub documents archive/unarchive as a read-only lifecycle transition, visibility changes with effects on forks, features and public logs, and rename redirects that do not preserve Action references. The [lifecycle specification](../features/repository-lifecycle.feature.md#official-contracts) records the primary sources and implementation evidence limits.

## Decision

Adopt an optional `settings.repository.archived` boolean for a separate lifecycle phase, not the ordinary repository PATCH. Omission is unmanaged; false requests unarchive; true requests archive. Read a complete repository ID, canonical owner/name, and archive state before a declared transition. Reject unavailable identity/state and redirects rather than retargeting.

Unarchive, when explicitly requested and necessary, occurs before ordinary updates. Archive occurs only after all other declared updates succeed, with a fresh identity/state check. An already archived target with other managed settings cannot be temporarily unarchived without explicit false intent; apply rejects that combination before any write. Read-only plan may completely observe and compare those declarations because it does not attempt a mutation. An archive-only matching declaration performs no mutation. A requested false leaves an already active repository active and permits normal work.

Keep plan read-only and show the native boolean plus the transition's ordering consequence. A plan for an already archived target may establish that its other declared settings match or differ, but it does not make those settings writable or authorize apply; an operator must explicitly declare `archived: false` before changing them. Add archive to export only through explicit `--archive`, so ordinary export does not begin managing lifecycle state. Applying an archive-inclusive export is not promised to succeed on a still-archived repository when it also declares other settings; explicitly unarchive first. No interactive prompt or hidden automatic recovery is introduced.

Pin the observed numeric repository ID for response/readback checks throughout the declared lifecycle operation, without changing the checkout's origin. A successful PATCH response alone is insufficient: complete identity and requested archive state must agree, followed by a fresh read. Reject an intervening rename or inaccessible target; do not retry another URL or undo earlier writes. These checks are observations, not an atomic identity lock or concurrency guarantee for GitHub's name-addressed PATCH endpoint.

Defer visibility management. A future single `visibility` enum, rather than competing `private` and `visibility` fields, needs an explicitly reviewed exposure/entitlement/fork-effects contract and an independent operation boundary. Do not publish private code or logs as a side effect of ordinary settings reconciliation or infer internal visibility from a private boolean.

Defer name management. Renaming can invalidate the origin-derived next-run target and downstream Action references even when web/Git redirects exist. A future standalone identity-migration operation must establish immutable ID binding, destination conflicts, consumers, and the local-origin handoff. Do not add a generic `name` setter, follow redirects, or edit local Git configuration implicitly. Empty, same-name, and changed-name declarations all remain unsupported until that separate decision.

## Consequences

Archive adds a bounded lifecycle concern with explicit ordering, acquired state, opt-in export, and independently reported reads/transitions. All existing unmanaged-resource behavior and ordinary update order remain. Failure after successful unarchive or ordinary writes leaves those effects in place; failure around archive may leave it archived and must be reported as uncertain until observed. Existing public Action source/version and reporting contracts still apply.

The [specification](../features/repository-lifecycle.feature.md) now maps the implemented source behavior to local tests. No release, real repository archive, visibility/name change, permission expansion, or credential operation was performed. Fixture results do not establish live mutation success. Deferred candidates remain visible in the roadmap for a later adoption decision.
