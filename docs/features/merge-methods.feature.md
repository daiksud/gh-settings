---
type: Feature
title: Repository merge methods
description: Read three merge-method booleans through GraphQL and apply declared REST fields.
---

## Feature: Manage repository merge-method settings

CLI and Action v0.10.0 add three optional booleans under the existing `settings.repository` group and retain `schemaVersion: 1`. Published v0.9.0 does not support these keys; the pinned v0.15.0 consumer workflows do. No new resource group, auto-merge, merge queue, or commit-message policy is introduced.

```yaml
schemaVersion: 1
settings:
  repository:
    allow_merge_commit: false
    allow_squash_merge: true
    allow_rebase_merge: false
```

### Rule: Preserve declaration presence and strict types

- Omitted fields remain unmanaged. Explicit false is a declared value and is included in PATCH; GitHub defaults never fill omitted fields.
- Values must be YAML booleans spelled true or false under the existing parser rules. Null, strings, numbers, collections, duplicate keys and unsupported settings fail validation before API access. Aliases and merge keys retain their existing behavior.
- Base fields compare first, followed by `allow_merge_commit`, `allow_squash_merge`, `allow_rebase_merge`, then the `delete_branch_on_merge` field added in v0.11.0, then current-source auto-merge and branch-update booleans, then the squash and normal merge commit title/message enums, then topics and labels. Each declared boolean counts as one managed field.

### Rule: Read merge methods without write permissions

[REST repository GET](https://docs.github.com/en/rest/repos/repos#get-a-repository) requires Contents write as well as read to expose merge-related settings. Those response fields are optional and are absent in a public unauthenticated GET; missing values cannot safely mean false.

The source read adapter instead sends a fixed GraphQL **query** to `POST /graphql`, using the same origin-selected owner/name and existing gh credentials. Published v0.10.0 requests the three method fields below; v0.11.0 and current source also request the branch-deletion field using the same query. These [public-schema](https://docs.github.com/en/graphql/overview/public-schema) fields are required booleans:

| Declaration | GraphQL read field |
| --- | --- |
| `allow_merge_commit` | `mergeCommitAllowed: Boolean!` |
| `allow_squash_merge` | `squashMergeAllowed: Boolean!` |
| `allow_rebase_merge` | `rebaseMergeAllowed: Boolean!` |
| `delete_branch_on_merge` (v0.11.0) | `deleteBranchOnMerge: Boolean!` |

An [isolated Actions experiment](https://github.com/daiksud/gh-settings/actions/runs/39086857913) on this public repository verified all three values with the ordinary token's effective Contents read and Metadata read permissions. The job used no App secrets, Environment, write scopes, checkout, or exported credentials. Its temporary workflow is excluded from the implementation. This is hosted read-access evidence for that token/repository; fixtures alone do not establish another credential's access.

- Plan calls this deferred read only when a merge-related field is declared. Existing declarations that omit all four source fields retain their existing API reads and credentials.
- A merge-method-only plan needs only GraphQL; the base repository REST read is skipped. Metadata, Summary and unmanaged local no-ops do not initialize a GraphQL client.
- Export reads base repository settings, merge methods, topics and labels, then serializes all sixteen repository values in current source (fifteen in v0.14.0, thirteen in v0.13.0, eleven in v0.12.0, nine in published v0.11.0 and eight in v0.10.0) and complete collections. All reads and serialization finish before opening the destination. Published v0.9.0 export retains its original five-field repository shape.
- The adapter retains the existing gh host/authentication, Unix-socket fixture support and 30-second read timeout through the already-used go-gh dependency. There is no caller-supplied query, target or mutation path.
- Missing/null repository or fields, wrong scalar types, HTTP/transport/JSON failure, or any GraphQL errors fail the whole read. Partial data with errors is discarded. Neither plan nor export synthesizes false or falls back to a privileged REST read.

### Rule: Keep repository apply direct

Apply uses the existing `PATCH /repos/{owner}/{repo}` and [Administration write permission](https://docs.github.com/en/rest/repos/repos#update-a-repository). It sends only declared base/merge-method fields, together in one payload, preserving explicit false. It does not perform GraphQL, read current merge settings, or consume a saved plan.

Existing target resolution, write timeout, redirect rejection, failure propagation and partial-failure behavior remain unchanged. This feature does not change root managed configuration, live merge policy, App grants, PR token permissions, consumer pins, or release selection. GitHub may reject combinations that pass local boolean validation; no additional merge policy is inferred locally.

CLI and the same-version Actions report/Summary preserve native boolean before/after values and resource-qualified field paths. The maximum current-source managed count is eighteen: sixteen repository fields, topics and labels. Published v0.14.0 retains seventeen. Published v0.10.0 retains its ten-item limit. Reports remain presentation data, never apply input.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Presence, base/merge read selection, stable typed comparison | [Settings tests](../../internal/settings/merge_methods_test.go) |
| Strict booleans, null rejection, omission, aliases/merges, round trips | [Workspace tests](../../internal/workspace/merge_methods_test.go) |
| Fixed GraphQL query/target, complete booleans, errors/partial data, exact PATCH | [GitHub adapter tests](../../internal/github/merge_methods_test.go) |
| Conditional reads, direct apply, failed comparison/export preservation | [Application tests](../../internal/app/merge_methods_test.go) |
| Native booleans, report validation and Summary rendering | [Actions tests](../../internal/actions/merge_methods_test.go) |
| Built executable: read order, no extra read for apply, representative local validation failures, zero/null/missing response values, export preservation, report and credential-free Summary | [Process acceptance](../../tests/acceptance/merge_methods_test.go) |

Input validation matrices stay at their owning layers. `TestMergeMethodBooleanCodec` in the workspace tests rejects null and wrong-typed values for all six merge-related booleans, including the former process cases `deletion invalid local` and `PR auto invalid local`. The process cases `invalid local` (plan with null) and `PR update invalid local` (apply with a quoted boolean) retain representative checks for nonzero exit status, diagnostics, zero API requests, and unchanged input bytes. The same process assertions cover the retained squash enum and message/title dependency failures described in the [squash test mapping](squash-commit-defaults.feature.md#test-mapping).

Process fixtures admit only the exact merge-method query and containing repository variables as a read-only POST. Arbitrary GraphQL mutations or unexpected targets fail the fixture. Product tests use local API fixtures with dummy credentials and do not modify live repository policy.
