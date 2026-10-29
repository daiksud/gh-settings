---
type: Feature
title: Issue creation policy
description: Declarative Issue creation eligibility without changing issue availability.
---

## Contract

Current source supports `repository.settings.general.features.issue_permissions.creation_allowed_by` with exact UI choices `all_users` and `collaborators_only`. Omission leaves it unmanaged. Null, empty strings, wrong types, case variants and unknown policies are invalid. Declaring a policy with the Issues checkbox omitted never enables/disables Issues. An explicitly false Issues checkbox with a declared policy is rejected locally.

A separate narrow authenticated GraphQL query reads this policy only when requested; PR-only and existing repository declarations retain their reads. Export validates the complete observed policy after all reads succeed, but omits the UI permission when Issues is explicitly observed disabled. Missing/null/unsupported data and partial GraphQL errors fail rather than becoming `ALL`, including when issues are disabled. Failed export preserves the old file. Reads and ID lookup disable rename following to reject stale origins.

Apply reuses the repository-ID lookup, selected-only `updateRepository` mutation and target/response verification from PR access. Send the Issue policy only when declared, without synthesizing `hasIssuesEnabled` or PR settings. When both PR and Issue controls are declared, use one lookup and one mutation. Declared REST-backed controls run first, before GraphQL and later resources; an unrelated Projects checkbox can accompany policy-only intent without inventing Issues availability. Stop on first error and report earlier successful writes; there is no rollback, retry or saved-plan execution.

Verification uses isolated HTTP/files/Git fixtures with dummy credentials. A read-only disabled-Issue observation does not prove live mutation compatibility, permissions or publication. Caller grants are unchanged and never expanded automatically. No live settings or release changes are included.

## Acceptance

Test both policies, independent omission, disabled-Issue observations, invalid local types/enums, strict missing/null/unknown/partial-error reads, selective API access, exact combined and policy-only mutation payloads, target and response failures, redirects and mixed REST/GraphQL partial failure. Exercise export/validate/plan/apply, native Action report/Summary and failure file preservation with the built CLI.

## Existing test mapping

| Boundary | Tests |
| --- | --- |
| Presence and acquired observations | [Issue policy model](../../internal/settings/issue_creation_policy_test.go) |
| YAML types, omission and round trip | [Issue policy codec](../../internal/workspace/ui_general_test.go) |
| Selective reads, shared exact mutation and response failures | [Issue policy API](../../internal/github/issue_creation_policy_test.go) |
| Native enum report and Summary validation | [Issue policy presentation](../../internal/actions/issue_creation_policy_test.go) |
| Built CLI disabled-Issue lifecycle, exact writes and failure sequencing | [Issue policy acceptance](../../tests/acceptance/issue_creation_policy_test.go) |

The existing PR-access tests continue to check their unchanged selective reads, identity guards and failure semantics through the shared writer. The private same-version report retains native API tokens; full UI rendering/count integration remains #481. The read-only disabled-Issue sample returns policy `ALL`; fixtures cover explicit edits while availability remains false without asserting live write preservation.

## Feature: Retain provenance for out-of-UI creation policies

### Rule: Complete typed policy observations use the named UI default

#### Scenario: A successful GET returns an unknown policy string

- Given the acquired `issue_permissions.creation_allowed_by` policy is outside the two UI choices
- When the policy is projected
- Then the effective choice is `all_users` with native value `ALL`
- And one record retains its YAML path, exact raw string and named fallback
- And empty and whitespace strings remain exactly recorded acquired values
- But absent/null members, wrong types and failed or partial acquisition remain errors

#### Scenario: Acquisition and desired omission remain independent

- Given the policy was not acquired
- When its UI projection is prepared
- Then it remains unacquired without an invented default or provenance
- And an omitted desired policy remains unmanaged

### Rule: A fallback match does not establish raw equality

#### Scenario: The declaration equals or differs from the projected policy

- Given a successful acquired policy required the named `all_users` fallback
- When the declaration manages that policy as `all_users` or `collaborators_only`
- Then comparison retains one actionable decision for that native policy field
- And its provenance retains the exact original token
- And an omitted policy produces no policy decision or warning
- But inconsistent provenance or an incomplete managed observation fails without a partial comparison

### Rule: Export warns only for a visible acquired normalization

#### Scenario: A normalized policy is visible or hidden

- Given complete repository acquisition retained an original out-of-UI policy
- When export projects a visible creation-permission control
- Then it warns with the original token and named fallback and saves only the UI choice
- But an explicitly disabled feature hides its policy without warning or replay
- And failed repository acquisition produces no normalization warning or save
- And later acquisition failure can retain an earlier valid warning but never saves partial YAML
