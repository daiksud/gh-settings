# Independent PR message validation

The [message workflow](../../.github/workflows/pr-message.yml) validates the PR title and body on creation, push, reopening, and metadata edits. Its stable `pr-message` job uses the existing [commitlint policy](../../commitlint.config.mjs) and [policy tests](../../tests/github-actions/commitlint.sh). It has read-only permissions, no Draft/Ready changes, and no product build or acceptance work.

Types, scope, compatibility, and meaningful body content still require the [code-review policy](../../.agents/skills/code-review/SKILL.md); passing syntax checks cannot establish semantic correctness.

## Safe activation order

The original `ci-pr` message check and aggregate `pass` remain in place until the independent check is required. This temporary overlap prevents a gap in merge protection.

1. Merge the independent message workflow and verify that `pr-message` succeeds for the intended revision.
2. Require `pr-message` from the GitHub Actions app alongside `pass` and `gh-settings` in the active default-branch Ruleset. Preserve strict freshness, review and Code Owner requirements, thread resolution, and all unrelated rules. Inspect the effective live rule; a source declaration or successful workflow is not enforcement evidence.
3. Only then remove message validation from code CI and stop running product jobs for title/body-only edits. The code gate must reject activation until the independent message context is actually required.

The [repository settings declaration](../../.github/settings.yml) leaves Rulesets unmanaged. The ordinary PR token cannot observe complete bypass-actor information, so retain read-only PR credentials rather than exposing privileged credentials to candidate code. Manage protection through the authorized maintainer process and inspect the complete live inventory before changing it. Do not introduce an authoritative partial collection.

When a bounded main-only settings apply is used for activation, verify the complete live result and then omit the temporary `rulesets` group. Omission preserves the configured Ruleset and avoids privileged reads in later ordinary PR plans. Never replace the group with `rulesets: []`, which deletes the collection. The independent required check remains live after the source declaration is removed.

A title/body-only edit must not cancel an in-flight code run, turn skipped tests into a new successful code result, or change the last code-validation outcome. Code pushes and base changes still require product validation. Release execution, automatic version selection, and generated release notes are unrelated to this gate transition.
