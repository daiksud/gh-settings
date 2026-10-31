---
type: Feature
title: Automatic branch deletion after merge
description: Manage one optional repository boolean through the existing merge-related read and write paths.
---

## Feature: Manage automatic branch deletion

Current source supports `repository.settings.general.pull_requests.automatically_delete_head_branches`. Publication does not adopt root configuration, change live policy, perform branch operations or grant permissions.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    general:
      pull_requests: {automatically_delete_head_branches: true}
```

### Rule: Preserve boolean presence

Omitting the field leaves it unmanaged. Explicit false remains a declared value in encode, comparison, report and PATCH. Null, quoted booleans, numbers and collections fail local validation before API access; the existing strict boolean, duplicate-key, alias and merge rules apply.

Repository comparison retains registry order for repository fields, followed by collections and policy resources in their documented order. Each declared scalar counts once and each managed collection once. Private report bounds are retained internally.

### Rule: Use the existing complete GraphQL read

The [GraphQL Repository field](https://docs.github.com/en/graphql/reference/objects#repository) is `deleteBranchOnMerge: Boolean!`, as defined by GitHub's [public schema](https://docs.github.com/en/graphql/overview/public-schema). It joins the existing fixed `POST /graphql` query for the three merge-method booleans, using only the origin-selected owner/name. There is no caller-supplied query or mutation and no extra read when both settings are declared.

The shared `MergeMethodsState`, adapter and deferred callback retain their internal names but hold four merge-related booleans; current source also includes two PR option booleans and four commit enums. A declaration containing only the new field calls GraphQL without the base REST read. Omitting all merge-related fields skips GraphQL.

Missing/null repository or field values, invalid types, transport/HTTP/JSON failures, and GraphQL errors including partial data fail the whole read. They never silently become false or fall back to a write-privileged REST read. A read failure preserves an existing export file and discards any partial plan comparison.

An [isolated PR Actions run](https://github.com/daiksud/gh-settings/actions/runs/39098804967) returned `deleteBranchOnMerge: true` with effective Contents read and Metadata read only. It used no checkout, App credentials, Environment or write permission. Its temporary workflow is excluded from the final implementation diff. This proves access for that token/repository, not all possible credentials.

### Rule: Keep apply direct

Apply uses the existing [repository update endpoint](https://docs.github.com/en/rest/repos/repos#update-a-repository), `PATCH /repos/{owner}/{repo}`, requiring Administration write for GitHub App/fine-grained tokens. The payload contains only declared repository fields; the new boolean shares that one PATCH and adds no read. Existing redirect rejection, timeout, target selection, partial-failure behavior and caller approval responsibilities are unchanged.

This configures GitHub's after-merge behavior. It adds no branch-listing or branch-deletion API operation, and implementation does not alter the currently enabled live value.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Nil/false presence and ordered comparison | [Settings tests](../../internal/settings/merge_methods_test.go) |
| Strict types and encode/decode omission | [Workspace tests](../../internal/workspace/ui_general_test.go) |
| Exact fixed query, complete/error reads and false/true PATCH payloads | [Adapter tests](../../internal/github/merge_methods_test.go) |
| Deferred GraphQL-only plan and direct apply | [Application tests](../../internal/app/merge_methods_test.go) |
| Native booleans and private Summary validation | [Actions tests](../../internal/actions/merge_methods_test.go) |
| Built executable plan/apply, missing/null/partial errors, export and credential-free Summary | [Process acceptance](../../tests/acceptance/merge_methods_test.go) |

All mutation tests use local API fixtures with dummy credentials; no test deletes a real branch or changes live repository policy. The read-only fixture admits only the exact current-source ten-field query and fixture owner/name, never arbitrary GraphQL POSTs.
