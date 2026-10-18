# Normal merge commit title and message defaults

## Status and scope

Current source supports optional `settings.repository.merge_commit_title` and `merge_commit_message` under `schemaVersion: 1` (unreleased; planned v0.14.0). Published v0.13.0 and currently pinned examples reject these keys. They extend the existing repository group without enabling normal merge commits, changing squash defaults, or performing a merge.

The [configuration reference](../settings.md#merge-commit-defaults-unreleased-planned-v0140) owns declaration/value semantics. These defaults can be managed while `allow_merge_commit` is false; no merge-method declaration is required or synthesized.

## API contract

| Declaration | Exact values | GraphQL Repository observation |
| --- | --- | --- |
| `merge_commit_title` | `PR_TITLE`, `MERGE_MESSAGE` | `mergeCommitTitle: MergeCommitTitle!` |
| `merge_commit_message` | `PR_BODY`, `PR_TITLE`, `BLANK` | `mergeCommitMessage: MergeCommitMessage!` |

The [REST repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) requires title when message is specified. Local validation therefore rejects message-only declarations before target/authentication/API access, while title-only is valid. Each omitted field remains unmanaged. A squash title is not a substitute for a normal merge title. Null, empty, unknown and wrong-typed enums fail locally; `BLANK` is a literal enum.

The [GraphQL public schema](https://docs.github.com/en/graphql/overview/public-schema) defines the non-null enums. The existing fixed query/deferred `MergeMethods` read now observes six booleans and four enums. Declaring only a new enum skips the base REST read; omitting every merge-related field skips GraphQL. Missing/null/invalid values, HTTP/transport/JSON errors and partial GraphQL errors discard the whole observation, preserving an existing export file and rejecting earlier partial plan comparisons.

Apply includes declared fields only in the existing Administration-write REST PATCH. It adds no preliminary read, inferred companion value, merge-method update or new permission. Existing redirect/timeout/target and direct-apply race/partial-failure behavior remains unchanged. No PR mutation or branch-protection change is introduced.

Complete export saves fifteen repository fields plus complete topics/all label pages after all reads/serialization succeed. v0.13.0 retains thirteen. Stable comparison order appends normal merge title/message after squash title/message and before topics/labels, preserving prior field order. Each declared field counts once; current-source report/Summary supports seventeen managed units, while v0.13.0 retains fifteen. CLI/Summary quote enum strings, typed reports retain strings, and unsupported report enum values/types are rejected.

## Read-only hosted evidence

A temporary isolated Actions proof will read both new fields with effective Contents read/Metadata read only, without checkout, App credentials, Environment or writes. Its successful run/values will be recorded here and its workflow removed from the final diff. Local authenticated reads alone do not prove ordinary Actions-token access.

## Test mapping

| Layer | Evidence |
| --- | --- |
| Declaration/model | [Merge codec](../../internal/workspace/merge_commit_test.go): enum combinations, title-only, message-only rejection, null/type/value matrices, aliases and round trips. [Comparison](../../internal/settings/merge_commit_test.go): omission/equality/order and strings. |
| GitHub adapter | [Merge enum reads/PATCH](../../internal/github/merge_commit_test.go), [shared boundary](../../internal/github/merge_methods_test.go): incomplete/invalid/partial-error reads, declared-only title/pair PATCH and no GET. |
| Application | [Merge sequencing](../../internal/app/merge_commit_test.go), [shared failures](../../internal/app/merge_methods_test.go), [export](../../internal/app/export_test.go): selective reads, local validation, direct writes and file preservation. |
| Actions | [Merge report/Summary](../../internal/actions/merge_commit_test.go): native strings, quoted output, invalid enums/types and seventeen-unit bound. |
| Installed binary | [Process acceptance](../../tests/acceptance/merge_methods_test.go): title-only/pair plan, equality, missing/null/partial errors, exact direct PATCH, cross-family message-only rejection, full export, typed reports and credential-free Summary. Existing tests retain unmanaged/no-op and prior settings behavior. |

Local fixtures use dummy credentials, not live mutations. Hosted proof establishes access for its observed token/repository. Native release acceptance and published Action smoke establish the separate package identity boundary.
