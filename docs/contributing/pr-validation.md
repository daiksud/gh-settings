# Independent PR message validation

The [message workflow](../../.github/workflows/pr-message.yml) validates the PR title and body on creation, push, reopening, and metadata edits. Its stable `pr-message` job uses the existing [commitlint policy](../../commitlint.config.mjs) and [policy tests](../../tests/github-actions/commitlint.sh). It has read-only permissions, no Draft/Ready changes, and no product build or acceptance work.

Types, scope, compatibility, and meaningful body content still require the [code-review policy](../../.agents/skills/code-review/SKILL.md); passing syntax checks cannot establish semantic correctness.

## Safe activation order

The original `ci-pr` message check and aggregate `pass` remain in place until the independent check is required. This temporary overlap prevents a gap in merge protection.

1. Merge the independent message workflow through ordinary review and verify that `pr-message` succeeds for the intended revision.
2. In the active default-branch ruleset, require `pr-message` from the GitHub Actions app alongside `pass` and `gh-settings`. Preserve strict up-to-date checking, review/Code Owner requirements, thread resolution, and every other rule. Recheck the live rule instead of treating source configuration or a successful workflow as enforcement evidence.
3. Only then remove message validation from code CI and stop running product jobs for title/body-only edits. The code gate must reject activation until the independent message context is actually required.

The repository's current settings declaration does not manage Rulesets. Do not add an authoritative partial collection just to make this change: undeclared Rulesets or rules could be removed. Use the existing authorized repository-settings administration process and inspect the complete live configuration first. This document does not authorize an approval bypass or a broader policy change.

A title/body-only edit must not cancel an in-flight code run, turn skipped tests into a new successful code result, or change the last code-validation outcome. Code pushes and base changes still require product validation. Release execution, automatic version selection, and generated release notes are unrelated to this gate transition.
