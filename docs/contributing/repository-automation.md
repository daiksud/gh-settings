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
| [`approve-codex-review.yml`](../../.github/workflows/approve-codex-review.yml) | Codex summary updates or successful PR CI completion; conditionally create a bot approval |
| [`release.yml`](../../.github/workflows/release.yml) | Explicit manual dispatch from `main`; shared resolve, commit, acceptance, release, and verification orchestration delegates product work to local Actions |

Release publication has no Issue-comment bridge. The owner starts [`release.yml`](../../.github/workflows/release.yml) directly with `workflow_dispatch` after main verification; version calculation happens within the workflow. This repository's other `issue_comment` workflows do not authorize or dispatch releases.

## PR lifecycle

Every PR event sets Draft, resolves the static build and acceptance matrices, runs commit-stage, then marks the PR Ready before process and public-Action acceptance. Both acceptance legs verify the same manifested binary from commit-stage without rebuilding it. The final native `ci-pr` aggregate job succeeds only when every prerequisite succeeded. Title/body edits rerun the same stages rather than introducing skipped-success required checks.

The independent `prlint` job checks the current title and body together as the proposed squash message. GitHub's required-check settings enforce this separately from the code reporter; the reporter does not query or enforce rulesets. See [PR validation](pr-validation.md) for the independent checks, metadata separation, and failure behavior.

Jobs executing PR source have read permissions, no supplied secrets, and no persisted checkout credentials. Draft/Ready and code-status jobs have narrowly scoped write permissions but do not check out or execute PR code, local composites, or transferred artifacts. Fork token restrictions can block automatic state/status changes; the workflow does not bypass them.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results. GitHub's protected-main and Environment configuration determine whether a reviewed change can reach apply; a passing source test does not configure those controls.

## Protected main and required checks

