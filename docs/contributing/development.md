# Local development

Use this guide when changing gh-settings. The [architecture](architecture.md) explains implementation ownership, and [testing](testing.md) explains where checks belong and how CI verifies the built product.

## Set up the tools

Install [mise](https://mise.jdx.dev/getting-started.html), then run from the repository root:

```sh
mise install
```

[mise.toml](../../mise.toml) pins the development tools used locally and in CI. [go.mod](../../go.mod) declares the minimum Go version and library dependencies. Mise's npm backend installs commitlint and its conventional preset with Bun; Node.js, npm, a package manifest, and local Git hooks are not required. Transitive npm dependencies are resolved during installation. Run development commands through mise to use the pinned tool versions.

## Pull request messages

PR titles and bodies are validated by the [PR CI workflow](../../.github/workflows/ci-pr.yml) as one squash commit message: the title is the subject, followed by a blank line and the optional PR body. The message follows [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

```text
<type>[optional scope][!]: <description>

[optional body]

[optional footers]
```

Commitlint uses the conventional preset with only these additional style rules:

- Types are restricted to `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, and `test`.
- Types must be lowercase.
- Scopes must be lowercase.
- Subjects must not be sentence-case, start-case, PascalCase, or uppercase.
- Subjects must not end in a period (`.`).
- Titles must not start or end with whitespace.

The preset already enforces all of these style rules except lowercase scopes. The local configuration only enables `scope-case` and disables the preset's three line-length checks; all other preset rules are inherited unchanged.

The configuration intentionally has no header, body, or footer line-length limit. Commitlint permits an empty body and either breaking-change marker alone. Mandatory structure remains enforced, including type and description, a blank line before the body, and a blank line before footers. Unlike local commit linting, PR checks do not ignore titles resembling generated Git messages such as `fixup!` or `Merge branch`.

Apply the [code-review policy](../../.agents/skills/code-review/SKILL.md) in addition to these syntax checks. It requires one coherent type/scope, a durable explanation in the body, and both `!` in the title and a `BREAKING CHANGE:` footer with impact and migration for every incompatible change. [ADR 0024](../adr/0024-code-review-policy.md) records this stricter review requirement.

To check a proposed squash message locally after `mise install`:

```sh
printf '%s\n\n%s\n' 'feat(settings): manage repository topics' 'Describe why this changed.' |
  mise exec -- bun run --bun --no-install "$(mise which commitlint)" --strict
```

Keep the PR body consistent with [Write PR bodies for the Git history](../../CONTRIBUTING.md#write-pr-bodies-for-the-git-history). The repository's PR CI rechecks when the title or body changes. Local `git commit` messages are not validated; the final squash message must still be confirmed at merge time.

## Build and run from source

```sh
mise exec -- go build -o gh-settings .
./gh-settings --help
./gh-settings --version
```

A source build reports version `dev` unless a value is supplied through the existing linker symbol `main.version`. The root [entry point](../../main.go) remains buildable with `go build .`; release builds use `-ldflags '-X main.version=...'`.

The source-built CLI and the [published v0.14.0 Action/CLI pair](../github-actions.md#published-package-and-consumer-pins) include labels, merge methods, automatic branch deletion, squash/normal merge commit defaults, auto-merge and branch-update options. Source test evidence remains distinct from verifying the published binary selected by the Action.

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

1. Extend `RepositoryDeclaration` and `RepositoryState` in [the model](../../internal/settings/repository.go). Preserve the optional declaration and complete observed-state distinction: omitted values differ from declared false or empty strings. Update presence and comparison owner tests.
2. Extend the private YAML document, validation, and encode mapping in [the codec](../../internal/workspace/config.go). Keep the versioned/grouped document, accepted aliases and merges, and strict rejection rules. Check the accepted type, omission, and round trip at this owner.
3. Map a complete read into state in [the read adapter](../../internal/github/repository.go), and declaration pointers into the private request DTO in [the write adapter](../../internal/github/apply.go). Check missing/null/type handling and exact declared-only PATCH payloads. Extend [export's state-to-declaration mapping](../../internal/app/export.go) so export includes the newly observed value.
4. Extend the [ordered comparison](../../internal/settings/comparison.go), retaining managed counts, native values, and omission. Review both presentations: the CLI renderer accepts native pairs, while [Actions Summary validation](../../internal/actions/summary.go) explicitly validates field names, value types, and the maximum managed count. The current maximum is seven: five repository fields, topics once, and labels once, independently of the number of label operations. Update that contract and its owner tests together. The report remains ephemeral presentation data, independent of apply.
5. Carry the field through one representative existing [built-product acceptance path](../../tests/acceptance/), checking the actual interface and boundary payload/output. Update complete-read fixtures where needed. Keep detailed parser/type matrices with their owners; extend composite or caller cases when their wiring contract changes.

Update user references and behavior specifications with the implementation. The [test ownership table](testing.md#test-ownership) helps avoid repeating the same case at every layer.

## Add an agreed resource group

Topics demonstrates a direct resource extension: [`settings.Declaration`](../../internal/settings/settings.go) holds concrete resource declarations, [`app.Reader` and `app.Writer`](../../internal/app/resources.go) defer resource operations, and [the topics adapter](../../internal/github/topics.go) owns GET/PUT mapping.

Labels demonstrates a collection extension with [typed declaration/state/operation comparisons](../../internal/settings/labels.go), a strict [YAML boundary](../../internal/workspace/labels.go), and [paged read/CRUD mapping](../../internal/github/labels.go). Its `Writer.ReadLabels` dependency obtains fresh state after the direct repository/topics writes; the shared model determines operations before `Writer.Labels` sends them. CLI and Actions own their escaped operation presentation. The [labels specification and tests](../features/labels.feature.md) cover authoritative membership, attribute omission, pagination/completeness, permission needs, and partial failures.

For another resource, agree its semantics first. Put declaration/state/comparison at the settings owner, YAML grouping at the workspace boundary, endpoint mapping at the GitHub boundary, and operation sequencing in the application. Adapt CLI/Actions presentation to those shared results. The current codec accepts repository, topics, and labels; introducing another group requires an explicit contract change.

Extend useful existing structure for the requested resource. A registry, generic resource framework, compatibility mode, or second planning implementation is not required by this process. The [future direction](architecture.md#future-direction) distinguishes agreed future semantics from unresolved choices.
