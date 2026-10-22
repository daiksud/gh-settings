# Independent PR message validation

The [message workflow](../../.github/workflows/pr-message.yml) validates the PR title and body on creation, push, reopening, and metadata edits. Its stable `pr-message` job uses the existing [commitlint policy](../../commitlint.config.mjs) and [policy tests](../../tests/github-actions/commitlint.sh). It has read-only permissions, no Draft/Ready changes, and no product build or acceptance work.

Types, scope, compatibility, and meaningful body content still require the [code-review policy](../../.agents/skills/code-review/SKILL.md); passing syntax checks cannot establish semantic correctness.

## Safe activation order

The original `ci-pr` message check and aggregate `pass` remain in place until the independent check is required. This temporary overlap prevents a gap in merge protection.

1. Merge the independent message workflow and verify that `pr-message` succeeds for the intended revision.
2. Apply the [repository settings declaration](../../.github/settings.yml), which adds `pr-message` from GitHub Actions alongside `pass` and `gh-settings`. Inspect the exact main apply and effective live rules before continuing; a successful source check is not enforcement evidence.
3. Only then remove message validation from code CI and stop running product jobs for title/body-only edits. The code gate must reject activation until the independent message context is actually required.

The declaration owns the complete repository-owned branch/tag Ruleset membership. Its initial inventory contains only `default`, with all five existing rule types retained. Only the required-check list is declared in rule parameters; omitted review parameters, strict freshness, conditions, enforcement, and bypass actors preserve their freshly observed values. Before adding or removing a Ruleset or rule, inventory the complete live collection and update the declaration together. An undeclared member can be deleted by reconciliation. This activation does not authorize weakening review or bypass settings.

A title/body-only edit must not cancel an in-flight code run, turn skipped tests into a new successful code result, or change the last code-validation outcome. Code pushes and base changes still require product validation. Release execution, automatic version selection, and generated release notes are unrelated to this gate transition.
