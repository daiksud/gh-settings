# gh-settings

A GitHub CLI extension for exporting the current repository's five basic settings as YAML. Only `export` is implemented; `validate`, `plan`, `apply`, and GitHub Action delivery remain future work.

## Build and run from source

Requires Go 1.25 or later, Git, and GitHub CLI with existing authentication for the repository. No release binaries are published by this change.

```sh
go build -o gh-settings .
gh extension install .
```

From the repository whose settings you want to read:

```sh
gh settings export
```

The current implementation selects the current Git checkout's `origin` remote, accepts GitHub.com HTTPS and SSH URLs, and ignores `GH_REPO`. A configuration file is not required to bootstrap export. The command performs a read only and writes exactly `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki` as a root YAML mapping to standard output, including false and empty strings. GitHub's null description/homepage values are exported as quoted empty strings, never YAML null. Missing response fields and null booleans are errors. Errors go to standard error with a nonzero exit status.

To save the output explicitly, redirect it after creating `.github`:

```sh
mkdir -p .github
gh settings export > .github/settings.yml
```

Shell redirection can truncate an existing destination even if export fails. Inspect the output before replacing an existing configuration you need to preserve. `export` itself does not write files or change GitHub settings.

## Development checks

```sh
go test ./...
go vet ./...
go build ./...
```

The [export behavior specification](docs/features/export.feature.md) maps its user-visible scenarios to ordinary Go tests. No BDD runner is required.

- [Concept, design philosophy, and future direction](docs/concept.md)
- [Supported settings and configuration semantics](docs/settings.md)
- [Commands and failure behavior](docs/commands.md)
