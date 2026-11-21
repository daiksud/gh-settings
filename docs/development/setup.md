# Local development

Use this guide when changing gh-settings. Build the checkout first, then use the [architecture guide](architecture.md) to locate your change and the [testing guide](testing.md) to choose its checks.

## Set up the tools

Install [mise](https://mise.jdx.dev/getting-started.html), then run from the repository root:

```sh
mise install
mise exec -- bun install --frozen-lockfile
```

[mise.toml](../../mise.toml) enables idiomatic version files only. Go is selected
from [go.mod](../../go.mod), and Bun from `packageManager` in
[package.json](../../package.json). The same manifest declares commitlint,
its conventional preset and rumdl; [bun.lock](../../bun.lock) fixes their
resolved dependencies. CI uses setup-go and setup-bun with these information
sources and installs dependencies with `bun install --frozen-lockfile`.
Node.js, npm and local Git hooks are not required. Use `bun run` for the
package-managed tools; Go commands may run through mise locally.

## Build and run from source

```sh
mise exec -- go build -o gh-settings .
./gh-settings --help
./gh-settings --version
```

A source build reports version `dev` unless a value is supplied through the existing linker symbol `main.version`. The root [entry point](../../main.go) remains buildable with `go build .`; release builds use `-ldflags '-X main.version=...'`.

To use the built checkout as a GitHub CLI extension, run `gh extension install .`. Commands that read or change settings use the current checkout; see the [command reference](../references/commands.md) before running them against a repository.

## Development checks

The [PR workflow](../../.github/workflows/pr.yml) calls focused [lint](../../.github/actions/lint/action.yml), [unit-test](../../.github/actions/unit-test/action.yml), and [build](../../.github/actions/build/action.yml) Actions directly from its `screening` job, with tool setup and Go cache restore/save as separate steps. Locally, inspect any filenames reported by `gofmt`, format those Go files, and run the relevant checks:

```sh
mise exec -- gofmt -l .
mise exec -- go mod download
mise exec -- go mod verify
mise exec -- go mod tidy -diff
mise exec -- go vet ./...
mise exec -- go test -race -count=1 -vet=off ./...
mise exec -- go build ./...
mise exec -- bash tests/release.sh
mise exec -- bash tests/github-actions/action-install.sh
mise exec -- bun run lint:markdown
```

[Process acceptance](testing.md#local-process-acceptance) runs separately against an explicitly supplied executable. For documentation changes, the Markdown check validates local file links and heading fragments as well as formatting.

## Pull request messages

Except for [Dependabot-authored PRs](pr-validation.md#message-and-settings-validation), PR titles and bodies are validated by the [independent PRLint workflow](../../.github/workflows/prlint.yml) as one squash commit message: the title is the subject, followed by a blank line and the optional PR body. The message follows [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

```text
<type>[optional scope][!]: <description>

[optional body]

[optional footers]
```

The [commitlint configuration](../../commitlint.config.mjs) only extends the
`@commitlint/config-conventional` preset. Its defaults apply unchanged,
including default ignores, style rules, and line-length limits. PRLint adds
no rules or separate policy tests.

Apply the [code-review policy](../../.agents/skills/code-review/SKILL.md) for PR scope and semantic impact. Either breaking marker is sufficient. PRLint itself only invokes commitlint on the title and body; it does not implement additional semantic review rules.

To check a proposed squash message locally after installing the development dependencies:

```sh
printf '%s\n\n%s\n' 'feat(settings): manage repository topics' 'Describe why this changed.' |
  mise exec -- bun run commitlint --strict
```

Keep the PR body consistent with [Write PR bodies for the Git history](../../CONTRIBUTING.md#write-pr-bodies-for-the-git-history). Message validation rechecks title/body edits independently of product builds; the [PR validation guide](pr-validation.md) explains required-check activation and code/base events. Local `git commit` messages are not validated; the final squash message must still be confirmed at merge time.

## Add an agreed repository field

First define the field's meaning, type, omission/null rules, and API mapping in the [settings contract](../references/settings/README.md) and affected [feature specification](features). Then carry that contract through the existing owners:

1. Extend `RepositoryDeclaration` and the appropriate complete state in [the model](../../internal/settings/repository.go), such as `RepositoryState` or a policy-specific observation. Preserve the optional declaration and complete observed-state distinction: omitted values differ from declared false or empty strings. Update presence and comparison owner tests.
2. Add the UI control to [the typed UI model](../../internal/settings/uimodel) and its current [codec](../../internal/workspace/config.go) contract. Keep the exact version/grouping, safe YAML aliases/merges and strict types/presence checks. API-shaped aliases and old envelopes are not compatibility options. Test literal UI round trips and independent UI-to-native adapter expectations.
3. Map complete state and declared-only payloads in the adapter owning that API: base fields use [REST reads](../../internal/github/repository.go) and [PATCH](../../internal/github/apply.go), while PR/Issue policies use [repository access](../../internal/github/repository_access.go). Check missing/null/type handling, target isolation and exact mutation payloads. Update [requested observation routing](../../internal/settings/requested_observation.go), [native export](../../internal/app/export.go) and [UI export projection](../../internal/github/ui_export.go) so acquired values are projected only after complete reads. Never fabricate acquisition markers, IDs or caps to invoke an observation wrapper.
4. Extend the [requested-state comparison](../../internal/settings/requested_observation.go), preserving native values and omission. Review CLI/rendered Summary separately from serialized private reports: UI controls can expand to multiple native members. Update the report's accepted fields and count bounds when a native comparison changes. Include any observation normalization needed to reconstruct a UI choice. Check the terminal output and the Action Summary against the same expected path and values.
5. Carry the field through one representative existing [built-product acceptance path](../../tests/acceptance), checking the actual interface and boundary payload/output. Update complete-read fixtures where needed. Keep detailed parser/type matrices with their owners; extend composite or caller cases when their wiring contract changes.

Update user references and behavior specifications with the implementation. The [test ownership table](testing.md#test-ownership) helps avoid repeating the same case at every layer.

## Add an agreed resource group

Topics demonstrates a direct resource extension: [`settings.Declaration`](../../internal/settings/settings.go) holds concrete resource declarations, [`app.Reader` and `app.Writer`](../../internal/app/resources.go) defer resource operations, and [the topics adapter](../../internal/github/topics.go) owns GET/PUT mapping.

Labels demonstrates ID-selected whole-object ownership with [typed comparisons](../../internal/settings/labels.go), a strict [YAML boundary](../../internal/workspace/config.go) and [paged ID observations](../../internal/github/labels.go). Fresh apply reads precede preflight and POST/PATCH. Confirmed creation IDs pass through the application result and snapshot-bound workspace recorder; the CLI owns opt-out, recovery output and failure status. The [labels specification](features/labels.feature.md) maps collision, rename, default, writeback, pagination and partial-failure tests.

For another resource, agree its semantics first. Put declaration/state/comparison at the settings owner, YAML grouping at the workspace boundary, endpoint mapping at the GitHub boundary, and operation sequencing in the application. Adapt CLI/Actions presentation to those shared results. For collection pruning, also define the identity, evidence and rechecks needed to delete an unlisted object. See [global preflight and reconciliation](architecture.md#reconcile-and-verify).

Use the [coverage ledger](settings-coverage.md) to record which UI and API paths are confirmed and which parts of the implementation still need verification.
