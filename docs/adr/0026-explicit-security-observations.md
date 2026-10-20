---
type: ADR
title: Require explicit, complete security observations
description: Separate repository security intent from permission, entitlement, and policy uncertainty.
---

## Status

Accepted design for [#343](https://github.com/daiksud/gh-settings/issues/343). The Secret scanning subset from #345 is implemented in current source, unreleased; other families remain planned/gated. Published v0.16.0 and older packages reject security keys/options. Implementation remains split across #344–#347, subject to the gates below.

## Context

Repository security controls span independent APIs, product entitlements, and organization policies. A missing field or an ambiguous 404 cannot safely mean disabled. A repository read can succeed while hiding security data. A successful update can also leave an organization-enforced setting unchanged. Treating any of these as a complete boolean observation would produce misleading plans or destructive declarations.

The existing declared-field, direct-apply, and no-rollback decisions still apply. Security does not justify a separate ownership database, automatic enrollment in a paid product, broader default export privileges, or a transaction abstraction.

## Decision

Add optional boolean declarations under `settings.security` only in the corresponding implementation PR. Omission is unmanaged, false is an explicit disable, null is invalid, and an empty mapping manages nothing. Use separate, validated observations for actual values and operational metadata. Unknown, denied, and inapplicable observations are errors for requested comparisons/export, never false values or successful partial results.

Choose explicit, repeatable per-family export selection: proposed `export --security dependabot`, `--security secret-scanning`, or `--security private-vulnerability-reporting`. There is no implicit all-security selection. Existing export requests and privileges remain unchanged. Reserved product-family names must not become accepted options before adoption.

The [security specification](../features/security.feature.md) owns the schema, endpoint and permission matrix, dependency ordering, failure semantics, implementation gates, and acceptance plan. It chooses declared-only nested PATCHes, fresh prerequisite reads, and complete post-write verification, including the case where an enforced policy returns apparent success without changing state. No automatic retries, rollback, parent enabling, or product substitution are introduced.

Adopt the narrow Secret scanning and public-repository private vulnerability reporting contracts. Dependabot's ambiguous disabled-state reads require an authorization discriminator before full disabled-state round-trip claims; without one, defer #344 rather than infer grants from another endpoint. Keep #346 a distinct product candidate: reserve separate names for Code Security and legacy Advanced Security, reject treating them as aliases, and require evidence of product applicability and collateral effects before adopting either for mutation. These gates do not block independent features.

## Consequences

Opted-in export can fail when requested data is unavailable; it cannot silently omit it or overwrite a file with partial configuration. Paid-product availability and write authorization are not implied by API schema presence, a repository role, or an unchanged plan. Plan remains read-only, and apply remains direct and nontransactional with honest partial progress.

The design changes documentation only. Later implementations must add the public references, native CLI/Action presentation, closed report validation, and fixture coverage together. Local fixtures establish behavior against modeled responses, not live licensing, enforcement, permission grants, or scan outcomes. No settings, scans, contracts, grants, workflows, releases, or package versions change with this decision.
