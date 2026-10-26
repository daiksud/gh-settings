---
type: Feature
title: Repository lifecycle design
description: Define archival sequencing and the boundary for deferred identity and visibility changes.
---

## Status and scope

Archive/unarchive is supported according to [ADR 0027](../adr/0027-repository-lifecycle.md). No real repository lifecycle operation or token permission was used as release evidence.

The target stays the checkout's GitHub.com origin. There is no repository creation, deletion, transfer, fork migration, organization policy update, local-origin rewrite, saved-plan execution, or automatic rollback.

## Archive declaration and observation

`settings.repository.archived` is an optional strict boolean. Omission acquires no new lifecycle read and performs no lifecycle work; explicit null, strings, and numbers are rejected locally. False requests an active repository; true requests an archived repository. This is not a declaration of individual issue, release, or collaborator states.

A declared archive plan/apply and `export --archive` require a complete fresh repository identity: positive numeric ID, canonical owner/name matching the selected target, and a present boolean `archived`. Missing/null/incorrect types, 401/403/404, redirects, wrong IDs, and unavailable canonical names fail. Absence is never interpreted as active, archived, or a missing repository to create.

Plan adds one native boolean comparison and preserves current nonzero error behavior. When the transition changes state, both CLI and Action Summary explicitly state unarchive-before-other-settings or archive-after-other-settings ordering. An already archived target with `archived: true` plus other managed resources remains readable: plan completely observes those resources and reports an honest matching, changed, or failed result without attempting to make them writable. Apply still rejects that combination before any write because changing an archived repository requires explicit `archived: false`. A failed or partial observation never becomes a complete plan. The Action Summary uses the same boolean and escapes identifiers. Ordinary export omits archive; opted-in export writes it only after all selected reads complete and preserves the previous file on failure.

## Transition and operation order

| Observed state | Declaration | Other managed settings | Planned/apply behavior |
| --- | --- | --- | --- |
| Either | Omitted | Any | Existing behavior; no new lifecycle read or write |
| Active | false | Any | No lifecycle write; perform ordinary updates |
| Archived | false | Any | Verify target; unarchive and verify before ordinary updates |
| Active | true | None | Archive and verify |
| Active | true | Present | Perform ordinary updates; then freshly verify target/state, archive and verify |
| Archived | true | None | Matching read; no lifecycle write |
| Archived | true | Present | Plan completely observes and compares; apply rejects before any write; do not implicitly unarchive and rearchive |
| Unknown or mismatched | Either | Any | Fail before starting writes |

Ordinary resource order is unchanged between lifecycle boundaries. Unarchive uses a separate body containing only `archived: false`; final archive uses only `archived: true`. Do not combine either with unrelated fields. A fresh pre-archive observation is required after earlier resource work, and response/readback ID and canonical name must match the initial identity.

A concurrent state change is not an authorization to undo it blindly. Re-read at the transition boundary and stop on an unexpected identity or incompatible state. Read/response checks cannot provide a transaction, lock, or unconditional protection against a name being reassigned between requests. No successful plan or prior read is used as an apply precondition token.

## Failure and recovery observations

| Failure point | Required result |
| --- | --- |
| Invalid local declaration or initial identity/state read | Nonzero; no update has started |
| Unarchive request/response or its readback | Stop; report attempted transition as uncertain unless complete requested state was observed; no ordinary updates |
| Ordinary update after a confirmed unarchive | Preserve/report unarchive and earlier confirmed updates, including in the returned error if a caller discards the detailed progress; do not rearchive to simulate rollback |
| Ordinary update before requested final archive | Preserve/report earlier updates; final archive is not attempted |
| Fresh pre-archive read | Stop; preserve earlier writes and report archive as not attempted |
| Archive request, transport failure or verification | Nonzero; report that a dispatched PATCH may have applied even if no response was received; do not write further settings |
| Output/Summary failure after verified changes | Retain failure and known remote progress; never repeat or undo writes to repair presentation |

