# Normal merge commit title and message defaults

## Status and scope

CLI and Action v0.14.0 support optional `settings.repository.merge_commit_title` and `merge_commit_message` under `schemaVersion: 1`. Published v0.13.0 rejects these keys; the pinned v0.18.0 examples support them. They extend the existing repository group without enabling normal merge commits, changing squash defaults, or performing a merge.

The [configuration reference](../settings.md#merge-commit-defaults-v0140) owns declaration/value semantics. These defaults can be managed while `allow_merge_commit` is false; no merge-method declaration is required or synthesized.

## API contract

| Declaration | Exact values | GraphQL Repository observation |
| --- | --- | --- |
| `merge_commit_title` | `PR_TITLE`, `MERGE_MESSAGE` | `mergeCommitTitle: MergeCommitTitle!` |
| `merge_commit_message` | `PR_BODY`, `PR_TITLE`, `BLANK` | `mergeCommitMessage: MergeCommitMessage!` |

The [REST repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) requires title when message is specified. Local validation therefore rejects message-only declarations before target/authentication/API access, while title-only is valid. Each omitted field remains unmanaged. A squash title is not a substitute for a normal merge title. Null, empty, unknown and wrong-typed enums fail locally; `BLANK` is a literal enum.

The [GraphQL public schema](https://docs.github.com/en/graphql/overview/public-schema) defines the non-null enums. The existing fixed query/deferred `MergeMethods` read now observes six booleans and four enums. Declaring only a new enum skips the base REST read; omitting every merge-related field skips GraphQL. Missing/null/invalid values, HTTP/transport/JSON errors and partial GraphQL errors discard the whole observation, preserving an existing export file and rejecting earlier partial plan comparisons.

Apply includes declared fields only in the existing Administration-write REST PATCH. It adds no preliminary read, inferred companion value, merge-method update or new permission. Existing redirect/timeout/target and direct-apply race/partial-failure behavior remains unchanged. No PR mutation or branch-protection change is introduced.

Current-source complete export saves twenty-five repository fields plus complete topics/all label pages after all reads and serialization succeed. Stable comparison order appends normal merge title/message after squash title/message and before later repository fields and resources. Each declared scalar counts once and each managed collection once; current-source report/Summary supports fifty-one managed units across all adopted resources. Older packages retain their documented lower limits. CLI/Summary quote enum strings, typed reports retain strings, and unsupported report enum values/types are rejected.

## Read-only hosted evidence

An [isolated PR Actions proof](https://github.com/daiksud/gh-settings/actions/runs/39120474618) returned `mergeCommitTitle: MERGE_MESSAGE` and `mergeCommitMessage: PR_TITLE` with all prior fields and the existing squash-only booleans. Effective permissions were Contents read and Metadata read only, with no checkout, App credentials, Environment or writes. Its temporary workflow is removed from the final diff. This proves access for that token/repository, not every credential.

## Test mapping

| Layer | Evidence |
| --- | --- |
| Declaration/model | [Merge codec](../../internal/workspace/merge_commit_test.go): enum combinations, title-only, message-only rejection, null/type/value matrices, aliases and round trips. [Comparison](../../internal/settings/merge_commit_test.go): omission/equality/order and strings. |
| GitHub adapter | [Merge enum reads/PATCH](../../internal/github/merge_commit_test.go), [shared boundary](../../internal/github/merge_methods_test.go): incomplete/invalid/partial-error reads, declared-only title/pair PATCH and no GET. |
| Application | [Merge sequencing](../../internal/app/merge_commit_test.go), [shared failures](../../internal/app/merge_methods_test.go), [export](../../internal/app/export_test.go): selective reads, local validation, direct writes and file preservation. |
| Actions | [Merge report/Summary](../../internal/actions/merge_commit_test.go): native strings, quoted output, invalid enums/types and the current-source fifty-one-unit bound. |
| Installed binary | [Process acceptance](../../tests/acceptance/merge_methods_test.go): title-only/pair plan, equality, shared representative read failures, exact direct PATCH, full export, typed reports and credential-free Summary. Existing tests retain unmanaged/no-op and prior settings behavior. |

Detailed local validation stays at its owning layers. The former process case `merge unknown rejected` maps to `TestMergeCommitCodec` in the workspace tests, whose invalid-value matrix explicitly rejects `COMMIT_OR_PR_TITLE` for normal merge defaults. The former `merge message-only rejected` case maps to the `squash title cannot satisfy merge message` subtest of `TestCommitMessageDependencies` in the [settings tests](../../internal/settings/commit_dependencies_test.go). Representative executable enum and dependency failures remain in the [squash acceptance mapping](squash-commit-defaults.feature.md#test-mapping), including zero API calls and unchanged input bytes; successful merge cases and representative API response failures remain at the process boundary.

`TestMergeCommitReadRejectsIncompleteEnums` owns the per-field response matrix formerly repeated by `merge missing title` and `merge null message`. The complete-data-with-errors fixture in `TestReadMergeMethodsGraphQLBoundary` covers the former `merge partial error`; the [shared process read-failure cases](merge-methods.feature.md#test-mapping) retain failed-plan and export-preservation evidence.

Local fixtures use dummy credentials, not live mutations. Hosted proof establishes access for its observed token/repository. Native release acceptance and published Action smoke establish the separate package identity boundary.
