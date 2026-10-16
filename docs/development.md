---
type: Guide
title: Local development
description: Run local checks and understand the repository CI validation boundaries.
---

Use Go 1.25 or later and install [mise](https://mise.jdx.dev/getting-started.html). From the repository root, use the commands below to install the exact tool versions in [mise.toml](../mise.toml), including Go 1.26.0, and run checks through mise so the local versions match CI.

## Refactoring direction

The [v0 simplification contract](concept.md#v0-simplification-contract) defines retained requirements and the shared Go ownership model. The [ordered series](https://github.com/daiksud/gh-settings/issues/133) delivers each issue through its own reviewed PR, tests and documentation; later planned behavior is not current behavior. Retired wrappers and historical fixture combinations may be deleted without forwarding adapters. The checks below describe the current tree and must lose obsolete invocations with the corresponding implementation change. Source integration does not publish a release, change active released consumer pins or approve a settings deployment.

The [root entry point](../main.go) retains `main.version` for `-ldflags '-X main.version=...'` and wires deferred GitHub read/write dependencies into the four shared application operations. The [CLI adapter](../internal/cli/run.go) owns arguments, terminal rendering and status over those typed operations; its [public tests](../internal/cli/run_test.go) preserve dispatch, silent commands and output failures. Help/version do not invoke an application operation or need checkout/authentication. Root and extension builds still use `go build .`.

## Development checks

```sh
mise install
mise exec -- go test -race -count=1 -vet=off ./...
mise exec -- go vet ./...
mise exec -- go build ./...
mise exec -- go mod verify
mise exec -- go mod tidy -diff
mise exec -- actionlint -shellcheck shellcheck .github/workflows/*.yml examples/github-actions/*.yml
mise exec -- shellcheck scripts/publish-release.sh tests/release.sh examples/github-actions/*.sh tests/github-actions/*.sh action/*.sh
mise exec -- bash tests/release.sh
mise exec -- bash tests/github-actions/action-install.sh
mise exec -- bash tests/github-actions/action-command.sh
python3 tests/github-actions/action-summary.py
mise exec -- rumdl check --no-cache --deny-config-warnings README.md docs examples/github-actions
```

## Local process acceptance

Build the product once, then run the tagged Go acceptance tests against that executable:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The [acceptance tests](../tests/acceptance/acceptance_test.go) require an explicit existing binary path and use a local Unix-socket HTTP fixture and go-gh v2.16.2's `http_unix_socket` configuration in a temporary `GH_CONFIG_DIR`. Their dummy token never authenticates with GitHub. Checks cover YAML values, empty strings and false, GET null normalization, current-repository selection, declared-field planning, export → edit → validate → plan → apply, exit codes, stderr, and failure paths. Plan cases verify GET-only access and unchanged configuration; apply cases verify exact PATCH payloads, omission preservation, and no GET. Invalid input and plan/apply with no managed fields make no API request. Temporary repositories/configuration are isolated and cleaned up. Product code needs no testing hook or test-only modification; Unix sockets are required.

The source-only [Actions metadata entry](commands.md#source-actions-metadata) is covered by `TestActionMetadata` in the [built-product acceptance suite](../tests/acceptance/metadata_test.go): the executable lives outside a real committed consumer checkout, runs from its subdirectory without settings or credentials, ignores an unrelated `GH_REPO` and returns the actual candidate's digest/version and consumer commit/target without API requests. [Public adapter tests](../internal/actions/run_test.go) cover missing checkout/origin/commit, empty and CR/LF output values, writer failure and argument rejection. The composite still uses its packaged Python metadata helper; switching that caller is a separate change.

## CI validation

The [PR workflow](../.github/workflows/ci-pr.yml) follows Draft → commit-stage → Ready → acceptance-stage and composite-smoke → `pass`. It runs on opened, synchronized, reopened, and edited PRs. The [main workflow](../.github/workflows/ci-push.yml) runs commit-stage → acceptance-stage and composite-smoke → `pass` on pushes to `main`. Failure, cancellation, or skipping either validation job prevents a successful `pass` result. This CI does not merge, publish, or deploy.

Both workflows call [commit-stage](../.github/actions/commit-stage/action.yml) for formatting, modules, vet, race tests, GitHub workflow/shell checks, a single product build, and Markdown/local links. Ordinary CI injects the committed `action/cli-version`; release keeps its explicit version input and separate equality guard. The producer exposes its actual checked-out Git SHA, CLI version and binary checksum. PR validation uses the merge source, and both downstream jobs check out that exact producer SHA rather than mixing a merge binary with a head-only Action.

CI installs the same pinned development tools from [mise.toml](../mise.toml) with the official mise Action. The acceptance stage installs only Go from that configuration. Tool caches are disabled so PR jobs do not supply installed executables to later runs. Both downstream jobs download that same run's binary artifact with digest mismatch treated as an error. [Acceptance-stage](../.github/actions/acceptance-stage/action.yml) verifies its SHA-256 and embedded version before launching it as a real process. The tagged acceptance Go tests are compiled separately with caching disabled; the product is not rebuilt and the ordinary Go unit suite is not run again. Artifacts are retained for one day; PR artifacts are not reused by privileged publication or settings workflows.

The separate `composite-smoke` job uses the public Action's supported GitHub-hosted Ubuntu 24.04 x64 runner; the existing stages remain on `ubuntu-slim`. It checks out the producer's exact source into `.action`, verifies the transferred candidate's checksum and the Action checkout's source/version through the [bounded fixture preparation](../tests/github-actions/prepare-candidate-download.sh), then creates a Git consumer tracking only `.github/settings.yml`. The nested `.action` checkout remains separate and untracked. The consumer has no root `action.yml` or `action/` helpers.

A test-only [curl stand-in](../tests/github-actions/candidate-curl.sh) serves only the installer's two fixed versioned asset/checksum URLs from the verified candidate fixture, without network forwarding. Actual local `uses: ./.action` expansion runs the unchanged installer, checksum/version checks, metadata and `command: validate` with no token input. No committed version override, public binary/download-URL input or old-release fallback is added. This proves candidate integration before publication, not a real release download or immutable released-asset identity.

The following step asserts all four actual composite outputs against the separate consumer commit/origin and this run's candidate version/checksum, and logs distinct source/consumer roots and SHAs. Missing/tampered artifacts, fixture, composite or assertion failures fail the existing aggregate gate; none becomes skipped success. Inspect the exact-source hosted `composite-smoke` log for actual Ubuntu/`uses` evidence. Local native installer/metadata/validation checks, including simulated runner attributes, remain distinct from that hosted proof. Existing installer/command/renderer and consumer tests retain their own boundaries; no separate fixture-testing framework is required.

PR-code validation jobs have read permissions, no supplied secrets, and checkout credentials are not persisted. Draft/Ready control jobs have write permissions but do not check out or execute PR code, composites, or artifacts. Normal fork tokens are read-only, so automatic PR-state changes can be blocked on forks; this workflow does not bypass that restriction.

Markdown checks use the rumdl version pinned in [mise.toml](../mise.toml) and the rules in [.rumdl.toml](../.rumdl.toml). MD051 and MD057 validate local headings/fragments and file links; external HTTP availability is not a CI dependency. mise resolves rumdl, actionlint, and ShellCheck through its standard aqua registry. Go module library dependencies remain in `go.mod` and `go.sum`.

## Codex approval control

The separate [approval workflow](../.github/workflows/approve-codex-review.yml) requires a Codex summary showing both Code Review and Security Review as Completed and a Codex 👍 reaction to the PR body. It waits up to three minutes for that reaction, then may create a bot Approve. It runs trusted default-branch workflow code without checking out PR code.

This mechanism does not verify the reviewed HEAD or latest Ready-triggered review cycle, and can reuse older completion/reaction information. A successful workflow run alone does not prove that an approval was created. Inspect new Ready-triggered reviews separately. PR body edits also start the CI state sequence again, even when the HEAD has not changed.

The approval workflow becomes available for comment events only after it exists on the default branch. For this workflow to create an approving review, repository settings must allow GitHub Actions to create and approve pull requests. Verify that setting before relying on the workflow. The workflow does not configure tokens, secrets, Rulesets, branch protection, or whether `pass` is a required check.

## Settings example verification

The inert [settings examples](../examples/github-actions/README.md) are linted and tested in commit-stage. Focused shell/Python fixtures cover each helper, while ordinary Go tests run the real YAML commands with a mock CLI and local approval/token stubs and check the cross-job credential/input boundary. Actual Environment approval, branch/tag enforcement, and real token revocation remain unverified until the separately authorized [adoption checklist](github-actions.md#ordered-owner-setup).
