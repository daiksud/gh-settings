# Contributing to gh-settings

This guide is for people changing gh-settings itself: its code, tests, documentation, or repository automation. To use the published CLI or Action in your own repository, start with the [README](README.md) and [user documentation](docs/README.md#for-users).

## Getting started

1. Read the relevant [public contract](docs/README.md#for-users) before changing behavior.
2. Follow the [development guide](docs/contributing/development.md) to install the pinned tools, build, and run from source.
3. Use the [architecture guide](docs/contributing/architecture.md) and [model glossary](docs/contributing/glossary.md) to find the code that owns the change.
4. Validate the affected behavior using the [testing guide](docs/contributing/testing.md). It explains local checks, acceptance tests, CI, and the limits of each kind of evidence.
5. Open a PR explaining the problem, the resulting behavior, and how you verified it. Apply the [code-review policy](.agents/skills/code-review/SKILL.md), address review feedback, and let the required checks finish before merging.

The repository's CI moves PRs through Draft and Ready states. Its review automation and settings workflows are described in [repository automation](docs/contributing/repository-automation.md); a successful bot workflow alone does not establish a current approval.

## Issues and pull requests

When [opening an issue](https://github.com/daiksud/gh-settings/issues/new/choose), choose the template that fits the work:

- **Bug report**: describe actual and expected behavior, a minimal reproduction with the relevant starting state, and the exact CLI version or Action ref. Add environment details when they affect reproduction.
- **Feature or improvement**: explain the current obstacle and an observable result that would solve it, with a concrete usage example.
- **Task**: identify the maintenance problem and the concrete result that completes it, including the affected area and how to check the result.

Blank issues remain available for questions or work that does not fit these forms. Use the same information structure when submitting through the web, CLI, or API. Add constraints or references to the relevant answer only when they affect the work.

### Write PR bodies for the Git history

The PR body is intended to become the squash commit message. Use the [PR template](.github/pull_request_template.md) to leave a durable explanation that is understandable from `git log` alone:

- **Why**: state the concrete problem and its impact, including the essential background even when an Issue already describes it.
- **What changed**: explain the resulting behavior and how it solves the problem. Include non-obvious design reasons and any compatibility, migration, or operational effects needed to use or maintain the change.
- **Verification**: summarize the behavior or cases checked, the method, and observed results. State any gap that limits the claimed outcome. The [testing guide](docs/contributing/testing.md) helps select appropriate checks.

Issue, test, and CI links supplement this explanation; the problem, decisions, and verification results must remain understandable if those links are unavailable. Use `Closes #123` only when the PR completes the issue.

Keep routine CI command lists, full logs, file-by-file diff narration, unrelated unchanged areas, and agent work history out of the body. Use PR comments or GitHub's review/check records for temporary progress, review requests, and approval status. Include the specific evidence and consequences that explain this change.

Before merging, update the body to match the final diff and results, remove template comments and temporary notes, and confirm the final squash message contains that body. The repository declaration selects `squash_merge_commit_message: PR_BODY`; this default takes effect after the settings workflow applies it.

## Where documentation belongs

Choose the reader and the question first. Use one authoritative page for a detailed explanation, and link to it from other documents.

| Content | Authoritative home |
| --- | --- |
| What the product does, installation, and a first successful use | [README](README.md) |
| Navigation by audience and task | [Documentation index](docs/README.md) |
| Public CLI syntax, target selection, output, and errors | [Command reference](docs/commands.md) |
| Supported settings, YAML schema, omission, and value semantics | [Configuration reference](docs/settings.md) |
| Public Action inputs/outputs, prerequisites, caller setup, and recovery | [GitHub Actions guide](docs/github-actions.md) |
| Copyable consumer workflows and instructions specific to those files | [Examples](examples/github-actions/README.md) |
| Tool setup, building, and making a code change | [Development guide](docs/contributing/development.md) |
| Current design, code ownership, terminology, and future design boundaries | [Architecture](docs/contributing/architecture.md) and [glossary](docs/contributing/glossary.md) |
| Test ownership, local/CI checks, and verification boundaries | [Testing guide](docs/contributing/testing.md) |
| PR scope, squash-message requirements, compatibility review, and actionable findings | [Code-review policy](.agents/skills/code-review/SKILL.md) |
| Internal metadata, plan-report, and Summary interfaces | [Actions internals](docs/contributing/actions-internals.md) |
| This repository's PR controls, settings workflows, Environment, and App bindings | [Repository automation](docs/contributing/repository-automation.md) |
| Maintainer release preparation, publication, and recovery | [Release guide](docs/contributing/releases.md) |
| Why an architectural decision was accepted or superseded | [Architecture decision records](docs/adr/README.md) |
| Detailed behavior scenarios and their existing Go test mappings | [Behavior specifications](docs/features/README.md) |
| Proposed work, implementation progress, and release-specific changes | [Issues](https://github.com/daiksud/gh-settings/issues), PRs, and [release notes](https://github.com/daiksud/gh-settings/releases) |

### Keep the audience boundary clear

User documentation explains how to use the published interface and understand its effects. Keep operational constraints users need, such as direct-apply behavior, partial failure, supported runners, and credential requirements, alongside the relevant instructions.

Contributor documentation explains how the implementation works and how this project is developed and maintained. Internal report schemas, fixture wiring, this repository's credential names, and release publication procedures belong under `docs/contributing/`.

ADRs and behavior specifications remain at their existing paths so decision numbers and scenario links stay stable. Read ADR context as a historical record; update status and supersession links when decisions change, and describe the current implementation in the architecture guide. Keep proposed behavior clearly separate from supported behavior. Mark implemented but unreleased behavior explicitly, and identify which CLI or Action version can use it; source support does not make the currently pinned Action capable of that behavior.

### Keep each contract in one place

- Link to the detailed reference from quick starts and examples instead of copying its full explanation.
- Update user references and affected behavior specifications with a behavior change; do not require users to infer the contract from test code or ADRs.
- Keep `examples/github-actions/README.md` focused on its adjacent workflow files; the Action guide owns general setup and recovery.
- Use [Action metadata](action.yml), the [committed CLI version](action/cli-version), and workflow pins as the source for implementation/version facts. Avoid repeating release digests, rollout progress, or the active repository's deployment state across guides.
- Describe external GitHub settings as prerequisites or things to verify unless their current state has been checked; workflow YAML alone cannot establish Environment protection or live token revocation.

## Documentation checks

After installing the pinned tools, run from the repository root:

```sh
mise exec -- rumdl check --no-cache --deny-config-warnings README.md CONTRIBUTING.md AGENTS.md .agents .github/pull_request_template.md docs examples/github-actions
git diff --check
```

The existing Markdown check validates relative files and heading fragments. When moving a page or heading, update links in Markdown, frontmatter source references, and any plain-text commands or paths. When adding a root document, include it in the [CI Markdown check](.github/actions/commit-stage/action.yml).

## Releases

Merging a PR does not publish a release. Maintainers use the separately initiated [release procedure](docs/contributing/releases.md), which covers explicit version selection, tested binary identity, and immutable release recovery.
