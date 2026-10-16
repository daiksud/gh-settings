# gh-settings

A GitHub CLI extension for exporting, validating, planning, and applying the current repository's five basic settings. Export saves `.github/settings.yml`; omitted fields remain unmanaged. A [public composite GitHub Action](docs/github-actions.md) exposes validate, plan and direct apply; the caller owns approval and credentials.

This checkout documents current **unreleased v0.7.0** source. The latest published package is [v0.6.1](https://github.com/daiksud/gh-settings/releases/tag/v0.6.1); installation and pinned consumer examples use that release. Source-only plan no-op/origin and Go Action changes below do not publish it or change active consumer pins. See the [released package boundary](docs/github-actions.md#published-package-and-consumer-pins) before adopting an Action revision.

## Install

Install [GitHub CLI](https://cli.github.com/) and Git, then authenticate to GitHub.com with `gh auth login` if needed. Check your authentication with `gh auth status`. The published extension has binaries for macOS on Apple Silicon (`arm64`) and Linux on x86-64 (`amd64`); see the [latest release](https://github.com/daiksud/gh-settings/releases/latest) for its assets and checksums.

```sh
gh extension install daiksud/gh-settings
gh settings --version
```

To update or remove the installed extension, use `gh extension upgrade gh-settings` or `gh extension remove gh-settings`. See the [GitHub CLI extension manual](https://cli.github.com/manual/gh_extension) for extension management.

## Quick start

In the Git checkout whose settings you want to read, run:

```sh
gh settings export
```

In current source, export saves root `schemaVersion: 1` and `settings.repository` with `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki` to `.github/settings.yml` at the Git checkout root, including from subdirectories. It creates a missing `.github` directory and succeeds silently. Export reads that checkout's GitHub.com `origin`, ignores `GH_REPO`, and does not change GitHub settings. See [commands](docs/commands.md) and [supported settings](docs/settings.md) for details.

Export always overwrites an existing regular configuration file after reading the settings and serializing the complete YAML. A write failure reports an error and can leave the destination truncated or partially written; use Git to restore tracked configuration. Symlinks at `.github` or `settings.yml`, including dangling links, and non-regular destinations are rejected.

Export produces no stdout. Read the saved file after success rather than redirecting the command; shell redirection can truncate a destination before export runs:

```sh
cat .github/settings.yml
```

No output flags are supported.

## Validate a saved configuration

To check a saved configuration without contacting GitHub, run this from the repository root or a subdirectory:

```sh
gh settings validate
```

`validate` reads the Git root's `.github/settings.yml`. For example, this declares one repository field:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
```

The [configuration contract](docs/settings.md#configuration-semantics) explains the five supported fields, `schemaVersion`, grouping and value types. Validation succeeds silently and reports errors on standard error. It does not read or change GitHub settings or edit the configuration file. See the [command behavior](docs/commands.md#validate-behavior) for its failure cases.

## Plan a saved configuration

Edit `.github/settings.yml`, validate it, then inspect differences:

```sh
gh settings validate
gh settings plan
```

Plan validates before reading GitHub and compares only declared fields. For the example above, when issues are currently enabled, it prints:

```text
settings.repository.has_issues: true → false
```

Successful comparison exits 0 with or without differences. Strings are quoted and escaped; omitted fields are unmanaged. The command reports no differences or no managed fields with a short message. An empty declaration succeeds locally without origin or authentication in current source. It does not change the file or repository settings. See [plan behavior](docs/commands.md#plan-behavior).

## Apply a saved configuration

Send declared values directly:

```sh
gh settings apply
```

Apply validates `.github/settings.yml` before contacting GitHub, then sends only declared fields to the checkout's origin repository. False and empty strings are explicit values; omitted fields stay outside the request. A declaration with no managed fields succeeds without origin or authentication. Successful updates print `Applied repository settings.` and exit 0.

Apply works independently of plan and does not fetch current settings. Remote changes to declared fields can be overwritten. Errors exit 1; an output failure can occur after an update succeeds, and there is no automatic rollback. See [apply behavior](docs/commands.md#apply-behavior).

## Use as a GitHub Action

Commit `.github/settings.yml` in the configuration repository, then use a fresh GitHub-hosted Ubuntu 24.04 x64 job. Checkout belongs at `GITHUB_WORKSPACE`, with a GitHub `origin` and no persisted credentials. This credential-free validation example uses the reviewed full Action SHA:

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
      - uses: daiksud/gh-settings@9d01d3258d9f42239b2daeb1ac26bc532cd56105 # Action revision; CLI v0.6.1
        id: settings
        with:
          command: validate
```

`command` is exactly `validate`, `plan`, or `apply`. Validate needs no token; plan requires an explicit read token; apply requires a target-scoped App installation token with Administration write. The caller owns its Environment/approval, token mint/revoke and concurrency. Use the [inert PR/main examples](examples/github-actions/README.md) for read-only review followed by separately approved apply; do not give privileged credentials to PR code.

Successful calls expose `config-sha`, `target`, `cli-version` and `cli-sha256` through `steps.settings.outputs`. This snippet pins the published v0.6.1 Action and CLI, including that package's Python 3 requirement. Current source selects **unreleased v0.7.0** in `action/cli-version` and uses the verified executable's Go metadata and typed plan/Summary entries, without Python. It requires a directly configured GitHub.com HTTPS, git@ SCP or full SSH origin; rewrite-only aliases and username-less SCP are unsupported. Candidate CI proves source integration without published v0.7.0 assets. The released snippet and active consumer pins retain their original implementation. See the [complete contract, released profile and adoption boundary](docs/github-actions.md).

## Build and run from source

To build from source, use Go 1.25 or later. The [development guide](docs/development.md) covers pinned tools and local checks.

```sh
go build -o gh-settings .
gh extension install .
```

`gh settings --version` prints the build version (`dev` for a source build).

## Development checks

See the [development guide](docs/development.md) for tool setup and local checks, [CI stages](docs/development.md#ci-validation), and [Codex approval control](docs/development.md#codex-approval-control).

The [export](docs/features/export.feature.md), [validate](docs/features/validate.feature.md), [plan](docs/features/plan.feature.md), and [apply](docs/features/apply.feature.md) behavior specifications map their user-visible scenarios to ordinary Go tests. No BDD runner is required.

## Maintainer release

Releases require an explicit run of the manual workflow from `main`; merging code does not publish a release. The workflow tests platform binaries before publishing them with checksums and release notes. See the [maintainer release procedure](docs/releases.md#start-a-manual-release) for initiation, identity checks, and recovery.

- [Concept, design philosophy, and future direction](docs/concept.md)
- [GitHub Actions setup, review, and recovery](docs/github-actions.md)
- [Supported settings and configuration semantics](docs/settings.md)
- [Commands and failure behavior](docs/commands.md)
- [Release design and future versioning](docs/releases.md)
