# Protect the apply path

An apply job receives authority to change repository settings. Protect both the code that reaches `main` and the job that receives the write credential. Validation checks the settings schema; it cannot authorize other commands or Actions sharing that job.

| Control | What it protects | What it does not establish |
| --- | --- | --- |
| Required PR reviews and status checks on `main` | Whether candidate settings and workflow changes can merge | Whether a particular job should receive a credential now |
| Environment reviewers and deployment restrictions | Whether a referencing job may start and access its Environment secrets | That only the intended workflow can reference that Environment |
| Trusted workflow-scope policy | Whether the inspected candidate keeps privileged Environment use on the approved path | That an ordinary green check name uniquely identifies the policy that ran |

The examples use Environment approval. If PR review is intended to be the only manual approval, first satisfy the [separate enforcement conditions](#when-pr-review-is-the-only-manual-approval). Copying the workflows does not configure GitHub protection settings or install a workflow-scope policy.

## Set up the two Environments

Create `settings-plan` and `settings-apply` before any workflow refers to them. An unknown name can create an Environment with no protection rules or secrets. Choose **Selected branches and tags** with a **Branch** rule for exactly `main` and no Tag rule. Require approval for apply and for a write-capable plan job. Disable administrator bypass where available. Enable **Prevent self-review** and choose another eligible reviewer if independent approval is required. One eligible reviewer can approve a pending job. Verify that these controls are available for the repository. [GitHub Environment settings](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)

## Environment approval and workflow identity

An Environment's protection rules are checked before the referencing job starts, and its secrets are unavailable until the gate passes. In the example, this places checkout, token creation and apply after approval. Rejecting the deployment stops that job. See [GitHub's deployment review procedure](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/review-deployments).

Environment branch rules match the run's ref, rather than selecting a workflow file or event. A different workflow running with an allowed `main` ref could also request the Environment. In particular, a privileged event such as `pull_request_target` must not be treated as safe merely because it uses a trusted base ref. Review the workflow identity as well as the configuration. Avoid **Protected branches only** as a substitute for the exact Branch rule: when no branches have protection rules, that option permits every branch. These boundaries follow [GitHub's Environment rules](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments).

Keep plan and apply jobs limited to reviewed code. Passing `token` only to intended data/API steps limits ordinary exposure, but all steps share the job's runner; another step can compromise later credential use. Keep untrusted PR builds, local Actions, scripts and artifacts out of credential-bearing jobs. Never give this App key/token to an ordinary PR-code job. Review third-party Action code at selected revisions; SHA pinning fixes the selected code, not whether it is trustworthy. See [GitHub's secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use).

## Protect main with required checks

Use an **Active** ruleset under **Settings → Rules → Rulesets**, or an equivalent branch protection rule, targeting `main`:

- Require a pull request and the intended approving reviews; dismiss stale approvals when the reviewed diff changes, and require resolution of review discussions.
- Require code-owner review for `.github/settings.yml`, `.github/workflows/`, the policy implementation and its dependencies, and `CODEOWNERS` itself. A CODEOWNERS file alone only assigns ownership; the GitHub rule must enforce the review requirement.
- Require the relevant CI and a dedicated workflow-scope check with a repository-wide unique name. Select the expected GitHub App producer, and require results for the latest candidate revision. Keep the branch up to date, or configure equivalent merge-queue validation.
- Restrict force pushes and deletion, and review bypass permissions so direct pushes or privileged actors cannot silently skip the intended review/check path.

