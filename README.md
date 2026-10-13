# gh-settings

A GitHub CLI extension for exporting and locally validating the current repository's five basic settings as YAML. The published release implements read-only `export`; this source also implements `validate`. `plan`, `apply`, and GitHub Action delivery remain future work.

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

The source version prints root `schemaVersion: 1` and `settings.repository` with `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki`. The published release still prints these fields at the YAML root. Export reads the checkout's GitHub.com `origin` remote; it does not use `GH_REPO` or require an existing `.github/settings.yml`. Export does not change repository settings or write a file. See [commands](docs/commands.md) and [supported settings](docs/settings.md) for details.

To save a copy, create `.github` first and redirect the output:

```sh
mkdir -p .github
gh settings export > .github/settings.yml
```

Shell redirection can truncate an existing destination even if export fails. Inspect the output before replacing an existing configuration you need to preserve.

## Build and run from source

To build from source, use Go 1.25 or later. The [development guide](docs/development.md) covers pinned tools and local checks.

```sh
go build -o gh-settings .
gh extension install .
```

To check a saved configuration without contacting GitHub, run this from the repository root or a subdirectory:

```sh
gh settings validate
```

`validate` reads the Git root's `.github/settings.yml` and accepts the five supported fields under `settings.repository` with their string or boolean types. For example, this declares one field:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
```

It rejects missing or unsupported schema versions and duplicate or unknown keys. A versioned document with empty mappings or partial declarations is valid; false and empty strings remain valid values. It succeeds silently and reports errors on standard error. It does not read or change GitHub settings or edit the configuration file. Explicit null fails the five fields' string/boolean type check; [ADR 0019](docs/adr/0019-reject-explicit-null.md) records this decision. Earlier flat files need `schemaVersion: 1` added and their fields moved under `settings.repository`; they fail validation without that migration.

`gh settings --version` prints the build version (`dev` for a source build).

The current source implementation accepts GitHub.com HTTPS and SSH `origin` URLs. It outputs `schemaVersion: 1` and all five fields under `settings.repository`, including false and empty strings. GitHub's null description/homepage values become quoted empty strings, never YAML null. Missing response fields and null booleans are errors. Errors go to standard error with a nonzero exit status.

## Development checks

See the [development guide](docs/development.md) for tool setup and local checks, [CI stages](docs/development.md#ci-validation), and [Codex approval control](docs/development.md#codex-approval-control).

The [export](docs/features/export.feature.md) and [validate](docs/features/validate.feature.md) behavior specifications map their user-visible scenarios to ordinary Go tests. No BDD runner is required.

## Maintainer release

The [manual release workflow](.github/workflows/release.yml) accepts a `vMAJOR.MINOR.PATCH` version when run from `main`. It freezes the latest main SHA at start, builds Linux amd64 and macOS arm64 on matching runners, runs commit and process acceptance checks, then stages a draft and publishes it only after validating the tested assets. It publishes raw GitHub CLI extension binaries, `SHA256SUMS`, and release notes. The workflow is not a scheduled or automatic release; merging its definition does not start a release.

To start a release after reviewing the version and main commit:

```sh
gh workflow run release.yml --repo daiksud/gh-settings --ref main -f version=v1.2.3
```

The workflow never moves an existing tag or replaces an existing release asset. Re-running the same version resolves the then-current main again; recovery succeeds only if its SHA and freshly tested binary hashes match the tag and release manifest. If main has advanced, the old workflow run's frozen SHA remains available in its logs, but a new run will stop at the existing tag. Keep incomplete drafts for inspection; do not delete or replace them automatically. See the [release design](docs/releases.md) for identity and recovery details.

- [Concept, design philosophy, and future direction](docs/concept.md)
- [Supported settings and configuration semantics](docs/settings.md)
- [Commands and failure behavior](docs/commands.md)
- [Release design and future versioning](docs/releases.md)
