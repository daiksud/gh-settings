# Repository automation

This guide is for maintainers of **daiksud/gh-settings**. For installing gh-settings workflows in another repository, use the [GitHub Actions guide](../github-actions.md) and [examples](../../examples/github-actions/README.md). The public guide explains [how to protect the apply path](../github-actions.md#protect-the-apply-path); this page maps those responsibilities to this repository's workflows and controls.

The workflow files describe source-controlled behavior. Required reviews/checks, Environment protection rules, App installations, secrets, and event-policy eligibility are configured separately on GitHub; inspect those settings when assessing the current operational state.

## Repository settings changes

Change this repository's managed settings in
[`.github/settings.yml`](../../.github/settings.yml) through the normal PR
review and squash-merge process. Use declarations supported by the
[pinned operational packages](#active-settings-workflows).

1. Open a configuration PR and inspect its checks. The `settings-pr`
   check validates the proposed merge configuration with the operational Action
   selected by the PR workflow definition. It runs only validate; it does not observe live GitHub settings or produce a plan.
2. Satisfy the [normal merge requirements](#protected-main-and-required-checks)
   and squash-merge the PR. Merge and push events do not apply settings.
3. When application is intended, manually dispatch
   [settings-apply.yml](../../.github/workflows/settings-apply.yml) from `main`.
   Its job rejects other refs and checks out the dispatch event's exact
   `github.sha` before running direct apply.
4. Inspect that apply run and read back the affected GitHub settings before
   reporting them as applied. A passing PR validation is not evidence of apply.
   Direct apply does not consume a saved plan. Check partial progress before
   retrying a failed or cancelled run.

The manual apply workflow uses the existing repository-scoped App authentication
and `gh-settings` Environment. It does not depend on a local CLI login or a
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
Review the declaration and read back those objects to confirm their IDs before
manually dispatching another apply. Rerunning the original ID-less configuration
after creation can fail with a resource-specific collision, including after a partial
failure. Recover confirmed IDs before retrying; follow the
[apply contract](../commands.md#apply-behavior) for partial progress and
resource-specific behavior.

## Workflow inventory

| Workflow | Trigger and responsibility |
| --- | --- |
| [`pr.yml`](../../.github/workflows/pr.yml) | Default pull_request activities run the single `screening` job for source checks, unit tests, and a native build |
| [`prlint.yml`](../../.github/workflows/prlint.yml) | PR creation, push, reopening, and edits validate title/body; the job skips Dependabot-authored PRs |
| [`ci-push.yml`](../../.github/workflows/ci-push.yml) | Pushes to `main`; install the manifest-selected toolchain, build native release-ready binaries, and inspect them, ending in `ci-push` |
| [`settings-pr.yml`](../../.github/workflows/settings-pr.yml) | Default pull_request events validate the PR merge declaration only; the validation check is `settings-pr` |
| [`settings-apply.yml`](../../.github/workflows/settings-apply.yml) | Manual dispatch from `main`; one Environment-bound direct apply job |
| [`approve-codex-review.yml`](../../.github/workflows/approve-codex-review.yml) | Codex summary updates or PR CI completion; approve completed reviews matching HEAD with thumbs-up |
| [`release.yml`](../../.github/workflows/release.yml) | Explicit manual dispatch from `main`; source checks, native build, packaging, acceptance, publication, and verification run as distinct steps or jobs |

Release publication has no Issue-comment bridge. The owner starts [`release.yml`](../../.github/workflows/release.yml) directly with `workflow_dispatch` after main verification; version calculation happens within the workflow. This repository's other `issue_comment` workflows do not authorize or dispatch releases.

## Workflow dependency lockfile

[The generated lockfile](../../.github/workflows/actions.lock) covers
`approve-codex-review`, `pr`, `ci-push`, `labeler`, `prlint`, `release`,
`settings-apply` and `settings-pr`. The run-only approval
workflow has an empty dependency list. There is no dedicated lockfile
verification workflow.

Use the official `github/gh-actions-lock` CLI at v0.4.1 to regenerate:

```sh
gh actions-lock --no-interactive --no-narrow --ignore-local-actions
```

Omit workflow path arguments so the CLI scans the full workflow directory and
removes lockfile entries for deleted workflows.

Commit the generated lockfile with dependency changes, then run the same scope
with `--verify` instead of `--no-interactive` to check upstream provenance.
Keep `--no-narrow` to retain SHA references. `--verify-local` is an optional
offline coverage check; it does not verify upstream identity or reachability.
Never edit the lockfile by hand or automatically accept moved pins.

The existing [Dependabot configuration](../../.github/dependabot.yml) checks
GitHub Actions weekly. GitHub documents that Dependabot updates lockfile entries
alongside dependency updates for onboarded workflows. Inspect both files and the
verification result in each update PR; local generation alone does not prove a
hosted Dependabot update. See [GitHub's maintenance
instructions](https://docs.github.com/en/actions/how-tos/secure-your-work/lock-workflow-dependencies).

### Workspace composite coverage

`pr`, `ci-push`, and `release` keep their workspace-local Action references.
The main and release acceptance jobs check frozen source out under `.action`, create a separate
configuration-only consumer at the workspace root, and execute that candidate
Action. Release verification separately checks the published tag out under
`.published`. A `$/` reference instead resolves from the workflow's repository at
its commit, independent of those workspace checkouts. Replacing these paths
mechanically would change which Action is exercised; `.action` and `.published`
are checkout destinations, not source repository paths.

PR Screening and native build jobs use setup-go with `go-version-file: go.mod`
and setup-bun with `bun-version-file: package.json`. Bun installs the frozen
package lock for Markdown and message validation. Setup-go owns the standard
module/compilation cache; no mise-action or custom Go cache Action is used.

`ci-push` and `release` still use the
[full dependency inventory](../../.github/actions/lock-workspace-dependencies/action.yml)
through `$/` in a step guarded by `${{ false }}`. The step never runs and
only enumerates transitive external dependencies for the lockfile generator.
`--ignore-local-actions` preserves the workspace-relative action paths. See
[GitHub's local-path guidance](https://docs.github.com/en/actions/how-tos/secure-your-work/lock-workflow-dependencies#keeping-a-local-path-action).

[Inventory tests](../../tests/github-actions/actions_lockfile_test.go) walk the
runtime workflow and nested local composites, including the root Action reached
through the frozen `.action` and `.published` checkouts. For PR Screening, they compare external
references directly against the same workflow's lock entries. For main and
release, they compare the external references with the inert composite and the
same workflow's generated lock entries. Changing a nested dependency requires
regenerating the lockfile and updating the main/release inventory. Existing candidate, consumer and release verification
checks still own checkout identity and execution isolation; the dependency
inventory does not replace those checks or authorize a release.

### Security boundary

Lockfiles inventory transitive Action dependencies and record repository and
owner IDs plus an upstream ref for runtime provenance checks. Existing direct
execution commits stay the same. The CLI reports a nonblocking bare-SHA warning
because readable tag references are preferred upstream. Review generated refs,
commit SHAs, and identities, including dependencies reached through local
composites.

Locking does not establish that an Action is safe. It does not lock downloaded
CLI binaries, package-manager dependencies, runner images, inputs, secrets,
or job-level reusable workflow calls. Local fixture checks do not establish
hosted runtime behavior for workflows that did not run. In particular, release
workflow coverage can be generated and checked without dispatching or publishing
a release. See [GitHub's lockfile
concepts](https://docs.github.com/en/actions/concepts/security/workflow-dependency-lockfiles)
and the [general availability
announcement](https://github.blog/changelog/2026-10-28-github-actions-workflow-lockfiles-are-generally-available/).

## PR lifecycle

The `pr` workflow uses the default `pull_request` activities: opened, synchronize, and reopened. Its single `screening` job runs source checks, unit tests, and a native Linux build on `ubuntu-slim`; GitHub reports that job's result directly. The workflow leaves the PR's Draft or Ready state unchanged. Title/body edits and draft/readiness transitions do not rerun Screening. Main CI owns process and public-Action Inspection of release-ready artifacts.

The independent `prlint` job checks the current title and body together as the proposed squash message, except for Dependabot-authored PRs. It reruns on metadata-only edits. Its job-level skip preserves the required check. GitHub's required-check settings enforce message validation separately from Screening. See [PR validation](pr-validation.md#message-and-settings-validation) for the author condition, required-check semantics and failure behavior.

The `screening` job has Contents read permission and no supplied secrets. Checkout uses its default
credential persistence, so the read-only `GITHUB_TOKEN` is available to subsequent steps through
Git configuration until checkout's post-job cleanup. PR-controlled source and checks can access
that credential during the job. Its steps call the setup, cache, lint, unit-test, and build
Actions directly. It does not request permission to change PR state or publish commit statuses.

This validation pipeline does not merge, publish, or deploy. Use [testing](testing.md#ci-validation) to interpret its results. GitHub's protected-main and Environment configuration determine whether a reviewed change can reach apply; a passing source test does not configure those controls.

## Protected main and required checks

The [declared default-branch Ruleset](https://github.com/daiksud/gh-settings/rules/24911852) selects the existing Ruleset by ID in [settings.yml](../../.github/settings.yml): one approval, stale approval dismissal, mandatory CODEOWNER approval, resolved review threads, linear history (without mandatory commit signatures), and strict required checks. The declaration requires `screening`, `prlint` and `settings-pr` from GitHub Actions (App ID `15368`). Repository merge configuration allows squash only. The declaration retains the existing repository-administrator bypass in `for_pull_requests_only` mode, CodeQL with Alerts `errors` and Security alerts `high_or_higher`, and code-quality severity `errors`. Retaining this configured bypass does not authorize its use: every change must satisfy ordinary approval, CI, and resolved-thread rules; see the [review policy](../../AGENTS.md#code-review). Do not add or broaden bypass actors. The manual main workflow applies this declaration only when dispatched after merge; inspect the actual live Ruleset and Actions permission rather than assuming that source is already enforced.

| Check | Source and purpose | Limit |
| --- | --- | --- |
| `screening` | The [pr workflow](../../.github/workflows/pr.yml) runs source checks, unit tests, and a native Linux build in one job | PR source controls its tests and CI; a successful result is not independent authorization for privileged code |
| `prlint` | The [independent PRLint workflow](../../.github/workflows/prlint.yml) checks the current title/body | Syntax validation does not establish semantic type, coherent scope, or disclosed compatibility impact |
| `settings-pr` | The [PR workflow](../../.github/workflows/settings-pr.yml) validates the default PR merge declaration with its pinned operational consumer, without an Environment or App token | Validate only; no remote settings observation or PR plan is performed |

See [required-check setup](../github-actions.md#protect-main-with-required-checks) for branch freshness and trusted check-source requirements. Requiring these names from GitHub Actions does not by itself prove that their intended workflows evaluated the proposed change.

After a permitted merge, application requires an explicit manual dispatch from
`main`. Whether the selected Environment adds a separate deployment approval
depends on its live reviewer configuration. Inspect actual check-source identity
and [verify the protection](../github-actions.md#verify-the-protection-before-use).
Neither a source comment nor `environment: gh-settings` establishes a reviewer gate.

## Operational declaration and current-source scope

The two settings callers select the immutable v0.44.1 package at
`2079ede46bcb3369d9575c1941ac88e4a37c081d` so the operational codec matches the
expanded declaration. Their events, credential sources, Environment and token
permission requests are unchanged; pin adoption grants no access and dispatches
no apply.

The declaration records supported, applicable settings acquired through the
published CLI's real read-only export and checked with a fresh local plan.
It includes all observed Labels, the existing main Ruleset and credential
Environment, General, Actions cache retention, persistent PR limits and
Security/CodeQL. Supported empty inventories are explicit;
without prune they do not request unlisted deletion. Existing Ruleset, reviewer
and check-source IDs are retained. No new resource is requested.

Autolinks are omitted from this repository's declaration under the explicit
[Free-plan v1 scope decision](../settings.md#autolink-references). The existing v0
implementation and automatic export acquisition remain; an API-observed empty
collection does not establish Free-plan UI entitlement. Removing the declaration
does not delete live references or request pruning.

The classic `repository.settings.branch_protection_rules` declaration is also
omitted under the [v1 Rulesets-only decision](../settings.md#classic-branch-protection).
The complete existing Rulesets declaration is retained. Classic v0 implementation
and export behavior remain; this omission neither migrates nor deletes live
classic protection and does not authorize an apply or prune.

Environment variable payloads remain omitted because they are internal data.
Secrets, private keys, inactive normal-merge message controls, inactive
selected-Actions controls and private-only fork/sharing controls are not copied
into desired state. Repository/Agents variable inventories were observed empty;
no variable values are published. A matching plan is point-in-time evidence,
not a lock or proof of effective enforcement.

Owner-readable settings are not automatically safe additions to the App-backed
apply declaration. These newly observed controls remain deferred without changing
any existing declaration or expanding grants:

| Omitted path | Existing credential boundary |
| --- | --- |
| `repository.settings.actions.general.cache.cache_size_eviction_limit` | Needs Actions read even for matching apply |
| `repository.settings.actions.oidc.subject_claim` | Needs Actions read/write |
| `repository.settings.advanced_security.code_security.ai_scan` | Needs Code scanning alerts read in addition to Administration write |
| `repository.settings.copilot.code_review` | Exact App-token grants remain independently unconfirmed |
| `repository.settings.pages` | Needs Pages read/write; an owner-observed absent site is not App visibility evidence |
| `repository.settings.environments` entry `copilot` | Additional Environment ownership needs the applicable Actions observation path; retain the existing credential Environment only |
| `repository.settings.environments[].variables` | Internal payloads are excluded from this public declaration; no variable grant is requested |

CodeQL default setup/configuration is separately documented with Administration
read/write; it does not use the AI Scan permission path. Empty collection
snapshots select no objects without pruning.

The current `settings-pr` check validates only; it is not a hosted live plan.
Local owner-authenticated plan evidence does not establish complete same-issued-
App-token grants or live reads for the expanded declaration. The existing manual
apply retains Administration/Issues token requests, which are not universal
permissions for every managed family. No token or grant is expanded here; any
future manual apply needs its own complete authorized observation and scope
assessment. The separate evidence limits in #381 remain deferred.

## Active settings workflows

The active callers use independently pinned released packages; see the [published package and consumer pins](../github-actions.md#published-package-and-consumer-pins) for adoption requirements. The PR workflow definition selects the PR validation pin; changes to the pin require review. Before a declaration uses a schema or capability those packages do not support, review the compatible published package pins together with the declaration and verify the resulting PR workflow. The ordinary `pull_request` workflow is candidate-controlled, not a trusted-base validator. Keep the main apply pin compatible with the tracked declaration. The operational pins are independent of the local development [`action/cli-version`](../../action/cli-version); the invoked published tag or its matching pinned SHA selects the CLI. Publishing a release or changing examples does not automatically update these callers.

The PR caller uses ordinary `pull_request` events with the default opened,
synchronize and reopened activities targeting main. There is no
workflow-level path filter. Its `settings-pr` job performs
[credential-free merge validation](#pr-settings-validation) for every supported
event, including PRs without settings changes. Metadata-only edits do not rerun it.

The PR caller has no plan job, Environment access, App token creation, or
plan-eligibility output. Validation uses the standard GitHub Actions run lifecycle. The job is named `settings-pr`; it does not produce a plan
or publish a replacement `settings-plan` success.

The apply caller runs only on `workflow_dispatch`, with a job condition requiring
`refs/heads/main`. Its single `settings-apply` job references Environment `gh-settings`
and contains checkout of the triggering `github.sha`, App token creation, and
direct apply. Repository-scoped concurrency retains the existing
`settings-main` group and does not cancel an in-progress apply. No push or PR
event starts this workflow.

The token-creation step references `vars.ANSWER26_CLIENT_ID` and `secrets.ANSWER26_PRIVATE_KEY`, limits installation scope to the triggering owner/repository, and requests Administration write and Issues write. The resulting token is passed only to the Action's apply input; the token Action owns its normal post-job revocation. Source checks verify those bindings, not the remote App installation or actual token revocation.

### Bounded candidate Action verification

The separate manual [Topics probe](../../.github/workflows/verify-public-action.yml)
also uses Environment `gh-settings`, only from this repository's `main` after its
normal Environment approval. It reuses the existing App credential on the runner
without copying or displaying it. The token selects only
`daiksud/test-gh-settings-public` and requests Administration write; that repository
must already be included in the App installation. No App grant or Environment
protection is changed by the probe.

Before issuing the App token, the job requires the latest exact-SHA main
`ci-push` to have completed successfully, using the existing release CI gate.
It downloads that run's Linux artifact with archive-digest checking and verifies
the source/version/target manifest and product digest with `release-artifact.sh`.
Missing, failed, unfinished, expired or mismatching evidence stops the probe;
there is no rebuild or published-release fallback. The source checkout must be
clean and match the artifact's source and dispatched workflow SHA.

The existing `prepare-candidate-download.sh` supplies the verified product through
the unchanged root Action installer. Its bounded curl fixture serves only the
two selected release-shaped asset/checksum URLs, without network forwarding or
credentials. This is candidate installation evidence, not a public-release
download. No API fixture or Unix-socket transport is installed: the App token and
independent `gh api` reads use the real GitHub.com repository. The workflow token
has only Contents/Actions read for producer inspection and artifact download;
the App token remains scoped to the dedicated test repository.

The job creates a configuration-only Git consumer in its ephemeral workspace,
with a direct test-repository origin and a local commit; it never pushes that
commit or changes the test repository's files. The local root Action runs
validate, plan and apply with its manifest-verified candidate CLI.
Only Topics are declared. Original membership and repository identity are
recorded before adding one fixed probe topic; fresh checks precede apply and the
matching repeat. Status-only plans must report `no-difference` for the repeat and
restored state, alongside independent API readback.

An `always()` cleanup prepares and applies original membership after an armed
attempt, including an apply with uncertain effects. Unknown competing membership
or identity drift blocks restoration instead of overwriting it. No atomic lock
exists between a check and an Action's write. Cancellation, runner loss, token
expiry or API failures can prevent cleanup: inspect the original Topics in the
prepare log and independently read the test repository before manual recovery.
Do not cancel an active probe or run another Topics writer concurrently. The
token Action retains its normal post-job cleanup; verify the actual run's logs
before claiming revocation or restoration succeeded.

This is a hosted candidate test targeting a dedicated live repository, not a
workflow installed in that target. Export uses the existing fixture/CLI evidence
rather than expanding the live token's grants. Pruning and other live settings
remain outside this probe. Normal public installation of unreleased main remains
unverified; supplying a candidate download fixture does not publish it.

After separately authorized publication, reuse `release.yml`'s existing
`verify-release` job and `release-verify` Action to compare public assets against
accepted manifests and verify the published root Action. Do not dispatch a release
to run that check during a pre-release pass. Candidate live results, public-package
live results and post-publication verification remain separate evidence; retain
their exact sources and run URLs in the tracking Issue.

### Managed credential Environment

The configuration manages only the `gh-settings` Environment. It requires
`daiksud` (User ID `155234749`) as reviewer, with self-review permitted, no wait
timer (`wait_timer: 0`), administrator bypass disabled, and only the typed Branch
`main` pattern. A manual dispatch requires the Environment's reviewer approval.
The live protection policy governs each run: changing the declaration does not
bypass an existing wait timer or reviewer gate. Read back the live timer, reviewer
policy, bypass setting and typed patterns after applying; source declarations
do not establish deployed protection.

`copilot` is absent from the declaration so subsequent apply does not recreate
it. Unlisted Environments remain unmanaged; removing a declaration does not
itself delete the live Environment. Deletion is a separate authorized operation.

An earlier zero-difference PR plan does not make a later apply write-free: REST-backed repository controls and Topics reconcile against new complete reads before any PATCH or PUT. Earlier successful writes remain applied if a later operation fails. Assess the exact execution rather than inferring no writes from a prior plan or the generic success message.

There is no prepare/handoff job or main-side pre-approval plan. The validated declaration and exact committed configuration provide review context; direct apply does not consume a saved plan and may overwrite intervening changes to managed values. The presence of `environment: gh-settings` identifies the selected Environment but does not establish whether reviewers are currently configured.

## Repository Environment scope tests

The ordinary Go suite retains the
[Environment scope tests](../../tests/github-actions/settings_environment_scope_test.go)
in `pr` and `ci-push`. They allow the literal `gh-settings` Environment only
in `.github/workflows/settings-apply.yml/jobs.settings-apply` and the bounded
`.github/workflows/verify-public-action.yml/jobs.topics` probe, require manual dispatch,
and require a single credential-free validation job in `settings-pr.yml`.
They reject dynamic Environment names, workflow/job aliases and merges,
duplicate keys, reusable workflow jobs, and retired check-name collisions.

There is no independent `settings-env` workflow. Candidate code controls the
scope tests in PR CI, so a passing result is not a trusted-main policy decision
or proof of safe privileged step bodies. Review the allowed apply job, its
referenced Actions, and the tests themselves under the
[apply-path guidance](../github-actions.md#protect-the-apply-path). Source tests
do not configure live Environment, token, ruleset, or review settings.

### Moderation format and operational package

Current source accepts and exports
`repository.settings.moderation.interaction_limits`. Its direct predecessor is
rejected without an alias, including when both paths are present. The tracked
self-management YAML deliberately retains `repository.settings.interaction_limits`
because both protected workflows consume the immutable v0.44.1 package, which
rejects the new group. Source validation of that unchanged operational file
therefore fails at the known format boundary; `settings-pr` validates it with the
actual pinned published package. Neither workflow is repinned to development main.

A future, separately reviewed adoption must select a compatible immutable
published package first. The configuration change is limited to wrapping the
existing Interaction limits mapping in `moderation`, at the same sidebar position
after Collaborators. Keep every value, list, ID, comment and other mapping intact.
Validate with the adopted package and compare a fresh read-only plan before
merging; this is not authorization to publish or apply settings. There is no
automatic file migration or legacy input path in the product.

The [operational migration fixture](../../tests/github-actions/managed_moderation_test.go)
projects that one-file candidate only in tests, while explicitly guarding the
existing package pin and rejection of the original by current source. It checks
unchanged cap/list ownership and all sibling semantics. Remove this temporary
projection when the compatible package and canonical operational file are adopted.
Local released-old-file and candidate-canonical-file plans are distinct from
App-authenticated hosted evidence; the hosted PR job still validates only.

### Required-check verification

The desired Ruleset requires `screening`, `prlint`, and `settings-pr`,
all from GitHub Actions (App ID `15368`). The tracked
[settings declaration](../../.github/settings.yml) names the same three checks.

A green required check alone does not authenticate its workflow identity.
Verify the current job producer, workflow
revision, event and candidate SHA, as well as the live Ruleset and its source
App. Review and merge declaration changes normally, manually apply from `main`,
and read back the affected rule before reporting it as enforced. The
[settings change procedure](#repository-settings-changes) owns application;
PR validation does not authorize or perform apply.

## PR settings validation

The `settings-pr` job in
[settings-pr.yml](../../.github/workflows/settings-pr.yml) validates
`.github/settings.yml` from GitHub's default `pull_request` merge checkout.
There is no explicit checkout `with`, merge-parent verification step, or
`pull_request_target` privileged context. The job has no conditional skip or
prerequisite job. The default opened, synchronize and reopened activities
trigger validation.

If the base or head changes, inspect a fresh PR merge candidate and the
current run. Metadata-only edits remain the responsibility of `prlint`.

It calls only `command: validate`, with no supplied token. The job uses the
public Action on `ubuntu-slim`, has only Contents read,
selects no Environment and creates no App token. It does not execute PR scripts
or local Actions. A validation failure fails the required check. No PR plan or remote settings observation follows validation.

This checks the operational declaration against its actual package contract.
The separate `screening` job screens and builds the candidate source. Main
`ci-push` owns process and public Action Inspection of the built products. The candidate decoder still
rejects obsolete configuration; no compatibility decoder or translation is used.
The tracked declaration uses the current UI contract. Validation does not
apply it; a manual dispatch from main performs direct apply after a permitted merge.

Validation does not establish live App visibility or drift. There is no PR
plan. No validation result authorizes a later apply or locks remote state.

## Codex approval control

The [approval workflow](../../.github/workflows/approve-codex-review.yml) submits an approval when these conditions hold:

1. Codex has reacted with 👍 on the PR.
2. The latest Codex Review Summary marks **Code Review** and **Security Review** as **Completed**.
3. Both reviews' full reviewed commit SHAs match the latest PR HEAD SHA.
4. This PR has no prior `github-actions[bot]` review for that HEAD in `APPROVED` or `DISMISSED` state.

The workflow reads the visible Summary table for both Completed status cells. It reads each review's full `commit_sha` from the same Summary's `codex-review-state:v1` JSON marker and compares it to the PR API's full HEAD SHA. Both SHAs must be 40 hexadecimal characters and equal HEAD; case is normalized. Missing, malformed or ambiguous review metadata withholds approval. The abbreviated table commits are not a fallback. Other JSON fields, including finding flags, do not add approval conditions. Base identity, timestamps, receipts, prior requests and other reviewers' changes-requested decisions are not checked. A retained thumbs-up can be reused only while the reviewed SHAs still match HEAD.

Summary creation/edits and completion of `pr` trigger reevaluation. CI success and the PR's source repository are not additional approval conditions. GitHub does not provide a reaction trigger here, so the workflow polls up to three minutes for a delayed thumbs-up. The exact Codex login `chatgpt-codex-connector[bot]` identifies both the summary author and the reaction actor; numeric account IDs and App metadata are not required. The newest summary is selected by comment ID. These identify the requested Codex signals.

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

The manual `release` workflow uses the `actions: read` job permission only for reading CI runs and their artifacts. No CI artifact is executed or published without matching archive digest, source SHA, version, target and product SHA-256. Artifact incompatibility triggers a warning and a full fresh build/Inspection; GitHub API/authentication failures or failed `ci-push` abort. The privileged publication job repeats the main/CI checks before writing a release.
