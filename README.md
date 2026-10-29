# gh-settings

Manage a GitHub repository's settings as code, using a GitHub CLI extension or a [GitHub Action](docs/github-actions.md). Keep the intended values in `.github/settings.yml`, review differences with `plan`, and send declared values with `apply`.

The extension uses your existing `gh` authentication. Actions use one GitHub App with explicit, short-lived installation tokens scoped per job/target, without a required PAT or personal-login combination. Validation needs no API authentication; nonmutating plan may still require write grants for complete reads. See the [authentication and trust policy](docs/github-actions.md#authentication-and-app-permissions), including trusted-base PR data-only planning and protected-main apply.

The configuration belongs to one Git checkout, and that checkout's GitHub.com `origin` selects the repository. Omitted resources stay unmanaged. Topics manage a complete set; Labels and Rulesets own only listed objects; Environment omissions remain API-controlled.

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

Edit the saved `.github/settings.yml`. The default export covers the standard resources; other supported groups require explicit export options. See the [export reference](docs/commands.md#export-behavior). Remove any resources or fields you do not want to manage. For example:

```yaml
version: daiksud/gh-settings@v0
repository:
  about:
    description: My project
    topics: [github-cli, settings-as-code]
  settings:
    general:
      features: {issues: true}
```

Current source requires the exact `version` string above and uses UI wording in snake_case. Omission, false, empty strings and empty lists retain distinct meanings; `about.topics: []` clears the topic set. Old envelopes and API-shaped aliases are rejected without migration. See the [configuration reference](docs/settings.md) for every supported path and ownership rule.

This YAML requires a CLI release containing the activation change or a current-source build. Published packages and the pinned Action below keep their own contracts; installing an older release does not enable it. Out-of-UI normalization (#480) and complete reporting integration (#481) remain separate work.

Labels and repository Rulesets select existing objects by numeric ID. Only listed objects are managed; an omitted object or empty list does not delete it. Entries without IDs are create-only. Updating a listed Ruleset can change branch protection, reviews and checks; read the [Labels](docs/settings.md#labels) and [Rulesets](docs/settings.md#rulesets) contracts before applying.

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

Apply does not execute a saved plan or protect against intervening changes.
It updates managed resources directly; a later failure can leave earlier remote
writes and recorded IDs in place. There is no automatic rollback. The
[command reference](docs/commands.md#apply-behavior) owns operation ordering,
ID writeback, permissions, and failure recovery.

## Use as a GitHub Action

The pinned example below uses its published package contract. It cannot consume the new YAML until a release containing that change is selected. See the [package boundary](docs/github-actions.md#published-package-and-consumer-pins); no pin is upgraded here.

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
      - uses: daiksud/gh-settings@eb8795dca8bb42376ab575f646ffa11bfe7775c8 # v0.21.0
        with:
          command: validate
```

The example pins the immutable v0.21.0 Action, which supports `validate`, `plan`, and `apply` for its published resource set. Validate needs no token; plan/apply take an explicit installation token from the same GitHub App with permissions for each declared resource. The caller owns trusted workflow execution, approval, token issuance/revocation and concurrency. Never run PR code with privileged credentials; use the [data-only PR example](examples/github-actions/README.md).

Before enabling apply, follow the [approval and required-check guidance](docs/github-actions.md#protect-the-apply-path) to protect both `main` and the job that receives the write credential. Use the [Action guide](docs/github-actions.md) for the complete contract and setup, and the [PR/main workflow examples](examples/github-actions/README.md) for adoption instructions.

## Documentation

| I want to… | Read |
| --- | --- |
| Find a guide or reference | [Documentation index](docs/README.md) |
| Understand commands, targeting, output, and failures | [Command reference](docs/commands.md) |
| Write `.github/settings.yml` | [Configuration reference](docs/settings.md) |
| Automate validation, review, and apply | [GitHub Actions guide](docs/github-actions.md) |
| Change the code, tests, or documentation | [Contributing](CONTRIBUTING.md) |
