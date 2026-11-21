# Development policies

## Adding a setting

Start with the control in the GitHub Web UI for a personally owned public repository. Check the project's [feature adoption scope](../references/settings/scope.md#feature-adoption-scope) for applicable exclusions. In the [coverage ledger](settings-coverage.md), distinguish three questions.

1. Can a repository owner configure this control in the target UI?
2. Do the applicable official APIs let gh-settings observe its value and make and verify the requested change?
3. Which parts of export, validation, plan and apply does the current implementation support, and what evidence covers them?

Record incomplete API evidence as unconfirmed. When a control is ready to implement, define its public names, accepted values, ownership and omission semantics in the [settings reference](../references/settings/README.md). Carry that contract through the owning packages and tests using the [development guide](setup.md#add-an-agreed-repository-field).

## Changing a public contract

During v0.x, replace superseded CLI, Action and YAML interfaces together with their documentation and tests. Remove obsolete formats, aliases and migrations. Preserve strict validation, checkout and target isolation, credential and data protection, and accurate reporting of failures and confirmed partial changes. See the [v0.x compatibility policy](../../AGENTS.md#v0x-compatibility-policy).

Assess the change from the caller's perspective. Mark an incompatible change in the proposed squash message with `!` or a `BREAKING CHANGE:` / `BREAKING-CHANGE:` footer. Follow the [code-review policy](../../.agents/skills/code-review/SKILL.md) for classification, review and merge requirements.

## Documenting source, releases and live settings

The current source, a published package and settings applied on GitHub can be at different revisions. Identify which one the documentation describes.

- Document current CLI and YAML behavior in the public references. Keep runner requirements, credential setup and partial-failure recovery with the task they affect.
- For a workflow, use its Action pin, the [Action metadata](../../action.yml) at that revision and the corresponding [published release](https://github.com/daiksud/gh-settings/releases) to identify the executing package. Git history and immutable releases preserve earlier contracts.
- For this repository's applied Rulesets, Environment protections and token revocation, describe what was checked on GitHub. Source declarations and local fixtures establish intended settings and implementation behavior. See the [verification procedure](repository-automation.md#inspect-external-configuration).
- Keep the reason for a specific change in its [PR and squash message](../../CONTRIBUTING.md#write-pr-bodies-for-the-git-history). Consult the [architecture decision records](adr/README.md) for recorded decisions, their status and historical context.