The [declared default-branch Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) selects the existing Ruleset by ID in [settings.yml](../../.github/settings.yml): one approval, stale approval dismissal, mandatory CODEOWNER approval, resolved review threads, signed and linear history, and strict required checks. It requires `ci-pr`, `prlint`, `settings-plan` and `settings-env` from GitHub Actions. Repository merge configuration allows squash only. No bypass actors are declared, including administrators. Every change must satisfy ordinary approval, CI, signature, and resolved-thread rules; see the [review policy](../../AGENTS.md#code-review). The trusted main workflow applies this declaration only after merge; inspect the actual live Ruleset and Actions permission rather than assuming that source is already enforced.

| Required check | Source and purpose | Limit |
| --- | --- | --- |
| `ci-pr` | [PR code CI](../../.github/workflows/ci-pr.yml) aggregates the build and acceptance stages | PR source controls its tests and CI; a successful result is not independent authorization for privileged code |
| `prlint` | The [independent PRLint workflow](../../.github/workflows/prlint.yml) checks the current title/body | Syntax validation does not establish semantic type, coherent scope, or disclosed compatibility impact |
| `settings-plan` | [Settings plan workflow](../../.github/workflows/settings-plan.yml) checks the PR declaration with its pinned operational consumer, without write credentials | Live differences are observed by the separate privileged settings-pr workflow when relevant |
| `settings-env` | The [trusted guard](../../.github/workflows/settings-env.yml) uses main-branch policy to inspect candidate workflow YAML as data | It checks the limited Environment/trigger/name policy below |

See [required-check setup](../github-actions.md#protect-main-with-required-checks) for branch freshness and trusted check-source requirements. Requiring these names from GitHub Actions does not by itself prove that their intended workflows evaluated the proposed change.

After a permitted merge, a matching main push starts the settings workflow. Whether the selected Environment adds a separate deployment approval depends on its live reviewer configuration. If PR review is intended to be the only manual approval, apply the [documented conditions for that mode](../github-actions.md#when-pr-review-is-the-only-manual-approval) and [verify the protection](../github-actions.md#verify-the-protection-before-use), including check-source identity. Neither a source comment nor `environment: gh-settings` establishes whether a reviewer gate is currently enabled.

## Active settings workflows

The active callers use independently pinned released packages; see the [published package and consumer pins](../github-actions.md#published-package-and-consumer-pins) for adoption requirements. Their pins are independent of the local development [`action/cli-version`](../../action/cli-version); the invoked published tag or its matching pinned SHA selects the CLI. Publishing a release or changing examples does not automatically update these callers.

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
[#382](https://github.com/daiksud/gh-settings/issues/382) owns the repository's
new-format declaration and compatible package adoption. This validation split
does not rewrite or apply that declaration.

Neither check establishes live App visibility or drift. The separate
[trusted PR plan](#active-settings-workflows) performs authenticated observation;
no validation result authorizes a later apply or locks remote state.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) reacts to trusted summary updates from `chatgpt-codex-connector[bot]` and rechecks the most recent summary after successful PR CI completion. Both **Code Review** and **Security Review** must have completed on the *current exact PR HEAD*. The workflow verifies the Bot account and GitHub App identities, rejects unresolved or unknown code findings, and requires a new Codex 👍 created **after** the summary's final update. An old reaction on the same HEAD is insufficient. It waits up to three minutes for a delayed reaction; a same-second reaction remains ambiguous because GitHub's timestamps have second precision.

An explicit `false` security finding flag is accepted. A `null` flag is never interpreted as `false`: it requires separate positive evidence from the current manual review cycle. The latest unedited `@codex review` request must precede exactly one unedited code receipt and one unedited security receipt from the trusted App, both explicitly reporting no findings. Their creation times must fall before the corresponding manual completion times in the current summary. Both abbreviated commit IDs must resolve unambiguously to the full current HEAD. The structured security metadata must also identify this repository, PR and completed HEAD. An unfinished earlier request, automatic cycle without an identifiable boundary, missing or duplicate receipt, edited evidence, contradictory result or unfamiliar format withholds approval. This narrow receipt path supports the observed nullable security summary; it does not establish a general meaning for `null` or `mergeGateEnabled: false`.

Every review-request update must precede the completed summary, including edits to older requests. The workflow reads the complete PR timeline for manual or automatic base retargets. After any retarget, it requires a new unedited `@codex review` request created strictly after every retarget event; a HEAD-only summary without that boundary is insufficient. A same-second retarget/request remains ambiguous.

Immediately before submitting, the workflow rereads the relevant comments, base-retarget events, reaction and PR HEAD. The base branch name and commit must also match the initial observation. A new request or changed evidence on the same SHA withholds approval. It handles draft and ready PRs, refuses closed/stale heads and outstanding change requests, avoids a duplicate same-HEAD bot approval, and pins the approving review to the verified SHA. Logs distinguish approval, an existing approval and a withheld approval with its reason. Failed or malformed API reads fail the job; a successful job that reports a withheld approval has not approved the PR.

The workflow runs trusted default-branch code without checking out PR source. GitHub Actions must be permitted to approve PRs, and a submitted approval is distinct from Codex's comment/reaction or workflow success. Recheck live Ruleset approvals, any review threads, and required CI for the final HEAD. No administrative bypass is permitted; an unavailable reviewer leaves the PR blocked until an eligible approval is obtained.

## Inspect external configuration

Use [repository rules](https://github.com/daiksud/gh-settings/rules), [Environments](https://github.com/daiksud/gh-settings/settings/environments), and [Actions settings](https://github.com/daiksud/gh-settings/settings/actions) when diagnosing blocked or unexpected execution. Follow [verification before use](../github-actions.md#verify-the-protection-before-use): confirm the required check results and their actual workflow/source for the intended commit, the selected Environment's branch/tag and reviewer restrictions, bypass settings, App installation/credential bindings, and whether GitHub permits the workflow event.

[Operational fixtures](../../tests/github-actions/operational_test.go) and the [test evidence boundaries](testing.md#release-and-publication-evidence) help distinguish source correctness from those external settings. Release dispatch and recovery have their own [maintainer runbook](releases.md).
