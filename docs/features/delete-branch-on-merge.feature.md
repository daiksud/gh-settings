---
type: Feature
title: Automatic branch deletion after merge
description: Manage one optional repository boolean through the existing merge-related read and write paths.
---

## Feature: Manage automatic branch deletion

CLI and Action v0.11.0 support `settings.repository.delete_branch_on_merge` under `schemaVersion: 1`. Consumers using v0.10.0 reject the key; the pinned v0.14.0 examples support it. Publication does not adopt root configuration, change live policy, perform branch operations or grant permissions.

```yaml
schemaVersion: 1
settings:
  repository:
    delete_branch_on_merge: true
```

### Rule: Preserve boolean presence

Omitting the field leaves it unmanaged. Explicit false remains a declared value in encode, comparison, report and PATCH. Null, quoted booleans, numbers and collections fail local validation before API access; the existing strict boolean, duplicate-key, alias and merge rules apply.

The stable repository comparison order is the six current-source REST fields (including `has_discussions` after `has_wiki`), the three merge-method booleans, then `delete_branch_on_merge`, the current-source auto-merge and branch-update booleans, then squash and normal merge title/message enums, topics and labels. Each declared repository field counts once. v0.11.0 Summary accepts up to eleven managed units: nine repository fields plus topics and labels; v0.12.0 adds two squash commit enums for a maximum of thirteen; v0.13.0 adds two more booleans for fifteen; v0.14.0 adds two normal merge enums for seventeen; current source adds Discussions for eighteen; published v0.10.0 retains ten.

### Rule: Use the existing complete GraphQL read

The [GraphQL Repository field](https://docs.github.com/en/graphql/reference/objects#repository) is `deleteBranchOnMerge: Boolean!`, as defined by GitHub's [public schema](https://docs.github.com/en/graphql/overview/public-schema). It joins the existing fixed `POST /graphql` query for the three merge-method booleans, using only the origin-selected owner/name. There is no caller-supplied query or mutation and no extra read when both settings are declared.

The shared `MergeMethodsState`, adapter and deferred callback retain their internal names but hold four merge-related booleans; current source also includes two PR option booleans and four commit enums. A declaration containing only the new field calls GraphQL without the base REST read. Omitting all merge-related fields skips GraphQL. A configuration declaring existing merge-method fields reads the complete source observation (four fields in v0.11.0; six in v0.12.0; eight in v0.13.0; ten in current source) but still compares and writes only declared values.

Export reads all repository fields (nine in v0.11.0, eleven in v0.12.0, thirteen in v0.13.0, fifteen in v0.14.0, sixteen in current source), complete topics and all label pages before saving. Missing/null repository or field values, invalid types, transport/HTTP/JSON failures, and GraphQL errors including partial data fail the whole read. They never silently become false or fall back to a write-privileged REST read. A read failure preserves an existing export file and discards any partial plan comparison.

An [isolated PR Actions run](https://github.com/daiksud/gh-settings/actions/runs/39098804967) returned `deleteBranchOnMerge: true` with effective Contents read and Metadata read only. It used no checkout, App credentials, Environment or write permission. Its temporary workflow is excluded from the final implementation diff. This proves access for that token/repository, not all possible credentials.

### Rule: Keep apply direct

Apply uses the existing [repository update endpoint](https://docs.github.com/en/rest/repos/repos#update-a-repository), `PATCH /repos/{owner}/{repo}`, requiring Administration write for GitHub App/fine-grained tokens. The payload contains only declared repository fields; the new boolean shares that one PATCH and adds no read. Existing redirect rejection, timeout, target selection, partial-failure behavior and caller approval responsibilities are unchanged.

This configures GitHub's after-merge behavior. It adds no branch-listing or branch-deletion API operation, and implementation does not alter the currently enabled live value.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Nil/false presence and ordered comparison | [Settings tests](../../internal/settings/merge_methods_test.go) |
| Strict types and encode/decode omission | [Workspace tests](../../internal/workspace/merge_methods_test.go) |
| Exact fixed query, complete/error reads and false/true PATCH payloads | [Adapter tests](../../internal/github/merge_methods_test.go) |
| Deferred GraphQL-only plan and direct apply | [Application tests](../../internal/app/merge_methods_test.go) |
| Native booleans, Summary validation and eleven-unit limit | [Actions tests](../../internal/actions/merge_methods_test.go) |
| Built executable plan/apply, missing/null/partial errors, export and credential-free Summary | [Process acceptance](../../tests/acceptance/merge_methods_test.go) |

All mutation tests use local API fixtures with dummy credentials; no test deletes a real branch or changes live repository policy. The read-only fixture admits only the exact current-source ten-field query and fixture owner/name, never arbitrary GraphQL POSTs.