See [branch protection settings](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches) and [code-owner enforcement](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners#codeowners-and-branch-protection).

Ensure the required policy reports on every applicable PR update. The example `plan` workflow is path-filtered; making it globally required without changing when it reports can leave unrelated PRs waiting forever. GitHub can accept skipped or neutral checks, so a security gate must explicitly fail when its required validation fails, is cancelled or does not execute; an always-running final gate can check prerequisite results. See [required-check troubleshooting](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks).

### What the workflow-scope policy should check

Inspect all candidate workflow YAML as data and permit each credential-bearing Environment only in its approved trusted workflow/job: target-event data-only plan or protected-main push apply. Account for scalar and mapping Environment syntax, dynamic names, YAML aliases/merges and reusable-workflow calls; reject constructs the policy cannot resolve safely. A text search for one spelling of `environment:` is insufficient.

Run the policy from a separately protected source, so a PR cannot change both an unsafe workflow and the test that is supposed to reject it. A trusted `pull_request_target` checker must have minimal read permissions and no apply credential. Never execute candidate code or install candidate-selected dependencies; do not run candidate local Actions or downloaded PR artifacts as code. Confirm that GitHub's applicable event policy allows the checker to run; see [secure use of `pull_request_target`](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target).

This repository's [Environment scope tests](../development/repository-automation.md#repository-environment-scope-tests) illustrate the scope check and its limits. They run with candidate source in ordinary CI, so they do not supply an independent trusted policy. They are not included in the two consumer examples. Owner review must still cover apply steps, Action revisions and credential handling.

### A check name is not a workflow identity

GitHub's required-status-check selection does not distinguish workflow files, matrices or event triggers. Selecting **GitHub Actions** as the expected App does not distinguish two jobs produced by that App. Unique names reduce collisions, but a same-named success does not establish that the intended trusted policy ran. Inspect the producer, workflow revision, event and candidate commit; do not infer those facts from a green badge. See [GitHub's documented status-check boundary](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules#troubleshooting-required-status-checks).

## When PR review is the only manual approval

PR review can be the sole manual gate only when enforced controls protect the privileged path without relying on a reviewer to catch an unexpected Environment request. Before removing Environment reviewers, establish all of the following:

1. Candidate changes cannot disable or rewrite the enforcing policy, satisfy it with an unrelated same-named job, or bypass required reviews/checks. Use an independently controlled enforcement source, such as an organization/enterprise ruleset requiring a specific protected workflow, or an external GitHub App check that validates the intended candidate and is required from that App.
2. The policy covers every route to the write credential, including changes to permitted job steps/dependencies and alternate Environment references. Keep the App key confined to the intended Environment, retain branch restrictions and least privilege, and require owner review of the privileged code.
3. Prove that invalid candidates and absent or substituted policy results cannot merge or reach credential issuance. A passing source test alone is not that evidence.

[Required ruleset workflows](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-workflows-to-pass-before-merging) depend on account capabilities and must have protected source and dependencies. Separately configured [Actions execution policies](https://docs.github.com/en/actions/how-tos/administer/control-workflow-execution) can restrict workflow paths, events and actors; verify their coverage and enforcement for the target repository. Neither capability is installed by these examples.

Until these conditions are met, retain Environment approval. Requiring an ordinary GitHub Actions check by name, including this project's `gh-settings` check, is not by itself evidence that a replacement approval boundary is enforceable.

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

## Review a pending run

Before clicking **Approve and deploy**, verify the following for the selected job:

| Review item | Expected evidence |
| --- | --- |
| Workflow and event | The approved main workflow and `apply` job, triggered by `push` to `refs/heads/main`; reject an unexpected Environment request from another path or event. |
| Actors and revisions | Original actor, rerun initiator, workflow-source revision and exact configuration commit; a rerun retains historical inputs. |
| Code and credential use | Reviewed checkout/token/Action steps and their pinned revisions, intended target repository and approved App/token permissions, with no added untrusted code in the job. |
| PR and checks | The relevant approved changes and the intended successful checks for the latest reviewed candidate, including the actual policy producer. |
| Configuration and effects | Complete PR plan Summary plus `.github/settings.yml` at the exact main commit, including planned creations and updates, any deletions enabled through explicit pruning, and the effects of a partial apply. |

The PR revision and resulting main commit can differ. Main does not depend on PR success or rerun plan before approval; its checkout uses `github.sha`, never a moving main branch. Inspect those actual revisions, and reject missing, incomplete or unexplained review evidence. Approval authorizes the job that receives the credential, including its code, rather than only the visible settings diff.

A plan is informational. The pinned package reads fresh managed state, skips matching values and verifies supported changes independently under each family's contract. Apply does not consume a saved plan. Apply can overwrite a change made after the plan. Check what each selected object owns. For example, omitting a selected label's description clears it, and omitting a supported rule from a selected Ruleset removes that rule. Review the [ownership rules](../references/settings/configuration.md#configuration-semantics) and [partial-failure behavior](../references/commands.md#apply) before approving.

The main job mints its token before the settings Action installs the CLI and validates the configuration. Installation or validation failure stops apply; the token Action still performs its normal post-job cleanup.

PR concurrency cancels superseded runs. Main uses a repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. These examples provide no custom queue, stale-commit check, drift/freshness guard or reapproval controller.

Return to [workflow setup](github-actions.md#ordered-owner-setup) to bind the App and run the first plan.
