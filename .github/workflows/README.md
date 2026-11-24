# GitHub Actions workflows

These YAML files define the events, jobs and permissions for gh-settings itself. Use the
[consumer examples](../../examples/github-actions/README.md) when adapting workflows for another repository.

| Task | Workflow |
| --- | --- |
| Change PR screening, the squash-message check or settings validation | [pr.yml](pr.yml), [prlint.yml](prlint.yml), [settings-pr.yml](settings-pr.yml) |
| Label a PR or record the result of a Codex review | [labeler.yml](labeler.yml), [approve-codex-review.yml](approve-codex-review.yml) |
| Build and test the exact commit pushed to `main` | [ci-push.yml](ci-push.yml) |
| Publish a release, apply reviewed repository settings or verify a candidate Action against live Topics | Manual workflows for [release](release.yml), [settings apply](settings-apply.yml) and [Topics verification](verify-public-action.yml) |

The generated [actions.lock](actions.lock) records dependencies by workflow. Follow the
[lockfile procedure](../../docs/development/repository-automation.md#workflow-dependency-lockfile)
when updating those dependencies.
