---
type: Feature
title: Pull request access
description: Declarative pull request availability and creation policy.
---

## Contract

Current source supports `repository.settings.general.features.pull_requests` (boolean) and `features.pull_request_permissions.creation_allowed_by` under the same General root (`all_users` or `collaborators_only`). Omission leaves each setting unmanaged; false disables PR access without synthesizing a creation policy. Null, wrong types and unknown choices are invalid. Explicit false plus a permission choice is invalid; an omitted enabling checkbox does not enable implicitly.

The [GraphQL repository contract](https://docs.github.com/en/graphql/reference/repos) exposes both fields and `updateRepository` inputs. Read both policy values completely when either is needed, rejecting missing/null/unknown values and partial-data errors. Export includes the checkbox and its visible permission only after complete observation; disabled PRs hide the permission without inferring a default or losing acquired raw state. Failure preserves the old file.

Plan uses query operations only. Both policy and identity queries set `followRenames: false` so stale checkout origins fail instead of following renamed repositories. Direct apply resolves the repository's opaque ID from the checkout owner/name, verifies target identity, and sends only declared PR values through `updateRepository`. It never uses undocumented REST keys or changes the omitted companion setting. Verify the declared mutation response values and target; errors after a mutation have uncertain effects. Existing repository REST writes run first, then PR access, topics, labels and Rulesets. Stop on first error, retaining earlier writes without rollback.

No live settings, token grants, individual PR operations or release changes are part of implementation verification. Local dummy-credential fixtures prove requests and behavior, not actual write compatibility or permissions. Reads need complete policy visibility; writes need a credential authorized to update the repository. No permission is expanded automatically.

## Acceptance

Cover true/false, both policies, independent omission, invalid YAML/enum/null, disabled observation, complete and partial-error reads, exact lookup/mutation identity and selected values, response mismatches and redirects. Exercise export/validate/plan/apply and native private Action reports and current boolean CLI paths with the built CLI against isolated local fixtures. Mixed REST/GraphQL failures stop later resources and report earlier writes honestly.

## Existing test mapping

| Boundary | Tests |
| --- | --- |
| Presence, observation and comparison | [PR access model](../../internal/settings/pr_access_test.go) |
| YAML types, omission and round trip | [PR access codec](../../internal/workspace/ui_general_test.go) |
| Selective complete reads, exact mutations, identity and errors | [PR access API](../../internal/github/pr_access_test.go) |
| Native report values and invalid enum/types | [PR access Action presentation](../../internal/actions/pr_access_test.go) |
| Built CLI lifecycle, native Action report/Summary and failure sequencing | [PR access process acceptance](../../tests/acceptance/pr_access_test.go) |

The process suite also retains the existing repository export/read/write fixtures, now including the two additional fields and narrow query. Verification uses local files/Git and dummy credentials; the read-only public-repository example of disabled PRs with policy `ALL` is observation evidence, not a write grant or preservation guarantee.

Public YAML choices convert to internal GraphQL enums; those API tokens are not YAML aliases. The private report and compound permission display retain existing native protocol values pending #481.

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
- And failed repository acquisition produces no normalization warning or save
- And later acquisition failure can retain an earlier valid warning but never saves partial YAML
