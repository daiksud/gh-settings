---
name: code-review
description: Review gh-settings pull requests and diffs for correct behavior, compatibility, security, focused scope, and durable Conventional Commits squash messages. Use for code review, PR review, self-review before opening a PR, and reassessment after review fixes, including changes to code, tests, documentation, Actions, or repository automation.
---

# gh-settings code review

Review the proposed change against its actual base and public contracts. Apply the PR policy to every PR and the implementation checks only to affected behavior. Keep findings concrete and proportionate to their impact.

## Establish the review context

1. Read the applicable agent instructions and [CONTRIBUTING](../../../CONTRIBUTING.md).
2. Identify the PR's base and current head, read the complete diff and relevant surrounding code, and obtain its current title, body, checks, and unresolved findings. For a local review, identify the comparison base and any proposed PR metadata supplied by the author.
3. Read the affected contracts: [commands](../../../docs/commands.md), [configuration](../../../docs/settings.md), or [public Action](../../../docs/github-actions.md). Use [architecture](../../../docs/contributing/architecture.md) and [testing](../../../docs/contributing/testing.md) to find the owners. Treat historical ADRs and proposed designs as context, not automatically as current behavior.
4. State any unavailable evidence. A diff-only review cannot establish that an unseen PR title/body complies or that hosted checks passed. Reassess changed code and metadata when they change after review.

## Require one coherent change per PR

- Require one change purpose that can be described by one Conventional Commits type and, when present, one meaningful scope. Reject independently useful changes that need different types or scopes; request separate PRs.
- Treat implementation, tests, specifications, documentation, and necessary wiring for that same change as one unit. File extensions, directories, and the number of layers touched do not themselves define types or scopes. For example, a settings feature can include its codec, API adapter, CLI presentation, tests, and documentation in one `feat(settings)` PR.
- Split an unrelated bug fix, feature, cleanup, dependency update, or formatting sweep even if discovered while doing the main work. Sharing a type or scope does not justify combining unrelated purposes.
- Reject combined types such as `feat/fix` and multiple scopes such as `cli,actions`, `cli/actions`, or `cli\actions`. Do not accept scope omission or a vague scope such as `misc` as a way to hide unrelated changes.
- Keep scope optional. When used, select a single lowercase area that describes the change, such as `settings`, `cli`, `actions`, or `release`; these are examples, not an allowlist. Do not require a scope for each touched package.

## Review the complete squash message

