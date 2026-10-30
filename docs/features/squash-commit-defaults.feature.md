---
type: Feature
title: Squash commit message menu
description: Select one UI squash message choice and convert its complete API pair.
sources:
  - id: ui-inventory
    resource: ../contributing/ui-repository-mapping.md
  - id: repository-api
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: graphql-schema
    resource: https://docs.github.com/en/graphql/overview/public-schema
  - id: hosted-read-proof
    resource: https://github.com/daiksud/gh-settings/actions/runs/39108650790
---

## Status and scope

Current source accepts optional `repository.settings.general.pull_requests.squash_merge.default_commit_message`. It selects one UI menu item; it does not merge a PR, edit PR contents or implicitly enable a merge method. The [configuration reference](../settings.md#squash-commit-defaults) owns adoption and presence semantics.[^ui-inventory]

## API contract

| Public UI choice | Internal API title | Internal API message |
| --- | --- | --- |
| `default_message` | `COMMIT_OR_PR_TITLE` | `COMMIT_MESSAGES` |
| `pull_request_title` | `PR_TITLE` | `BLANK` |
| `pull_request_title_and_commit_details` | `PR_TITLE` | `COMMIT_MESSAGES` |
| `pull_request_title_and_description` | `PR_TITLE` | `PR_BODY` |

The [repository REST contract](https://docs.github.com/en/rest/repos/repos#update-a-repository) and [GraphQL public schema](https://docs.github.com/en/graphql/overview/public-schema) define the underlying API fields.[^repository-api][^graphql-schema]

A declared menu converts to both API members together. Omission leaves the menu unmanaged. Separate title/message fields, API tokens, null, empty/unknown choices and wrong types are not public YAML. Explicit `allow_squash_merging: false` with the menu fails before origin/authentication/API access. An omitted enabling checkbox remains omitted.

The existing fixed GraphQL query uses only checkout-selected owner/name and complete native observations. Missing/null members, wrong types, transport/HTTP/JSON failures and GraphQL errors including partial data discard the observation. There is no privileged read fallback. After complete typed acquisition, an out-of-UI pair uses the named `default_message` projection and retains both original strings in one provenance record. Export warns for a visible normalized menu and hides a menu behind an observed disabled method without replaying latent values as intent. API-only pairs never become additional UI choices.

Apply selectively reads both members of each declared menu, sends the translated pair in the existing Administration-write repository PATCH, then verifies a fresh raw GraphQL readback. A retained out-of-UI pair fails verification even when its projection equals the declared choice. Preflight failure makes no repository PATCH; later errors report possible retained writes. No companion is borrowed from live state and no enabling checkbox is synthesized. Other undeclared fields are omitted. Existing target, timeout, redirect rejection and nontransactional partial-failure semantics remain.

## Export, ordering and presentation

Visible acquired menu choices export under UI paths after all requested reads/conversion succeed. Exact supported observations retain stable native comparison order; one menu can produce two component changes. A normalized observation instead produces one menu-level warning and write-required decision, retaining the existing managed count of two native members even when the desired choice equals the projected default. Complete UI menu rendering is #481: exact-observation CLI/Summary and the private same-version report can still show native component paths/tokens. Normalized decisions use the UI menu path/choices and retain both raw strings separately; status-only Summary suppresses policy values. They are not YAML aliases or saved apply input. Report type validation remains closed; this change does not claim the UI count/report audit complete.

## Feature: Project complete out-of-UI squash observations with provenance

### Rule: One UI menu retains its complete raw title and message

#### Scenario: A known API-only pair uses the adopted default

- Given a successful complete read returns title `COMMIT_OR_PR_TITLE` and message `BLANK`
- When the squash menu is projected to UI choices
- Then the effective choice is `default_message`
- And its effective title/message pair is `COMMIT_OR_PR_TITLE` and `COMMIT_MESSAGES`
- And one provenance record retains the new YAML field, both exact raw strings and the fallback

#### Scenario: Unknown and empty typed strings are acquired observations

- Given a successful complete read contains a previously unseen or empty title/message string
- When that pair is outside the UI choices
- Then the same adopted default is projected with both raw strings retained
- But missing/null members, invalid types or failed acquisition are not fallback observations

#### Scenario: An unacquired menu does not become a default

- Given no squash menu observation was acquired
- When UI choices are prepared
- Then the menu remains unacquired without a fallback or warning record

### Rule: Fallback equality remains actionable

#### Scenario: Plan and apply retain the distinction between raw and effective pairs

- Given the acquired raw squash pair is outside UI choices
- When a declared choice equals or differs from its projected default
- Then plan emits one warning and one menu decision requiring a write
- And an omitted menu emits neither warning nor menu decision
- And export warns for a visible normalized menu and saves only its effective UI choice
- And apply verifies the declared pair using a fresh raw GET after PATCH
- But a retained unknown pair fails verification even when its fallback equals the desired choice

## Read-only hosted evidence

An [isolated PR Actions proof](https://github.com/daiksud/gh-settings/actions/runs/39108650790)[^hosted-read-proof] returned `squashMergeCommitTitle: COMMIT_OR_PR_TITLE` and `squashMergeCommitMessage: COMMIT_MESSAGES` alongside the existing four booleans. The job had effective Contents read and Metadata read only, with no checkout, App credential, Environment or write permission. Its temporary workflow is removed from the final implementation diff. This establishes access for that token/repository, not every possible credential.

## Test mapping

| Boundary | Evidence |
| --- | --- |
| Public YAML omission/types/roundtrip/API-alias rejection | [General UI codec](../../internal/workspace/ui_general_test.go) |
| Every menu choice, exact pair and disabled/unknown rejection | [UI menu conversion](../../internal/github/ui_commit_message_test.go) |
| Internal API enum reads and declared-only payloads | [Squash transport](../../internal/github/squash_commit_test.go), [merge transport](../../internal/github/merge_commit_test.go), [shared GraphQL boundary](../../internal/github/merge_methods_test.go) |
| Complete typed normalization and raw verification | [Full reader](../../internal/github/commit_message_full_read_test.go), [selective reads](../../internal/github/commit_message_read_test.go), [raw verification](../../internal/github/commit_message_verification_test.go), [apply transport](../../internal/github/commit_message_apply_test.go) |
| Normalized public commands and private presentation | [Process normalization](../../tests/acceptance/normalization_commit_message_test.go), [paired reports](../../internal/actions/normalization_commit_message_test.go) |
| Existing native dependency/comparison behavior | [Commit dependencies](../../internal/settings/commit_dependencies_test.go), [merge application](../../internal/app/merge_commit_test.go), [squash application](../../internal/app/squash_commit_test.go) |
| Private native report validation | [Merge Summary](../../internal/actions/merge_commit_test.go), [squash Summary](../../internal/actions/squash_commit_test.go) |
| Built CLI current choices, exact paired PATCH, disabled-method failure, hidden export and Summary boundary | [Process acceptance](../../tests/acceptance/merge_methods_test.go), [export/validate](../../tests/acceptance/export_validate_test.go) |

The process menu cases use independently expected API pairs, not a test translator or production-derived expectations. Obsolete API-key rejection, unknown UI choice and explicit/observed disabled prerequisites retain zero-network or exact-read boundaries and unchanged source bytes. Native transport tests remain internal protocol coverage, not an alternate public codec.

Local fixtures use dummy credentials without live mutations. Hosted read evidence establishes only its observed token/repository access. Published package identity and release authorization remain separate.

[^ui-inventory]: Existing read-only UI wording inventory from 2026-10-28; current local fixtures check UI/API translation.
[^repository-api]: Original primary REST reference retained for underlying request fields; API tokens are not public YAML choices.
[^graphql-schema]: Original primary schema reference retained for native observation fields and types.
[^hosted-read-proof]: Previously cited historical read-only run, not rerun or reclassified as new live verification by YAML activation.
