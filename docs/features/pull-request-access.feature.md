---
type: Feature
title: Pull request access
description: Declarative pull request availability and creation policy.
---

## Contract

Current source supports `repository.settings.general.features.pull_requests` (boolean) and `features.pull_request_permissions.creation_allowed_by` under the same General root (`all_users` or `collaborators_only`). Omission leaves each setting unmanaged; false disables PR access without synthesizing a creation policy. Desired YAML null, wrong types and unknown choices are invalid. Explicit false plus a permission choice is invalid; an omitted enabling checkbox does not enable implicitly.

The [GraphQL repository contract](https://docs.github.com/en/graphql/reference/repos) exposes both fields and `updateRepository` inputs. Public observations read PR availability and its policy completely when either is needed, rejecting missing/null members, wrong types and failed or partial acquisition. Complete typed policy strings outside the UI choices, including empty/whitespace strings, project to the named `all_users`/`ALL` fallback while preserving the original token. Export includes the checkbox and its visible permission only after complete observation; disabled PRs hide the permission without emitting a hidden-policy warning or replaying acquired raw state as intent. Visible normalized policies warn and export only the UI choice. An unavailable PR-access group follows the [readable-settings contract](readable-settings-export.feature.md): emit info, retain compatible prior intent or omit it on first export, and continue other groups. Plan remains strict.

Plan uses query operations only and retains its complete PR observation group. Access queries set `followRenames: false` so stale checkout origins fail. Direct apply resolves the opaque ID and acquires only originally declared access members. A complete native match makes no mutation or second read. Otherwise one `updateRepository` sends only changed declared values; its response and an independent same-ID readback verify every original declared member, including matching members omitted from the payload. Unknown raw policies remain write-required at fallback equality. Checkbox-only intent does not acquire or warn about an omitted policy; policy-only intent does not read or implicitly enable an omitted checkbox.

A failed selected preflight makes no access mutation. Mutation or readback failures have uncertain effects and stop later resources. REST controls run first; earlier completed REST writes take precedence over an inner access read-failure marker and retain partial progress. An access acknowledgment followed by failed verification alone is not a completed earlier substep. Successful reads and no-ops do not count as writes. Preserve error provenance and existing archive/Topics/Labels/Rulesets ordering without rollback, polling or retries.

No live settings, token grants, individual PR operations or release changes are part of implementation verification. Local dummy-credential fixtures prove requests and behavior, not actual write compatibility or permissions. Reads need complete policy visibility; writes need a credential authorized to update the repository. No permission is expanded automatically.

## Acceptance

Cover true/false, both policies, independent omission, invalid YAML/enum/null, disabled observation, complete and partial-error reads, exact lookup/mutation identity and selected values, response mismatches and redirects. Exercise export/validate/plan/apply and native private Action reports and current boolean CLI paths with the built CLI against isolated local fixtures. Mixed REST/GraphQL failures stop later resources and report earlier writes honestly.

## Existing test mapping

| Boundary | Tests |
| --- | --- |
| Presence, observation and comparison | [PR access model](../../internal/settings/pr_access_test.go) |
| YAML types, omission and round trip | [PR access codec](../../internal/workspace/ui_general_test.go) |
| Selective complete reads, exact mutations, identity and errors | [PR access API](../../internal/github/pr_access_test.go) |
| Typed-string normalization and fresh raw verification | [Complete reader](../../internal/github/creation_policy_full_read_test.go), [selected reads](../../internal/github/creation_policy_read_test.go), [apply verification](../../internal/github/creation_policy_apply_test.go) |
| Normalized report, UI privacy and public commands | [Private presentation](../../internal/actions/normalization_creation_policy_test.go), [normalization process](../../tests/acceptance/normalization_creation_policy_test.go) |
| Native report values and invalid enum/types | [PR access Action presentation](../../internal/actions/pr_access_test.go) |
| Selected fresh comparison and actual composite progress | [Access delta](../../internal/settings/repository_access_test.go), [API reconciliation](../../internal/github/repository_access_reconcile_test.go), [composition](../../repository_write_test.go), [process regression](../../tests/acceptance/repository_access_reconcile_test.go) |
| Built CLI lifecycle, native Action report/Summary and failure sequencing | [PR access process acceptance](../../tests/acceptance/pr_access_test.go) |

The process suite also retains the existing repository export/read/write fixtures, now including the two additional fields and narrow query. Verification uses local files/Git and dummy credentials; the read-only public-repository example of disabled PRs with policy `ALL` is observation evidence, not a write grant or preservation guarantee.

Public YAML choices convert to internal GraphQL enums; those API tokens are not YAML aliases. The private report retains native values; CLI and Summary use UI paths and choices.

## Feature: Retain provenance for out-of-UI creation policies

### Rule: Complete typed policy observations use the named UI default

#### Scenario: A successful GET returns an unknown policy string

- Given the acquired `pull_request_permissions.creation_allowed_by` policy is outside the two UI choices
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
- And unavailable policy acquisition produces no normalization warning for retained or omitted intent
- And later acquisition failure can retain an earlier valid warning while complete independent groups still export
- And no incomplete group observation is saved
