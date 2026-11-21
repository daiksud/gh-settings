# gh-settings

Manage a GitHub repository's settings from `.github/settings.yml` with the GitHub CLI extension or the [GitHub Action](docs/guides/github-actions.md). You choose which settings to declare, preview differences with `plan`, and apply the values you reviewed.

The project targets active, personally owned public repositories. Check the [settings reference](docs/references/settings/README.md) for the controls available in current source and the [support policy](docs/references/settings/scope.md#feature-adoption-scope) for v1 scope and exceptions.

## Install

Install [GitHub CLI](https://cli.github.com/) and Git. Check your login with `gh auth status`. If needed, sign in with `gh auth login`. Then install the extension and check its version.

```sh
gh extension install daiksud/gh-settings
gh settings --version
```

Published binaries support macOS on Apple Silicon (`arm64`) and Linux on x86-64 (`amd64`). [Release assets and checksums](https://github.com/daiksud/gh-settings/releases/latest) and [source build instructions](docs/development/setup.md#build-and-run-from-source) are available. To update, run `gh extension upgrade gh-settings`. To uninstall, run `gh extension remove gh-settings`.

## Quick start

In a checkout of the repository you want to manage, export the current settings. The checkout's GitHub.com `origin` selects the target. Export writes to `.github/settings.yml` and overwrites an existing regular file, so save any local edits first.

```sh
gh settings export
```

Open `.github/settings.yml`. Keep the fields you want gh-settings to manage and edit their values. For example, change `repository.about.description` to update the text in the repository’s **About** section. Validate the file, then preview the changes.

```sh
gh settings validate
gh settings plan
```

When the plan shows the changes you intend, `gh settings apply` writes the declared values to GitHub. A later failure can leave earlier writes in place. For a worked example, [manage the description of a practice repository](docs/tutorials.md).

## Use as a GitHub Action

This pull request workflow checks the configuration with the published v0.44.0 Action. Checkout uses Contents read access. Local validation needs no settings API credential.

```yaml
name: Validate repository settings
on: pull_request
permissions:
  contents: read
jobs:
  validate:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          repository: ${{ github.repository }}
          ref: ${{ github.sha }}
          persist-credentials: false
      - uses: daiksud/gh-settings@948873d260be87aecc3d0597f715a8c09e1347b9 # v0.44.0
        with:
          command: validate
```

For export, plan, and apply, the caller supplies a short-lived GitHub App installation token scoped to the target and operation. Follow the [GitHub Actions guide](docs/guides/github-actions.md) to plan with trusted workflow code, review changes, and protect the job that can apply them. The [workflow examples](examples/github-actions/README.md) include the PR and main workflows.

## Documentation

| Task | Start here |
| --- | --- |
| Understand which checkout and settings a command manages | [How gh-settings manages settings](docs/concepts.md) |
| Look up command syntax, output, or failure behavior | [Command reference](docs/references/commands.md) |
| Find a YAML field or its required GitHub App permissions | [Settings reference](docs/references/settings/README.md) · [Permission matrix](docs/references/settings/permissions.md) |
| Find a guide, reference, or feature specification | [Documentation index](docs/README.md) |
| Change or maintain gh-settings | [Contributing](CONTRIBUTING.md) |
