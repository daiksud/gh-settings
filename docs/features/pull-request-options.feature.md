# Repository auto-merge and branch-update options

## Status and scope

CLI and Action v0.13.0 support optional `settings.repository.allow_auto_merge` and `allow_update_branch` under `schemaVersion: 1`. Published v0.12.0 rejects these keys; the pinned v0.13.0 examples support them. This extends the existing repository group, with no new resource family or credential permissions.

The [configuration reference](../settings.md#auto-merge-and-branch-updates-v0130) owns values, omission and GitHub plan availability. `allow_auto_merge` permits PR auto-merge at repository level; it never schedules individual PRs or bypasses reviews/checks. `allow_update_branch` permits behind-head branch updates; it performs no branch operation or protection change.

## API and execution contract

| YAML boolean | GraphQL Repository field | Write |
| --- | --- | --- |
| `allow_auto_merge` | `autoMergeAllowed: Boolean!` | `allow_auto_merge` in repository REST PATCH |
| `allow_update_branch` | `allowUpdateBranch: Boolean!` | `allow_update_branch` in repository REST PATCH |

The [REST update endpoint](https://docs.github.com/en/rest/repos/repos#update-a-repository) accepts both booleans using existing Administration write. The [GraphQL public schema](https://docs.github.com/en/graphql/overview/public-schema) defines both observations. There is no PR auto-merge mutation, update-branch call, privilege fallback or permission expansion.

Omission is unmanaged, false is a declared value, and null/non-booleans fail local validation before target/authentication/API access. Existing strict spelling, alias/merge and duplicate-key rules apply. Fields are independent; neither requires or infers any other repository setting.

The shared `MergeMethodsState`, fixed query and deferred read retain their names and now contain six booleans and four commit enums. Declaring only either new option skips the base REST read. Omitting all merge-related fields skips GraphQL. A complete ten-field current-source observation (eight in v0.13.0) is required for any merge-related plan/export read: missing/null/invalid values or GraphQL errors including partial data discard the entire observation. Earlier partial plan comparisons are discarded, and export preserves the prior file on failure.

Apply sends declared fields only in the existing repository PATCH, with no preliminary read, inferred defaults or extra requests. The existing target, timeout, redirect, partial-failure and direct-apply race behavior is retained.

Current-source export saves fifteen repository fields; v0.13.0 saves thirteen, plus complete topics and all label pages after every read/serialization succeeds. v0.12.0 retains eleven fields. Stable comparison order is five base fields, three merge-method booleans, branch deletion, auto-merge, branch updates, squash title/message, current-source normal merge title/message, topics and labels. Each declared repository field counts once; Current-source report/Summary supports seventeen managed units; v0.13.0 retains fifteen, while v0.12.0 retains thirteen. Boolean changes remain native booleans, and Summary rejects wrong-typed values.

## Hosted read evidence

An [isolated PR Actions proof](https://github.com/daiksud/gh-settings/actions/runs/39112485520) returned `autoMergeAllowed: true` and `allowUpdateBranch: true` alongside all prior merge-related fields. Its effective permissions were Contents read and Metadata read only, without checkout, App credentials, Environment or writes. The temporary workflow is removed from the final implementation diff. This proves access for that token/repository, not every credential.

## Test mapping

| Layer | Tests |
| --- | --- |
| Boolean presence, omission, order and equality | [Settings comparison](../../internal/settings/merge_methods_test.go) |
| Strict types/spelling, aliases and round trips | [YAML codec](../../internal/workspace/merge_methods_test.go) |
| Complete fixed GraphQL read, missing/null/type failures, partial errors and declared-only PATCH | [Shared adapter boundary](../../internal/github/merge_methods_test.go), [new boolean reads](../../internal/github/pr_options_test.go) |
| Deferred read, direct write, local validation and export preservation | [PR options](../../internal/app/pr_options_test.go), [shared failures](../../internal/app/merge_methods_test.go), [export](../../internal/app/export_test.go) |
| Native boolean report/Summary, wrong types and managed bound | [Boolean presentation](../../internal/actions/merge_methods_test.go), [count validation](../../internal/actions/squash_commit_test.go) |
| Real binary composition | [Installed-process acceptance](../../tests/acceptance/merge_methods_test.go): equality/differences, missing/null/partial errors, direct one/pair PATCH, validation before requests, full export and credential-free Summary. Existing acceptance covers unmanaged/no-op and prior fields. |

Local tests use dummy tokens and isolated fixtures. Hosted proof establishes access for the observed token/repository, not all credentials; release native acceptance and published Action smoke establish the separate installed-package identity.
