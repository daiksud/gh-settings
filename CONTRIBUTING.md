# Contributing to gh-settings

This guide is for people changing gh-settings itself: its code, tests, documentation, or repository automation. To use the published CLI or Action in your own repository, start with the [README](README.md) and [user documentation](docs/README.md#for-users).

## Getting started

1. Install the development tools, build the CLI and run it locally with the [setup guide](docs/development/setup.md).
2. Find the relevant [public contract](docs/README.md#for-users) and [behavior specification](docs/development/features/README.md). For a new setting or a change to an existing interface, check the [development policies](docs/development/policies.md).
3. Locate the code that owns the change in the [architecture guide](docs/development/architecture.md). The [model glossary](docs/development/glossary.md) explains terms used across packages.
4. Use the [testing guide](docs/development/testing.md) to choose checks for the affected behavior.
5. Submit a focused PR to `main`. Describe the problem, change and verification, and address findings under the [code-review policy](.agents/skills/code-review/SKILL.md). See [PR validation](docs/development/pr-validation.md) for the checks and [repository automation](docs/development/repository-automation.md#codex-approval-control) for how an approving review is recorded.

## Repository directory guide

| Directory | What you can change here |
| --- | --- |
| [.agents](.agents/README.md) | Review instructions and skill metadata for agents working on gh-settings. |
| [.github](.github/CONTENTS.md) | This repository's settings declaration, workflows, CI Actions and contribution templates. |
| [action](action/README.md) | The installer that the public GitHub Action uses to fetch and verify the CLI. |
| [docs](docs/README.md) | User guides, public references, behavior specifications and maintainer documentation. |
| [examples](examples/README.md) | Workflows that users can copy into the repository they manage. |
| [internal](internal/README.md) | Go packages that implement settings management, the CLI and Action reporting. |
| [scripts](scripts/README.md) | Tools for generating release notes and publishing verified binaries. |
| [tests](tests/README.md) | CLI and Action acceptance tests, shared fixtures and offline release checks. |

## Issues and pull requests

When [opening an issue](https://github.com/daiksud/gh-settings/issues/new/choose), choose the template that fits the work:

- **Bug report**: describe actual and expected behavior, a minimal reproduction with the relevant starting state, and the exact CLI version or Action ref. Add environment details when they affect reproduction.
- **Feature or improvement**: explain the current obstacle and an observable result that would solve it, with a concrete usage example.
- **Task**: identify the maintenance problem and the concrete result that completes it, including the affected area and how to check the result.

Blank issues remain available for questions or work that does not fit these forms. Use the same information structure when submitting through the web, CLI, or API. Add constraints or references to the relevant answer only when they affect the work.

### Branch names and change classification

Use a topic branch to identify the work. Choose the Conventional Commits type and any compatibility markers from the final effect on CLI or public Action users. Correct the PR title directly if that classification changes. Keep the code, tests, documentation and wiring for one behavior together. The [code-review policy](.agents/skills/code-review/SKILL.md) defines PR scope, breaking markers and squash requirements.

### Write PR bodies for the Git history

The reviewed title and body become the squash commit. Use the [PR template](.github/pull_request_template.md) to write a message that a maintainer can understand from `git log`.

- **Why** the change is needed. Describe the concrete problem and its impact. Include enough background to explain it even if the linked issue becomes unavailable.
- **What changed** for the caller or maintainer. Explain how the change solves the problem and why you chose the approach. Include compatibility or operational effects when relevant.
- **Verification** of the result. Name the behavior you checked, how you checked it, what you observed and any gap that limits your conclusion.

Use `Closes #123` when the PR completes the issue. Put review requests, temporary status, and detailed logs in comments or check records. Before merging, remove template comments, reconcile the message with the final diff and confirm that the squash commit contains the reviewed title and body.

## Where documentation belongs

Start with the [documentation index](docs/README.md) to find the page that owns the subject.

- For someone using gh-settings, put the exact CLI, Action or YAML contract in `docs/references/`. Use the tutorial for a first successful run, a guide for an operational task and the concepts page for how settings management works. Keep copyable workflows and instructions specific to those files together in [examples](examples/github-actions/README.md).
- For someone changing gh-settings, put setup, implementation, testing and maintenance guidance under `docs/development/`. Record current project constraints in the [development policies](docs/development/policies.md).
- When observable product behavior changes, update its public reference and [feature specification](docs/development/features/README.md) together. Write the expected result and identify the tests that exercise it. Describe internal workflow or release checks in the corresponding maintainer guide.
- Record proposed work in issues and the reasons for a particular change in the PR. Release-specific changes belong in [release notes](https://github.com/daiksud/gh-settings/releases).

## Documentation checks

After installing the pinned tools, run from the repository root:

```sh
mise exec -- bun run lint:markdown
git diff --check
```

The Markdown check validates relative file links and heading fragments. When moving a page or heading, update those links, frontmatter source paths and plain-text references. If you add a root document, include it in the [lint:markdown script](package.json) used by CI.

When changing the [settings reference](docs/references/settings/README.md), run the tests that parse its YAML examples and check permission matrix coverage against the public configuration fields.

```sh
mise exec -- go test -count=1 ./internal/workspace -run 'TestSettings(ReferenceYAMLExamples|PermissionMatrixCoversPublicCodec)$'
```

## Releases

Publication requires an explicit release request after merge. Follow the [maintainer release guide](docs/development/releases.md) for version calculation, acceptance, immutable publication and recovery.
