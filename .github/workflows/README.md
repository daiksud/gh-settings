# GitHub Actions workflows

These YAML files define the events, jobs and permissions for gh-settings itself. Use the
[consumer examples](../../examples/github-actions/README.md) when adapting workflows for another repository.

| Task | Workflow |
| --- | --- |
| Change PR screening, commit and squash-message checks or settings validation | [pr.yml](pr.yml), [commitlint.yml](commitlint.yml), [settings-pr.yml](settings-pr.yml) |
| Label a PR or record the result of a Codex review | [labeler.yml](labeler.yml), [approve-codex-review.yml](approve-codex-review.yml) |
| Deploy the inspected binaries for each commit pushed to `main` | [push.yml](push.yml) |
| Change the native builds and inspections shared by main and release | [candidate.yml](candidate.yml) |
| Publish a release, apply reviewed repository settings or verify a candidate Action against live Topics | Manual workflows for [release](release.yml), [settings apply](settings-apply.yml) and [Topics verification](verify-public-action.yml) |

The generated [actions.lock](actions.lock) records dependencies by workflow. Follow the
[lockfile procedure](../../docs/development/repository-automation.md#workflow-dependency-lockfile)
when updating those dependencies.
