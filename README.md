# gh-settings

A GitHub CLI extension for exporting, validating, planning, and applying the current repository's five basic settings. The published v0.3.0 release implements read-only `export`, local `validate`, and read-only `plan`. Current source also implements `apply`; it has not been released. GitHub Action delivery remains future work.

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

The published v0.2.0 release and current source print root `schemaVersion: 1` and `settings.repository` with `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki`. Releases before v0.2.0 used a flat YAML format. Export reads the checkout's GitHub.com `origin` remote; it does not use `GH_REPO` or require an existing `.github/settings.yml`. Export does not change repository settings or write a file. See [commands](docs/commands.md) and [supported settings](docs/settings.md) for details.

To save a copy, create `.github` first and redirect the output:

```sh
mkdir -p .github
gh settings export > .github/settings.yml
```

Shell redirection can truncate an existing destination even if export fails. Inspect the output before replacing an existing configuration you need to preserve.

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

The [configuration contract](docs/settings.md#configuration-semantics) explains the five supported fields, `schemaVersion`, grouping, value types, and migration from earlier flat files. Validation succeeds silently and reports errors on standard error. It does not read or change GitHub settings or edit the configuration file. See the [command behavior](docs/commands.md#validate-behavior) for its failure cases.

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

Successful comparison exits 0 with or without differences. Strings are quoted and escaped; omitted fields are unmanaged. The command reports no differences or no managed fields with a short message. It does not change the file or repository settings. See [plan behavior](docs/commands.md#plan-behavior). Plan is available in v0.3.0.

## Apply a saved configuration

With a source build, send declared values directly:

```sh
gh settings apply
```

Apply validates `.github/settings.yml` before contacting GitHub, then sends only declared fields to the checkout's origin repository. False and empty strings are explicit values; omitted fields stay outside the request. A declaration with no managed fields succeeds without origin or authentication. Successful updates print `Applied repository settings.` and exit 0.

Apply works independently of plan and does not fetch current settings. Remote changes to declared fields can be overwritten. Errors exit 1; an output failure can occur after an update succeeds, and there is no automatic rollback. See [apply behavior](docs/commands.md#apply-behavior). Apply is not in v0.3.0.

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
- [Supported settings and configuration semantics](docs/settings.md)
- [Commands and failure behavior](docs/commands.md)
- [Release design and future versioning](docs/releases.md)