Permission loss or an inaccessible repository after a transition is an unknown result, not proof that it is deleted or that the transition failed remotely. Operators inspect and recover deliberately. Do not promise the repository's own workflow can later unarchive it; a trusted external checkout or GitHub's administrator interface may be needed. No additional credential is requested or provisioned automatically.

## Examples of the intended interface

```yaml
schemaVersion: 1
settings:
  repository:
    archived: false
    description: Active maintenance has resumed
```

A plan for an archived target shows `repository.archived: true -> false` and identifies that unarchive precedes the description update. Applying it performs the isolated unarchive first. With `archived: true` on an active target, the description update precedes archival. With both fields on an already archived target, plan reads and reports whether the description matches or differs, but the unchanged boolean does not imply that the description is writable: apply rejects and requires explicit unarchive intent.

A source CLI/Action failure identifies `repository.archive` progress without inventing success for later operations. Status-only reporting must retain its existing bounded outcomes and confidentiality: do not expose hidden values merely to explain a lifecycle failure.

## Visibility and rename disposition

Visibility and rename remain unsupported, including same-value requests. No `private` alias, guessed default, empty-string reset, or auto-normalized name is accepted. No export option silently includes them. Visibility changes can alter disclosure and feature availability; rename can break the next invocation and Action consumers. They require a separate user-reviewed operation/identity design before #333 or #334 is adopted. This is a deliberate boundary, not a claim that GitHub lacks those APIs.

Future visibility design must select one enum, specify applicability and the effects of public/private/internal transitions, avoid changing visibility before other writes, and define failures after access loss. Future rename design must specify exact same-name behavior, reject empty/invalid destinations, detect conflicts, bind stable identity, and explicitly hand off the origin and dependent references. Neither operation may be a fallback from a failed read or be combined with automatic rollback.

## Implementation verification and remaining operational evidence

Model and codec tests cover omission, true/false, null/type rejection, and retained scalar identity. The [archive API tests](../../internal/github/archive_test.go) exercise complete identity/state, narrow PATCHes, permission and response failures, redirects, rejected updates, and post-write readback. The [application tests](../../internal/app/archive_test.go) verify selective plan, opt-in export, ordering, matching no-ops, early rejection, and retained confirmed or uncertain progress.

The [built-CLI acceptance fixture](../../tests/acceptance/archive_test.go) covers ordinary and opt-in export, preserved output files on incomplete reads, read-only matching/different evaluation while archived, apply rejection without writes, separate Action plan/Summary, archive after other writes, unarchive before them, and repeated matching archive. Existing composite checks cover the same-source Action interface and the status-only reporting boundary; they do not simulate an actual repository archive or prove remote permission. All mutation tests use local fixtures and dummy credentials, not this repository's real archive setting.

Before claiming operational archival support, an explicitly authorized disposable-repository experiment must separately verify actual permissions, transition/readback behavior, and a documented recovery route. Do not archive a production or development repository merely to make a fixture claim stronger. No such experiment has been authorized or performed by this implementation.

## Official contracts

Reviewed October 22, 2026:

- [Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents `archived` and explicit false to unarchive, and requires Administration write for fine-grained mutation tokens. Metadata reads do not establish mutation permission.
- [Archive and unarchive](https://docs.github.com/en/repositories/archiving-a-github-repository/archiving-repositories) describes repository read-only behavior and administrator recovery. A few separately documented security operations do not imply ordinary settings remain writable.
- [Visibility changes](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/setting-repository-visibility) describes organization restrictions and collateral effects, including public logs/forks and changed feature availability.
- [Repository rename](https://docs.github.com/en/repositories/creating-and-managing-repositories/renaming-a-repository) distinguishes ordinary redirects from unsupported Action-reference redirects and recommends an explicit origin update.

These primary contracts justify the design; they are not evidence of this installation's grants, a live state transition, or already implemented product behavior.
