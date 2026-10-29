---
type: Feature
title: Merge commit message menu
description: Select one UI merge message choice and retain disabled-method prerequisites.
sources:
  - id: ui-inventory
    resource: ../contributing/ui-repository-mapping.md
  - id: repository-api
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: graphql-schema
    resource: https://docs.github.com/en/graphql/overview/public-schema
  - id: hosted-read-proof
    resource: https://github.com/daiksud/gh-settings/actions/runs/39120474618
---

## Status and scope

Current source accepts optional `repository.settings.general.pull_requests.merge_commit.default_commit_message`. It selects one UI menu item; it does not merge a PR, edit PR contents or implicitly enable a merge method. The [configuration reference](../settings.md#merge-commit-defaults) owns adoption and presence semantics.[^ui-inventory]

## API contract

| Public UI choice | Internal API title | Internal API message |
| --- | --- | --- |
| `default_message` | `MERGE_MESSAGE` | `PR_TITLE` |
| `pull_request_title` | `PR_TITLE` | `BLANK` |
| `pull_request_title_and_description` | `PR_TITLE` | `PR_BODY` |

The [repository REST contract](https://docs.github.com/en/rest/repos/repos#update-a-repository) and [GraphQL public schema](https://docs.github.com/en/graphql/overview/public-schema) define the underlying API fields.[^repository-api][^graphql-schema]

A declared menu converts to both API members together. Omission leaves the menu unmanaged. Separate title/message fields, API tokens, null, empty/unknown choices and wrong types are not public YAML. Explicit `allow_merge_commits: false` with the menu fails before origin/authentication/API access. An omitted enabling checkbox remains omitted.

With the checkbox omitted, plan also checks observed merge availability and fails if disabled. Explicit true satisfies that prerequisite without synthesizing intent. Direct apply cannot pre-resolve an omitted allow checkbox; declare true when enabling the method with its menu.

The existing fixed GraphQL query uses only checkout-selected owner/name and complete native observations. Missing/null members, wrong types, transport/HTTP/JSON failures and GraphQL errors including partial data discard the observation. There is no privileged read fallback. After complete typed acquisition, an out-of-UI pair uses the named `default_message` projection and retains both original strings in one provenance record. Export warns for a visible normalized menu and hides a menu behind an observed disabled method without replaying latent values as intent. API-only pairs never become additional UI choices.

Apply selectively reads both members of each declared menu, sends the translated pair in the existing Administration-write repository PATCH, then verifies a fresh raw GraphQL readback. A retained out-of-UI pair fails verification even when its projection equals the declared choice. Preflight failure makes no repository PATCH; later errors report possible retained writes. No companion is borrowed from live state and no enabling checkbox is synthesized. Other undeclared fields are omitted. Existing target, timeout, redirect rejection and nontransactional partial-failure semantics remain.

## Export, ordering and presentation

Visible acquired menu choices export under UI paths after all requested reads/conversion succeed. Exact supported observations retain stable native comparison order; one menu can produce two component changes. A normalized observation instead produces one menu-level warning and write-required decision, retaining the existing managed count of two native members even when the desired choice equals the projected default. Complete UI menu rendering is #481: exact-observation CLI/Summary and the private same-version report can still show native component paths/tokens. Normalized decisions use the UI menu path/choices and retain both raw strings separately; status-only Summary suppresses policy values. They are not YAML aliases or saved apply input. Report type validation remains closed; this change does not claim the UI count/report audit complete.

## Feature: Preserve normalized merge observations as uncertain wire state

### Rule: Complete typed pairs outside UI choices use the named default with provenance

#### Scenario: An API-only or future pair is acquired

- Given a complete successful GET returns `MERGE_MESSAGE` and `BLANK`, unknown strings, or empty strings
- When the merge menu is projected
- Then the effective choice is `default_message` with native pair `MERGE_MESSAGE` and `PR_TITLE`
- And one record retains both exact raw strings, the YAML menu path and named fallback
- But missing/null members, invalid types and failed or partial acquisition remain errors

#### Scenario: A projected default does not prove matching

- Given the raw merge pair lies outside UI choices
- When a declared choice equals or differs from the projected default
- Then plan emits a warning and a menu decision requiring a write
- And an omitted menu emits neither warning nor menu decision
- And apply verifies the declared native pair against a fresh raw GET after PATCH
- But a retained unknown pair fails verification even if it projects to the desired default

## Read-only hosted evidence

An [isolated PR Actions proof](https://github.com/daiksud/gh-settings/actions/runs/39120474618)[^hosted-read-proof] returned `mergeCommitTitle: MERGE_MESSAGE` and `mergeCommitMessage: PR_TITLE` with all prior fields and the existing squash-only booleans. Effective permissions were Contents read and Metadata read only, with no checkout, App credentials, Environment or writes. Its temporary workflow is removed from the final diff. This proves access for that token/repository, not every credential.

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
