---
type: ADR
title: Manage existing Pages configuration with explicit lifecycle boundaries
description: Separate Pages configuration from site creation, deletion, visibility and deployment readiness.
---

## Status

Accepted design for #349. The existing-site subset of #350 is published in v0.18.0; published v0.17.0 and older packages reject Pages keys/options. Site creation/deletion and site visibility mutation remain deferred pending the gates below, and #350 retains those unresolved lifecycle criteria.

## Context

Pages combines a singleton site's existence, publishing mode, a branch source, a nullable custom domain and asynchronous deployment/certificate state. Its GET can return 404 for an inaccessible resource. Public API readability does not establish write authorization, repository privacy does not establish site privacy, and a successful settings update does not establish that a deployment is serving.

The [Pages specification](../features/pages.feature.md#official-contracts) records the official API and product sources. The existing declared-only updates, direct apply and no-rollback contracts remain. [ADR 0019](0019-reject-explicit-null.md) applies to its original repository fields and explicitly permits separate decisions for future endpoints; Pages' documented domain-clearing null merits such a decision.

## Decision

Adopt `settings.pages` as an optional mapping for configuration of an existing, positively observed site. Adopt optional `build_type`, complete `source: {branch, path}`, `cname`, and `https_enforced` under the strict field contract in the specification. Omission preserves ownership; an empty top-level mapping manages nothing. Only `cname` accepts explicit null, meaning remove the custom domain. Neither omitted groups nor null groups mean delete the site. Keep branch source as a complete tuple instead of inventing partial-source merging.

Use explicit `export --pages` to acquire this resource; ordinary export and its privileges remain unchanged. Export only supported configured state after a complete read, never ephemeral build/certificate state or a guessed disabled value. A missing or inaccessible requested site fails export without replacing the destination. Preserve nullable domain presence through decode, comparison, export and strict native CLI/Action reports without making unrelated fields nullable.

Separate configured publishing state from operational readiness. Retain branch source only for legacy mode; workflow mode must not acquire a publishing branch by inference. An explicit transition to legacy requires an explicit complete source. A transition to workflow rejects a source declaration. Fresh preflight and readback verify the owned changes and unchanged applicable companions, including site visibility. Mode changes may make a previously active source inactive; do not silently activate a stale source when returning to legacy.

The adopted API path is GET/PUT on the origin-derived repository's Pages singleton, with redirects rejected. Require complete observations and use declared-only update bodies. A 404 remains unknown/inaccessible: never translate it to absent, disabled, a creation request, an empty export or a successful deletion. Other endpoints, actor roles, token requirements headers and App installation permissions do not prove the exact observation token's grants. Apply never uses a write as an existence probe.

Defer site creation and deletion. Do not accept `enabled`, `state`, a deletion marker or a visibility key in the first implementation. Before adopting lifecycle syntax, establish an endpoint-specific absence/authorization discriminator, creation exposure and entitlement conditions, creation-required values, and verifiable deletion/idempotence. Until those prerequisites are met, neither POST nor DELETE is permitted. An existing-site update must not silently create on 404. #350 must retain the unresolved lifecycle work visibly rather than claiming full CRUD completion.

Do not write `public` back just because GET returns it. Site visibility and access control need their own adopted write contract. Do not change repository visibility, billing, organization policy, workflows, DNS, source files or deployment settings to work around a denied operation. Configuration changes may nevertheless cause GitHub-managed publishing or domain-file effects; the tool must explain those effects and cannot promise that API settings updates are isolated from publication.

## Consequences

The first implementation can manage positively observed existing configuration without guessing site absence or expanding access. It needs one independently acquired Pages observation, strict nullable-domain intent, bounded native comparisons, opt-in export, lifecycle-aware sequencing and representative binary tests. No general resource registry, ownership database, saved plan, transaction or automatic rollback is required.

The specification defines bounded immediate verification, not waiting for DNS, certificates or deployment. Denied preflight makes no Pages write; uncertain update/readback errors retain earlier successful operations. Explicit unarchive is before Pages updates; final archive is after them. Product implementation does not authorize any live Pages/DNS/publication change or credential operation in this repository.
