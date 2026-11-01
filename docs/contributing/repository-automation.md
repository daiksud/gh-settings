# Repository automation

This guide is for maintainers of **daiksud/gh-settings**. For installing gh-settings workflows in another repository, use the [GitHub Actions guide](../github-actions.md) and [examples](../../examples/github-actions/README.md). The public guide explains [how to protect the apply path](../github-actions.md#protect-the-apply-path); this page maps those responsibilities to this repository's workflows and controls.

The workflow files describe source-controlled behavior. Required reviews/checks, Environment protection rules, App installations, secrets, and event-policy eligibility are configured separately on GitHub; inspect those settings when assessing the current operational state.

## Workflow inventory

| Workflow | Trigger and responsibility |
| --- | --- |
| [`ci-pr.yml`](../../.github/workflows/ci-pr.yml) | PR events run staged code validation and publish the native `ci-pr` aggregate check |
| [`prlint.yml`](../../.github/workflows/prlint.yml) | PR creation, push, reopening, and edits validate the current title/body through `prlint` |
| [`ci-push.yml`](../../.github/workflows/ci-push.yml) | Pushes to `main`; validate source and the built candidate, ending in `pass` |
| [`labeler.yml`](../../.github/workflows/labeler.yml) | PR-target events suggest add-only labels for unambiguous all-files path categories without executing candidate code |
| [`settings-pr.yml`](../../.github/workflows/settings-pr.yml) | Relevant PR path changes; read-only settings plan and review Summary |
| [`settings-plan.yml`](../../.github/workflows/settings-plan.yml) | Credential-free validation of the merge declaration with its pinned operational consumer |
| [`settings-main.yml`](../../.github/workflows/settings-main.yml) | Pushes to `main` affecting settings or this workflow; one Environment-bound direct apply job |
| [`settings-env.yml`](../../.github/workflows/settings-env.yml) | PR target events and pushes to `main`; inspect workflow YAML with trusted source, reporting `settings-env` |
| [`approve-codex-review.yml`](../../.github/workflows/approve-codex-review.yml) | Codex summary updates or PR CI completion; approve on thumbs-up and both Completed statuses |
| [`release.yml`](../../.github/workflows/release.yml) | Explicit manual dispatch from `main`; shared resolve, commit, acceptance, release, and verification orchestration delegates product work to local Actions |

Release publication has no Issue-comment bridge. The owner starts [`release.yml`](../../.github/workflows/release.yml) directly with `workflow_dispatch` after main verification; version calculation happens within the workflow. This repository's other `issue_comment` workflows do not authorize or dispatch releases.

## PR lifecycle

Every PR event sets Draft, resolves the static build and acceptance matrices, runs commit-stage, then marks the PR Ready before process and public-Action acceptance. Both acceptance legs verify the same manifested binary from commit-stage without rebuilding it. The final native `ci-pr` aggregate job succeeds only when every prerequisite succeeded. Title/body edits rerun the same stages rather than introducing skipped-success required checks.

The independent `prlint` job checks the current title and body together as the proposed squash message. GitHub's required-check settings enforce this separately from the code reporter; the reporter does not query or enforce rulesets. See [PR validation](pr-validation.md) for the independent checks, metadata separation, and failure behavior.

Jobs executing PR source have read permissions, no supplied secrets, and no persisted checkout credentials. Draft/Ready and code-status jobs have narrowly scoped write permissions but do not check out or execute PR code, local composites, or transferred artifacts. Fork token restrictions can block automatic state/status changes; the workflow does not bypass them.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results. GitHub's protected-main and Environment configuration determine whether a reviewed change can reach apply; a passing source test does not configure those controls.

## Protected main and required checks

The [declared default-branch Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) selects the existing Ruleset by ID in [settings.yml](../../.github/settings.yml): one approval, stale approval dismissal, mandatory CODEOWNER approval, resolved review threads, signed and linear history, and strict required checks. It requires `ci-pr`, `prlint`, `settings-plan` and `settings-env` from GitHub Actions. Repository merge configuration allows squash only. The declaration retains the existing repository-administrator bypass in `for_pull_requests_only` mode, CodeQL with Alerts `errors` and Security alerts `high_or_higher`, and code-quality severity `errors`. Retaining this configured bypass does not authorize its use: every change must satisfy ordinary approval, CI, signature, and resolved-thread rules; see the [review policy](../../AGENTS.md#code-review). Do not add or broaden bypass actors. The trusted main workflow applies this declaration only after merge; inspect the actual live Ruleset and Actions permission rather than assuming that source is already enforced.

| Required check | Source and purpose | Limit |
| --- | --- | --- |
| `ci-pr` | [PR code CI](../../.github/workflows/ci-pr.yml) aggregates the build and acceptance stages | PR source controls its tests and CI; a successful result is not independent authorization for privileged code |
| `prlint` | The [independent PRLint workflow](../../.github/workflows/prlint.yml) checks the current title/body | Syntax validation does not establish semantic type, coherent scope, or disclosed compatibility impact |
| `settings-plan` | [Settings plan workflow](../../.github/workflows/settings-plan.yml) checks the PR declaration with its pinned operational consumer, without write credentials | Live differences are observed by the separate privileged settings-pr workflow when relevant |
| `settings-env` | The [trusted guard](../../.github/workflows/settings-env.yml) uses main-branch policy to inspect candidate workflow YAML as data | It checks the limited Environment/trigger/name policy below |

See [required-check setup](../github-actions.md#protect-main-with-required-checks) for branch freshness and trusted check-source requirements. Requiring these names from GitHub Actions does not by itself prove that their intended workflows evaluated the proposed change.

After a permitted merge, a matching main push starts the settings workflow. Whether the selected Environment adds a separate deployment approval depends on its live reviewer configuration. If PR review is intended to be the only manual approval, apply the [documented conditions for that mode](../github-actions.md#when-pr-review-is-the-only-manual-approval) and [verify the protection](../github-actions.md#verify-the-protection-before-use), including check-source identity. Neither a source comment nor `environment: gh-settings` establishes whether a reviewer gate is currently enabled.

## Active settings workflows

The active callers use independently pinned released packages; see the [published package and consumer pins](../github-actions.md#published-package-and-consumer-pins) for adoption requirements. The PR plan pin is selected independently from the main apply and merge-validation pins: trusted default-branch code must understand a candidate declaration before that declaration can be adopted. Main apply and merge validation must stay compatible with the tracked declaration. Their pins are independent of the local development [`action/cli-version`](../../action/cli-version); the invoked published tag or its matching pinned SHA selects the CLI. Publishing a release or changing examples does not automatically update these callers.

The PR caller uses trusted default-branch `pull_request_target` code for opened, synchronize and reopened events targeting main. Its path filter covers `.github/settings.yml`, both operational caller workflows, and the GitHub Actions examples. The exact main-ref/repository/base condition and Environment `gh-settings` constrain this credential consumer. Per-PR concurrency cancels superseded plans.

Before minting a token, the caller anonymously verifies this public repository's fixed identity, fetches only filtered commit/tree metadata for the event's PR head, and downloads the single regular `.github/settings.yml` blob. It validates the fetched head, file mode, bounded size and Git blob hash. It creates a configuration-only consumer with candidate Git identity; no PR code, hooks, local Actions, dependency configuration or scripts are checked out or executed. Public API rate limits, a visibility/identity change, an unavailable blob or a moved PR head fail the preparation; there is no credential fallback. This public-only retrieval deliberately needs no Contents grant on ANSWER26.

The plan job then mints a short-lived repository-scoped token from the same ANSWER26 App used by main apply, requesting Administration write and Issues read, and passes it explicitly to the pinned released Action. GitHub's [Ruleset GET contract](https://docs.github.com/en/rest/repos/rules#get-a-repository-ruleset) returns bypass actors only to callers with Ruleset write access. Nonmutating plan therefore needs a write-capable token for complete observation; it is not a read-only credential. This token can change repository administration settings if trusted executing code misuses it. The existing installation grants are unchanged, and its repository selection is limited to this repository. No additional Environment consumer or API mutation is authorized by this permission request. Feature adoption must still verify complete observation, endpoint support and installation permissions; the broader token alone does not establish that a Ruleset is safely manageable. Local fixtures establish data isolation and failure handling, not effective remote grants or successful live execution. A PR changing this caller itself runs the previous default-branch caller; subsequent relevant events must verify the updated trusted plan.

The main caller runs only on a push to `main` changing `.github/settings.yml` or `settings-main.yml`. Its single `apply` job references Environment `gh-settings` and contains three steps: checkout of the triggering `github.sha`, App token creation, and direct apply. Repository-scoped concurrency does not cancel an in-progress apply.

The token-creation step references `vars.ANSWER26_CLIENT_ID` and `secrets.ANSWER26_PRIVATE_KEY`, limits installation scope to the triggering owner/repository, and requests Administration write and Issues write. The resulting token is passed only to the Action's apply input; the token Action owns its normal post-job revocation. Source checks verify those bindings, not the remote App installation or actual token revocation.

The operational callers manage the twenty existing labels in [this repository's configuration](../../.github/settings.yml) by their verified numeric IDs, with every supported value explicit. Unlisted objects remain unmanaged and are never implicitly deleted. The main caller requests Issues write for label mutations; installation grants are unchanged. This permission also permits operations on issues, comments, assignees and milestones. See the [label permission contract](../github-actions.md#labels-and-permissions) before changing label definitions or token requests.

The configuration also selects the existing `gh-settings` and `copilot`
Environments by name with complete supported observations. `gh-settings`
retains administrator bypass disabled and only the typed Branch `main` pattern;
`copilot` retains administrator bypass enabled and no branch/tag restriction.
Both retain zero wait time and no reviewers. Unlisted Environment names remain
unmanaged. A fresh complete match skips Environment writes; a prior plan is
not a lock. Read back the policy and typed patterns after any apply rather than
inferring effective deployment security from a successful workflow alone.

When the fresh labels comparison performed by apply has no operations, that invocation sends no label mutations. An earlier zero-difference PR plan does not guarantee that result: labels can change before apply reads them. It also does not make the whole apply write-free, because declared repository fields and topics retain their direct PATCH/PUT behavior. A labels read or mutation failure occurs after those earlier writes and does not roll them back. Any claim that an adoption or run performed no label writes needs evidence for that exact execution, not just a prior plan or the generic success message.

There is no prepare/handoff job or main-side pre-approval plan. The PR Summary and exact committed configuration provide review context; direct apply does not consume a saved plan and may overwrite intervening changes to managed values. The presence of `environment: gh-settings` identifies the selected Environment but does not establish whether reviewers are currently configured.

## Active repository Environment guard

The [guard workflow](../../.github/workflows/settings-env.yml) checks out trusted `main` policy into `.policy` and the candidate PR merge into `.candidate`. The candidate is inspected only as YAML data. The job has a read-only token, installs Go using trusted configuration, and runs the main-branch [scope test](../../tests/github-actions/settings_environment_scope_test.go) with `GH_SETTINGS_WORKFLOWS_DIR` pointing at the candidate workflows.

The `gh-settings` Environment allowlist contains exactly two locations: `.github/workflows/settings-main.yml/jobs.apply` on push to main, and `.github/workflows/settings-pr.yml/jobs.plan` on trusted `pull_request_target` targeting main. The PR job must have the exact event/main-ref/repository/base condition, and only opened/synchronize/reopened activities are permitted. All other `gh-settings` references are rejected. The existing main reference remains mandatory; the new PR reference may be absent while the guard is deployed before its caller, but absence cannot grant another path access. Dynamic Environment names, workflow/job aliases or merges, duplicate mapping keys, dynamic job names and job-level reusable-workflow calls remain rejected. The retired `gh-settings-plan` Environment is forbidden in every workflow. Unrelated literal Environment names retain their existing boundaries.

The guard was deployed before the operational PR credential consumer. GitHub [evaluates `pull_request_target` using the default branch](https://github.blog/changelog/2025-11-07-actions-pull_request_target-and-environment-branch-protections-changes/), including Environment branch matching; existing main-only deployment restrictions do not need to be widened. The permitted PR job still needs trusted executing code and candidate YAML treated only as data; path/event allowlisting does not inspect every step body. The allowlist does not configure live Environment restrictions or issue tokens itself.

The guard does not inspect the allowed apply job's step bodies, approve Action revisions, restrict its API operations, validate actual token grants, or configure remote Environment/ruleset settings. The allowed job could still be changed to execute code with privileged credentials. Review that job, its referenced code, and changes to the guard/policy itself under the [apply-path protection guidance](../github-actions.md#protect-the-apply-path); later runs use whatever policy has been merged into `main`.

The dedicated check name is **`settings-env`**. The policy reserves that job ID/display name for the guard and rejects collisions elsewhere in inspected workflows. The ordinary Go suite also runs the scope assertion and contributes to `ci-pr` or `ci-push`.

These checks detect violations in the inspected source, but the required status name does not identify a unique workflow. GitHub documents that [required checks do not take the workflow or event trigger into account](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks). Selecting GitHub Actions as the required check provider identifies the app, not this guard's workflow file. Another Actions job can report the same name, so a green `settings-env` result alone is not proof that the trusted guard ran. The name-reservation test prevents collisions only when the trusted policy actually executes.

When evaluating enforcement, inspect the applicable required checks and allowed check providers, actual guard run, Environment deployment branches/reviewers, and [`pull_request_target` event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target). The repository YAML does not establish those external settings or enforce policy across other repositories. This guide records the guard's behavior and limits without prescribing a change to external approval configuration.

## Local PR settings validation

The required `settings-plan` job in [settings-plan.yml](../../.github/workflows/settings-plan.yml)
validates `.github/settings.yml` from the PR merge checkout with the same immutable
public Action pin used by [settings-pr.yml](../../.github/workflows/settings-pr.yml)
and [settings-main.yml](../../.github/workflows/settings-main.yml). It calls only
`command: validate`, with no token. The job uses the public Action's supported
fresh Ubuntu 24.04 x64 VM, has only Contents read, selects no Environment and
creates no App token. A validation failure fails the required check.

This checks the operational declaration against its actual package contract.
The separate required `ci-pr` builds and tests the candidate source, including
current-format process and public Action fixtures. The candidate decoder still
rejects obsolete configuration; no compatibility decoder or translation is used.
The tracked declaration uses the current UI contract. Validation and plan do
not apply it; the main caller performs direct apply only after a permitted
merge.

Neither check establishes live App visibility or drift. The separate
[trusted PR plan](#active-settings-workflows) performs authenticated observation;
no validation result authorizes a later apply or locks remote state.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) submits an approval when both conditions hold:

1. Codex has reacted with 👍 on the PR.
2. The latest Codex Review Summary marks **Code Review** and **Security Review** as **Completed**.

The workflow reads the visible Summary table: the status cell for each of **Code Review** and **Security Review** must contain **Completed**. Hidden JSON markers are not approval conditions. It does not inspect HEAD/base, timestamps, receipts, finding flags, prior requests or another reviewer's changes-requested decision. A completed summary and retained thumbs-up may therefore be reused across changed commits or comparisons; the workflow makes no claim that the current diff was reviewed.

Summary creation/edits and completion of `ci-pr` trigger reevaluation. CI success and the PR's source repository are not additional approval conditions. GitHub does not provide a reaction trigger here, so the workflow polls up to three minutes for a delayed thumbs-up. The exact Codex login `chatgpt-codex-connector[bot]` identifies both the summary author and the reaction actor; numeric account IDs and App metadata are not required. The newest summary is selected by comment ID. These identify the requested Codex signals rather than add another review gate.

Each polling attempt reads the latest Summary and the current reactions. Immediately before submission, it reads the same two conditions again; changed bodies or reaction IDs are not additional gates. When both conditions hold, it submits `event=APPROVE` without a commit SHA, so GitHub selects the PR's latest commit. Repeated matching events can submit another approval; the workflow does not read review history or suppress duplicates based on prior decisions. Failed or malformed API responses are reported without claiming a successful approval. The workflow runs default-branch code without checking out PR source and uses only its existing pull-request write permission.

Maintainer decision (2026-11-01): unresolved findings or review conversations do not add an approval condition when Codex thumbs-up and both Completed statuses are present. The repository owner accepts this tradeoff and keeps clean-finding flags, clean receipts and thread resolution outside this workflow's decision. Whether conversations must be resolved before merging is determined by the target's effective Rulesets. With `required_review_thread_resolution` enabled, submitting an approval does not remove that merge requirement.

Conversation resolution rules apply to GitHub review threads, not every hidden Codex finding flag. Rulesets do not resolve or dismiss threads automatically; maintainers and reviewers record the disposition and resolve threads when appropriate. A risk-acceptance reply records that decision without marking a thread resolved. This responsibility split does not claim that Rulesets independently certify the review evidence as clean.

The controls have separate responsibilities:

| Control | Responsibility |
| --- | --- |
| GitHub Rulesets | Determine merge eligibility: required approvals, stale-approval dismissal, required checks, branch freshness and other configured rules |
| Codex approval workflow | Interpret Codex thumbs-up and both Completed statuses, then submit an approving review |
| Maintainer/reviewer | Assess changes to the comparison or review evidence, resolve findings, and verify effective rules and required checks before a normal merge |

With stale-approval dismissal enabled, GitHub evaluates changes to the approved **diff**, not simply elapsed time or a changed SHA. Pushes, Update branch, or related changes merged into the target can change that diff; a merge-base change introducing new changes can also dismiss approval. Strict required checks separately require the branch to be up to date before merging. A target-branch advance is not a blanket claim that every approval is dismissed. See GitHub's [Ruleset review requirements](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-a-pull-request-before-merging) and [strict required checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-status-checks-to-pass-before-merging).

The workflow does not reproduce those merge gates. Its approval does not establish that a changed comparison received a new review or that the PR can merge. A subsequent run can submit another approval after dismissal if the two Codex signals still exist; Rulesets do not validate the provenance or freshness of those signals. GitHub evaluates applicable merge rules independently of this workflow's decision.

Read-only verification on 2026-11-01 found this repository's [active main Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) enables stale-review dismissal, required review-thread resolution and strict required checks. Treat this as a dated observation, not a guarantee of future enforcement. In the shared five-repository audit, all five had stale-review dismissal; gh-settings, gh-qw, agents and apex also had strict required checks. elite-miko-engineer's active Ruleset had no required-status-check rule, so that Ruleset does not establish latest-base enforcement. Its classic branch-protection read returned 403, leaving any additional protection unverified; this is not evidence that no other protection exists. No setting was changed or compensated for by the approval workflow. Recheck the actual target's effective rules before relying on them.

Logs distinguish approval, withholding because a required Codex signal is absent, and API/execution failure. A successful workflow that reports withholding has not approved the PR. Recheck live Ruleset approvals, review threads and required CI before merging. The retained administrator bypass must not be used to merge; this workflow does not authorize settings changes or a release.

## Inspect external configuration

Use [repository rules](https://github.com/daiksud/gh-settings/rules), [Environments](https://github.com/daiksud/gh-settings/settings/environments), and [Actions settings](https://github.com/daiksud/gh-settings/settings/actions) when diagnosing blocked or unexpected execution. Follow [verification before use](../github-actions.md#verify-the-protection-before-use): confirm the required check results and their actual workflow/source for the intended commit, the selected Environment's branch/tag and reviewer restrictions, bypass settings, App installation/credential bindings, and whether GitHub permits the workflow event.

[Operational fixtures](../../tests/github-actions/operational_test.go) and the [test evidence boundaries](testing.md#release-and-publication-evidence) help distinguish source correctness from those external settings. Release dispatch and recovery have their own [maintainer runbook](releases.md).
