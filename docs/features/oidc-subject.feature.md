---
type: Feature
title: Manage the repository OIDC subject template
description: Reconcile independently selected active-template and immutable configuration.
---

## Feature: Manage the Subject claim active template

The [configuration reference](../settings.md#actions-oidc-subject-template) owns
YAML, permissions, active-template ownership and cloud trust coordination.

### Scenario: Validate the complete active union before remote work

Given an omitted or empty group, it remains unmanaged. Given default true, keys
must be absent. Given custom false, a unique ordered ASCII key array is required,
including explicit empty. Null, unknown members, coercions and incomplete unions
fail locally; no closed claim-name enum or invented array minimum is applied.
The independent immutable boolean accepts true or false without a template
selection. Declaring it does not relax the template union or invent a date-based
eligibility rule. Configuration intent is not effective token-format evidence.

### Scenario: Export acquired state independently

A complete known default exports true without inactive keys. A complete custom
template exports false with exact ordered keys. An unavailable family retains
valid prior whole intent or is omitted on first export; sibling reads continue.
Known immutable values export including false. Missing optional immutable does not
become false or block active export: refresh the readable template and retain only
the prior immutable leaf, or omit it on first export, with bounded info. Retained
intent never passes through acquired-only checks. Failed diagnostic delivery
prevents saving; malformed present immutable makes the whole family unavailable.

### Scenario: Reconcile with exact companion preservation

Plan compares one managed group. Apply rereads target identity and complete active
state. Known matching selected values send no PUT. A template-only match still
no-ops with unknown immutable, but selecting the unknown leaf fails comparison.
A needed update requires known immutable and a complete active template. Unselected
active units are replayed from the fresh read, including exact ordered custom
keys for immutable-only writes and the immutable value for template-only writes. Custom keys
are sent only for false mode; default selects the default template without a
hidden inactive-key retention promise. An archived or incomplete target prevents
a needed write. Combined intent sends both explicit units. No fallback or retry is
attempted when GitHub rejects a transition.

### Scenario: Confirm a write independently

Only HTTP 201 with a bounded empty body, JSON null or strict empty object is a valid
acknowledgment. A fresh independent GET must confirm the same repository/owner
identity, archive state, original selected values and replayed active companions.
Prefix changes are allowed; missing requested immutable cannot confirm success.
Wrong status, malformed acknowledgment, denied readback or drift fails with
uncertain effects. Prewrite failures remain distinct, earlier verified writes
remain recorded and later work stops. No automatic retry or rollback occurs.

### Scenario: Keep private evidence out of public output

CLI and detailed Action output show ordered UI intent, with strict same-version
report reconstruction. Before/after must select the same units; replayed companions
never become declared changes. Status-only output hides the group. Raw API bodies and
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
| Immutable-only/combined intent, unknown evidence, selected reports and clone isolation | [Immutable model](../../internal/settings/oidc_immutable_test.go), [codec](../../internal/workspace/ui_oidc_immutable_test.go) and [report tests](../../internal/actions/oidc_subject_test.go) |
| Fresh custom replay, true/false no-ops, rejection without retry, prefix changes and readback drift | [Immutable adapter](../../internal/github/oidc_immutable_test.go) |
| Independent export leaf retention and diagnostic failure | [Immutable app tests](../../internal/app/oidc_immutable_test.go) |
| Immutable export/plan/apply/repeat, failure privacy and unchanged YAML | [CLI](../../tests/acceptance/oidc_immutable_test.go) and [actual Action](../../tests/github-actions/oidc_immutable_acceptance_test.go) |

The unsaved personal-public UI and official [OIDC API](https://docs.github.com/en/rest/actions/oidc)
provide configuration evidence. Fixtures do not establish live mutation behavior
or cloud trust; no live settings changes are part of this work.
The immutable-only YAML regression fails on unchanged source with an unknown key.
The editable legacy control was inspected without saving on 2026-11-14; persisted
legacy opt-out and effective cloud-token format were not tested.
