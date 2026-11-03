# Repository automation

This guide is for maintainers of **daiksud/gh-settings**. For installing gh-settings workflows in another repository, use the [GitHub Actions guide](../github-actions.md) and [examples](../../examples/github-actions/README.md). The public guide explains [how to protect the apply path](../github-actions.md#protect-the-apply-path); this page maps those responsibilities to this repository's workflows and controls.

The workflow files describe source-controlled behavior. Required reviews/checks, Environment protection rules, App installations, secrets, and event-policy eligibility are configured separately on GitHub; inspect those settings when assessing the current operational state.

## Repository settings changes

Change this repository's managed settings in
[`.github/settings.yml`](../../.github/settings.yml) through the normal PR
review and squash-merge process. Use declarations supported by the
[pinned operational packages](#active-settings-workflows).

1. Open a configuration PR and inspect its checks. The required `settings-plan`
   check validates the proposed merge configuration with the operational Action
   selected by trusted main code; it does not observe GitHub settings. The same
   `settings-pr` workflow selects a separate authenticated plan for relevant
   file changes. Metadata edits still validate but do not start a plan. Review
   the eligible plan run's Summary for the current PR head.
2. Satisfy the [normal merge requirements](#protected-main-and-required-checks)
   and squash-merge the PR. A push to `main` changing `.github/settings.yml`
   or `.github/workflows/settings-main.yml` starts `settings-main`, which
   checks out that event's exact `github.sha` and runs `apply`. Other changes
   alone do not trigger this apply workflow.
3. Inspect the matching main apply run and read back the affected GitHub
   settings before reporting them as applied. A passing PR validation or
   plan is not evidence of a successful apply, and direct apply does not
   consume a saved PR plan. Check any partial progress before retrying a
   failed or cancelled run.

The hosted workflows use the existing repository-scoped App authentication
and `gh-settings` Environment. They do not depend on a local CLI login or a
manual pre-merge apply. Inspect the actual run and live App/Environment
configuration when diagnosing a blocked apply; see
[active settings workflows](#active-settings-workflows) for those bindings.

### Created resource IDs

For supported resources whose contract treats an entry without an ID as
create-only, the normal main apply can perform that initial creation.
Confirmed creation IDs appear in the apply logs. GitHub Actions disables
[configuration ID writeback](../settings.md#apply-id-writeback), and the
workflow does not commit or push IDs back to the declaration.

Read back the created objects, confirm their IDs, and adopt those IDs in a
follow-up settings PR before another plan or apply of the same resources.
Review that PR's plan to confirm it selects the existing objects instead of
requesting another creation. Rerunning the original ID-less configuration
after creation can fail with a name collision, including after a partial
failure. Recover confirmed IDs before retrying; follow the
[apply contract](../commands.md#apply-behavior) for partial progress and
resource-specific behavior.

## Workflow inventory

| Workflow | Trigger and responsibility |
| --- | --- |
| [`ci-pr.yml`](../../.github/workflows/ci-pr.yml) | PR events run staged code validation and publish the native `ci-pr` aggregate check |
| [`prlint.yml`](../../.github/workflows/prlint.yml) | PR creation, push, reopening, and edits validate title/body; the job skips Dependabot-authored PRs |
| [`ci-push.yml`](../../.github/workflows/ci-push.yml) | Pushes to `main`; validate source and the built candidate, ending in `pass` |
| [`settings-pr.yml`](../../.github/workflows/settings-pr.yml) | Trusted PR-target events always validate the merge declaration as `settings-plan`; relevant file changes separately select a read-only head plan |
| [`settings-main.yml`](../../.github/workflows/settings-main.yml) | Pushes to `main` affecting settings or this workflow; one Environment-bound direct apply job |
| [`settings-env.yml`](../../.github/workflows/settings-env.yml) | PR target events and pushes to `main`; inspect workflow YAML with trusted source, reporting `settings-env` |
| [`approve-codex-review.yml`](../../.github/workflows/approve-codex-review.yml) | Codex summary updates or PR CI completion; approve completed reviews matching HEAD with thumbs-up |
| [`release.yml`](../../.github/workflows/release.yml) | Explicit manual dispatch from `main`; shared resolve, commit, acceptance, release, and verification orchestration delegates product work to local Actions |

Release publication has no Issue-comment bridge. The owner starts [`release.yml`](../../.github/workflows/release.yml) directly with `workflow_dispatch` after main verification; version calculation happens within the workflow. This repository's other `issue_comment` workflows do not authorize or dispatch releases.

## Workflow dependency lockfile

[The generated lockfile](../../.github/workflows/actions.lock) onboards
`approve-codex-review`, `labeler`, `prlint`, `release-version-tests`,
`settings-env`, `settings-main`, and `settings-pr`, plus the
[`verify-lockfile` workflow](../../.github/workflows/verify-lockfile.yml).
Existing workflow SHA references, triggers, and permissions are retained.
The run-only approval workflow has an empty dependency list.

The verification workflow installs the official `github/gh-actions-lock` CLI
at a fixed release version and runs online `--verify` for this explicit scope on PRs
and main pushes. It checks coverage and upstream provenance; it is not a
required check or a Require lockfile policy. Do not run an unscoped generation
or verification command during this rollout: it also scans the deferred
workflows. To regenerate, run the scoped command from that workflow with
`--verify` replaced by `--no-interactive`. Keep `--no-narrow` to retain SHA
references; without it the CLI may rewrite them to release tags.
Commit the generated lockfile with dependency changes, and rerun the scoped
verification command. `--verify-local` is an optional offline coverage check;
it does not verify upstream identity or reachability. Never edit the lockfile
by hand or automatically accept moved pins.

The existing [Dependabot configuration](../../.github/dependabot.yml) checks
GitHub Actions weekly. GitHub documents that Dependabot updates lockfile
entries alongside dependency updates for onboarded workflows, leaving other
workflows unonboarded. Inspect both files and the verification result in each
update PR; local generation alone does not prove a hosted Dependabot update.
See [GitHub's maintenance instructions](https://docs.github.com/en/actions/how-tos/secure-your-work/lock-workflow-dependencies).

### Deferred composite workflows

`ci-pr`, `ci-push`, and `release` retain their SHA pins and workspace-local
Actions. Their acceptance jobs check source out under `.action`, create a
separate configuration-only consumer at the workspace root, and execute that
candidate Action. Release verification separately checks the published tag
out under `.published`. A `$/` reference instead resolves from the workflow's
repository at its commit, independent of those workspace checkouts. Replacing
these paths mechanically would change which Action is exercised; `.action`
and `.published` are checkout destinations, not source repository paths.

An isolated `--migrate-local-actions --no-narrow` experiment rewrote nine
references and reported success, but left the four `.action` acceptance and
verification call sites as unresolved local-path warnings. The pinned
actionlint 1.7.12 then rejected all seven generated workflow `$/` references
as missing an external Action ref. CLI success alone therefore does not
establish complete transitive coverage or lint compatibility. Before onboarding
these workflows, validate each call site's source and working-directory
semantics and resolve that linter incompatibility.
GitHub documents a possible bridge: a never-run `$/` composite enumerating
the workspace Action dependencies, generated with `--ignore-local-actions`.
Such entries must cover every dependency for the **same workflow**, including
nested composites and the published Action actually tested. Validate that
inventory and candidate/published checkout isolation before adopting it.
Simply suppressing local-action errors can leave missing runtime pins.
See [GitHub's local-path guidance](https://docs.github.com/en/actions/how-tos/secure-your-work/lock-workflow-dependencies#keeping-a-local-path-action).

### Security boundary

Compared with direct SHA pinning, lockfiles also inventory transitive Action
dependencies and record repository and owner IDs plus an upstream ref for
runtime provenance checks. This pilot's five distinct remote Actions add no
new transitive Action entries; existing direct execution commits stay the same.
The CLI reports a nonblocking bare-SHA warning because readable tag references
are preferred upstream. Review generated refs, commit SHAs, and identities.

Locking does not establish that an Action is safe. It does not lock downloaded
CLI binaries, package-manager dependencies, runner images, inputs, secrets,
or job-level reusable workflow calls. Local fixture checks do not establish
hosted runtime behavior for workflows that did not run. See
[GitHub's lockfile concepts](https://docs.github.com/en/actions/concepts/security/workflow-dependency-lockfiles)
and the [general availability announcement](https://github.blog/changelog/2026-10-28-github-actions-workflow-lockfiles-are-generally-available/).

## PR lifecycle

Every PR event sets Draft, resolves the static build and acceptance matrices, runs commit-stage, then marks the PR Ready before process and public-Action acceptance. Both acceptance legs verify the same manifested binary from commit-stage without rebuilding it. The final native `ci-pr` aggregate job succeeds only when every prerequisite succeeded. Title/body edits rerun the same stages rather than introducing skipped-success required checks.

The independent `prlint` job checks the current title and body together as the proposed squash message, except for Dependabot-authored PRs. Its job-level skip preserves the required check. GitHub's required-check settings enforce this separately from the code reporter; the reporter does not query or enforce rulesets. See [PR validation](pr-validation.md#message-and-settings-validation) for the author condition, required-check semantics and failure behavior.

Jobs executing PR source have read permissions, no supplied secrets, and no persisted checkout credentials. Draft/Ready and code-status jobs have narrowly scoped write permissions but do not check out or execute PR code, local composites, or transferred artifacts. Fork token restrictions can block automatic state/status changes; the workflow does not bypass them.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results. GitHub's protected-main and Environment configuration determine whether a reviewed change can reach apply; a passing source test does not configure those controls.

## Protected main and required checks

The [declared default-branch Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) selects the existing Ruleset by ID in [settings.yml](../../.github/settings.yml): one approval, stale approval dismissal, mandatory CODEOWNER approval, resolved review threads, signed and linear history, and strict required checks. It requires `ci-pr`, `prlint`, `settings-plan` and `settings-env` from GitHub Actions. Repository merge configuration allows squash only. The declaration retains the existing repository-administrator bypass in `for_pull_requests_only` mode, CodeQL with Alerts `errors` and Security alerts `high_or_higher`, and code-quality severity `errors`. Retaining this configured bypass does not authorize its use: every change must satisfy ordinary approval, CI, signature, and resolved-thread rules; see the [review policy](../../AGENTS.md#code-review). Do not add or broaden bypass actors. The trusted main workflow applies this declaration only after merge; inspect the actual live Ruleset and Actions permission rather than assuming that source is already enforced.

| Required check | Source and purpose | Limit |
| --- | --- | --- |
| `ci-pr` | [PR code CI](../../.github/workflows/ci-pr.yml) aggregates the build and acceptance stages | PR source controls its tests and CI; a successful result is not independent authorization for privileged code |
| `prlint` | The [independent PRLint workflow](../../.github/workflows/prlint.yml) checks the current title/body | Syntax validation does not establish semantic type, coherent scope, or disclosed compatibility impact |
| `settings-plan` | The [trusted PR workflow](../../.github/workflows/settings-pr.yml) validates the verified PR merge declaration with its pinned operational consumer, without an Environment or App token | The separate `plan` job observes the PR head against GitHub only when its event and path conditions match |
| `settings-env` | The [trusted guard](../../.github/workflows/settings-env.yml) uses main-branch policy to inspect candidate workflow YAML as data | It checks the limited Environment/trigger/name policy below |

See [required-check setup](../github-actions.md#protect-main-with-required-checks) for branch freshness and trusted check-source requirements. Requiring these names from GitHub Actions does not by itself prove that their intended workflows evaluated the proposed change.

After a permitted merge, a matching main push starts the settings workflow. Whether the selected Environment adds a separate deployment approval depends on its live reviewer configuration. If PR review is intended to be the only manual approval, apply the [documented conditions for that mode](../github-actions.md#when-pr-review-is-the-only-manual-approval) and [verify the protection](../github-actions.md#verify-the-protection-before-use), including check-source identity. Neither a source comment nor `environment: gh-settings` establishes whether a reviewer gate is currently enabled.

## Active settings workflows

The active callers use independently pinned released packages; see the [published package and consumer pins](../github-actions.md#published-package-and-consumer-pins) for adoption requirements. The trusted default-branch workflow selects the PR validation and plan pins. Before a declaration uses a schema or capability those packages do not support, first merge the compatible published package pins into main and verify the updated trusted workflow. Changing a pin only inside the declaration PR does not change that PR's trusted validator or planner. Keep the main apply pin compatible with the tracked declaration. The operational pins are independent of the local development [`action/cli-version`](../../action/cli-version); the invoked published tag or its matching pinned SHA selects the CLI. Publishing a release or changing examples does not automatically update these callers.

The PR caller uses trusted default-branch `pull_request_target` code for opened,
synchronize, reopened and edited events targeting main. There is no
workflow-level path filter. Its required `settings-plan` job performs
[credential-free merge validation](#pr-settings-validation) for every supported
event, including metadata edits and PRs without settings changes.

The separate `plan` job requires successful validation and an opened,
synchronize or reopened event. Validation checks the verified merge against
its base parent for changes to `.github/settings.yml`,
`.github/workflows/settings-pr.yml`, `.github/workflows/settings-main.yml` or
`examples/github-actions/**`. Only a matching change selects the plan; an
edited event never does. The exact main-ref/repository/base condition and
Environment `gh-settings` still constrain this credential consumer. Validation
uses its own per-PR concurrency group without cancelling active runs; the
separate plan group cancels superseded plans.

Before minting a token, the plan job anonymously verifies this public repository's fixed identity, fetches only filtered commit/tree metadata for the event's PR head, and downloads the single regular `.github/settings.yml` blob. It validates the fetched head, file mode, bounded size and Git blob hash. It creates a configuration-only consumer with candidate Git identity; no PR code, hooks, local Actions, dependency configuration or scripts are checked out or executed in this job. Public API rate limits, a visibility/identity change, an unavailable blob or a moved PR head fail the preparation; there is no credential fallback. This public-only retrieval deliberately needs no Contents grant on ANSWER26.

The plan job then mints a short-lived repository-scoped token from the same ANSWER26 App used by main apply, requesting Administration write and Issues read, and passes it explicitly to the pinned released Action. GitHub's [Ruleset GET contract](https://docs.github.com/en/rest/repos/rules#get-a-repository-ruleset) returns bypass actors only to callers with Ruleset write access. Nonmutating plan therefore needs a write-capable token for complete observation; it is not a read-only credential. This token can change repository administration settings if trusted executing code misuses it. The existing installation grants are unchanged, and its repository selection is limited to this repository. No additional Environment consumer or API mutation is authorized by this permission request. Feature adoption must still verify complete observation, endpoint support and installation permissions; the broader token alone does not establish that a Ruleset is safely manageable. Local fixtures establish data isolation and failure handling, not effective remote grants or successful live execution. A PR changing this caller itself runs the previous default-branch caller; subsequent relevant events must verify the updated trusted plan.

The main caller runs only on a push to `main` changing `.github/settings.yml` or `settings-main.yml`. Its single `apply` job references Environment `gh-settings` and contains three steps: checkout of the triggering `github.sha`, App token creation, and direct apply. Repository-scoped concurrency does not cancel an in-progress apply.

The token-creation step references `vars.ANSWER26_CLIENT_ID` and `secrets.ANSWER26_PRIVATE_KEY`, limits installation scope to the triggering owner/repository, and requests Administration write and Issues write. The resulting token is passed only to the Action's apply input; the token Action owns its normal post-job revocation. Source checks verify those bindings, not the remote App installation or actual token revocation.

The configuration also selects the existing `gh-settings` and `copilot`
Environments by name with complete supported observations. `gh-settings`
retains administrator bypass disabled and only the typed Branch `main` pattern;
`copilot` retains administrator bypass enabled and no branch/tag restriction.
Both retain zero wait time and no reviewers. Unlisted Environment names remain
unmanaged. A fresh complete match skips Environment writes; a prior plan is
not a lock. Read back the policy and typed patterns after any apply rather than
inferring effective deployment security from a successful workflow alone.

An earlier zero-difference PR plan does not make a later apply write-free: declared repository fields and topics retain their direct PATCH/PUT behavior. Earlier successful writes remain applied if a later operation fails. Assess the exact execution rather than inferring no writes from a prior plan or the generic success message.

There is no prepare/handoff job or main-side pre-approval plan. The PR Summary and exact committed configuration provide review context; direct apply does not consume a saved plan and may overwrite intervening changes to managed values. The presence of `environment: gh-settings` identifies the selected Environment but does not establish whether reviewers are currently configured.

## Active repository Environment guard

The [guard workflow](../../.github/workflows/settings-env.yml) checks out trusted `main` policy into `.policy` and the candidate PR merge into `.candidate`. The candidate is inspected only as YAML data. The job has a read-only token, installs Go using trusted configuration, and runs the main-branch [scope test](../../tests/github-actions/settings_environment_scope_test.go) with `GH_SETTINGS_WORKFLOWS_DIR` pointing at the candidate workflows.

The `gh-settings` Environment allowlist contains exactly two locations: `.github/workflows/settings-main.yml/jobs.apply` on push to main, and `.github/workflows/settings-pr.yml/jobs.plan` on trusted `pull_request_target` targeting main. Both references are mandatory. All other `gh-settings` references are rejected.

The PR caller must contain exactly the required `settings-plan` validation job
and the separate `plan` job. It handles opened, synchronize, reopened and edited
events without a workflow-level path filter. Validation runs unconditionally;
plan must depend on that validation, retain the exact
event/main-ref/repository/base condition, exclude edited events and require
`needs.settings-plan.outputs.plan-required == 'true'`. Validation and plan
have independent per-PR concurrency groups; validation does not cancel active
runs and plan cancels superseded plans.

The `settings-plan` job ID and display name are reserved exclusively for
`.github/workflows/settings-pr.yml/jobs.settings-plan`. That validator is
mandatory, and a `.github/workflows/settings-plan.yml` file is rejected.
The guard requires a merge checkout without persisted credentials and
the pinned Action's `command: validate`, with only Contents read and no
Environment or injected credentials. Conditional validation steps, ignored
failures and additional execution steps are rejected. The validator
cannot have a job condition or prerequisite job; its only extra step handles
the explicit event base/head inputs and produces plan eligibility.

Dynamic Environment names, workflow/job aliases or merges, duplicate mapping keys, dynamic job names and job-level reusable-workflow calls remain rejected. The retired `gh-settings-plan` Environment is forbidden in every workflow. Unrelated literal Environment names retain their existing boundaries.

The guard was deployed before the operational PR credential consumer. GitHub [evaluates `pull_request_target` using the default branch](https://github.blog/changelog/2025-11-07-actions-pull_request_target-and-environment-branch-protections-changes/), including Environment branch matching; existing main-only deployment restrictions do not need to be widened. The permitted PR job still needs trusted executing code and candidate YAML treated only as data; path/event allowlisting does not inspect every step body. The allowlist does not configure live Environment restrictions or issue tokens itself.

The guard checks validation structure, not the inline merge/path handler's implementation. It does not inspect the allowed apply job's step bodies, approve Action revisions, restrict its API operations, validate actual token grants, or configure remote Environment/ruleset settings. The allowed job could still be changed to execute code with privileged credentials. Review the trusted step bodies, their referenced code, and changes to the guard/policy itself under the [apply-path protection guidance](../github-actions.md#protect-the-apply-path); later runs use whatever policy has been merged into `main`.

The dedicated check name is **`settings-env`**. The policy reserves that job ID/display name for the guard and rejects collisions elsewhere in inspected workflows. The ordinary Go suite also runs the scope assertion and contributes to `ci-pr` or `ci-push`.

These checks detect violations in the inspected source, but the required status name does not identify a unique workflow. GitHub documents that [required checks do not take the workflow or event trigger into account](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks). Selecting GitHub Actions as the required check provider identifies the app, not this guard's workflow file. Another Actions job can report the same name, so a green `settings-env` result alone is not proof that the trusted guard ran. The name-reservation test prevents collisions only when the trusted policy actually executes.

When evaluating enforcement, inspect the applicable required checks and allowed check providers, actual guard run, Environment deployment branches/reviewers, and [`pull_request_target` event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target). The repository YAML does not establish those external settings or enforce policy across other repositories. This guide records the guard's behavior and limits without prescribing a change to external approval configuration.

## PR settings validation

The required `settings-plan` job in
[settings-pr.yml](../../.github/workflows/settings-pr.yml) validates
`.github/settings.yml` from the PR merge checkout. Trusted main code checks
that the merge commit has exactly the event's base and head SHAs as its two
parents before invoking its pinned public Action. An unavailable or mismatched
merge fails validation. The job has no conditional skip or prerequisite job;
every supported PR event executes the validation.

If the base or head moved, obtain a fresh PR event for the current candidate.
Rerunning the earlier event retains its earlier identities; validation does
not substitute a head or main checkout. A synchronization or reopening event
can also select a relevant authenticated plan, while an edit only validates.

It calls only `command: validate`, with no supplied token. The job uses the
public Action's supported fresh Ubuntu 24.04 x64 VM, has only Contents read,
selects no Environment and creates no App token. It does not execute PR scripts
or local Actions. A validation failure fails the required check. The validator
passes only the relevant-path match to the separate plan job, whose data comes
from the event's PR head rather than the merge checkout.

This checks the operational declaration against its actual package contract.
The separate required `ci-pr` builds and tests the candidate source, including
current-format process and public Action fixtures. The candidate decoder still
rejects obsolete configuration; no compatibility decoder or translation is used.
The tracked declaration uses the current UI contract. Validation and plan do
not apply it; the main caller performs direct apply only after a permitted
merge.

Validation does not establish live App visibility or drift. The separate
[trusted plan job](#active-settings-workflows) performs authenticated observation;
no validation result authorizes a later apply or locks remote state.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) submits an approval when these conditions hold:

1. Codex has reacted with 👍 on the PR.
2. The latest Codex Review Summary marks **Code Review** and **Security Review** as **Completed**.
3. Both reviews' full reviewed commit SHAs match the latest PR HEAD SHA.
4. This PR has no prior `github-actions[bot]` review for that HEAD in `APPROVED` or `DISMISSED` state.

The workflow reads the visible Summary table for both Completed status cells. It reads each review's full `commit_sha` from the same Summary's `codex-review-state:v1` JSON marker and compares it to the PR API's full HEAD SHA. Both SHAs must be 40 hexadecimal characters and equal HEAD; case is normalized. Missing, malformed or ambiguous review metadata withholds approval. The abbreviated table commits are not a fallback. Other JSON fields, including finding flags, do not add approval conditions. Base identity, timestamps, receipts, prior requests and other reviewers' changes-requested decisions are not checked. A retained thumbs-up can be reused only while the reviewed SHAs still match HEAD.

Summary creation/edits and completion of `ci-pr` trigger reevaluation. CI success and the PR's source repository are not additional approval conditions. GitHub does not provide a reaction trigger here, so the workflow polls up to three minutes for a delayed thumbs-up. The exact Codex login `chatgpt-codex-connector[bot]` identifies both the summary author and the reaction actor; numeric account IDs and App metadata are not required. The newest summary is selected by comment ID. These identify the requested Codex signals.

Each polling attempt reads the latest Summary and current reactions once. When the reviews are Completed and Codex thumbs-up is present, it obtains PR HEAD once, compares both full reviewed SHAs to that HEAD once, and reads every page of this PR's review history before submitting `event=APPROVE` with that verified SHA as `commit_id`. There is no second HEAD, Summary or reaction read in that attempt. The API response must confirm approval of the specified SHA.

Verification and the review POST are separate operations. A push after the HEAD read is not detected by this workflow; the request still names the previously verified SHA. Signal changes after their reads are not rechecked either. This is not an atomic guarantee that HEAD and signals remain unchanged at submission. The workflow makes no assumption that Rulesets necessarily invalidate an old-SHA approval created after a concurrent push. The log identifies the reviewed commit named by the POST.

The history check matches the exact approver login `github-actions[bot]` and full `commit_id` (case normalized), retaining both `APPROVED` and `DISMISSED` entries. It does not require a body marker, numeric account ID or App metadata, so existing automatic approvals with empty bodies also count. Any matching entry withholds another approval even after a new Codex review on the same HEAD. Other accounts, review states and commit SHAs do not consume this PR/HEAD allowance. A different HEAD or a different PR is evaluated normally.

The existing per-PR workflow concurrency group serializes these workflow runs without cancelling an active run. Review-history retrieval and approval creation are separate API operations, not an atomic reservation: another workflow or independent actor using the same bot can approve after the history read, and an older in-flight workflow revision may not apply this guard. The workflow cannot guarantee global exactly-once creation against such writers. A failed/uncertain POST is not retried inside the run; a later run checks the recorded history again. Failed or malformed API responses are reported without claiming a successful approval. The workflow runs default-branch code without checking out PR source and uses only its existing pull-request write permission.

Maintainer decision (2026-11-01): unresolved findings or review conversations do not add an approval condition when Codex thumbs-up, both Completed statuses and matching full reviewed HEAD SHAs are present. The repository owner accepts this tradeoff and keeps clean-finding flags, clean receipts and thread resolution outside this workflow's decision. Whether conversations must be resolved before merging is determined by the target's effective Rulesets. With `required_review_thread_resolution` enabled, submitting an approval does not remove that merge requirement.

Conversation resolution rules apply to GitHub review threads, not every hidden Codex finding flag. Rulesets do not resolve or dismiss threads automatically; maintainers and reviewers record the disposition and resolve threads when appropriate. A risk-acceptance reply records that decision without marking a thread resolved. This responsibility split does not claim that Rulesets independently certify the review evidence as clean.

The controls have separate responsibilities:

| Control | Responsibility |
| --- | --- |
| GitHub Rulesets | Determine merge eligibility: required approvals, stale-approval dismissal, required checks, branch freshness and other configured rules |
| Codex approval workflow | Check Codex thumbs-up, both Completed statuses and reviewed HEAD SHA equality; approve that PR/HEAD only if no prior automatic APPROVED or DISMISSED review exists |
| Maintainer/reviewer | Assess changes to the comparison or review evidence, resolve findings, and verify effective rules and required checks before a normal merge |

With stale-approval dismissal enabled, GitHub evaluates changes to the approved **diff**, not simply elapsed time or a changed SHA. Pushes, Update branch, or related changes merged into the target can change that diff; a merge-base change introducing new changes can also dismiss approval. Strict required checks separately require the branch to be up to date before merging. A target-branch advance is not a blanket claim that every approval is dismissed. See GitHub's [Ruleset review requirements](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-a-pull-request-before-merging) and [strict required checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-status-checks-to-pass-before-merging).

The workflow does not reproduce those merge gates. Its approval does not establish that a changed base comparison received a new review or that the PR can merge. After an automatic approval is dismissed, subsequent runs do not restore it for the same PR/HEAD, even when Codex reviews that HEAD again. This prevents a retained review result from reviving the dismissed approval, but does not detect a base change between review and the first approval. Rulesets do not validate the provenance or freshness of the Codex signals. GitHub evaluates applicable merge rules independently of this workflow's decision.

Read-only verification on 2026-11-01 found this repository's [active main Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) enables stale-review dismissal, required review-thread resolution and strict required checks. Treat this as a dated observation, not a guarantee of future enforcement. In the shared five-repository audit, all five had stale-review dismissal; gh-settings, gh-qw, agents and apex also had strict required checks. elite-miko-engineer's active Ruleset had no required-status-check rule, so that Ruleset does not establish latest-base enforcement. Its classic branch-protection read returned 403, leaving any additional protection unverified; this is not evidence that no other protection exists. No setting was changed or compensated for by the approval workflow. Recheck the actual target's effective rules before relying on them.

Logs distinguish approval, withholding because a required Codex signal is absent, and API/execution failure. A successful workflow that reports withholding has not approved the PR. Recheck live Ruleset approvals, review threads and required CI before merging. The retained administrator bypass must not be used to merge; this workflow does not authorize settings changes or a release.

## Inspect external configuration

Use [repository rules](https://github.com/daiksud/gh-settings/rules), [Environments](https://github.com/daiksud/gh-settings/settings/environments), and [Actions settings](https://github.com/daiksud/gh-settings/settings/actions) when diagnosing blocked or unexpected execution. Follow [verification before use](../github-actions.md#verify-the-protection-before-use): confirm the required check results and their actual workflow/source for the intended commit, the selected Environment's branch/tag and reviewer restrictions, bypass settings, App installation/credential bindings, and whether GitHub permits the workflow event.

[Operational fixtures](../../tests/github-actions/operational_test.go) and the [test evidence boundaries](testing.md#release-and-publication-evidence) help distinguish source correctness from those external settings. Release dispatch and recovery have their own [maintainer runbook](releases.md).
