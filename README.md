# gh-settings

Manage a GitHub repository's settings as code, using a GitHub CLI extension or a [GitHub Action](docs/github-actions.md). Keep the intended values in `.github/settings.yml`, review differences with `plan`, and send declared values with `apply`.

The configuration belongs to one Git checkout, and that checkout's GitHub.com `origin` selects the repository. Omitted resources stay unmanaged; declared collections control their complete membership.

## Install

Install [GitHub CLI](https://cli.github.com/) and Git. Authenticate with `gh auth login` if needed, then check your credentials with `gh auth status`.

```sh
gh extension install daiksud/gh-settings
gh settings --version
```

Published binaries support macOS on Apple Silicon (`arm64`) and Linux on x86-64 (`amd64`). See the [latest release](https://github.com/daiksud/gh-settings/releases/latest) for assets and checksums, or the [development guide](docs/contributing/development.md) to build from source.

Update with `gh extension upgrade gh-settings`, or uninstall with `gh extension remove gh-settings`.

The published [v0.17.0 CLI and Action](https://github.com/daiksud/gh-settings/releases/tag/v0.17.0) support repository settings including Discussions, merge methods, automatic branch deletion, squash/normal merge commit defaults, auto-merge and branch-update options, topics and labels. The Action examples below select this release. See the [label configuration contract](docs/settings.md#labels) before managing that collection.

[v0.10.0](https://github.com/daiksud/gh-settings/releases/tag/v0.10.0) adds three [repository merge-method booleans](docs/features/merge-methods.feature.md). The pinned examples include them; v0.9.0 and older packages do not accept these keys.

[v0.11.0](https://github.com/daiksud/gh-settings/releases/tag/v0.11.0) adds the [`delete_branch_on_merge` setting](docs/features/delete-branch-on-merge.feature.md). The pinned v0.17.0 package includes this key; publication does not change the repository's live policy.

[v0.12.0](https://github.com/daiksud/gh-settings/releases/tag/v0.12.0) adds [squash commit title/body defaults](docs/features/squash-commit-defaults.feature.md). The pinned v0.17.0 package includes these keys; publication does not change live policy.

[v0.13.0](https://github.com/daiksud/gh-settings/releases/tag/v0.13.0) adds [`allow_auto_merge` and `allow_update_branch`](docs/features/pull-request-options.feature.md). The pinned v0.17.0 package includes these keys; publication does not change live policy.

[v0.14.0](https://github.com/daiksud/gh-settings/releases/tag/v0.14.0) adds [merge commit title/body defaults](docs/features/merge-commit-defaults.feature.md). The pinned v0.17.0 package includes these keys; publication does not change live policy.

[v0.15.0](https://github.com/daiksud/gh-settings/releases/tag/v0.15.0) adds [`has_discussions`](docs/features/discussions.feature.md). The pinned package includes this optional boolean; v0.14.0 and older packages reject it.

## Quick start

Run these commands inside the Git checkout whose settings you want to manage, including from a subdirectory.

### 1. Export the current settings

Export saves `.github/settings.yml` at the checkout root and overwrites an existing regular file. It does not change GitHub settings and produces no stdout, so do not redirect it into the configuration file.

```sh
gh settings export
```

### 2. Edit the declaration

Edit the saved `.github/settings.yml`. Export includes every supported resource; remove resources or fields you want to leave unmanaged. For example:

```yaml
schemaVersion: 1
settings:
  repository:
    description: "My project"
    has_issues: true
  topics:
    names:
      - github-cli
      - settings-as-code
```

`schemaVersion: 1` is required. Omission leaves a field unmanaged; `false` and empty strings are explicit values. Declaring `topics.names` manages the complete topic set, and `names: []` clears it. See the [configuration reference](docs/settings.md) for supported fields, types, and validation rules.

A source build also exports the complete label collection. A declared label list is authoritative: undeclared remote labels are deleted, and `labels: []` deletes every label and its issue/PR associations. Omit `settings.labels` to leave labels unmanaged. The [configuration reference](docs/settings.md#labels) explains retained attributes and name identity.

### 3. Validate and review differences

```sh
gh settings validate
gh settings plan
```

Validation is local and succeeds silently. Plan reads only managed resources, shows their differences, and exits successfully whether or not changes are found.

### 4. Apply the declaration

After reviewing the plan and configuration, apply with credentials that can write the declared resources; see [authentication and permissions](docs/commands.md#repository-and-authentication):

```sh
gh settings apply
```

Apply does not execute a saved plan or protect against intervening changes. It writes declared repository settings, then topics, directly. In current source, declared labels are then read afresh and reconciled with creates, updates, and deletes. A later read or write failure leaves earlier successful updates applied, with no automatic rollback. See the [command reference](docs/commands.md) for authentication, output, and failure behavior.

## Use as a GitHub Action

Commit `.github/settings.yml`, then validate it in a GitHub-hosted Ubuntu 24.04 x64 job:

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
      - uses: daiksud/gh-settings@cccd06a151c2dce3b9eb71520d3c01b4c73ac92f # v0.17.0
        with:
          command: validate
```

The pinned v0.17.0 Action supports `validate`, `plan`, and `apply` for repository settings, topics and labels. Validate needs no token; plan takes an explicit read token; repository/topic apply takes a target-scoped App installation token with Administration write; label mutations additionally require Issues or Pull requests write. The caller owns approval, credential issuance and revocation, and concurrency.

Before enabling apply, follow the [approval and required-check guidance](docs/github-actions.md#protect-the-apply-path) to protect both `main` and the job that receives the write credential. Use the [Action guide](docs/github-actions.md) for the complete contract and setup, and the [PR/main workflow examples](examples/github-actions/README.md) for adoption instructions.

## Documentation

| I want to… | Read |
| --- | --- |
| Find a guide or reference | [Documentation index](docs/README.md) |
| Understand commands, targeting, output, and failures | [Command reference](docs/commands.md) |
| Write `.github/settings.yml` | [Configuration reference](docs/settings.md) |
| Automate validation, review, and apply | [GitHub Actions guide](docs/github-actions.md) |
| Change the code, tests, or documentation | [Contributing](CONTRIBUTING.md) |
