# Repository automation

This guide is for maintainers of **daiksud/gh-settings**. For installing gh-settings workflows in another repository, use the [GitHub Actions guide](../github-actions.md) and [examples](../../examples/github-actions/README.md). The public guide explains [how to protect the apply path](../github-actions.md#protect-the-apply-path); this page maps those responsibilities to this repository's workflows and controls.

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

The PR CI workflow sets Draft, runs commit-stage, then marks the PR Ready before acceptance-stage and composite-smoke. A separate `commitlint` job checks the PR title and body together as the proposed squash message, including on title/body edits. The final `pass` job rejects failed, cancelled, or skipped validation, including commitlint.

Jobs executing PR source have read permissions, no supplied secrets, and no persisted checkout credentials. Draft/Ready jobs have write permissions but do not check out or execute PR code, local composites, or transferred artifacts. Fork token restrictions can block automatic state changes; the workflow does not bypass them.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results. GitHub's protected-main and Environment configuration determine whether a reviewed change can reach apply; a passing source test does not configure those controls.

## Protected main and required checks

The active [default-branch ruleset](https://github.com/daiksud/gh-settings/rules/24911852) requires one approving review, dismisses stale approvals when new commits are pushed, requires CODEOWNER review and resolved review threads, and requires strict `pass` and `gh-settings` checks from the GitHub Actions App. No bypass actors are configured. Recheck the live ruleset when assessing enforcement: these are GitHub settings, not properties created by the workflow files.

| Required check | Source and purpose | Limit |
| --- | --- | --- |
| `pass` | [PR CI](../../.github/workflows/ci-pr.yml) aggregates commitlint, acceptance, and composite smoke; the ordinary suite also includes the Environment scope assertion | PR source controls its tests and CI; a successful result is not independent authorization for privileged code |
| `gh-settings` | The [trusted guard](../../.github/workflows/settings-environment-guard.yml) uses main-branch policy to inspect candidate workflow YAML as data | It checks the limited Environment/trigger/name policy below, and the required check's App/name does not identify a unique workflow |

See [required-check setup](../github-actions.md#protect-main-with-required-checks) for branch freshness and trusted check-source requirements. Requiring both names from GitHub Actions does not by itself prove that their intended workflows evaluated the proposed change.

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

These checks detect violations in the inspected source, but the required status name does not identify a unique workflow. GitHub documents that [required checks do not take the workflow or event trigger into account](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks). Selecting GitHub Actions as the required check provider identifies the App, not this guard's workflow file. Another Actions job can report the same name, so a green `gh-settings` result alone is not proof that the trusted guard ran. The name-reservation test prevents collisions only when the trusted policy actually executes.

When evaluating enforcement, inspect the applicable required checks and allowed check providers, actual guard run, Environment deployment branches/reviewers, and [`pull_request_target` event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target). The repository YAML does not establish those external settings or enforce policy across other repositories. This guide records the guard's behavior and limits without prescribing a change to external approval configuration.

## Candidate settings preparation (not activated)

The standalone [pr-settings helper](../../scripts/pr-settings/main.go) prepares candidate data for a future trusted observer. No current workflow calls it, selects a new Environment or supplies App credentials. Its implementation does not authorize activation, release publication, credential exposure or token issuance.

Build the helper from a reviewed, exact main revision in a separate trusted source directory. Invoke it from the canonical `GITHUB_WORKSPACE` root, which must not already contain `.git` or `.github` or be inside another Git repository. The trusted source can be under `.trusted`; it must never be PR source. Set an existing ordinary workflow read token in `GH_TOKEN` and an absolute `RUNNER_TEMP` on the same filesystem as the workspace. Contents read, Pull requests read and automatic Metadata read suffice for these preparation APIs; no App token or setting-write grant is needed. The helper does not issue tokens or inspect their actual grants.

The command interface is `pr-settings prepare PR_NUMBER EXPECTED_HEAD_SHA`, then `pr-settings verify PR_NUMBER EXPECTED_HEAD_SHA` after the caller's plan. PR numbers are positive canonical decimal integers and expected heads are full lowercase 40-character hex SHAs. There is no target, URL, file-path, command or Action override. The target is this repository's fixed name and repository ID, with base branch `main` and path `.github/settings.yml`. Public visibility is required because Git metadata is fetched anonymously; visibility changes require a separate design rather than a credential fallback.

Preparation requires an open, same-repository PR with a calculated mergeable test-merge commit; forks, pending/conflicted merges, wrong identities and missing fields fail closed. Git reads fetch commit/tree metadata only with `blob:none`, without checkout, hooks, submodules, credential helpers or inherited Git configuration. Unexpected downloaded blobs are rejected. The real test-merge object's ordered parents must match the observed base/head. Native tree entries must identify a `.github` tree and a nonexecutable regular `settings.yml` blob; symlinks, submodules and missing paths are rejected.

Only the selected blob is read through the fixed GitHub API, with redirects forbidden, a 2 MiB response bound, a 1 MiB configuration bound, and base64/size/Git-object-hash verification. YAML is handled only as bytes at this stage. The future caller must run published configuration validation before requesting any privileged credential. No candidate workflow, script, local Action, attribute filter, build, cache or artifact is executed or restored.

The prepared root contains real candidate Git metadata with the fixed origin and actual merge HEAD, plus only the candidate settings file. A private `.git/gh-settings-plan.json` receipt records verified revisions and blob/content digests; it contains no token or policy values. New directories use mode 0700 and settings/receipt files use 0600. Existing destinations are refused rather than overwritten. Staging is removed and newly owned destinations are removed on ordinary preparation failure; filesystem/forced-termination failures are not transactional cleanup guarantees. Successful stdout consists only of target, PR number, head/base/configuration SHAs and configuration SHA-256, never YAML or API/Git diagnostics.

Preparation checks PR/main state again before returning. Verification checks regular local paths, the file against both its SHA-256 and actual candidate Git blob, origin/HEAD/tree/receipt identity, and current PR/main state. A changed head, base or test-merge commit makes the result stale. Call `verify` with the ordinary read token immediately after plan, and require both steps to succeed before accepting an observation. Neither check locks external state, binds a later apply or makes a saved plan executable. The caller owns the private workspace and must not execute untrusted code or allow other steps to rewrite it.

`scripts/pr-settings` fixtures use real synthetic merge/tree/blob identities and bounded read-only API stand-ins. They cover unsafe paths, fork/identity failures, stale revisions, missing/oversized/forged data, local tampering, preservation/cleanup, inherited Git programs/credentials and refusal of redirects. These tests establish preparation boundaries, not App visibility, a live protected workflow or deployment approval. Existing `gh-settings` Environment/guard restrictions and main apply remain unchanged; a protected observer still needs a compatible published private-reporting package and separately approved Environment/guard and credential paths.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) reacts only to a PR comment from `chatgpt-codex-connector[bot]` whose body starts with the expected review-summary marker. It fetches that summary and requires both **Code Review** and **Security Review** to say **Completed**, plus a Codex 👍 reaction to the PR body. It waits up to three minutes for the reaction, then may create an approving bot review.

The workflow executes trusted default-branch code without checking out PR source. It becomes available for comment events after the workflow exists on the default branch. GitHub must also allow Actions to create and approve pull requests for the final approval call to succeed.

The mechanism does not verify the reviewed HEAD or the latest Ready-triggered review cycle, and an older completion/reaction can satisfy its conditions. A successful workflow run alone does not prove that an approval was created. Inspect the actual current review, relevant findings, and resulting approval separately.

## Inspect external configuration

Use [repository rules](https://github.com/daiksud/gh-settings/rules), [Environments](https://github.com/daiksud/gh-settings/settings/environments), and [Actions settings](https://github.com/daiksud/gh-settings/settings/actions) when diagnosing blocked or unexpected execution. Follow [verification before use](../github-actions.md#verify-the-protection-before-use): confirm the required check results and their actual workflow/source for the intended commit, the selected Environment's branch/tag and reviewer restrictions, bypass settings, App installation/credential bindings, and whether GitHub permits the workflow event.

[Operational fixtures](../../tests/github-actions/operational_test.go) and the [test evidence boundaries](testing.md#release-and-publication-evidence) help distinguish source correctness from those external settings. Release dispatch and recovery have their own [maintainer runbook](releases.md).
