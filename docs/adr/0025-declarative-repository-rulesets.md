---
type: ADR
title: Reconcile repository Rulesets as declarative collections
description: Adopt label-style ownership for repository Rulesets while preserving complete observations.
---

## Status

Accepted; the basic branch/tag subset is implemented in current source, unreleased. Published v0.16.0 and older packages do not support it. Design issue [#335](https://github.com/daiksud/gh-settings/issues/335) precedes the separate basic lifecycle and PR/check implementation issues. PR/check support remains separate future work.

## Context

The goal is to codify existing repository Rulesets: export their configuration, obtain a zero-difference plan without editing it, then reconcile deliberate edits. Labels already distinguish unmanaged collections from authoritative membership and preserve omitted attributes on retained objects. Inventing a separate replacement mode, ownership marker, or adoption database would make the same declaration mean different things across resources.

Rulesets differ at the API boundary: names must resolve to numeric write IDs, inherited policies have different owners, updates use PUT, and a successful read may hide bypass actors. A partial typed projection can silently remove protection when a rules array is rewritten. The previous architecture left deletion and inner collection semantics unresolved.

## Decision

Use the existing labels declaration model for the outer Rulesets collection and declared inner collections. Omission leaves a field unmanaged; a declared collection owns membership, including deletion; omitted attributes of retained members preserve observed values. Reject YAML null rather than treating it as omission or clearing. No rename is inferred.

The [Rulesets design specification](../features/rulesets.feature.md) owns the exact planned schema, identity keys, ownership boundary, complete-read requirements, reconciliation, failure behavior, and acceptance matrix. Outer Rulesets use exact names, not labels' Unicode case folding: no verified Rulesets API guarantee establishes that differently cased names are the same object. Resolve unique names to fresh observed IDs; ambiguous identity is an error.

Manage only branch/tag Rulesets owned by the target repository. Keep inherited organization/enterprise policies, push Rulesets, and legacy branch protection outside that collection. Defer legacy protection implementation under [#338](https://github.com/daiksud/gh-settings/issues/338); do not migrate, replace, or delete it through Rulesets.

Add explicit Rulesets export opt-in when implementing this feature, preserving ordinary export's existing read/permission boundary. Require complete writable observations for opted-in export and declared Rulesets reconciliation. Missing bypass information, unknown writable content, or unsupported rule types stop those operations; they never become empty state or a successful partial export. Read-only metadata is excluded by an explicit allowlist.

## Consequences

Declared Rulesets membership can remove protection, just as declared labels membership can delete labels. An explicit empty list authorizes deletion within the documented repository branch/tag boundary; it does not erase inherited or legacy protection. A zero plan is not a lock, transaction, rollback guarantee, or proof that a particular workflow enforces the desired policy.

Failing on unsupported content limits initial adoption, but makes the boundary honest. The basic lifecycle implementation must not claim to round-trip repositories containing PR/check rules before the extension supports their complete writable shape. Newly observed fields require classification and evidence before support; raw unknown JSON is not blindly replayed into privileged requests.

This decision changes no executable, workflow, permission, credential, live protection, or release. Implementation PRs must add the public contract, native CLI/Action presentation, and the specified tests together, under normal review and CI.
