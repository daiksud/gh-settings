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

This Security mapping was prepared for [#485](https://github.com/daiksud/gh-settings/issues/485) and composed by [#479](https://github.com/daiksud/gh-settings/issues/479). The current page/section structure follows [#541](https://github.com/daiksud/gh-settings/issues/541); the [public reference](../settings.md#secret-scanning-and-push-protection) owns accepted YAML. No compatibility alias, migration or entitlement is added.

The [UI model](../../internal/settings/uimodel/security.go) contains optional booleans with display terminology. The [public document](../../internal/workspace/ui_settings_document.go) groups those independent controls for the single YAML codec. The [GitHub adapter](../../internal/github/ui_security.go) translates the family controls to existing validated security declarations and consumes acquired observations without declaration normalization. A complete inconsistent source remains visible as acquired state and fails conversion to a desired declaration; it is not silently repaired.

## Verified controls and placement

English UI headings and labels were inspected read-only on 2026-10-28. The inspected navigation was Security and quality / Advanced Security. The repository feature formerly represented by API `secret_scanning` appeared as **Secret Protection**, with Enable/Disable actions. Push protection was the adjacent toggle beneath that heading. Private vulnerability reporting was a separate control on the same page. Release immutability was located under General / Releases. No control was changed.[^advanced-security-ui][^general-ui]

| UI location / control | Current public UI path | Internal API read / write |
| --- | --- | --- |
| Security and quality / Advanced Security / Secret Protection | `repository.settings.advanced_security.secret_protection.enabled` | Repository GET `security_and_analysis.secret_scanning.status`; narrow PATCH same member, `enabled` / `disabled` |
| Security and quality / Advanced Security / Push protection | `repository.settings.advanced_security.secret_protection.push_protection` | Repository GET `security_and_analysis.secret_scanning_push_protection.status`; narrow PATCH same member |
| Security and quality / Advanced Security / Private vulnerability reporting | `repository.settings.advanced_security.private_vulnerability_reporting` | Public applicability GET, then endpoint boolean `enabled`; bodyless PUT / DELETE |
| General / Releases / Enable release immutability | `repository.settings.general.releases.immutable_release` | Immutable-releases endpoint `enabled`; bodyless PUT / DELETE |

The internal `SecuritySettings.GeneralReleases` member remains a family adapter container, not a literal public path. Current composition places its control in General / Releases beside General fields from #482. Security owns this control's API conversion; it does not move the control into an invented Security page.

The navigation-only Security and quality heading is omitted from YAML. The Secret Protection section contains independently optional `enabled` and `push_protection` booleans; private reporting remains a sibling of that section. These approved document keys do not claim to be HTML form attributes or introduce product enrollment. Dependency Scanning, Code Security, AI scanning, user-level protection, partner notifications, bypasses, alert patterns and other candidates remain unsupported.

Dependabot now maps `repository.settings.advanced_security.dependabot.alerts` and `security_updates` to two independently optional native booleans. The [public Dependabot contract](../settings.md#dependabot-alerts-and-security-updates) owns permission, dependency-graph effects, pause metadata and the explicit GET 404/accepted-disable risks. These controls add no grouped-updates or auto-triage setting. [Codec fixtures](../../internal/workspace/ui_dependabot_test.go) and the [executable cycle](../../tests/acceptance/dependabot_test.go) verify literal current UI declarations through the existing native adapters.

## Presence, defaults and acquired provenance

All six desired fields retain omission independently; explicit false remains managed. Empty internal containers manage nothing. Declarations copy selected pointers, validate the existing explicit Secret Protection=false / Push protection=true conflict and never implicitly enable a prerequisite. Observation overlays copy the affected nested container and newly acquired values while preserving unrelated incoming fields; they do not deep-copy an arbitrary incoming tree.

Secret Protection and Push protection are UI booleans sourced from finite API status strings. Supported `enabled` / `disabled` translate to true / false. Named `DefaultSecretProtection=false` and `DefaultPushProtection=false` are active per-field fallbacks under #480 when a correctly typed unknown API status is normalized. They describe client observation policy, not the inspected repository's current state or a promise about new repository defaults. Complete typed unknown, empty and whitespace statuses retain exact raw provenance alongside their effective booleans; missing/null fields, denied requests and unacquired observations remain errors, never fallback triggers. Reporting and immutability are sourced from actual API booleans and need no unknown-enum fallback. None of these defaults populate omitted desired fields.

Immutable release owner enforcement remains outside the UI model as `PreparedSecurityImmutable.EnforcedByOwner`: nil means unknown, a false pointer means known not enforced and a true pointer means enforced. The existing product decision treats HTTP 404 as acquired disabled with unknown enforcement. A 200 response requires both booleans and rejects enforced=true with enabled=false. The existing authorization ambiguity of 404 is preserved, not presented as proof of access. Dependabot has its own separately adopted 404 reading and accepted-disable policies.[^current-contract]

## Dependency, applicability and endpoint semantics

Existing Secret Protection preflight requires already-enabled scanning or explicit true intent before enabling Push protection. Disabling scanning while Push protection is active or unverified requires explicit false intent for both. Enable scanning before protection; disable protection before scanning. Each narrow write refreshes prerequisites, sends only the affected status member and verifies exact fresh raw statuses for the desired member and preserved companion. Unknown raw companions cannot establish prerequisite certainty; normalized false-to-false intent remains a required write. No unrelated security field is replayed.

Private vulnerability reporting remains public-repository-only; unavailable or inapplicable state is not false. Exact existing bodyless PUT/DELETE, fresh applicability checks and readback remain unchanged. Immutable policy remains a policy endpoint only: owner enforcement blocks disabling, only 204 writes succeed and fresh readback follows. Individual releases, tags and assets are never accessed by these helpers. There is no release operation, entitlement purchase, permission expansion, retry, rollback or protection bypass. Earlier confirmed and uncertain write effects retain existing failure semantics.[^current-contract]

## Validation and activation handoff

Independent [boolean/dependency tests](../../internal/github/ui_security_test.go), [reporting fixtures](../../internal/github/ui_security_reporting_test.go), [immutable fixtures](../../internal/github/ui_security_immutable_test.go) and [raw/PATCH composition fixtures](../../internal/github/ui_security_composition_test.go) specify expected values separately from production mappings. They cover all supported controls both ways, omission, copied ownership, raw inconsistency, missing acquisition, fresh visibility, exact payload/bodyless toggles, 404 owner-unknown and enforced-owner rejection. Existing [scanning](../../internal/github/security_test.go), [reporting](../../internal/github/private_reporting_test.go) and [immutability](../../internal/github/immutable_releases_test.go) endpoint matrices continue covering malformed/null data, permission failures and uncertain writes.

The [literal page/section fixtures](../../internal/workspace/ui_page_sections_test.go) cover independent omission, explicit false, export round trips and rejection of superseded shapes. Current UI composition preserves narrow ownership, applicability and dependency order. The [HTTP matrix](../../internal/github/security_status_apply_test.go) and [executable cycle](../../tests/acceptance/normalization_security_status_test.go) verify scoped diagnostics, raw verification, dependencies and inconsistency using the current paths. Acquired booleans and owner-known metadata remain distinct from desired intent. Source behavior does not upgrade a published package or apply live settings.

[^advanced-security-ui]: GitHub, Advanced Security settings, read-only UI inspection, 2026-10-28.
[^general-ui]: GitHub, General / Releases settings, read-only UI inspection, 2026-10-28.
[^current-contract]: Current repository documentation, Security settings and immutable release policy, consulted 2026-10-28 before public YAML activation; the current link now reflects the activated contract.
