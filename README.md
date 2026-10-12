# gh-settings

A GitHub CLI extension for exporting the current repository's five basic settings as YAML. Only `export` is implemented; `validate`, `plan`, `apply`, and GitHub Action delivery remain future work.

## Build and run from source

Requires Go 1.25 or later, Git, and GitHub CLI with existing authentication for the repository. For development, install [mise](https://mise.jdx.dev/getting-started.html) and run `mise install` to get the exact tool versions in [mise.toml](mise.toml), including Go 1.26.0.

```sh
go build -o gh-settings .
gh extension install .
```

From the repository whose settings you want to read:

```sh
gh settings export
```

`gh settings --version` prints the build version (`dev` for a source build).

The current implementation selects the current Git checkout's `origin` remote, accepts GitHub.com HTTPS and SSH URLs, and ignores `GH_REPO`. A configuration file is not required to bootstrap export. The command performs a read only and writes exactly `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki` as a root YAML mapping to standard output, including false and empty strings. GitHub's null description/homepage values are exported as quoted empty strings, never YAML null. Missing response fields and null booleans are errors. Errors go to standard error with a nonzero exit status.

To save the output explicitly, redirect it after creating `.github`:

```sh
mkdir -p .github
gh settings export > .github/settings.yml
```

Shell redirection can truncate an existing destination even if export fails. Inspect the output before replacing an existing configuration you need to preserve. `export` itself does not write files or change GitHub settings.

## Development checks

Install the tools declared in [mise.toml](mise.toml), then run checks through mise so the local versions match CI:

```sh
mise install
mise exec -- go test -race -count=1 -vet=off ./...
mise exec -- go vet ./...
mise exec -- go build ./...
mise exec -- go mod verify
mise exec -- go mod tidy -diff
mise exec -- actionlint -shellcheck shellcheck .github/workflows/*.yml
mise exec -- shellcheck scripts/publish-release.sh tests/release.sh
mise exec -- bash tests/release.sh
mise exec -- rumdl check --no-cache --deny-config-warnings README.md docs
```

The [PR workflow](.github/workflows/ci-pr.yml) follows Draft → commit-stage → Ready → acceptance-stage → `pass`, matching the current apex/agents control flow. It runs on opened, synchronized, reopened, and edited PRs. The [main workflow](.github/workflows/ci-push.yml) runs commit-stage → acceptance-stage → `pass` on pushes to `main`. Failure, cancellation, or skipped acceptance prevents a successful `pass` result. This CI does not merge, publish, or deploy.

Both workflows call [commit-stage](.github/actions/commit-stage/action.yml) for formatting, modules, vet, race tests, GitHub workflow/shell checks, a single product build, and Markdown/local links. CI installs the same pinned development tools from [mise.toml](mise.toml) with the official mise Action. The acceptance stage installs only Go from that configuration. Tool caches are disabled so PR jobs do not supply installed executables to later runs. They transfer that same binary through an artifact, then call [acceptance-stage](.github/actions/acceptance-stage/action.yml) to verify its SHA-256 and launch it as a real process. The acceptance harness is compiled separately; the product is not rebuilt and the Go unit suite is not run again. Artifacts are retained for one day and downloaded only from the current workflow run.

PR-code validation jobs have read permissions, no supplied secrets, and checkout credentials are not persisted. Draft/Ready control jobs have write permissions but do not check out or execute PR code, composites, or artifacts. Normal fork tokens are read-only, so automatic PR-state changes can be blocked on forks; this workflow does not bypass that restriction.

### Local process acceptance

Build the product once, then run the small Go harness against that executable:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go run ./tests/acceptance --binary /tmp/gh-settings-acceptance
```

The [harness](tests/acceptance/main.go) uses a local Unix-socket HTTP fixture and go-gh v2.16.2's `http_unix_socket` configuration in a temporary `GH_CONFIG_DIR`. Its dummy token never authenticates with GitHub. Checks cover YAML values, empty strings and false, GET null normalization, current-repository selection, exit codes, stderr, and failure paths. Temporary repositories/configuration are isolated and cleaned up. Product code needs no testing hook or test-only modification; Unix sockets are required.

### Codex approval control

The separate [approval workflow](.github/workflows/approve-codex-review.yml) uses the current apex/agents mechanism: a Codex summary must show both Code Review and Security Review as Completed, and Codex must have reacted 👍 to the PR body. It waits up to three minutes for that reaction, then may create a bot Approve. It runs trusted default-branch workflow code without checking out PR code.

This mechanism does not verify the reviewed HEAD or latest Ready-triggered review cycle, and can reuse older completion/reaction information. A successful workflow run alone does not prove that an approval was created. Inspect new Ready-triggered reviews separately. PR body edits also start the CI state sequence again, even when the HEAD has not changed.

The approval workflow becomes available for comment events only after it exists on the default branch. The repository currently has Actions review approvals disabled; enabling them requires a separate authorized settings change. No new token, secret, Ruleset, or branch protection is configured here. Making `pass` required is also a separate settings decision.

Markdown checks use the rumdl version pinned in [mise.toml](mise.toml) and the rules in [.rumdl.toml](.rumdl.toml). MD051 and MD057 validate local headings/fragments and file links; external HTTP availability is not a CI dependency. mise resolves rumdl, actionlint, and ShellCheck through its standard aqua registry. Go module library dependencies remain in `go.mod` and `go.sum`.

The [export behavior specification](docs/features/export.feature.md) maps its user-visible scenarios to ordinary Go tests. No BDD runner is required.

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
