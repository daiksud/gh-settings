# Local development

Use Go 1.25 or later and install [mise](https://mise.jdx.dev/getting-started.html). From the repository root, use the commands below to install the exact tool versions in [mise.toml](../mise.toml), including Go 1.26.0, and run checks through mise so the local versions match CI.

## Development checks

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

## Local process acceptance

Build the product once, then run the tagged Go acceptance tests against that executable:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The [acceptance tests](../tests/acceptance/acceptance_test.go) require an explicit existing binary path and use a local Unix-socket HTTP fixture and go-gh v2.16.2's `http_unix_socket` configuration in a temporary `GH_CONFIG_DIR`. Their dummy token never authenticates with GitHub. Checks cover YAML values, empty strings and false, GET null normalization, current-repository selection, exit codes, stderr, and failure paths. Temporary repositories/configuration are isolated and cleaned up. Product code needs no testing hook or test-only modification; Unix sockets are required.

## CI validation

The [PR workflow](../.github/workflows/ci-pr.yml) follows Draft → commit-stage → Ready → acceptance-stage → `pass`. It runs on opened, synchronized, reopened, and edited PRs. The [main workflow](../.github/workflows/ci-push.yml) runs commit-stage → acceptance-stage → `pass` on pushes to `main`. Failure, cancellation, or skipped acceptance prevents a successful `pass` result. This CI does not merge, publish, or deploy.

Both workflows call [commit-stage](../.github/actions/commit-stage/action.yml) for formatting, modules, vet, race tests, GitHub workflow/shell checks, a single product build, and Markdown/local links. CI installs the same pinned development tools from [mise.toml](../mise.toml) with the official mise Action. The acceptance stage installs only Go from that configuration. Tool caches are disabled so PR jobs do not supply installed executables to later runs. They transfer that same binary through an artifact, then call [acceptance-stage](../.github/actions/acceptance-stage/action.yml) to verify its SHA-256 and launch it as a real process. The tagged acceptance Go tests are compiled separately with caching disabled; the product is not rebuilt and the ordinary Go unit suite is not run again. Artifacts are retained for one day and downloaded only from the current workflow run.

PR-code validation jobs have read permissions, no supplied secrets, and checkout credentials are not persisted. Draft/Ready control jobs have write permissions but do not check out or execute PR code, composites, or artifacts. Normal fork tokens are read-only, so automatic PR-state changes can be blocked on forks; this workflow does not bypass that restriction.

Markdown checks use the rumdl version pinned in [mise.toml](../mise.toml) and the rules in [.rumdl.toml](../.rumdl.toml). MD051 and MD057 validate local headings/fragments and file links; external HTTP availability is not a CI dependency. mise resolves rumdl, actionlint, and ShellCheck through its standard aqua registry. Go module library dependencies remain in `go.mod` and `go.sum`.

## Codex approval control

The separate [approval workflow](../.github/workflows/approve-codex-review.yml) requires a Codex summary showing both Code Review and Security Review as Completed and a Codex 👍 reaction to the PR body. It waits up to three minutes for that reaction, then may create a bot Approve. It runs trusted default-branch workflow code without checking out PR code.

This mechanism does not verify the reviewed HEAD or latest Ready-triggered review cycle, and can reuse older completion/reaction information. A successful workflow run alone does not prove that an approval was created. Inspect new Ready-triggered reviews separately. PR body edits also start the CI state sequence again, even when the HEAD has not changed.

The approval workflow becomes available for comment events only after it exists on the default branch. For this workflow to create an approving review, repository settings must allow GitHub Actions to create and approve pull requests. Verify that setting before relying on the workflow. The workflow does not configure tokens, secrets, Rulesets, branch protection, or whether `pass` is a required check.
