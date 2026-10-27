# Change labels

[Repository settings](../../.github/settings.yml) owns the 20 label names, descriptions, and colors shared with gh-qw: eleven Conventional Commit types, two compatibility labels (`BREAKING CHANGE` and `DEPRECATED`), and seven independent issue-triage labels.

Give each focused PR one Conventional Commit type. Use `feat`, `fix`, and `perf` only for user-observable changes to the CLI or public Action, not development infrastructure. Compatibility labels may accompany a type. The initial Issue label, branch name, title alone, or file extension is not proof of a change type. Reviewers compare labels against the final diff under the [review policy](../../.agents/skills/code-review/SKILL.md).

Issue forms suggest `fix` for bug reports and `feat` for feature requests. Tasks and blank Issues have no default classification; an implementing PR may use another type. The seven manually managed triage labels are `accessibility`, `duplicate`, `good first issue`, `help wanted`, `invalid`, `question`, and `wontfix`.

Version selection uses durable squash messages, not labels. Reviewed PR labels classify generated release notes: breaking, deprecated, feature/performance, fix, and maintenance. See [releases](releases.md).

## Automatic path suggestions

The [Labeler configuration](../../.github/labeler.yml) has four deliberately conservative categories. A label is suggested only when **every changed file** matches the same category. Product changes including incidental tests or documentation are not automatically labeled as test-only or docs-only.

| Label | Automatic scope |
| --- | --- |
| `docs` | Root README/contributing/agent docs, `docs/**`, skill instructions, PR template |
| `test` | `tests/**` and `**/*_test.go` |
| `build` | `go.mod`, `go.sum`, and `mise.toml` |
| `ci` | CI/PRLint/Labeler workflows, CI-specific local Actions/scripts, Labeler rules |

Release workflows and tooling, repository-settings workflows, public Action source, and consumer examples are classified manually; their paths do not determine user impact or whether the work is `build` versus `ci`. Dependabot suggests `build` for Go modules and `ci` for GitHub Actions updates, without automatic merging.

The [Labeler workflow](../../.github/workflows/labeler.yml) uses base-branch rules over the GitHub API under `pull_request_target` without executing PR code. Permissions are Contents read and Pull requests write. `dot: true` includes hidden files; `sync-labels: false` preserves manually assigned labels. Reviewers remove stale suggestions when the PR changes; a later matching run can re-add them. All label names must exist before Labeler can apply them.
