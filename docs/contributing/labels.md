# Change labels

[Repository settings](../../.github/settings.yml) owns label names, colors, and descriptions. Use the conventional types `feat`, `fix`, `perf`, `revert`, `docs`, `refactor`, `style`, `test`, `build`, `ci`, and `chore` for change classification. `BREAKING CHANGE` and `DEPRECATED` record compatibility information alongside the applicable type. Independent triage labels, including `accessibility`, `duplicate`, `good first issue`, `help wanted`, `invalid`, `question`, and `wontfix`, remain available.

Authors and reviewers classify the actual change, not its branch name, file extension, or associated Issue's initial label. A new public Action capability can be `feat` even when implemented in YAML or shell; an internal CI feature is `ci`. Follow the [review policy](../../.agents/skills/code-review/SKILL.md) and retain its single-purpose PR and breaking-change requirements.

Issue forms use `fix` for bug reports and `feat` for feature requests as initial triage classifications. They do not prescribe the eventual PR type: a reported problem might be resolved by documentation or tests instead of a product fix.

Labels do not currently calculate versions or generate release notes. The [release procedure](releases.md) retains explicit version selection and the Git-log-based generator. Automatic version selection and `.github/release.yml` integration are separate work in [#385](https://github.com/daiksud/gh-settings/issues/385).

## Migrate legacy names without losing assignments

The settings workflow automatically reconciles the declared label collection after a relevant main push. Removing a name from the declaration can delete the live label and its assignments, so migration must be staged:

1. Declare the new labels while retaining `bug`, `enhancement`, and `documentation`. Update Issue forms to use the new names and verify the settings apply created them.
2. Enumerate all open and closed Issues **and PRs** carrying each old name. Preserve the inventory and add `fix`, `feat`, or `docs`, respectively, without replacing unrelated labels. Recheck every recorded assignment and repeat the inventory to catch intervening additions. This transfers historical classification, not a new judgment about past changes.
3. Only after all assignments have the corresponding replacement, remove the three legacy definitions through a separate reviewed PR. Verify the resulting apply and replacement assignments. Do not delete first and attempt to reconstruct lost associations afterward.

A matching name is not evidence that an earlier settings run succeeded. Inspect the actual run and live labels before each destructive step. Until migration is verified, keep both old and new definitions in the declaration.
