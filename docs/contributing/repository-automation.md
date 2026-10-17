# Repository automation

This guide is for maintainers of **daiksud/gh-settings**. For installing gh-settings workflows in another repository, use the [GitHub Actions guide](../github-actions.md) and [examples](../../examples/github-actions/README.md).

The workflow files describe source-controlled behavior. Required reviews/checks, Environment protection rules, App installations, secrets, and event-policy eligibility are configured separately on GitHub; inspect those settings when assessing the current operational state.

## Workflow inventory

| Workflow | Trigger and responsibility |
| --- | --- |
| [`ci-pr.yml`](../../.github/workflows/ci-pr.yml) | Opened, synchronized, reopened, or edited PRs; Draft/Ready transitions and validation ending in `pass` |
| [`ci-push.yml`](../../.github/workflows/ci-push.yml) | Pushes to `main`; validate source and the built candidate, ending in `pass` |
| [`settings-pr.yml`](../../.github/workflows/settings-pr.yml) | Relevant PR path changes; read-only settings plan and review Summary |
| [`settings-main.yml`](../../.github/workflows/settings-main.yml) | Pushes to `main` affecting settings or this workflow; one Environment-bound direct apply job |
| [`settings-environment-guard.yml`](../../.github/workflows/settings-environment-guard.yml) | PR target events and pushes to `main`; inspect workflow YAML with trusted source, reporting `gh-settings` |
| [`approve-codex-review.yml`](../../.github/workflows/approve-codex-review.yml) | Qualifying Codex summary comments; conditionally create a bot approval |
| [`release.yml`](../../.github/workflows/release.yml) | Explicit manual dispatch from `main`; [versioned release procedure](releases.md) |

## PR lifecycle

The PR CI workflow sets Draft, runs commit-stage, then marks the PR Ready before acceptance-stage and composite-smoke. The final `pass` job rejects failed, cancelled, or skipped validation. Editing the PR body can start this sequence again even when its HEAD commit has not changed.

Jobs executing PR source have read permissions, no supplied secrets, and no persisted checkout credentials. Draft/Ready jobs have write permissions but do not check out or execute PR code, local composites, or transferred artifacts. Fork token restrictions can block automatic state changes; the workflow does not bypass them.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results and inspect the applicable [repository rules](https://github.com/daiksud/gh-settings/rules) for the current merge requirements. A passing source test does not show which checks GitHub requires externally.

## Active settings workflows

The active callers pin the released package shown in the [published package and consumer pins](../github-actions.md#published-package-and-consumer-pins). Their pins are independent of [`action/cli-version`](../../action/cli-version), which selects the CLI packaged by current source. Publishing a release or changing examples does not automatically update these callers.

The PR caller checks out the triggering merge configuration without persisted credentials and explicitly passes the read-only workflow token to plan. Its path filter covers `.github/settings.yml`, its own workflow, and the GitHub Actions examples. It supplies no App credentials or Environment, and per-PR concurrency cancels superseded runs.

The main caller runs only on a push to `main` changing `.github/settings.yml` or `settings-main.yml`. Its single `apply` job references Environment `gh-settings` and contains three steps: checkout of the triggering `github.sha`, App token creation, and direct apply. Repository-scoped concurrency does not cancel an in-progress apply.

The token-creation step references `vars.ANSWER26_CLIENT_ID` and `secrets.ANSWER26_PRIVATE_KEY`, limits installation scope to the triggering owner/repository, and requests Administration write. The resulting token is passed only to the Action's apply input; the token Action owns its normal post-job revocation. Source checks verify those bindings, not the remote App installation or actual token revocation.

There is no prepare/handoff job or main-side pre-approval plan. The PR Summary and exact committed configuration provide review context; direct apply does not consume a saved plan and may overwrite intervening changes to managed values. The presence of `environment: gh-settings` identifies the selected Environment but does not establish whether reviewers are currently configured.

## Active repository Environment guard

The [guard workflow](../../.github/workflows/settings-environment-guard.yml) checks out trusted `main` policy into `.policy` and the candidate PR merge into `.candidate`. The candidate is inspected only as YAML data. The job has a read-only token, installs Go using trusted configuration, and runs the main-branch [scope test](../../tests/github-actions/settings_environment_scope_test.go) with `GH_SETTINGS_WORKFLOWS_DIR` pointing at the candidate workflows.

The policy requires exactly one literal `gh-settings` Environment reference, in `.github/workflows/settings-main.yml` under job `apply`. It also verifies that this workflow's only event is a push to the exact `main` branch. It rejects additional `gh-settings` references, dynamic Environment names, workflow/job aliases or merges, duplicate mapping keys, dynamic job names, and job-level reusable-workflow calls that could select an unseen Environment. Unrelated literal Environment names are allowed.

The dedicated check name is **`gh-settings`**. The policy reserves that job ID/display name for the guard and rejects collisions elsewhere in inspected workflows. The ordinary Go suite also runs the scope assertion and contributes to `pass`.

These checks detect violations in the inspected source, but the required status name does not identify a unique workflow. GitHub documents that [required checks do not take the workflow or event trigger into account](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks). A same-named GitHub Actions job can therefore be ambiguous; the name-reservation test only applies when the trusted policy actually executes.

When evaluating enforcement, inspect the applicable required checks and allowed check providers, actual guard run, Environment deployment branches/reviewers, and [`pull_request_target` event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target). The repository YAML does not establish those external settings or enforce policy across other repositories. This guide records the guard's behavior and limits without prescribing a change to external approval configuration.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) reacts only to a PR comment from `chatgpt-codex-connector[bot]` whose body starts with the expected review-summary marker. It fetches that summary and requires both **Code Review** and **Security Review** to say **Completed**, plus a Codex 👍 reaction to the PR body. It waits up to three minutes for the reaction, then may create an approving bot review.

The workflow executes trusted default-branch code without checking out PR source. It becomes available for comment events after the workflow exists on the default branch. GitHub must also allow Actions to create and approve pull requests for the final approval call to succeed.

The mechanism does not verify the reviewed HEAD or the latest Ready-triggered review cycle, and an older completion/reaction can satisfy its conditions. A successful workflow run alone does not prove that an approval was created. Inspect the actual current review, relevant findings, and resulting approval separately.

## Inspect external configuration

Use [repository rules](https://github.com/daiksud/gh-settings/rules), [Environments](https://github.com/daiksud/gh-settings/settings/environments), and [Actions settings](https://github.com/daiksud/gh-settings/settings/actions) when diagnosing blocked or unexpected execution. Confirm the required check results for the intended commit, the selected Environment's restrictions, the App installation/credential bindings, and whether GitHub permits the workflow event.

[Operational fixtures](../../tests/github-actions/operational_test.go) and the [test evidence boundaries](testing.md#release-and-publication-evidence) help distinguish source correctness from those external settings. Release dispatch and recovery have their own [maintainer runbook](releases.md).