Treat the PR title as the header and the PR body as the body/footers of **one** [Conventional Commits 1.0.0 message](https://www.conventionalcommits.org/en/v1.0.0/). Review `PR_TITLE + "\n\n" + PR_BODY`; do not require a second Conventional Commits header in the PR body or require rewriting every local commit.

### Title and type

Require `type: description` or `type(scope): description`, adding `!` immediately before `:` for a breaking change. Select the type by the **result for gh-settings users**, not by whether the implementation added code or fixed a problem somewhere in the repository.

Branch names are identifiers, not classification evidence. Do not require type prefixes or request a branch rename or replacement PR to correct a type. Authors and reviewers edit the PR title and labels instead. Check labels against the actual diff, independently of an associated Issue's initial classification; remove stale path suggestions when the PR changes scope. File paths alone do not determine feature, fix, performance, refactoring, or compatibility intent. Reconcile the final title, labels, body, and squash message before merging.

For `feat` and `fix`, **users** are people consuming the gh-settings CLI or public GitHub Action in their own repositories, not contributors maintaining this repository. Compare the observable product behavior before and after: supported commands, configuration, output/errors, Action inputs/outputs, installation, and execution. Product features intended for an upcoming release count even before publication. Build scripts, CI, release tooling, and test harnesses used only to maintain this repository are not themselves gh-settings product features.

Decide in order:

1. Does the change add a capability or supported behavior that gh-settings consumers can use? Choose `feat`.
2. Does it correct a defect or regression in gh-settings product behavior that consumers could encounter? Choose `fix`.
3. Otherwise, **do not choose `feat` or `fix`**, even when an internal system gains functionality or a defect in that system is repaired. Choose the most specific internal-work type, such as `build`, `ci`, `test`, `docs`, or `refactor`.

| Type | Use for |
| --- | --- |
| `feat` | A new capability or supported behavior for users of the gh-settings CLI or public Action, not an internal tooling feature |
| `fix` | Correcting an existing user-observable defect in gh-settings CLI or public Action behavior, not an internal tooling defect |
| `perf` | Improving the runtime performance experienced by gh-settings users without adding a separate capability or correcting incorrect behavior |
| `refactor` | Restructuring implementation without changing user-observable product behavior; assess compatibility separately |
| `docs` | Documentation or contributor/reviewer guidance changes |
| `test` | Test-only additions or corrections without changing product behavior |
| `ci` | Adding, changing, or repairing this repository's CI checks or development automation |
| `build` | Adding, changing, or repairing internal build, packaging, release tooling, or dependency management |
| `style` | Code formatting or whitespace changes without a behavior change |
| `revert` | Reverting a previous change; identify the reverted commit and reason in the body |
| `chore` | Maintenance that does not fit a more specific type |

Examples of the product-user boundary:

| Change | Type | Reason |
| --- | --- | --- |
| Add a new supported settings field, including its tests and documentation | `feat` | Users gain a gh-settings capability |
| Correct a `gh settings plan` result or public Action failure on a supported runner | `fix` | Users encounter the incorrect behavior |
| Add a build target or repair a broken internal build script | `build` | The tool's supported behavior does not change |
| Add a CI job or repair a failing PR workflow | `ci` | Repository maintenance automation changes, not the product |

Do not label internal work `feat` merely because it adds functionality or `fix` merely because it repairs a bug. A faster CI pipeline is `ci`, not a gh-settings `perf` change. Conversely, a public Action behavior change can be `feat` or `fix` even when implemented in YAML or shell. Tests, documentation, and required wiring for the same product change remain part of that change's type. Classify by demonstrated user-visible consequences, not speculative indirect benefits such as faster future releases.

Use the pinned [commitlint configuration](../../../commitlint.config.mjs) and [development guide](../../../docs/contributing/development.md#pull-request-messages) for syntax/style checks: the conventional 11-type list, lowercase type/scope, subject-case restrictions, no final subject period, and no surrounding title whitespace. Do not add arbitrary title/body line-length limits. A passing lint result does not establish that the type matches the diff, that the scope is coherent, or that breaking changes have been disclosed.

### Body and durable history

Require the [PR template](../../../.github/pull_request_template.md)'s explanation of **Why**, **What changed**, and **Verification**, following [the Git history guidance](../../../CONTRIBUTING.md#write-pr-bodies-for-the-git-history). The body must explain the problem, resulting behavior, material decisions, and observed verification results without requiring access to an Issue, CI log, or review conversation. An empty body may pass commitlint but does not meet this review policy.

Include compatibility, migration, and operational effects when relevant. Keep links as supporting evidence; use `Closes #123` only for work actually completed. Remove template comments, temporary progress notes, approval requests, full logs, and file-by-file diff narration. Require the final title/body to match the final diff and verify the final squash message before merging.

### Breaking changes: follow Conventional Commits

Apply the [v0.x compatibility policy](../../../AGENTS.md#v0x-compatibility-policy):
do not preserve obsolete behavior merely to avoid an incompatible change.
Identify user-observable changes to supported CLI commands, options,
configuration, outputs, exit codes, and public Action inputs and outputs.

Conventional Commits marks a breaking change with **either** a `!` immediately
before the title colon (`type!:` or `type(scope)!:`) **or** a
`BREAKING CHANGE:` footer. Either is sufficient; do not require both,
automatic migration, or upgrade guides during v0.x. Reviewers assess whether
the PR accurately describes the user impact. PRLint only checks the title
and body together with commitlint, not a separate semantic policy.
An internal-only change is not automatically a public breaking change.

## Check the affected implementation contracts

Use the linked contract as the source of detail; do not infer supported release behavior from an unmerged proposal or source code alone.

- **Declared state and collections:** Preserve the distinction between omission, explicit `false`, empty strings/collections, and invalid `null`. Update only managed values. For topics, retain authoritative collection membership. For labels and Rulesets, review ID-selected whole-object ownership, create-only collisions, default-derived changes, created-ID writeback/recovery and preservation of unlisted objects. Environments instead use name-selected whole-object upsert with fixed supported defaults, no numeric-ID writeback and no unlisted deletion; unknown protection must fail rather than be erased. Review these boundaries against the [configuration contract](../../../docs/settings.md).
- **Validation and targeting:** Preserve validation before remote work and the isolation of the target checkout and its direct GitHub origin. Check the documented no-op cases without confusing an omitted resource with an explicitly empty managed collection. Preserve the differences between [CLI](../../../docs/commands.md) and [Action](../../../docs/github-actions.md) prerequisites.
- **Plan, apply, and failures:** Keep plan read-only. Preserve direct apply semantics, fresh label reconciliation, operation ordering, and honest partial-failure reporting. Do not present a saved plan, a zero-difference plan, or an output error as proof of transactional application, no writes, drift protection, or rollback. Review any intended contract change explicitly.
- **API and file boundaries:** Require complete, correctly typed reads and pagination before comparison/export; do not convert missing data or partial API failures into plausible empty state. Check declared-only payloads, target/path handling, redirect policy, and failure order. Preserve documented file-safety guarantees without inventing stronger atomicity promises.
- **Ownership and simplicity:** Keep declaration/state/comparison in `internal/settings`, YAML/files/Git in `internal/workspace`, API mapping in `internal/github`, sequencing in `internal/app`, and presentation in `internal/cli` or `internal/actions`. Reuse common use cases. Request new abstractions or compatibility layers only for a concrete need; do not force unrelated architecture work into a focused PR.
- **Action and release integrity:** Check pinned Action/CLI identity, checksum/version verification, safe escaping, token boundaries, failure status propagation, and output publication after required work succeeds. For release changes, preserve the identity of tested and published artifacts and immutable releases. Confirm availability using the actual Action source, `action/cli-version`, consumer pin, and relevant publication evidence; use the [Action guide](../../../docs/github-actions.md) and [release guide](../../../docs/contributing/releases.md).
- **Repository automation:** For workflow, credential, guard, or privileged apply-path changes, inspect the code that receives the token, its target, permissions, and referenced Actions. Follow the [repository automation boundaries](../../../docs/contributing/repository-automation.md); a green Environment guard does not review the allowed apply job's step bodies. Verify live settings only when the change or a claimed outcome depends on them. Do not treat source YAML or local fixtures as proof of current approval, ruleset enforcement, or token revocation.
- **Tests and documentation:** Require evidence for the changed behavior and relevant failure/side-effect cases at their owning layer. For a bug fix, seek a regression case that fails for the original defect; for composition changes, select representative acceptance or Action checks. Avoid repeating every low-level case at every layer or requiring unrelated product tests for documentation-only changes. Update the authoritative public contract and affected behavior specifications together, following [CONTRIBUTING](../../../CONTRIBUTING.md#where-documentation-belongs) and the [testing guide](../../../docs/contributing/testing.md).

## Report actionable findings and a bounded verdict

- Prioritize defects, compatibility mistakes, unsafe mutations, credential exposure, and explicit PR-policy violations. For each finding, identify the file/line or PR title/body, triggering condition, expected versus actual behavior, concrete impact, and a focused correction or split. Describe likelihood when it affects priority; do not present hypothetical concerns as confirmed failures.
- Mark unmet mandatory type/scope, body, or breaking-change requirements as blocking even if CI passes. Distinguish these from product bugs and optional improvements. Keep personal style preferences and speculative generalization nonblocking; do not manufacture findings to fill a quota.
- Summarize what was actually reviewed and verified, remaining limitations, and whether changes are required. If no actionable issue was found, say so without claiming unavailable checks passed. Do not make unrelated pre-existing problems blockers for this change.
- Before reporting merge readiness, confirm current required checks, unresolved findings, and actual reviews for the intended revision. A prior bot completion marker, reaction, or successful approval workflow alone does not establish a current approval; follow [the approval-control guidance](../../../docs/contributing/repository-automation.md#codex-approval-control).
- Do not treat a successful merge as release authorization. Create release tags, dispatch the release workflow, or publish only after an explicit user request, following the release guide without bypassing validation or replacing published artifacts.
