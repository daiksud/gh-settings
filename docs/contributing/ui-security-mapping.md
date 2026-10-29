---
type: Reference
title: Prepared Security UI mappings
description: Internal supported Security controls, API conversion and activation handoff.
sources:
  - id: advanced-security-ui
    resource: https://github.com/daiksud/gh-settings/settings/security_analysis
  - id: general-ui
    resource: https://github.com/daiksud/gh-settings/settings
  - id: current-contract
    resource: ../settings.md
---

## Preparation boundary

This Security mapping was prepared for [#485](https://github.com/daiksud/gh-settings/issues/485) and is composed at its real UI locations by current-source [#479](https://github.com/daiksud/gh-settings/issues/479). See the [public reference](../settings.md#secret-scanning-and-push-protection). [#480](https://github.com/daiksud/gh-settings/issues/480) normalization/warnings and [#481](https://github.com/daiksud/gh-settings/issues/481) complete reporting remain separate. No compatibility alias, migration or entitlement is added.

The [UI model](../../internal/settings/uimodel/security.go) contains optional booleans with display terminology. The [GitHub adapter](../../internal/github/ui_security.go) translates those controls to existing validated security declarations and consumes acquired observations without declaration normalization. A complete inconsistent source remains visible as acquired state and fails conversion to a desired declaration; it is not silently repaired.

## Verified controls and placement

English UI headings and labels were inspected read-only on 2026-10-28. Current navigation is Security and quality / Advanced Security. The repository feature formerly represented by API `secret_scanning` appears as **Secret Protection**, with Enable/Disable actions. Push protection is the adjacent toggle beneath that heading. Private vulnerability reporting is a separate control on the same page. Release immutability is located under General / Releases. No control was changed.[^advanced-security-ui][^general-ui]

| UI location / control | Current public UI path | Internal API read / write |
| --- | --- | --- |
| Security and quality / Advanced Security / Secret Protection | `repository.settings.security_and_quality.advanced_security.secret_protection` | Repository GET `security_and_analysis.secret_scanning.status`; narrow PATCH same member, `enabled` / `disabled` |
| Security and quality / Advanced Security / Push protection | `repository.settings.security_and_quality.advanced_security.push_protection` | Repository GET `security_and_analysis.secret_scanning_push_protection.status`; narrow PATCH same member |
| Security and quality / Advanced Security / Private vulnerability reporting | `repository.settings.security_and_quality.advanced_security.private_vulnerability_reporting` | Public applicability GET, then endpoint boolean `enabled`; bodyless PUT / DELETE |
| General / Releases / Enable release immutability | `repository.settings.general.releases.enable_release_immutability` | Immutable-releases endpoint `enabled`; bodyless PUT / DELETE |

The internal `SecuritySettings.GeneralReleases` member remains a family adapter container, not a literal public path. Current composition places its control in General / Releases beside General fields from #482. Security owns this control's API conversion; it does not move the control into an invented Security page.

Secret Protection and Push protection are sibling boolean fields under Advanced Security: the first control itself is boolean, so no invented inner `enabled` UI label is needed. The navigation heading does not adopt Advanced Security product enrollment. Dependency Scanning, Dependabot, Code Security, AI scanning, user-level protection, partner notifications, bypasses, alert patterns and other candidates remain unsupported.

## Presence, defaults and acquired provenance

All four desired fields retain omission independently; explicit false remains managed. Empty internal containers manage nothing. Declarations copy selected pointers, validate the existing explicit Secret Protection=false / Push protection=true conflict and never implicitly enable a prerequisite. Observation overlays copy the affected nested container and newly acquired values while preserving unrelated incoming fields; they do not deep-copy an arbitrary incoming tree.

Secret Protection and Push protection are UI booleans sourced from finite API status strings. Supported `enabled` / `disabled` translate to true / false. Named `DefaultSecretProtection=false` and `DefaultPushProtection=false` are explicit per-field fallback candidates for #480 when a correctly typed unknown API status is normalized. They describe planned normalization policy, not the inspected repository's current state or a promise about new repository defaults. Current transport rejects unknown statuses; missing/null fields, denied requests and unacquired observations remain errors, never fallback triggers. Reporting and immutability are sourced from actual API booleans and need no unknown-enum fallback. None of these defaults populate omitted desired fields.

Immutable release owner enforcement remains outside the UI model as `PreparedSecurityImmutable.EnforcedByOwner`: nil means unknown, a false pointer means known not enforced and a true pointer means enforced. The existing product decision treats HTTP 404 as acquired disabled with unknown enforcement. A 200 response requires both booleans and rejects enforced=true with enabled=false. The existing authorization ambiguity of 404 is preserved, not generalized to other endpoints or presented as proof of access.[^current-contract]

## Dependency, applicability and endpoint semantics

Existing Secret Protection preflight requires already-enabled scanning or explicit true intent before enabling Push protection. Disabling scanning while Push protection is active requires explicit false intent for both. Enable scanning before protection; disable protection before scanning. Each narrow write refreshes prerequisites, sends only the affected status member and verifies the desired value and preserved companion. No unrelated security field is replayed.

Private vulnerability reporting remains public-repository-only; unavailable or inapplicable state is not false. Exact existing bodyless PUT/DELETE, fresh applicability checks and readback remain unchanged. Immutable policy remains a policy endpoint only: owner enforcement blocks disabling, only 204 writes succeed and fresh readback follows. Individual releases, tags and assets are never accessed by these helpers. There is no release operation, entitlement purchase, permission expansion, retry, rollback or protection bypass. Earlier confirmed and uncertain write effects retain existing failure semantics.[^current-contract]

## Validation and activation handoff

Independent [boolean/dependency tests](../../internal/github/ui_security_test.go), [reporting fixtures](../../internal/github/ui_security_reporting_test.go), [immutable fixtures](../../internal/github/ui_security_immutable_test.go) and [raw/PATCH composition fixtures](../../internal/github/ui_security_composition_test.go) specify expected values separately from production mappings. They cover all supported controls both ways, omission, copied ownership, raw inconsistency, missing acquisition, fresh visibility, exact payload/bodyless toggles, 404 owner-unknown and enforced-owner rejection. Existing [scanning](../../internal/github/security_test.go), [reporting](../../internal/github/private_reporting_test.go) and [immutability](../../internal/github/immutable_releases_test.go) endpoint matrices continue covering malformed/null data, permission failures and uncertain writes.

Current UI composition preserves actual paths, narrow ownership, applicability and dependency order. #480 must distinguish raw string statuses from UI boolean values before applying normalization and warnings; acquired boolean and owner-known metadata must survive into #481. Public user documentation now describes this UI YAML while keeping the published-package and unfinished-reporting boundaries explicit.

[^advanced-security-ui]: GitHub, Advanced Security settings, read-only UI inspection, 2026-10-28.
[^general-ui]: GitHub, General / Releases settings, read-only UI inspection, 2026-10-28.
[^current-contract]: Current repository documentation, Security settings and immutable release policy, consulted 2026-10-28 before public YAML activation; the current link now reflects the activated contract.
