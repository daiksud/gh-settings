# Change labels

[Repository settings](../../.github/settings.yml) owns the 20 label names, descriptions, and colors shared with gh-qw: thirteen labels for organizing work and compatibility information, and seven issue-triage labels.

Labels organize Issues and PRs independently of the proposed squash message. They need not match its type or compatibility markers, and title/body edits do not require relabeling. Reviewers assess the message against actual user impact under the [review policy](../../.agents/skills/code-review/SKILL.md); an Issue label, branch name, or file extension does not select that message's type.

Issue forms suggest the `fix` label for bug reports and the `feat` label for feature requests. Tasks and blank Issues have no default labels. These suggestions organize incoming Issues and do not select the Conventional Commit type of an implementation. The seven manually managed triage labels are `accessibility`, `duplicate`, `good first issue`, `help wanted`, `invalid`, `question`, and `wontfix`.

Version selection and release-note categories use the actual committed squash messages in the fixed release interval. Labels do not affect either result. Do not add automatic type/label synchronization, type-agreement requirements, or label-based PRLint/release checks. See [releases](releases.md#release-notes-from-committed-messages).

## Automatic path suggestions

The [Labeler configuration](../../.github/labeler.yml) has four deliberately conservative categories. A label is suggested only when **every changed file** matches the same category. Product changes including incidental tests or documentation are not automatically labeled as test-only or docs-only.

| Label | Automatic scope |
| --- | --- |
| `docs` | Root README/contributing/agent docs, `docs/**`, skill instructions, PR template |
| `test` | `tests/**` and `**/*_test.go` |
| `build` | `go.mod`, `go.sum`, and `mise.toml` |
| `ci` | CI/PRLint/Labeler workflows, CI-specific local Actions/scripts, Labeler rules |

Release workflows and tooling, repository-settings workflows, public Action source, and consumer examples are outside these automatic path suggestions. Dependabot suggests the `build` label for Go modules and the `ci` label for GitHub Actions updates, without automatic merging.

The [Labeler workflow](../../.github/workflows/labeler.yml) uses base-branch rules over the GitHub API under `pull_request_target` without executing PR code. Permissions are Contents read and Pull requests write. `dot: true` includes hidden files; `sync-labels: false` preserves manually assigned labels. Authors can adjust suggestions for organization; a later matching run can re-add them. All label names must exist before Labeler can apply them.
