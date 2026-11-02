---
type: ADR
title: Manage Pages lifecycle through UI Source and Branch controls
description: Declare publishing configuration and explicit Branch None deletion while keeping visibility and readiness separate.
---

## Status

Accepted for #349 and extended by the November 1, 2026 user decisions for #350. Creation and deletion are adopted; site visibility management is excluded. This decision supersedes the earlier lifecycle deferral and authorization/absence-discriminator requirement.

## Context

Pages combines singleton existence, Source, branch/folder, nullable custom domain and asynchronous deployment/certificate state. The UI creates branch publishing by selecting a source and deletes by selecting Deploy from a branch and Branch None, including for an Actions site. A separate enabled/state control would invent a UI setting.

All existing and future product specifications assume required caller permissions. For the target Pages GET, 404 therefore means absent. This is a specification precondition, not evidence of live grants or authorization for credential, security, site or publication changes. Other failures remain errors. Endpoint-specific 404 semantics elsewhere remain distinct; earlier permission-ambiguity rationale/reporting requires a separate consistency change.

## Decision

Use the current UI paths and values in the [configuration reference](../settings.md#pages-configuration). Omitted Pages and empty mappings remain unmanaged. A complete explicit branch Source/Branch/folder declaration or explicit GitHub Actions Source creates when absent and reconciles when present. Incomplete creation intent fails before writing; partial existing-site ownership is preserved. Real branch/folder intent remains an atomic tuple.

Reserve exact lowercase `branch: none` with explicit `source: deploy_from_a_branch` for deletion. Reject folder, domain or HTTPS intent alongside it. Other branch casing is literal. The user accepts the collision with a real branch of the reserved spelling. Do not introduce aliases, enabled/state controls or implicit deletion through null/omission.

Represent acquired absence separately from a complete present observation and the unknown zero value. Opt-in `export --pages` exports the deletion selector for absence, intentionally capable of deleting a site present at a later apply. Export only configured values for presence; never export visibility, readiness or a guessed branch. Reject an unrepresentable reserved literal branch rather than emitting contradictory YAML.

Use target-derived GET/POST/PUT/DELETE with redirect rejection, bounded requests and fresh identity/archive checks. POST contains only creation-supported source fields. Require 201 plus complete matching readback before any optional declared-only domain/HTTPS PUT. Omitted optional controls retain server defaults. PUT requires 204 and verifies exact declared values, omitted applicable companions and visibility. DELETE requires 204 (or a concurrent 404) plus fresh GET 404. All other failures, including POST/PUT readback 404, remain errors. No automatic write retry, rollback or compensating configuration is introduced.

Read-only plan shows creation/deletion and native declared values without fabricating absent before values. Internal lifecycle report metadata is not additional YAML syntax. Confirmed creation remains separately reported progress if its optional follow-up fails. Explicit unarchive runs first; final archive runs only after Pages succeeds.

Do not manage site visibility or infer it from repository privacy. Observe the server's creation choice and preserve it during optional updates. Do not alter billing, organization policy, DNS, workflows or content to work around a denial. GitHub itself can publish or create a CNAME-file commit as a configuration side effect.

## Consequences

Existing complete source declarations now create when absent. The reserved branch spelling intentionally changes its interpretation; accompanying folder intent is rejected. The v0.x public contract documents these effects without retaining an obsolete existing-site-only implementation.

Bounded verification establishes configuration convergence, not DNS, certificate or deployment readiness. Fixtures establish client behavior only. This implementation authorizes no live Pages changes, grant changes, release, commit, push or merge. The [behavior specification](../features/pages.feature.md) owns test mappings and detailed failures.
