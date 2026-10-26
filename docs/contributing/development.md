# Local development

Use this guide when changing gh-settings. The [architecture](architecture.md) explains implementation ownership, and [testing](testing.md) explains where checks belong and how CI verifies the built product.

## Set up the tools

Install [mise](https://mise.jdx.dev/getting-started.html), then run from the repository root:

```sh
mise install
```

[mise.toml](../../mise.toml) pins the development tools used locally and in CI. [go.mod](../../go.mod) declares the minimum Go version and library dependencies. Mise's npm backend installs commitlint and its conventional preset with Bun; Node.js, npm, a package manifest, and local Git hooks are not required. Transitive npm dependencies are resolved during installation. Run development commands through mise to use the pinned tool versions.

## Pull request messages

PR titles and bodies are validated by the [independent PRLint workflow](../../.github/workflows/prlint.yml) as one squash commit message: the title is the subject, followed by a blank line and the optional PR body. The message follows [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

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

To check a proposed squash message locally after `mise install`:

```sh
printf '%s\n\n%s\n' 'feat(settings): manage repository topics' 'Describe why this changed.' |
  mise exec -- bun run --bun --no-install "$(mise which commitlint)" --strict
```

Keep the PR body consistent with [Write PR bodies for the Git history](../../CONTRIBUTING.md#write-pr-bodies-for-the-git-history). Message validation rechecks title/body edits independently of product builds; the [PR validation guide](pr-validation.md) explains required-check activation and code/base events. Local `git commit` messages are not validated; the final squash message must still be confirmed at merge time.

## Build and run from source

```sh
mise exec -- go build -o gh-settings .
./gh-settings --help
./gh-settings --version
```

A source build reports version `dev` unless a value is supplied through the existing linker symbol `main.version`. The root [entry point](../../main.go) remains buildable with `go build .`; release builds use `-ldflags '-X main.version=...'`.

The source-built CLI and the [published v0.18.0 Action/CLI pair](../github-actions.md#published-package-and-consumer-pins) include the documented v0.18.0 resource set. Source test evidence remains distinct from verifying the published binary selected by the Action; current source includes the unpublished #412 Ruleset presentation correction.

To use the built checkout as a GitHub CLI extension, run `gh extension install .`. Commands that read or change settings use the current checkout; see the [command reference](../commands.md) before running them against a repository.

## Development checks

The [commit-stage Action](../../.github/actions/commit-stage/action.yml) defines the CI checks. Locally, inspect any filenames reported by `gofmt`, format those Go files, and run the relevant checks:

```sh
mise exec -- gofmt -l .
mise exec -- go mod download
mise exec -- go mod verify
mise exec -- go mod tidy -diff
mise exec -- go vet ./...
mise exec -- go test -race -count=1 -vet=off ./...
mise exec -- go build ./...
mise exec -- actionlint -shellcheck shellcheck .github/workflows/*.yml examples/github-actions/*.yml
mise exec -- shellcheck scripts/publish-release.sh tests/release.sh tests/github-actions/*.sh action/*.sh .github/scripts/*.sh
mise exec -- bash tests/release.sh
mise exec -- bash tests/github-actions/action-install.sh
mise exec -- rumdl check --no-cache --deny-config-warnings README.md CONTRIBUTING.md AGENTS.md .agents .github/pull_request_template.md docs examples/github-actions
```

[Process acceptance](testing.md#local-process-acceptance) runs separately against an explicitly supplied executable. For documentation changes, the Markdown check validates local file links and heading fragments as well as formatting.

## Add an agreed repository field

First define the field's meaning, type, omission/null rules, and API mapping in the [settings contract](../settings.md) and affected [feature specification](../features/). Then carry that contract through the existing owners:

1. Extend `RepositoryDeclaration` and the appropriate complete state in [the model](../../internal/settings/repository.go), such as `RepositoryState` or a policy-specific observation. Preserve the optional declaration and complete observed-state distinction: omitted values differ from declared false or empty strings. Update presence and comparison owner tests.
2. Extend the private YAML document, validation, and encode mapping in [the codec](../../internal/workspace/config.go). Keep the versioned/grouped document, accepted aliases and merges, and strict rejection rules. Check the accepted type, omission, and round trip at this owner.
3. Map complete state and declared-only payloads in the adapter owning that API: base fields use [REST reads](../../internal/github/repository.go) and [PATCH](../../internal/github/apply.go), while PR/Issue policies use [repository access](../../internal/github/repository_access.go). Check missing/null/type handling, target isolation and exact mutation payloads. Update [requested observation routing](../../internal/settings/requested_observation.go) and [export](../../internal/app/export.go) so the newly observed value is included only after complete reads.
4. Extend the [requested-state comparison](../../internal/settings/requested_observation.go), retaining managed counts, native values, and omission. Review both presentations: the CLI renderer accepts native pairs, while [Actions Summary validation](../../internal/actions/summary.go) explicitly validates field names, value types, and the maximum managed count. The current-source maximum is fifty: twenty-four repository fields, topics once, labels once, Rulesets once, Environments once, fourteen Actions policy fields, three security fields and five Pages fields. Update that contract and its owner tests together. The report remains ephemeral presentation data, independent of apply.
5. Carry the field through one representative existing [built-product acceptance path](../../tests/acceptance/), checking the actual interface and boundary payload/output. Update complete-read fixtures where needed. Keep detailed parser/type matrices with their owners; extend composite or caller cases when their wiring contract changes.

Update user references and behavior specifications with the implementation. The [test ownership table](testing.md#test-ownership) helps avoid repeating the same case at every layer.

## Add an agreed resource group

Topics demonstrates a direct resource extension: [`settings.Declaration`](../../internal/settings/settings.go) holds concrete resource declarations, [`app.Reader` and `app.Writer`](../../internal/app/resources.go) defer resource operations, and [the topics adapter](../../internal/github/topics.go) owns GET/PUT mapping.

Labels demonstrates ID-selected whole-object ownership with [typed comparisons](../../internal/settings/labels.go), a strict [YAML boundary](../../internal/workspace/labels.go) and [paged ID observations](../../internal/github/labels.go). Fresh apply reads precede preflight and POST/PATCH. Confirmed creation IDs pass through the application result and snapshot-bound workspace recorder; the CLI owns opt-out, recovery output and failure status. The [labels specification](../features/labels.feature.md) maps collision, rename, default, writeback, pagination and partial-failure tests.

For another resource, agree its semantics first. Put declaration/state/comparison at the settings owner, YAML grouping at the workspace boundary, endpoint mapping at the GitHub boundary, and operation sequencing in the application. Adapt CLI/Actions presentation to those shared results. The current codec also accepts the adopted Rulesets, Actions and security subsets; introducing another group requires an explicit contract change.

Extend useful existing structure for the requested resource. A registry, generic resource framework, compatibility mode, or second planning implementation is not required by this process. The [future direction](architecture.md#future-direction) distinguishes agreed future semantics from unresolved choices.
