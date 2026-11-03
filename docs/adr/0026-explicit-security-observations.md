---
type: ADR
title: Require explicit, complete security observations
description: Separate repository security intent from permission, entitlement, and policy uncertainty.
---

## Status

Accepted design for [#343](https://github.com/daiksud/gh-settings/issues/343). The Secret scanning subset from #345 and private vulnerability reporting for public repositories from #347 are implemented in v0.17.0; other public families remain planned/gated. Current source implements Dependabot under the scoped reading and verification decisions below. Published v0.16.0 and older packages reject security keys/options. Implementation remains split across #344–#347, subject to the gates below.

## Context

Repository security controls span independent APIs, product entitlements, and organization policies. A missing field or an ambiguous 404 cannot safely mean disabled. A repository read can succeed while hiding security data. A successful update can also leave an organization-enforced setting unchanged. Treating any of these as a complete boolean observation would produce misleading plans or destructive declarations.

The existing declared-field, direct-apply, and no-rollback decisions still apply. Security does not justify a separate ownership database, automatic enrollment in a paid product, broader default export privileges, or a transaction abstraction.

## Decision

Add optional boolean declarations under `settings.security` only in the corresponding implementation PR. Omission is unmanaged, false is an explicit disable, null is invalid, and an empty mapping manages nothing. Use separate, validated observations for actual values and operational metadata. Unknown, denied, and inapplicable observations are errors for requested comparisons/export, never false values or successful partial results, except for the explicitly scoped 404 reading decisions below.

Choose explicit, repeatable per-family export selection: proposed `export --security dependabot`, `--security secret-scanning`, or `--security private-vulnerability-reporting`. There is no implicit all-security selection. Existing export requests and privileges remain unchanged. Reserved product-family names must not become accepted options before adoption.

The [security specification](../features/security.feature.md) owns the schema, endpoint and permission matrix, dependency ordering, failure semantics, implementation gates, and acceptance plan. It chooses declared-only nested PATCHes, fresh prerequisite reads, and complete post-write verification, including the case where an enforced policy returns apparent success without changing state. No automatic retries, rollback, parent enabling, or product substitution are introduced.

Adopt the narrow Secret scanning and public-repository private vulnerability reporting contracts. Dependabot follows the later scoped 404 decisions below, with ordered narrow mutations and fresh target/companion readback. Keep #346 a distinct product candidate: reserve separate names for Code Security and legacy Advanced Security, reject treating them as aliases, and require evidence of product applicability and collateral effects before adopting either for mutation. These gates do not block independent features.

## Consequences

Opted-in export can fail when requested data is unavailable; it cannot silently omit it or overwrite a file with partial configuration. Paid-product availability and write authorization are not implied by API schema presence, a repository role, or an unchanged plan. Plan remains read-only, and apply remains direct and nontransactional with honest partial progress.

Public family implementations must add the public references, native CLI/Action presentation, closed report validation, and fixture coverage together. Dependabot verifies enabled intent under the required-permissions precondition while retaining unavailable pause metadata. Local fixtures establish behavior against modeled responses, not live licensing, enforcement, permission grants, or scan outcomes. No settings, scans, contracts, grants, workflows, releases, or package versions change with this decision.

## Later scoped 404 decisions

### Dependabot state reads

For [#344](https://github.com/daiksud/gh-settings/issues/344), GET 404 means
disabled for `vulnerability-alerts` and `automated-security-fixes` under the
required-permissions precondition, consistent with the adopted Pages contract.
No authorization discriminator, credential discovery, expanded grants or
GraphQL replacement is required. This interpretation does not certify actual
grants or independently establish live platform behavior. Retain 404 provenance;
updates 404 supplies no `paused` value. Explicit 401/403, incomplete 200 bodies,
timeouts and other read failures remain errors.

### Dependabot update verification

The permission-precondition reconciliation supersedes the earlier
accepted-unverified disable contract: DELETE 204 followed by fresh target
GET 404 now verifies disabled enabled intent under that precondition. Apply
records a completed operation without a permission-risk or accepted-disable
warning/notice. A GET alone does not establish that a write ran. PUT followed
by 404, a non-204 write, target/companion mismatch or failed/incomplete family
readback still fails with possible write effects.

Each changed control gets fresh whole-family preflight and readback. Enable
alerts before updates; disable updates before alerts. A completed child disable
can precede an explicitly declared parent disable. Later failure retains
completed prior progress; no retry, rollback or implicit parent enablement is
introduced. These endpoint-specific rules do not generalize 404 to other APIs.

Observation-only `VerifyEnabled` compares acquired enabled intent: 404 matches
false, alerts 204 matches true, and complete updates 200 matches its enabled
boolean independently of pause. Unknown pause remains unavailable and paused
updates are never implicitly resumed. The
[security specification](../features/security.feature.md#dependabot-state-reading-decision)
owns the public controls, endpoint rules, ordering and fixture mapping.

### Immutable release policy

The [immutable release policy](../features/immutable-releases.feature.md), adopted
separately for #348, interprets its GET 404 as disabled by explicit user decision.
Required permissions are a precondition; the response still supplies no owner
enforcement metadata. Its owner-enforcement gate and endpoint-specific
update verification remain distinct from Dependabot pause metadata. Neither
contract changes other security endpoints' missing/denied-response handling.
