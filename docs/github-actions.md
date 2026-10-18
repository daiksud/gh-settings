# Use the public settings Action

## Public Action and supported checkout

`daiksud/gh-settings` validates, plans or applies the checked-out repository's `.github/settings.yml`. The Action installs its selected CLI release and exposes normal `with` inputs and step outputs.[^composite-metadata] Consumers provide the checkout, credentials and approval policy; they do not install the CLI or copy Action internals. This guide covers the pinned v0.11.0 package for repository fields, topics and [labels](#labels-and-permissions).

Use a fresh GitHub-hosted `ubuntu-24.04` x64 job with Bash, curl, GNU sha256sum and Git. macOS, Windows, ARM, containers and self-hosted runners are unsupported. Checkout the configuration repository at `GITHUB_WORKSPACE`, without a checkout `path` or persisted credentials, and commit `.github/settings.yml` at its root.

A resolvable checkout commit and a directly configured GitHub.com `origin` are required even for validate's output metadata. Supported origins are HTTPS, `git@github.com:owner/repo.git` and full SSH URLs. The checkout's origin selects the target; the Action repository and `GH_REPO` do not. See [repository resolution](commands.md#repository-and-authentication) and [configuration semantics](settings.md#configuration-semantics).

Validate performs no settings API requests, but installing the public CLI still requires network access. Workflow, configuration and Action revisions must remain available for ordinary reruns.

## Inputs, outputs and failures

| Input | Contract |
| --- | --- |
| `command` | Required, no default; exactly `validate`, `plan` or `apply`. |
| `token` | Unused for validate; required and nonempty for plan/apply. Supply it explicitly; there is no ambient or implicit `github.token` fallback. |

Validate checks local configuration without credentials. Plan validates first, then reads settings with an explicit read-only token, such as `token: ${{ github.token }}` in the example PR job. For the supported repository fields and topics, apply uses a caller-created, target-scoped App installation token with Administration write. The Action neither creates the token nor verifies its actual grants.

There are no inputs for export, arbitrary arguments, configuration path, target, CLI version or download URL. The Action's checkout metadata and explicit plan/apply token requirements also apply to configurations with no managed fields.

| Success output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit (`HEAD`), not the settings file hash, workflow SHA or Action SHA. |
| `target` | Configuration checkout's resolved GitHub `origin`, in `OWNER/REPO` form. |
| `cli-version` | Verified installed CLI release selected by the Action package. |
| `cli-sha256` | SHA-256 of the verified installed binary. |

Use `steps.<id>.outputs.<name>` only after successful completion. Outputs are published after the command and, for plan, successful Summary rendering and writing. Differences, no differences and no managed fields are successful plan outcomes.

Missing command/token, unsupported prerequisites, install/metadata/output failures and CLI/configuration/API errors fail the Action. A nonzero plan status is preserved; otherwise Summary status determines success. Failed calls have no successful output contract. Plan Summary displays escaped settings values and categorical failure diagnostics, and distinguishes workflow, configuration, Action and CLI identities. Validate/apply report through the Action step's outcome and logs; they do not render a plan Summary.

## Published package and consumer pins

The examples use the published [v0.11.0 Action/CLI package](https://github.com/daiksud/gh-settings/releases/tag/v0.11.0), pinned to the full Action SHA `266b96ed81c713e32bb4f554718981aa4ad0bfc3`. Its selected CLI version is fixed by that package; use a reviewed full SHA when adopting or updating the workflows. Release assets and `SHA256SUMS` are available on the release page.

Installation downloads the selected `gh-settings-linux-amd64` asset and `SHA256SUMS` from fixed upstream HTTPS release paths. It requires one valid checksum for the exact asset, verifies before execution and checks exact `gh-settings <version>` output. There is no latest fallback or caller version/URL override. The binary and checksum come from the same GitHub release; the Action SHA fixes the selected version, not an independent binary digest.

Older pins retain their released behavior. Maintainers document immutable package history and known limitations in the [release guide](contributing/releases.md#historical-release-limitations).

### Labels and permissions

[Labels support](settings.md#labels) is published in v0.9.0 and included in the pinned package. Older v0.8.0 packages reject `settings.labels`. Checking out newer Action source does not build that source; the selected release determines CLI capabilities.

Listing needs Issues or Pull requests read, except public resources can be listed without those grants. Label mutations need either permission at write level; Administration write alone is insufficient. The default example App token still requests only Administration write for repository fields/topics. For label mutations, separately approve the App/installation permission and a matching token request scoped to the containing repository. Do not assume installing or pinning v0.11.0 grants access.

Labels apply reads fresh membership and performs only needed CRUD operations, with authoritative deletion of undeclared labels and their issue/PR associations. Review the [labels contract](settings.md#labels) before adoption.

## Consumer files and compatibility

The [copyable PR/main examples](../examples/github-actions/README.md) need two workflow files plus `.github/settings.yml`. No caller script, root `action.yml` or copied `action/` directory is required.

The PR workflow validates and plans with a read-only token and produces a review Summary. Main uses one job with three steps: checkout, App token creation and direct apply. It checks out the event's exact configuration SHA. Main does not consume a PR artifact, depend on the PR run, or produce another plan before approval.

## Protect the apply path

An apply job receives authority to change repository settings. Protect both the code that reaches `main` and the job that receives the write credential. Validation checks the settings schema; it cannot authorize other commands or Actions sharing that job.

| Control | What it protects | What it does not establish |
| --- | --- | --- |
| Required PR reviews and status checks on `main` | Whether candidate settings and workflow changes can merge | Whether a particular job should receive a credential now |
| Environment reviewers and deployment restrictions | Whether a referencing job may start and access its Environment secrets | That only the intended workflow can reference that Environment |
| Trusted workflow-scope policy | Whether the inspected candidate keeps privileged Environment use on the approved path | That an ordinary green check name uniquely identifies the policy that ran |

The examples use Environment approval. If PR review is intended to be the only manual approval, first satisfy the [separate enforcement conditions](#when-pr-review-is-the-only-manual-approval). Copying the workflows does not configure GitHub protection settings or install a workflow-scope policy.

### Ordered owner setup

Configure supported controls before supplying the apply credential; availability depends on the GitHub plan and repository visibility.[^environment-controls]

1. [Protect `main` with required reviews and checks](#protect-main-with-required-checks), including the settings and automation that can use privileged credentials.
2. Install a dedicated selected-repositories App on the target. Repository fields/topics need Administration write and automatic Metadata read; label mutations need separately approved [label permissions](#labels-and-permissions). The official token Action narrows the token with explicit `owner`, `repositories` and permission inputs; retain its default post-job revocation.[^app-token-action]
3. In **Settings → Environments**, create the example's exact `settings-apply` Environment before use. Enable **Required reviewers**, disable administrator bypass, and choose **Selected branches and tags** with a **Branch** rule for exactly `main` and no Tag rule. The single-owner example permits self-approval; enable **Prevent self-review** and provide another eligible reviewer when independent approval is required. One listed reviewer can approve; a list does not require every reviewer to approve.
4. Adopt the [two workflows](../examples/github-actions/README.md#copy-the-workflows) and root configuration through review. Retain the exact-event checkout, no persisted credentials, `push`/`refs/heads/main` guards, reviewed full-SHA Action pins and concurrency. [Verify the protection](#verify-the-protection-before-use) with an isolated setup before supplying a live write credential.
5. Store `SETTINGS_APP_CLIENT_ID` as an Environment variable and `SETTINGS_APP_PRIVATE_KEY` as a secret in that protected Environment. Keep the private key out of repository/organization secrets that an unprotected job could use as an alternate path. Issue the short-lived installation token only inside the approved apply job.

An unknown Environment name in workflow YAML can create an Environment with no protection rules or secrets. Merely adding `environment: settings-apply` does not configure approval. Required reviewers on Free, Pro and Team are available only for public repositories; check feature availability for the actual repository before relying on the gate.[^environment-controls]

### Environment approval and workflow identity

An Environment's protection rules are checked before the referencing job starts, and its secrets are unavailable until the gate passes. In the example, this places checkout, token creation and apply after approval. Rejecting the deployment stops that job. See [GitHub's deployment review procedure](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/review-deployments).

Environment branch rules match the run's ref, rather than selecting a workflow file or event. A different workflow running with an allowed `main` ref could also request the Environment. In particular, a privileged event such as `pull_request_target` must not be treated as safe merely because it uses a trusted base ref. Review the workflow identity as well as the configuration. Avoid **Protected branches only** as a substitute for the exact Branch rule: when no branches have protection rules, that option permits every branch. These boundaries follow [GitHub's Environment rules](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments).

Keep the apply job limited to reviewed code. Passing `token` only to the apply step limits ordinary exposure, but all steps share the job's runner; another step can compromise later credential use. Keep untrusted PR builds, local Actions, scripts and artifacts out of the privileged job, and keep privileged credentials out of PR jobs. Review third-party Action code at the selected revisions; SHA pinning fixes the selected code, not whether that code is trustworthy. See [GitHub's secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use).

### Protect main with required checks

Use an **Active** ruleset under **Settings → Rules → Rulesets**, or an equivalent branch protection rule, targeting `main`:

- Require a pull request and the intended approving reviews; dismiss stale approvals when the reviewed diff changes, and require resolution of review discussions.
- Require code-owner review for `.github/settings.yml`, `.github/workflows/`, the policy implementation and its dependencies, and `CODEOWNERS` itself. A CODEOWNERS file alone only assigns ownership; the GitHub rule must enforce the review requirement.
- Require the relevant CI and a dedicated workflow-scope check with a repository-wide unique name. Select the expected GitHub App producer, and require results for the latest candidate revision. Keep the branch up to date, or configure equivalent merge-queue validation.
- Restrict force pushes and deletion, and review bypass permissions so direct pushes or privileged actors cannot silently skip the intended review/check path.

See [branch protection settings](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches) and [code-owner enforcement](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners#codeowners-and-branch-protection).

Ensure the required policy reports on every applicable PR update. The example `plan` workflow is path-filtered; making it globally required without changing when it reports can leave unrelated PRs waiting forever. GitHub can accept skipped or neutral checks, so a security gate must explicitly fail when its required validation fails, is cancelled or does not execute; an always-running final gate can check prerequisite results. See [required-check troubleshooting](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks).

#### What the workflow-scope policy should check

Inspect all candidate workflow YAML as data and permit the privileged Environment only in the approved main workflow/job with the intended event and branch restriction. Account for scalar and mapping Environment syntax, dynamic names, YAML aliases/merges and reusable-workflow calls; reject constructs the policy cannot resolve safely. A text search for one spelling of `environment:` is insufficient.

Run the policy from a separately protected source, so a PR cannot change both an unsafe workflow and the test that is supposed to reject it. A trusted `pull_request_target` checker must have minimal read permissions and no apply credential. Never execute candidate code or install candidate-selected dependencies; do not run candidate local Actions or downloaded PR artifacts as code. Confirm that GitHub's applicable event policy allows the checker to run; see [secure use of `pull_request_target`](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target).

This repository's [Environment guard](contributing/repository-automation.md#active-repository-environment-guard) illustrates the scope check and its limits. It is not included in the two consumer examples. Its location/event check does not assess every command inside the permitted apply job; owner review must still cover steps, Action revisions and credential handling.

#### A check name is not a workflow identity

GitHub's required-status-check selection does not distinguish workflow files, matrices or event triggers. Selecting **GitHub Actions** as the expected App does not distinguish two jobs produced by that App. Unique names reduce collisions, but a same-named success does not establish that the intended trusted policy ran. Inspect the producer, workflow revision, event and candidate commit; do not infer those facts from a green badge. See [GitHub's documented status-check boundary](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks).

### When PR review is the only manual approval

PR review can be the sole manual gate only when enforced controls protect the privileged path without relying on a reviewer to catch an unexpected Environment request. Before removing Environment reviewers, establish all of the following:

1. Candidate changes cannot disable or rewrite the enforcing policy, satisfy it with an unrelated same-named job, or bypass required reviews/checks. Use an independently controlled enforcement source, such as an organization/enterprise ruleset requiring a specific protected workflow, or an external GitHub App check that validates the intended candidate and is required from that App.
2. The policy covers every route to the write credential, including changes to permitted job steps/dependencies and alternate Environment references. Keep the App key confined to the intended Environment, retain branch restrictions and least privilege, and require owner review of the privileged code.
3. Prove that invalid candidates and absent or substituted policy results cannot merge or reach credential issuance. A passing source test alone is not that evidence.

[Required ruleset workflows](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-workflows-to-pass-before-merging) depend on account capabilities and must have protected source and dependencies. Separately configured [Actions execution policies](https://docs.github.com/en/actions/how-tos/administer/control-workflow-execution) can restrict workflow paths, events and actors; verify their coverage and enforcement for the target repository. Neither capability is installed by these examples.

Until these conditions are met, retain Environment approval. Requiring an ordinary GitHub Actions check by name, including this project's `gh-settings` check, is not by itself evidence that a replacement approval boundary is enforceable.

## Review a pending run

Before clicking **Approve and deploy**, verify the following for the selected job:

| Review item | Expected evidence |
| --- | --- |
| Workflow and event | The approved main workflow and `apply` job, triggered by `push` to `refs/heads/main`; reject an unexpected Environment request from another path or event. |
| Actors and revisions | Original actor, rerun initiator, workflow-source revision and exact configuration commit; a rerun retains historical inputs. |
| Code and credential use | Reviewed checkout/token/Action steps and their pinned revisions, intended target repository and approved App/token permissions, with no added untrusted code in the job. |
| PR and checks | The relevant approved changes and the intended successful checks for the latest reviewed candidate, including the actual policy producer. |
| Configuration and effects | Complete PR plan Summary plus `.github/settings.yml` at the exact main commit, including label deletions, retained attributes and possible partial updates. |

The PR revision and resulting main commit can differ. Main does not depend on PR success or rerun plan before approval; its checkout uses `github.sha`, never a moving main branch. Inspect those actual revisions, and reject missing, incomplete or unexplained review evidence. Approval authorizes the job that receives the credential, including its code, rather than only the visible settings diff.

A plan is informational. The pinned package sends declared repository fields and topics directly; Labels alone reads fresh state before reconciliation. Neither consumes a saved plan. Intervening changes to managed fields can be overwritten; omitted fields stay outside the request. Review the [apply and partial-failure behavior](commands.md#apply-behavior) before approving changes.

The main job mints its token before the settings Action installs the CLI and validates the configuration. Installation or validation failure stops apply; the token Action still performs its normal post-job cleanup.

PR concurrency cancels superseded runs. Main uses a repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. These examples provide no custom queue, stale-commit check, drift/freshness guard or reapproval controller.

## Verify the protection before use

Use an isolated repository and dummy credentials for hostile or invalid workflow cases. Keep the live App key unavailable to those cases; these checks should establish blocking behavior without mutating production settings.

| Case | Expected result |
| --- | --- |
| Eligible main job awaiting approval | No checkout, token-creation or apply step has started. |
| Rejected approval or disallowed branch/tag | The job cannot reach token issuance or apply. |
| Extra privileged Environment reference, changed trigger or unresolved dynamic use | The trusted policy rejects the candidate and the enforced merge requirement blocks it. |
| Disabled policy or unrelated same-named success | The intended enforcing policy is still required; a replacement green badge cannot satisfy the chosen trust boundary. |
| New commit after approval of a PR | Applicable stale-review and current-revision check requirements prevent the old result from authorizing new content. |

Confirm the active GitHub rules and the actual blocked/pending runs. Local fixtures can prove parser and policy behavior, but cannot prove Environment protection, merge enforcement or real token revocation. Repeat the relevant verification when changing the guard, required-check source, privileged workflow, repository visibility or approval model.

## Failure and recovery

| Situation | Recovery |
| --- | --- |
| Unexpected Environment request or changed privileged code | Reject the request and inspect the workflow, policy and credential use before enabling further apply runs. If a credential was exposed, revoke/rotate it and review remote changes. |
| Missing/invalid config | Correct the declaration and rerun through normal review; validation fails before settings API access. |
| CLI download/checksum/version error | Inspect the release identity and reviewed pin. Do not execute an unchecked binary or switch to source/latest. |
| Plan read/API permission error | Check read-token access and the target origin. Keep administration credentials out of PR jobs. |
| Rejected Environment approval | The main job cannot start; review the configuration and protection policy before requesting another approval. |
| Missing App key/client ID or token error | Verify the protected Environment and dedicated installation scope using secure credential entry. |
| Failed/cancelled apply or output failure | Read current target settings before retrying: an update may already have succeeded. There is no atomicity or automatic rollback. |
| Token cleanup failure | Inspect the official Action's post-job logs and dedicated installation. Treat revocation as unverified and keep the token out of diagnostic output. |
| Disable further executions | Disable the adopted settings workflow in Actions and cancel pending/running executions as appropriate. Review updates already sent; disabling does not undo settings. |

## Contributor documentation

Implementation protocols belong in [Action internals](contributing/actions-internals.md), fixture and CI evidence in [testing](contributing/testing.md), and this project's own Environment/workflow policy in [repository automation](contributing/repository-automation.md). Those maintainer details are separate from consumer setup.

## References

- [GitHub contexts](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts)
- [Environment setup and supported controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
- [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510)

[^composite-metadata]: [GitHub composite metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax). Inputs and mapped step outputs; runtime checks remain Action-owned.
[^environment-controls]: [GitHub Environment controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments). Check availability and protection rules for the actual target before owner setup.
[^app-token-action]: [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510). Explicit target/permission inputs and default post-job revocation; actual revocation needs runtime evidence.
