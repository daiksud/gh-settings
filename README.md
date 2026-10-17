# gh-settings

Manage a GitHub repository's basic settings and topics as code, using a GitHub CLI extension or a [GitHub Action](docs/github-actions.md). Keep the intended values in `.github/settings.yml`, review differences with `plan`, and send declared values with `apply`.

Only declared settings are managed; omitted fields stay unchanged. The configuration belongs to one Git checkout, and that checkout's GitHub.com `origin` selects the repository.

## Install

Install [GitHub CLI](https://cli.github.com/) and Git. Authenticate with `gh auth login` if needed, then check your credentials with `gh auth status`.

```sh
gh extension install daiksud/gh-settings
gh settings --version
```

Published binaries support macOS on Apple Silicon (`arm64`) and Linux on x86-64 (`amd64`). See the [latest release](https://github.com/daiksud/gh-settings/releases/latest) for assets and checksums, or the [development guide](docs/contributing/development.md) to build from source.

Update with `gh extension upgrade gh-settings`, or uninstall with `gh extension remove gh-settings`.

## Quick start

Run these commands inside the Git checkout whose settings you want to manage, including from a subdirectory.

### 1. Export the current settings

Export saves `.github/settings.yml` at the checkout root and overwrites an existing regular file. It does not change GitHub settings and produces no stdout, so do not redirect it into the configuration file.

```sh
gh settings export
```

### 2. Edit the declaration

Edit the saved `.github/settings.yml`. Export includes every supported field and the complete topic set; remove fields you want to leave unmanaged. For example:

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

### 3. Validate and review differences

```sh
gh settings validate
gh settings plan
```

Validation is local and succeeds silently. Plan reads only managed resources, shows their differences, and exits successfully whether or not changes are found.

### 4. Apply the declaration

After reviewing the plan and configuration, apply with credentials that can administer the target repository:

```sh
gh settings apply
```

Apply sends declared values directly; it does not execute a saved plan or check for intervening changes. Repository settings are written before topics. If a later write fails, earlier updates remain applied, and there is no automatic rollback. See the [command reference](docs/commands.md) for authentication, output, and failure behavior.

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
      - uses: daiksud/gh-settings@fd2aef0efb15d6b4c4b19c45e47f5fa965c66b24 # v0.8.0
        with:
          command: validate
```

The Action supports `validate`, `plan`, and `apply`. Validate needs no token; plan takes an explicit read token; apply takes a target-scoped App installation token with Administration write. The caller owns approval, credential issuance and revocation, and concurrency.

Use the [Action guide](docs/github-actions.md) for the complete contract and setup, and the [PR/main workflow examples](examples/github-actions/README.md) for adoption instructions.

## Documentation

| I want to… | Read |
| --- | --- |
| Find a guide or reference | [Documentation index](docs/README.md) |
| Understand commands, targeting, output, and failures | [Command reference](docs/commands.md) |
| Write `.github/settings.yml` | [Configuration reference](docs/settings.md) |
| Automate validation, review, and apply | [GitHub Actions guide](docs/github-actions.md) |
| Change the code, tests, or documentation | [Contributing](CONTRIBUTING.md) |
