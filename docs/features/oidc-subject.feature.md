---
type: Feature
title: Manage the repository OIDC subject template
description: Reconcile the active template while preserving the immutable companion.
---

## Feature: Manage the Subject claim active template

The [configuration reference](../settings.md#actions-oidc-subject-template) owns
YAML, permissions, whole-template ownership and cloud trust coordination.

### Scenario: Validate the complete active union before remote work

Given an omitted or empty group, it remains unmanaged. Given default true, keys
must be absent. Given custom false, a unique ordered ASCII key array is required,
including explicit empty. Null, unknown members, coercions and incomplete unions
fail locally; no closed claim-name enum or invented array minimum is applied.

### Scenario: Export acquired state independently

A complete known default exports true without inactive keys. A complete custom
template exports false with exact ordered keys. An unavailable family retains
valid prior whole intent or is omitted on first export; sibling reads continue.
Missing optional immutable mode does not become false or block active export.

### Scenario: Reconcile with exact companion preservation

Plan compares one managed group. Apply rereads target identity and complete active
state. A match sends no PUT even with unknown immutable mode. A needed update
requires known immutable mode and replays its exact true/false value. Custom keys
are sent only for false mode; default selects the default template without a
hidden inactive-key retention promise. An archived or incomplete target prevents
a needed write. No mutable immutable-mode key exists.

### Scenario: Confirm a write independently

Only HTTP 201 with a bounded empty body, JSON null or strict empty object is a valid
acknowledgment. A fresh independent GET must confirm the same repository/owner
identity, archive state, complete active template and preserved immutable mode.
Wrong status, malformed acknowledgment, denied readback or drift fails with
uncertain effects. Prewrite failures remain distinct, earlier verified writes
remain recorded and later work stops. No automatic retry or rollback occurs.

### Scenario: Keep private evidence out of public output

CLI and detailed Action output show ordered UI intent, with strict same-version
report reconstruction. Status-only output hides the group. Raw API bodies and
subject prefixes remain private with GH_DEBUG enabled. The actual Action publishes
success outputs only after a verified update or no-op and does not modify YAML.
No scenario establishes actual tokens, cloud authentication, grants or billing.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Exact active union, acquired observations and ordered comparisons | [Settings tests](../../internal/settings/oidc_subject_test.go) |
| Literal YAML, strict types, omitted groups and explicit empty keys | [Workspace tests](../../internal/workspace/ui_oidc_subject_test.go) |
| Complete reads, no-op, exact payload and independent preservation | [API tests](../../internal/github/oidc_subject_test.go) |
| Export retention, one managed group and honest progress | [App tests](../../internal/app/oidc_subject_test.go) |
| Closed same-version report reconstruction | [Action tests](../../internal/actions/oidc_subject_test.go) |
| Built CLI lifecycle, unavailable retention and debug privacy | [Process tests](../../tests/acceptance/oidc_subject_test.go) |
| Supplied product, actual composite, status-only privacy and output timing | [Composite tests](../../tests/github-actions/oidc_subject_acceptance_test.go) |

The unsaved personal-public UI and official [OIDC API](https://docs.github.com/en/rest/actions/oidc)
provide configuration evidence. Fixtures do not establish live mutation behavior
or cloud trust; no live settings changes are part of this work.
