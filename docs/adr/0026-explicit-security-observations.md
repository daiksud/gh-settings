---
type: ADR
title: Require explicit, complete security observations
description: Separate repository security intent from permission, entitlement, and policy uncertainty.
---

## Status

Accepted design for [#343](https://github.com/daiksud/gh-settings/issues/343). The Secret scanning subset from #345 and private vulnerability reporting for public repositories from #347 are implemented in v0.17.0; other public families remain planned/gated. Current source implements Dependabot under the scoped reading and accepted-disable decisions below. Published v0.16.0 and older packages reject security keys/options. Implementation remains split across #344–#347, subject to the gates below.

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

Public family implementations must add the public references, native CLI/Action presentation, closed report validation, and fixture coverage together. Dependabot retains the distinction between verified state and an explicitly accepted disable with uncertain retained state. Local fixtures establish behavior against modeled responses, not live licensing, enforcement, permission grants, or scan outcomes. No settings, scans, contracts, grants, workflows, releases, or package versions change with this decision.

## Later scoped 404 decisions

### Dependabot state reads

For [#344](https://github.com/daiksud/gh-settings/issues/344), the user explicitly
decided on 2026-11-01 to stop distinguishing insufficient permission from
disabled state and interpret GET 404 as disabled for both `vulnerability-alerts`
and `automated-security-fixes`. This supersedes this ADR's requirement for an
authorization discriminator for those **state reads**. The choice permits
progress using the documented disabled response without additional credential
discovery, expanded grants, or a GraphQL replacement.

This is an accepted risk, not proof of authorization: an inaccessible setting
can be misclassified as disabled, so a plan can show a false change or
false equality and export can contain a false disabled value. Retain
404 provenance. Updates 404 supplies no `paused` value; do not invent false.
Explicit 401/403, incomplete 200 bodies and other read failures remain errors.

### Dependabot accepted disable

On 2026-11-01 the user separately chose to accept a disable when DELETE returns
204 and the subsequent target GET returns 404, while retaining uncertainty.
This is acceptance of an acknowledged request, **not independent verification**
that GitHub retained disabled state. Permission loss, concealment or enforcement
can still produce a misleading result. A 404 alone, a non-204 DELETE, or a PUT
followed by 404 never qualifies for this exception.

The adapter returns a distinct accepted-disable result. Apply retains it as
`OperationAcceptedUnverified`, emits a warning/notice, and preserves it as prior
progress if a later operation fails. It can proceed from an accepted child
disable to the explicitly declared parent disable; it must not describe the
child as independently verified. Fresh preflight and post-write family reads
still reject failures and inconsistent interpreted target/companion values.
Companion checks inherit the state-reading 404 risk rather than proving grants
or preservation under all policies. There is no retry, rollback or implicit
parent enablement.

Observation-only `VerifyEnabled` still rejects 404 for either desired value:
only the write adapter, after the required 204 acknowledgment, can accept the
scoped disabled result. A complete updates 200 verifies enabled independently
of acquired pause metadata; alerts 204 verifies enabled=true. The
[security specification](../features/security.feature.md#dependabot-state-reading-decision)
owns the public controls, endpoint rules, ordering and fixture mapping.

### Immutable release policy

The [immutable release policy](../features/immutable-releases.feature.md), adopted
separately for #348, interprets its GET 404 as disabled by explicit user decision.
An authorization-related 404 can therefore be misclassified; owner enforcement
is unknown on that response. Its separately adopted update-verification
behavior does not decide Dependabot mutation semantics or change other security
endpoints' missing/denied-response handling.
