# Independent PR message and code validation

The [message workflow](../../.github/workflows/pr-message.yml) validates the PR title and body on creation, push, reopening, and metadata edits. Its stable `pr-message` job uses the existing [commitlint policy](../../commitlint.config.mjs) and [policy tests](../../tests/github-actions/commitlint.sh). It has read-only permissions, no Draft/Ready changes, and no product build or acceptance work.

Types, scope, compatibility, and meaningful body content still require the [code-review policy](../../.agents/skills/code-review/SKILL.md); passing syntax checks cannot establish semantic correctness.

## Code-only validation

The [code workflow](../../.github/workflows/ci-pr.yml) still receives PR edits, but starts product work only on creation, push, reopening, or a base edit. A title/body-only edit skips the code path without writing a new `pass` result. Separate concurrency groups prevent a metadata edit from cancelling a running code validation.

A code run marks the PR head's `pass` commit status pending before Draft/Ready transitions, source checks, acceptance, and composite smoke. API-only reporting jobs have status-write permission but never check out or execute PR code or artifacts; product jobs remain read-only. The final reporter requires every prerequisite to succeed and the PR head/base to still match the evaluated revision. A failure, cancellation, skipped prerequisite, unavailable metadata, or missing independent message requirement cannot produce success. An interrupted run can leave pending status, which remains blocking until a valid code run completes.

The old workflow job named `pass` is replaced by the reported commit-status context of the same name. This avoids GitHub treating a metadata-only skipped job as a new successful required code check. The separate native `pr-message` check remains responsible for current text; `pass` no longer claims to validate it. The trusted `gh-settings` guard and all review/up-to-date protections are unchanged.

## Safe activation order

1. Merge the independent message workflow through ordinary review and verify that `pr-message` succeeds for the intended revision. During that preparatory stage, retain the original combined CI gate.
2. In the active default-branch ruleset, require `pr-message` from the GitHub Actions app alongside `pass` and `gh-settings`. Preserve strict up-to-date checking, review/Code Owner requirements, thread resolution, and every other rule. Recheck the live rule instead of treating source configuration or a successful workflow as enforcement evidence.
3. Only then merge the code-workflow split. Its final reporter reads the effective base-branch rules and deliberately rejects success until `pr-message` is required from GitHub Actions. A stacked PR targeting an unprotected topic branch also fails this activation guard until it is retargeted to the protected base.
4. Verify the reported `pass` context and provider for the intended PR head, as well as `pr-message` and the trusted guard. Edit the PR message and confirm that the previous code status remains unchanged while the current message is revalidated. Base changes and pushes must still run product checks.

The repository's current settings declaration does not manage Rulesets. Do not add an authoritative partial collection just to make this change: undeclared Rulesets or rules could be removed. Use the existing authorized repository-settings administration process and inspect the complete live configuration first. This document does not authorize an approval bypass or a broader policy change.

If activation or metadata lookup fails, correct the prerequisite and rerun code validation; do not force a successful status or disable the guard. Release execution, automatic version selection, and generated release notes are unrelated to this transition.
