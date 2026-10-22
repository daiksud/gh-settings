# Repository automation

This guide is for maintainers of **daiksud/gh-settings**. For installing gh-settings workflows in another repository, use the [GitHub Actions guide](../github-actions.md) and [examples](../../examples/github-actions/README.md). The public guide explains [how to protect the apply path](../github-actions.md#protect-the-apply-path); this page maps those responsibilities to this repository's workflows and controls.

The workflow files describe source-controlled behavior. Required reviews/checks, Environment protection rules, App installations, secrets, and event-policy eligibility are configured separately on GitHub; inspect those settings when assessing the current operational state.

## Workflow inventory

| Workflow | Trigger and responsibility |
| --- | --- |
| [`ci-pr.yml`](../../.github/workflows/ci-pr.yml) | PR creation, push, reopening, and base edits run code validation; metadata-only edits do not alter the reported `pass` result |
| [`pr-message.yml`](../../.github/workflows/pr-message.yml) | PR creation, push, reopening, and edits validate the current title/body through `pr-message` |
| [`ci-push.yml`](../../.github/workflows/ci-push.yml) | Pushes to `main`; validate source and the built candidate, ending in `pass` |
| [`labeler.yml`](../../.github/workflows/labeler.yml) | PR-target events suggest add-only labels for unambiguous all-files path categories without executing candidate code |
| [`settings-pr.yml`](../../.github/workflows/settings-pr.yml) | Relevant PR path changes; read-only settings plan and review Summary |
| [`settings-main.yml`](../../.github/workflows/settings-main.yml) | Pushes to `main` affecting settings or this workflow; one Environment-bound direct apply job |
| [`settings-environment-guard.yml`](../../.github/workflows/settings-environment-guard.yml) | PR target events and pushes to `main`; inspect workflow YAML with trusted source, reporting `gh-settings` |
| [`approve-codex-review.yml`](../../.github/workflows/approve-codex-review.yml) | Qualifying Codex summary comments; conditionally create a bot approval |
| [`release.yml`](../../.github/workflows/release.yml) | Explicit manual dispatch from `main`; [versioned release procedure](releases.md) |

## PR lifecycle

A code-changing PR event marks `pass` pending, sets Draft, runs commit-stage, then marks the PR Ready before acceptance-stage and composite-smoke. The final API-only reporter publishes the actual code result only while the PR head/base still match and every required code job succeeded. A title/body-only edit skips this path without writing a new `pass` status or repeating Draft/Ready changes. Metadata edits use a different concurrency group so they do not cancel code validation.

The independent `pr-message` job checks the current title and body together as the proposed squash message. Both code and message validation must be required independently; the code reporter refuses success until the effective base-branch rules require `pr-message` from GitHub Actions. See [PR validation](pr-validation.md) for the activation sequence, status semantics, and failure behavior.

Jobs executing PR source have read permissions, no supplied secrets, and no persisted checkout credentials. Draft/Ready and code-status jobs have narrowly scoped write permissions but do not check out or execute PR code, local composites, or transferred artifacts. Fork token restrictions can block automatic state/status changes; the workflow does not bypass them.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results. GitHub's protected-main and Environment configuration determine whether a reviewed change can reach apply; a passing source test does not configure those controls.

## Protected main and required checks

The inspected [default-branch Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) requires one approving review, dismisses stale approvals when new commits are pushed, requires CODEOWNER review and resolved review threads, and requires strict `pass`, `gh-settings`, and `pr-message` checks from the GitHub Actions app. The existing administrator-role bypass is available for the owner's bounded [review-quota fallback](../../AGENTS.md#code-review); that exception does not waive required verification. Recheck the live Ruleset and bypass actors when assessing enforcement: these are GitHub settings, not properties created by workflow files.

| Required check | Source and purpose | Limit |
| --- | --- | --- |
| `pass` | [PR code CI](../../.github/workflows/ci-pr.yml) reports source, acceptance, and composite results; metadata-only edits do not overwrite it | PR source controls its tests and CI; a successful result is not independent authorization for privileged code |
| `pr-message` | The [independent message workflow](../../.github/workflows/pr-message.yml) checks the current title/body separately from code validation | Syntax validation does not establish semantic type, coherent scope, or disclosed compatibility impact |
| `gh-settings` | The [trusted guard](../../.github/workflows/settings-environment-guard.yml) uses main-branch policy to inspect candidate workflow YAML as data | It checks the limited Environment/trigger/name policy below, and the required check's app/name does not identify a unique workflow |

See [required-check setup](../github-actions.md#protect-main-with-required-checks) for branch freshness and trusted check-source requirements. Requiring these names from GitHub Actions does not by itself prove that their intended workflows evaluated the proposed change.

After a permitted merge, a matching main push starts the settings workflow. Whether the selected Environment adds a separate deployment approval depends on its live reviewer configuration. If PR review is intended to be the only manual approval, apply the [documented conditions for that mode](../github-actions.md#when-pr-review-is-the-only-manual-approval) and [verify the protection](../github-actions.md#verify-the-protection-before-use), including check-source identity. Neither a source comment nor `environment: gh-settings` establishes whether a reviewer gate is currently enabled.

## Active settings workflows

The active callers pin the released package shown in the [published package and consumer pins](../github-actions.md#published-package-and-consumer-pins). Their pins are independent of [`action/cli-version`](../../action/cli-version), which selects the CLI packaged by current source. Publishing a release or changing examples does not automatically update these callers.

The PR caller checks out the triggering merge configuration without persisted credentials and explicitly passes the read-only workflow token to plan. Its path filter covers `.github/settings.yml`, its own workflow, and the GitHub Actions examples. It supplies no App credentials or Environment, and per-PR concurrency cancels superseded runs.

The main caller runs only on a push to `main` changing `.github/settings.yml` or `settings-main.yml`. Its single `apply` job references Environment `gh-settings` and contains three steps: checkout of the triggering `github.sha`, App token creation, and direct apply. Repository-scoped concurrency does not cancel an in-progress apply.

The token-creation step references `vars.ANSWER26_CLIENT_ID` and `secrets.ANSWER26_PRIVATE_KEY`, limits installation scope to the triggering owner/repository, and requests Administration write and Issues write. The resulting token is passed only to the Action's apply input; the token Action owns its normal post-job revocation. Source checks verify those bindings, not the remote App installation or actual token revocation.

The adopted package supports labels, and [this repository's configuration](../../.github/settings.yml) declares their definitions alongside repository fields and topics. The main caller requests Issues write for label mutations; the App installation must separately grant that permission. This grant also permits operations on issues, comments, assignees, and milestones. The PR caller retains its read-only workflow token. Public label listing can work without Issues or Pull requests grants; label mutations require one of them at write level. A release pin, public listing, or successful read does not establish mutation permission. See the [label permission contract](../github-actions.md#labels-and-permissions) before changing those definitions or token requests.

When the fresh labels comparison performed by apply has no operations, that invocation sends no label mutations. An earlier zero-difference PR plan does not guarantee that result: labels can change before apply reads them. It also does not make the whole apply write-free, because declared repository fields and topics retain their direct PATCH/PUT behavior. A labels read or mutation failure occurs after those earlier writes and does not roll them back. Any claim that an adoption or run performed no label writes needs evidence for that exact execution, not just a prior plan or the generic success message.

There is no prepare/handoff job or main-side pre-approval plan. The PR Summary and exact committed configuration provide review context; direct apply does not consume a saved plan and may overwrite intervening changes to managed values. The presence of `environment: gh-settings` identifies the selected Environment but does not establish whether reviewers are currently configured.

## Active repository Environment guard

The [guard workflow](../../.github/workflows/settings-environment-guard.yml) checks out trusted `main` policy into `.policy` and the candidate PR merge into `.candidate`. The candidate is inspected only as YAML data. The job has a read-only token, installs Go using trusted configuration, and runs the main-branch [scope test](../../tests/github-actions/settings_environment_scope_test.go) with `GH_SETTINGS_WORKFLOWS_DIR` pointing at the candidate workflows.

The policy requires exactly one literal `gh-settings` Environment reference, in `.github/workflows/settings-main.yml` under job `apply`. It also verifies that this workflow's only event is a push to the exact `main` branch. It rejects additional `gh-settings` references, dynamic Environment names, workflow/job aliases or merges, duplicate mapping keys, dynamic job names, and job-level reusable-workflow calls that could select an unseen Environment. Unrelated literal Environment names are allowed.

The guard does not inspect the allowed apply job's step bodies, approve Action revisions, restrict its API operations, validate actual token grants, or configure remote Environment/ruleset settings. The allowed job could still be changed to execute code with privileged credentials. Review that job, its referenced code, and changes to the guard/policy itself under the [apply-path protection guidance](../github-actions.md#protect-the-apply-path); later runs use whatever policy has been merged into `main`.

The dedicated check name is **`gh-settings`**. The policy reserves that job ID/display name for the guard and rejects collisions elsewhere in inspected workflows. The ordinary Go suite also runs the scope assertion and contributes to `pass`.

These checks detect violations in the inspected source, but the required status name does not identify a unique workflow. GitHub documents that [required checks do not take the workflow or event trigger into account](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks). Selecting GitHub Actions as the required check provider identifies the app, not this guard's workflow file. Another Actions job can report the same name, so a green `gh-settings` result alone is not proof that the trusted guard ran. The name-reservation test prevents collisions only when the trusted policy actually executes.

When evaluating enforcement, inspect the applicable required checks and allowed check providers, actual guard run, Environment deployment branches/reviewers, and [`pull_request_target` event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target). The repository YAML does not establish those external settings or enforce policy across other repositories. This guide records the guard's behavior and limits without prescribing a change to external approval configuration.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) reacts only to a PR comment from `chatgpt-codex-connector[bot]` whose body starts with the expected review-summary marker. It fetches that summary and requires both **Code Review** and **Security Review** to say **Completed**, plus a Codex 👍 reaction to the PR body. It waits up to three minutes for the reaction, then may create an approving bot review.

The workflow executes trusted default-branch code without checking out PR source. It becomes available for comment events after the workflow exists on the default branch. GitHub must also allow Actions to create and approve pull requests for the final approval call to succeed.

The mechanism does not verify the reviewed HEAD or the latest Ready-triggered review cycle, and an older completion/reaction can satisfy its conditions. A successful workflow run alone does not prove that an approval was created. Inspect the actual current review, relevant findings, and resulting approval separately.

## Inspect external configuration

Use [repository rules](https://github.com/daiksud/gh-settings/rules), [Environments](https://github.com/daiksud/gh-settings/settings/environments), and [Actions settings](https://github.com/daiksud/gh-settings/settings/actions) when diagnosing blocked or unexpected execution. Follow [verification before use](../github-actions.md#verify-the-protection-before-use): confirm the required check results and their actual workflow/source for the intended commit, the selected Environment's branch/tag and reviewer restrictions, bypass settings, App installation/credential bindings, and whether GitHub permits the workflow event.

[Operational fixtures](../../tests/github-actions/operational_test.go) and the [test evidence boundaries](testing.md#release-and-publication-evidence) help distinguish source correctness from those external settings. Release dispatch and recovery have their own [maintainer runbook](releases.md).
