# Independent PR message validation

The [message workflow](../../.github/workflows/pr-message.yml) validates the PR title and body on creation, push, reopening, and metadata edits. Its stable `pr-message` job uses the existing [commitlint policy](../../commitlint.config.mjs) and [policy tests](../../tests/github-actions/commitlint.sh). It has read-only permissions, no Draft/Ready changes, and no product build or acceptance work.

Types, scope, compatibility, and meaningful body content still require the [code-review policy](../../.agents/skills/code-review/SKILL.md); passing syntax checks cannot establish semantic correctness.

## Safe activation order

The original `ci-pr` message check and aggregate `pass` remain in place until the independent check is required. This temporary overlap prevents a gap in merge protection.

1. Merge the independent message workflow and verify that `pr-message` succeeds for the intended revision.
2. Apply the temporary [repository settings declaration](../../.github/settings.yml), which adds `pr-message` from GitHub Actions alongside `pass` and `gh-settings`. Inspect the exact main apply and compare the complete live Ruleset before and after; a successful source check is not enforcement evidence.
3. After verifying the new requirement, omit the `rulesets` group again in a follow-up PR. Omission leaves the live Ruleset unchanged and restores ordinary read-only PR plans. Never substitute `rulesets: []`, which deletes the managed collection.
4. Only then remove message validation from code CI and stop running product jobs for title/body-only edits. The code gate must reject activation until the independent message context is actually required.

The temporary declaration owns the complete repository-owned branch/tag Ruleset membership. Its verified inventory contains only `default`, with all five existing rule types retained. Only the required-check list is declared in rule parameters; omitted review parameters, strict freshness, conditions, enforcement, and bypass actors preserve their freshly observed values. Recheck the complete live collection before applying: an undeclared member can be deleted by reconciliation. This activation does not authorize weakening review or bypass settings.

The ordinary PR token cannot read complete bypass-actor information, so its settings plan fails closed while the temporary Ruleset declaration is present. Do not provide privileged credentials to PR code, suppress that failure, or describe the plan as successful. For this bounded activation, review the complete live Ruleset through an authorized maintainer read, inspect the exact declaration and passing source/message/guard checks, and verify the existing main-only apply separately. Do not retain this declaration as a permanent replacement for a privileged planning design.

A title/body-only edit must not cancel an in-flight code run, turn skipped tests into a new successful code result, or change the last code-validation outcome. Code pushes and base changes still require product validation. Release execution, automatic version selection, and generated release notes are unrelated to this gate transition.
