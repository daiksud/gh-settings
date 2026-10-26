# Squash commit title and message defaults

## Status and scope

CLI and Action Current source supports `settings.repository.squash_merge_commit_title` and `squash_merge_commit_message` under `schemaVersion: 1`. This extends the existing repository fields; it does not introduce a resource family, change merge methods, edit PR contents or perform a merge.

The [configuration reference](../settings.md#squash-commit-defaults-v0120) owns the public declaration and value semantics.

## API contract

GitHub's [Update a repository REST API](https://docs.github.com/en/rest/repos/repos#update-a-repository) accepts:

| Declaration | Values | GraphQL Repository observation |
| --- | --- | --- |
| `squash_merge_commit_title` | `PR_TITLE`, `COMMIT_OR_PR_TITLE` | `squashMergeCommitTitle: SquashMergeCommitTitle!` |
| `squash_merge_commit_message` | `PR_BODY`, `COMMIT_MESSAGES`, `BLANK` | `squashMergeCommitMessage: SquashMergeCommitMessage!` |

REST requires title when message is specified. Local validation rejects message-only declarations before target/authentication/API access; title-only is valid. No title is synthesized from live state. Null, empty, unknown and wrong-typed enums fail validation. Omitted fields remain unmanaged, independently of `allow_squash_merge`.

The [GraphQL public schema](https://docs.github.com/en/graphql/overview/public-schema) defines the non-null enums. It uses only the origin-selected owner/name, with no caller query or mutation. A declaration of only title (or both enums) skips the base REST read; omission of every merge-related field skips GraphQL.

Missing/null repository or fields, invalid enum strings/types, malformed JSON, HTTP/transport failures and any GraphQL errors (including partial data) fail the whole observation. They never become empty/default values or trigger a privileged fallback. Plan discards earlier partial comparisons; export preserves the prior file on read failure.

Apply uses the existing repository REST PATCH with Administration write. It sends declared fields only, without a preliminary read, implicit title/message, or merge-method updates. Normal direct-apply race and partial-failure semantics remain unchanged. No credential grant, visibility change, branch protection change or other field is managed by this feature.

## Export, ordering and presentation

Archive and private forking remain separate, mutually incompatible opt-in selections.

Repository comparison retains registry order for repository scalars, followed by collections and policy resources in their documented order. Each declared scalar counts once and each managed collection once. Current-source report/Summary accepts at most fifty-one managed units across all adopted resources; older packages retain their documented lower limits. Enum changes retain native JSON strings and quoted CLI/Summary values. Unknown, empty or wrong-typed enum report values are rejected. Reports remain presentation data, never apply input.

## Read-only hosted evidence

An [isolated PR Actions proof](https://github.com/daiksud/gh-settings/actions/runs/39108650790) returned `squashMergeCommitTitle: COMMIT_OR_PR_TITLE` and `squashMergeCommitMessage: COMMIT_MESSAGES` alongside the existing four booleans. The job had effective Contents read and Metadata read only, with no checkout, App credential, Environment or write permission. Its temporary workflow is removed from the final implementation diff. This establishes access for that token/repository, not every possible credential.

## Test mapping

| Layer | Tests and evidence |
| --- | --- |
| Declaration/model | [Codec](../../internal/workspace/squash_commit_test.go): every enum combination, title-only/omitted message, invalid values/types, aliases/merges, message-only rejection and round trips. [Comparison](../../internal/settings/squash_commit_test.go): strings, equality, omission and order. |
| GitHub transport | [Enum read and exact PATCH](../../internal/github/squash_commit_test.go), [shared GraphQL boundary](../../internal/github/merge_methods_test.go): missing/null/invalid values, complete response validation, partial errors, declared-only title/pair PATCH and no GET. |
| Application | [Sequencing](../../internal/app/squash_commit_test.go), [shared failure cases](../../internal/app/merge_methods_test.go), [export](../../internal/app/export_test.go): deferred reads, local validation, direct writes, earlier-comparison discard and export preservation. |
| Actions | [Typed report/Summary](../../internal/actions/squash_commit_test.go): strings, quoted changes, enum rejection and the current-source fifty-one-unit bound. |
| Installed binary | [Process acceptance](../../tests/acceptance/merge_methods_test.go): title-only/pair plan, equality, shared representative read failures, exact direct PATCH, message-only rejection, complete export, typed report and credential-free Summary. Existing acceptance retains empty/no-op and prior merge-field behavior. |

The process cases `squash unknown rejected` (plan) and `squash message-only rejected` (apply) represent enum and message/title dependency validation across the executable boundary. Both verify a nonzero exit status, the diagnostic, zero API requests, and preservation of the input file. Full enum/type/null matrices remain in `TestSquashCommitCodec`; `TestCommitMessageDependencies` in the [settings tests](../../internal/settings/commit_dependencies_test.go) owns dependencies within and across commit-default families.

`TestSquashCommitReadRejectsIncompleteEnums` owns the per-field response matrix formerly repeated by `squash missing title` and `squash null message`. The complete-data-with-errors fixture in `TestReadMergeMethodsGraphQLBoundary` covers the former `squash partial error`; the [shared process read-failure cases](merge-methods.feature.md#test-mapping) retain failed-plan and export-preservation evidence.

Owner/process fixtures use dummy credentials and local API responses. Hosted token proof establishes access only for the observed token/repository. Release native acceptance and published Action smoke establish the separate package identity boundary.
