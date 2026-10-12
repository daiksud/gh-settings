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
go test -race -count=1 -vet=off ./...
go vet ./...
go build ./...
go mod verify
go mod tidy -diff
rumdl check --no-cache --deny-config-warnings README.md docs
```

The [PR workflow](.github/workflows/ci-pr.yml) follows Draft → commit-stage → Ready → acceptance-stage → `pass`, matching the current apex/agents control flow. It runs on opened, synchronized, reopened, and edited PRs. The [main workflow](.github/workflows/ci-push.yml) runs commit-stage → acceptance-stage → `pass` on pushes to `main`. Failure, cancellation, or skipped acceptance prevents a successful `pass` result. This CI does not merge, publish, or deploy.

Both workflows call [commit-stage](.github/actions/commit-stage/action.yml) for formatting, modules, vet, race tests, a single product build, and Markdown/local links. They transfer that same binary through an artifact, then call [acceptance-stage](.github/actions/acceptance-stage/action.yml) to verify its SHA-256 and launch it as a real process. The acceptance harness is compiled separately; the product is not rebuilt and the Go unit suite is not run again. Artifacts are retained for one day and downloaded only from the current workflow run.

PR-code validation jobs have read permissions, no supplied secrets, and checkout credentials are not persisted. Draft/Ready control jobs have write permissions but do not check out or execute PR code, composites, or artifacts. Normal fork tokens are read-only, so automatic PR-state changes can be blocked on forks; this workflow does not bypass that restriction.

### Local process acceptance

Build the product once, then run the small Go harness against that executable:

```sh
go build -o /tmp/gh-settings-acceptance .
go run ./tests/acceptance --binary /tmp/gh-settings-acceptance
```

The [harness](tests/acceptance/main.go) uses a local Unix-socket HTTP fixture and go-gh v2.16.2's `http_unix_socket` configuration in a temporary `GH_CONFIG_DIR`. Its dummy token never authenticates with GitHub. Checks cover YAML values, empty strings and false, GET null normalization, current-repository selection, exit codes, stderr, and failure paths. Temporary repositories/configuration are isolated and cleaned up. Product code needs no testing hook or test-only modification; Unix sockets are required.

### Codex approval control

The separate [approval workflow](.github/workflows/approve-codex-review.yml) uses the current apex/agents mechanism: a Codex summary must show both Code Review and Security Review as Completed, and Codex must have reacted 👍 to the PR body. It waits up to three minutes for that reaction, then may create a bot Approve. It runs trusted default-branch workflow code without checking out PR code.

This mechanism does not verify the reviewed HEAD or latest Ready-triggered review cycle, and can reuse older completion/reaction information. A successful workflow run alone does not prove that an approval was created. Inspect new Ready-triggered reviews separately. PR body edits also start the CI state sequence again, even when the HEAD has not changed.

The approval workflow becomes available for comment events only after it exists on the default branch. The repository currently has Actions review approvals disabled; enabling them requires a separate authorized settings change. No new token, secret, Ruleset, or branch protection is configured here. Making `pass` required is also a separate settings decision.

Markdown checks use rumdl 0.2.79 with the repository's [.rumdl.toml](.rumdl.toml). MD051 and MD057 validate local headings/fragments and file links; external HTTP availability is not a CI dependency. The CI installation pins the Linux wheel's version and SHA-256 in [requirements-ci.txt](.github/requirements-ci.txt). Use an installed rumdl 0.2.79 for the local command above.

The [export behavior specification](docs/features/export.feature.md) maps its user-visible scenarios to ordinary Go tests. No BDD runner is required.

- [Concept, design philosophy, and future direction](docs/concept.md)
- [Supported settings and configuration semantics](docs/settings.md)
- [Commands and failure behavior](docs/commands.md)
