---
type: ADR
title: Require explicit, complete security observations
description: Separate repository security intent from permission, entitlement, and policy uncertainty.
---

## Status

The original per-family export selection and fail-fast export policy is superseded
by the later automatic readable export decision below. Strict observations for
plan/apply and product-switch exclusions remain.

Accepted design for [#343](https://github.com/daiksud/gh-settings/issues/343). Current source implements Secret scanning (#345), private vulnerability reporting (#347), Dependabot (#344), and the separately documented CodeQL and AI Scan controls. Product-wide Code Security and Advanced Security enablement (#346) is excluded under the later UI-scope decision below; it is not pending implementation.

## Context

Repository security controls span independent APIs, product entitlements, and organization policies. A missing field or an ambiguous 404 cannot safely mean disabled. A repository read can succeed while hiding security data. A successful update can also leave an organization-enforced setting unchanged. Treating any of these as a complete boolean observation would produce misleading plans or destructive declarations.

The existing declared-field, direct-apply, and no-rollback decisions still apply. Security does not justify a separate ownership database, automatic enrollment in a paid product, broader default export privileges, or a transaction abstraction.

## Decision

Add optional boolean declarations under `settings.security` only in the corresponding implementation PR. Omission is unmanaged, false is an explicit disable, null is invalid, and an empty mapping manages nothing. Use separate, validated observations for actual values and operational metadata. Unknown, denied, and inapplicable observations are errors for requested comparisons/export, never false values or successful partial results, except for the explicitly scoped 404 reading decisions below.

Choose explicit, repeatable per-family export selection: proposed `export --security dependabot`, `--security secret-scanning`, or `--security private-vulnerability-reporting`. There is no implicit all-security selection. Existing export requests and privileges remain unchanged. Reserved product-family names must not become accepted options before adoption.

The [security specification](../features/security.feature.md) owns the schema, endpoint and permission matrix, dependency ordering, failure semantics, implementation gates, and acceptance plan. It chooses declared-only nested PATCHes, fresh prerequisite reads, and complete post-write verification, including the case where an enforced policy returns apparent success without changing state. No automatic retries, rollback, parent enabling, or product substitution are introduced.

Adopt the narrow Secret scanning and public-repository private vulnerability reporting contracts. Dependabot follows the later scoped 404 decisions below, with ordered narrow mutations and fresh target/companion readback. The original product-applicability gate for #346 is superseded by the later UI-scope decision below. Code Security and legacy Advanced Security remain distinct API concepts, not aliases or adopted product mutations. This exclusion does not block independent supported controls.

## Consequences

Opted-in export can fail when requested data is unavailable; it cannot silently omit it or overwrite a file with partial configuration. Paid-product availability and write authorization are not implied by API schema presence, a repository role, or an unchanged plan. Plan remains read-only, and apply remains direct and nontransactional with honest partial progress.

Public family implementations must add the public references, native CLI/Action presentation, closed report validation, and fixture coverage together. Dependabot verifies enabled intent under the required-permissions precondition while retaining unavailable pause metadata. Local fixtures establish behavior against modeled responses, not live licensing, enforcement, permission grants, or scan outcomes. No settings, scans, contracts, grants, workflows, releases, or package versions change with this decision.

## Later product-switch scope decision

On 2026-11-11, [#346](https://github.com/daiksud/gh-settings/issues/346) was dispositioned not planned under the [Free/personal/public/visible-and-configurable Web UI policy](../settings.md#feature-adoption-scope). Read-only inspection of `daiksud/test-gh-settings-public` at Settings / Advanced Security found **Advanced Security** and **Code Security** headings, but no corresponding product-wide Enable/Disable control. CodeQL and AI Scan had individual controls; Secret Protection had its own Disable action. The exclusion follows the missing product switch in the target UI, not pricing, a Copilot subscription, or the absence of an API field.

A separately authorized, single repository PATCH on 2026-11-11 at 14:39:01 UTC sent only `security_and_analysis.code_security.status=enabled`. GitHub returned HTTP 422: "When billing is bundled Code Security can only be enabled if Advanced Security is enabled." Repository GET, AI Scan, CodeQL default setup and Actions workflow/run reads matched their pre-request values. No observable change was found; missing `code_security` and `advanced_security` fields remained unavailable, not disabled. This rejection does not prove a universal public-repository restriction or justify enabling another product. No retry, restoration PATCH, contract change or scan-execution request followed.

The [Security specification](../features/security.feature.md#product-switch-exclusion) records the resulting unsupported boundary. Existing CodeQL and AI Scan implementations remain separate from product enablement. Other individual UI controls with unconfirmed public API paths retain their own evidence gaps.

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

## Later automatic readable export decision

On 2026-11-12 the user replaced opt-in export and privilege compatibility with
automatic observation of supported Security settings: export readable values,
omit unreadable values, emit info and continue independent acquisition. The old
`--security FAMILY` interface is removed under the v0.x compatibility policy.

Unknown is not false. First export omits unavailable families; re-export retains
valid existing intent for them. Bound diagnostics identify the family and reason
without API bodies/messages/URLs or credentials. CodeQL state and details
use a single response. Confirmed disabled state exports false and removes prior
details with info; unknown state retains the prior pair. Configured state exports
true even if unavailable details must retain prior intent. Fundamental
target/core-read/file/diagnostic IO failures remain errors. Existing malformed
files or concurrent local edits are preserved by snapshot guards.

This supersedes the selection/partial-result clauses above, not strict plan/apply
reads, endpoint-specific 404 semantics, existing acquired-value normalization,
product adoption scope or mutation permissions. Other resource opt-ins retain
their existing scope. The [current behavior specification](../features/readable-security-export.feature.md)
owns acceptance. No live setting, scan, permission or release is changed.

## Later default readable-settings decision

On 2026-11-13 the user extended automatic export to every supported readable
setting family. This supersedes export opt-in/selection clauses above; strict
plan/apply and existing API-specific absence rules remain. See the
[automatic export contract](../features/readable-settings-export.feature.md).
Independent unavailable families emit bounded info, continue other reads and
retain valid prior intent; no secret API or new unsupported setting is adopted.
