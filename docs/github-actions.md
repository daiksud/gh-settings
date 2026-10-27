# Use the public settings Action

## Public Action and supported checkout

`daiksud/gh-settings` validates, plans or applies the checked-out repository's `.github/settings.yml`. The Action installs its selected CLI release and exposes normal `with` inputs and step outputs.[^composite-metadata] Consumers provide the checkout/data preparation, credentials and approval policy; they do not install the CLI or copy Action internals. The [selected Action pin](#published-package-and-consumer-pins) determines which CLI is installed.

Use a fresh GitHub-hosted `ubuntu-24.04` x64 job with Bash, curl, GNU sha256sum and Git. macOS, Windows, ARM, containers and self-hosted runners are unsupported. Checkout the configuration repository at `GITHUB_WORKSPACE`, without a checkout `path` or persisted credentials, and commit `.github/settings.yml` at its root.

A resolvable checkout commit and a directly configured GitHub.com `origin` are required even for validate's output metadata. Supported origins are HTTPS, `git@github.com:owner/repo.git` and full SSH URLs. The checkout's origin selects the target; the Action repository and `GH_REPO` do not. See [repository resolution](commands.md#repository-and-authentication) and [configuration semantics](settings.md#configuration-semantics).

Validate performs no settings API requests, but installing the public CLI still requires network access. Workflow, configuration and Action revisions must remain available for ordinary reruns.

## Inputs, outputs and failures

| Input | Contract |
| --- | --- |
| `command` | Required, no default; exactly `validate`, `plan` or `apply`. |
| `token` | Unused for validate; required and nonempty for plan/apply. Supply it explicitly; there is no ambient or implicit `github.token` fallback. |
| `plan-reporting` | `detailed` by default, or `status-only` for plan only. |

Validate checks local configuration without API authentication. The supported Actions setup uses one GitHub App for plan and apply, with one explicit installation-token input per Action call. The caller may mint different short-lived, repository/permission-scoped tokens from that same App for different jobs. Plan is nonmutating, but complete observation can require a write grant; see [authentication and App permissions](#authentication-and-app-permissions). The Action neither mints a token, identifies its credential type nor verifies its actual grants; there is no ambient-token or personal-credential fallback.

There are no inputs for export, arbitrary arguments, configuration path, target, CLI version or download URL. The Action's checkout metadata and explicit plan/apply token requirements also apply to configurations with no managed fields.

| Success output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit (`HEAD`), not the settings file hash, workflow SHA or Action SHA. |
| `target` | Configuration checkout's resolved GitHub `origin`, in `OWNER/REPO` form. |
| `cli-version` | Verified installed CLI release selected by the Action package. |
| `cli-sha256` | SHA-256 of the verified installed binary. |
| `plan-outcome` | `changed`, `no-difference` or `unmanaged` after a successful status-only plan and Summary write; empty for detailed/validate/apply calls. |

Use `steps.<id>.outputs.<name>` only after successful completion. Outputs are published after the command and, for plan, successful Summary rendering, writing and ephemeral report cleanup. Differences, no differences and no managed fields are successful plan outcomes.

Missing command/token, unsupported prerequisites, install/metadata/output failures and CLI/configuration/API errors fail the Action. A nonzero plan status is preserved; otherwise Summary status determines success. Failed calls have no successful output contract. Plan Summary displays escaped settings values and categorical failure diagnostics, and distinguishes workflow, configuration, Action and CLI identities. Validate/apply report through the Action step's outcome and logs; they do not render a plan Summary.

### Private plan reporting

For plan, `plan-reporting: status-only` It displays only a validated outcome and bounded managed count; it omits policy values, names, actors and raw diagnostics. Setup checks the installed binary's private-reporting capability before authenticated work and reports a fixed error for an unsupported package. This package capability does not activate this repository's protected observer: #380 remains open for its dedicated Environment, credentials, activation and real nonmutating verification.

Status-only failures use fixed diagnostics and preserve failure status. Detailed reports stay in the installer's private temporary directory with mode 0600 and are removed by an `always()` cleanup step, including after ordinary failures. No report is uploaded or cached. Forced runner termination can prevent cleanup; use a fresh supported hosted runner, and never arrange for other steps to cache/upload its temporary data. Metadata success outputs still identify the caller's checkout and installed package; the caller must supply a trusted target/revision and must not inject confidential data into public identities or caller logs.

This mode does not make a write-capable token read-only or protect it from other code in the shared job. Use trusted workflow/Action code, handle candidate configuration strictly as data, and keep credential acquisition and lifecycle under the caller's approved boundary. A no-difference result is an observation, not a lock or authority for a later apply.

## Published package and consumer pins

The examples pin the immutable [v0.21.0 Action/CLI package](https://github.com/daiksud/gh-settings/releases/tag/v0.21.0) by its full source SHA `eb8795dca8bb42376ab575f646ffa11bfe7775c8`. The Action installs that exact release, not arbitrary current source code or the latest release; future pin changes require separately reviewed adoption. Published assets and `SHA256SUMS` are available on the release page.

Installation downloads the selected `gh-settings-linux-amd64` asset and `SHA256SUMS` from fixed upstream HTTPS release paths. It requires one valid checksum for the exact asset, verifies before execution and checks exact `gh-settings <version>` output. There is no latest fallback or caller version/URL override. The binary and checksum come from the same GitHub release; the Action SHA fixes the selected version, not an independent binary digest.

### Labels and permissions

The selected published CLI determines supported [label behavior](settings.md#labels).
**Safety:** The pinned v0.21.0 CLI uses ID-selected labels and repository
Rulesets. Unlisted objects are left untouched, and omitted or explicitly empty
collections never request implicit deletion. An entry without an ID is
create-only. Updating a listed Ruleset still sends supported whole-object
values and defaults, which may alter required reviews, checks, or bypasses.
Review the [Rulesets contract](settings.md#rulesets), observed plan, and
pinned binary before approving apply.

Listing needs Issues or Pull requests read, except public resources can be listed without those grants. Label mutations need either permission at write level; Administration write alone is insufficient. The default example App token still requests only Administration write for repository fields/topics. For label mutations, separately approve the App/installation permission and a matching token request scoped to the containing repository. Installing or pinning the Action does not grant permissions.

The pinned v0.21.0 label apply is ID-based and never deletes unlisted labels. An ID-less collision fails with ID guidance. The CLI disables ID writeback whenever `GITHUB_ACTIONS=true`; created label/Ruleset IDs remain available in output or error diagnostics for manual adoption. See the [labels contract](settings.md#labels).

## Consumer files

The [copyable PR/main examples](../examples/github-actions/README.md) need two workflow files plus `.github/settings.yml`. No caller script, root `action.yml` or copied `action/` directory is required.

The PR workflow runs trusted base workflow code on `pull_request_target`, obtains only pinned candidate settings data, and plans using a token from the same App as main. Main uses one job with three steps: checkout, App token creation and direct apply. It checks out the event's exact protected-main configuration SHA after merge. Main does not consume a PR artifact, depend on the PR run, or produce another plan before approval.

## Authentication and App permissions

The two supported use cases are distinct:

- **GitHub CLI extension:** use the authentication already selected by `gh`, normally the user's own authority. There is no independent gh-settings credential store or retry with another identity. See [CLI authentication](commands.md#repository-and-authentication).
- **GitHub Actions:** use one installed GitHub App for the supported settings APIs. Do not require a PAT/personal login alongside the App to complete a feature. The App/installation grants are the ceiling; each job mints a short-lived token narrowed to its target and needed permissions, with normal post-job revocation. One App does not mean one endlessly reused token string. `validate` needs no settings API credential; installation/download and checkout metadata are separate prerequisites.

The following are resource-specific boundaries, not a request to grant all permissions. Metadata read is automatic for installations. Add only the permissions needed by the actual declaration to both the App installation and token request; no workflow here changes the installation's grants. Public unauthenticated reads do not establish private access or mutation permission.

| Feature/API | Plan/observation boundary | Apply boundary |
| --- | --- | --- |
| Repository REST fields, topics, archive | Metadata read | Administration write; archive includes fresh identity/readback |
| Merge/default/PR/Issue/Sponsor GraphQL fields | Authenticated access to the selected query/objects; verify exact query with the App | Repository update authorization; verify exact mutation with the App |
| Labels | Issues **or** Pull requests read | The same permission family at write level; Administration alone is insufficient |
| Repository Rulesets | Metadata read for GET **plus write access to the Ruleset for complete bypass visibility** | Administration write, with complete observation and account eligibility |
| Actions execution, workflow permissions, retention, contributor fork approval | Administration read | Administration write |
| Secret scanning/push protection | Complete repository security-block visibility; Metadata read alone is insufficient | Administration write plus applicable security authority/entitlement and complete readback |
| Private vulnerability reporting | Metadata read and supported public applicability | Administration write and applicable repository authority |
| Existing Pages | Pages read | Pages write plus Administration write and applicable repository authority |
| Named Environments (unreleased source only) | Actions read, including enabled custom-protection visibility | Administration write plus complete supported observation/readback |
| PR sample's candidate file retrieval | Contents read for filtered Git metadata and the exact settings blob | No mutation; not an apply credential |

GitHub documents installation-token support for the relevant [repository/topic](https://docs.github.com/en/rest/repos/repos), [label](https://docs.github.com/en/rest/issues/labels), [Ruleset](https://docs.github.com/en/rest/repos/rules#get-a-repository-ruleset), [Actions permissions](https://docs.github.com/en/rest/actions/permissions), [Pages](https://docs.github.com/en/rest/pages/pages) and [Environment](https://docs.github.com/en/rest/deployments/environments) REST APIs. The Ruleset GET explicitly limits `bypass_actors` to actors with write access: nonmutating plan is **not** proof that a read-only installation token is sufficient. Missing visibility must fail, never become an empty bypass list.

GitHub also [supports installation-token GraphQL and HTTP Git access](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation), but instructs callers to test the permissions for their exact GraphQL queries/mutations. Documentation of the API family is not live verification of every gh-settings field. App compatibility, complete visibility and required permissions are feature-adoption conditions; if the selected App cannot perform a needed operation, stop and investigate that boundary rather than combine it with personal credentials, weaken validation or silently skip the field. This change runs only local fixtures: it does not claim real App access, private-fork retrieval, token revocation, protection enforcement or successful live writes. Follow the selected package's feature/version contract; unreleased source support does not extend the pinned binary.

### Trusted PR data, not PR code

The PR example has no PR checkout, build, dependency install, local Action or artifact execution. A fixed script in the trusted base workflow validates the event's immutable head SHA and PR number, fetches commit/tree metadata with blob filtering, verifies the fetched head, and obtains only `.github/settings.yml` by its Git blob identity. It rejects symlinks/executable file mode, missing paths, malformed/oversized data, redirects, identity mismatch and failed reads. Git hooks, inherited Git programs/credentials and lazy object downloads are disabled. The prepared root contains configuration plus Git metadata; its `HEAD` identifies the candidate head and its fixed base-repository `origin` identifies the target. The workflow-source SHA remains the trusted base SHA, not that configuration SHA.

Candidate bytes are never evaluated as shell, expressions, scripts or workflow code. No PR-controlled executable file is checked out, and no token is persisted to Git configuration. The trusted pinned public Action alone consumes YAML as configuration data. Failed preparation stops before settings observation and exposes only a fixed diagnostic. PR refs/objects must be readable with the same App; absence or denied access fails instead of switching to a fork credential or PAT.

Strongly recommend exact Environment Branch `main` restrictions **and** enforced protection/review of workflow and credential-consuming paths, pinned Action dependencies and the enforcement policy itself. Protect the plan credential path as well as apply, especially when complete Ruleset reads require write authority. Branch restrictions identify a ref, not a workflow: another workflow on main could still request the same Environment. `pull_request_target` is safe only while its executing code remains trusted; it is not permission to execute candidate content. Check the repository's applicable [Actions event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target) before adoption. These examples do not opt out of that policy or configure protections.

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
2. Install one selected-repositories App on the target. The PR sample needs Contents read for candidate metadata/blob retrieval; repository fields/topics apply needs Administration write and automatic Metadata read. Other resources need the separately approved [feature permissions](#authentication-and-app-permissions). The official token Action narrows each token with explicit `owner`, `repositories` and permission inputs; retain its default post-job revocation.[^app-token-action]
3. Configure the examples' exact `settings-plan` and `settings-apply` Environments before use. Strongly recommend **Selected branches and tags** with a **Branch** rule for exactly `main` and no Tag rule, plus protection of their workflow/credential-consuming paths. For apply, enable **Required reviewers** and disable administrator bypass; use equivalent approval for a write-capable plan token. The single-owner example permits self-approval; enable **Prevent self-review** and provide another eligible reviewer when independent approval is required. One listed reviewer can approve; a list does not require every reviewer to approve.
4. Adopt the [two workflows](../examples/github-actions/README.md#copy-the-workflows) and root configuration through review. Retain the exact-event checkout, no persisted credentials, `push`/`refs/heads/main` guards, reviewed full-SHA Action pins and concurrency. [Verify the protection](#verify-the-protection-before-use) with an isolated setup before supplying a live write credential.
5. Bind `SETTINGS_APP_CLIENT_ID` and `SETTINGS_APP_PRIVATE_KEY` to the **same App** in the two protected Environments. Keep the private key out of alternate unprotected credential paths. Mint tokens only inside the intended trusted jobs; the PR sample requests Contents read/Administration read, while main requests Administration write. Adjust token permissions only for separately adopted features under the table above. This document is setup guidance, not authorization to create Environments, enter keys or change App grants.

An unknown Environment name in workflow YAML can create an Environment with no protection rules or secrets. Merely adding `environment: settings-apply` does not configure approval. Required reviewers on Free, Pro and Team are available only for public repositories; check feature availability for the actual repository before relying on the gate.[^environment-controls]

### Environment approval and workflow identity

An Environment's protection rules are checked before the referencing job starts, and its secrets are unavailable until the gate passes. In the example, this places checkout, token creation and apply after approval. Rejecting the deployment stops that job. See [GitHub's deployment review procedure](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/review-deployments).

Environment branch rules match the run's ref, rather than selecting a workflow file or event. A different workflow running with an allowed `main` ref could also request the Environment. In particular, a privileged event such as `pull_request_target` must not be treated as safe merely because it uses a trusted base ref. Review the workflow identity as well as the configuration. Avoid **Protected branches only** as a substitute for the exact Branch rule: when no branches have protection rules, that option permits every branch. These boundaries follow [GitHub's Environment rules](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments).

Keep plan and apply jobs limited to reviewed code. Passing `token` only to intended data/API steps limits ordinary exposure, but all steps share the job's runner; another step can compromise later credential use. Keep untrusted PR builds, local Actions, scripts and artifacts out of credential-bearing jobs. Never give this App key/token to an ordinary PR-code job. Review third-party Action code at selected revisions; SHA pinning fixes the selected code, not whether it is trustworthy. See [GitHub's secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use).

### Protect main with required checks

Use an **Active** ruleset under **Settings → Rules → Rulesets**, or an equivalent branch protection rule, targeting `main`:

- Require a pull request and the intended approving reviews; dismiss stale approvals when the reviewed diff changes, and require resolution of review discussions.
- Require code-owner review for `.github/settings.yml`, `.github/workflows/`, the policy implementation and its dependencies, and `CODEOWNERS` itself. A CODEOWNERS file alone only assigns ownership; the GitHub rule must enforce the review requirement.
- Require the relevant CI and a dedicated workflow-scope check with a repository-wide unique name. Select the expected GitHub App producer, and require results for the latest candidate revision. Keep the branch up to date, or configure equivalent merge-queue validation.
- Restrict force pushes and deletion, and review bypass permissions so direct pushes or privileged actors cannot silently skip the intended review/check path.

See [branch protection settings](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches) and [code-owner enforcement](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners#codeowners-and-branch-protection).

Ensure the required policy reports on every applicable PR update. The example `plan` workflow is path-filtered; making it globally required without changing when it reports can leave unrelated PRs waiting forever. GitHub can accept skipped or neutral checks, so a security gate must explicitly fail when its required validation fails, is cancelled or does not execute; an always-running final gate can check prerequisite results. See [required-check troubleshooting](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks).

#### What the workflow-scope policy should check

Inspect all candidate workflow YAML as data and permit each credential-bearing Environment only in its approved trusted workflow/job: target-event data-only plan or protected-main push apply. Account for scalar and mapping Environment syntax, dynamic names, YAML aliases/merges and reusable-workflow calls; reject constructs the policy cannot resolve safely. A text search for one spelling of `environment:` is insufficient.

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
| Plan read/API permission error | Check the same App's target/feature grants and complete visibility. A write grant may be required for observation; do not fall back to a PAT, personal login, empty state or PR-code execution. |
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

## Fork workflow policies

[Fork workflow policies](settings.md#fork-pr-workflow-policies), with public contributor approval separate from private fork execution/token/secrets/approval controls. Existing Administration read/write credentials are required; no Action input or grant is added. Summary shows exact enum and native boolean differences, including credentials passed to forks. It remains credential-free presentation; apply independently rereads state. Fixtures do not establish live mutation compatibility, workflow execution safety or organization-policy enforcement.

[Existing Pages configuration](settings.md#existing-pages-configuration) through validate/plan/apply and typed native Summary entries, including a domain-clearing null and inactive workflow source. Pages reads require Pages read; updates require Pages write and Administration write plus the applicable actor authorization. Existing caller grants and Action inputs are unchanged; no export input exists. Configuration success does not promise deployment/DNS/certificate readiness, and partial update failures retain prior effects. Site creation/deletion and visibility changes remain unsupported under #350.
