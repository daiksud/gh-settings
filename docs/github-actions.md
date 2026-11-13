# Use the public settings Action

## Public Action and supported checkout

Current-source `daiksud/gh-settings` exports, validates, plans or applies the checked-out repository's `.github/settings.yml`. The Action installs its selected CLI release and exposes normal `with` inputs and step outputs.[^composite-metadata] Consumers provide the checkout/data preparation, credentials and approval policy; they do not install the CLI or copy Action internals. The [selected Action pin](#published-package-and-consumer-pins) determines which CLI is installed.

Use a fresh GitHub-hosted `ubuntu-24.04` x64 job with Bash, curl, GNU sha256sum and Git. macOS, Windows, ARM, containers and self-hosted runners are unsupported. Checkout the configuration repository at `GITHUB_WORKSPACE`, without a checkout `path` or persisted credentials, and keep `.github/settings.yml` at its root. Export can create the file and its `.github` directory on first use; validate/plan/apply require an existing configuration.

A resolvable checkout commit and a directly configured GitHub.com `origin` are required even for validate's output metadata. Supported origins are HTTPS, `git@github.com:owner/repo.git` and full SSH URLs. The checkout's origin selects the target; the Action repository and `GH_REPO` do not. See [repository resolution](commands.md#repository-and-authentication) and [configuration semantics](settings.md#configuration-semantics).

Validate performs no settings API requests, but installing the public CLI still requires network access. Workflow, configuration and Action revisions must remain available for ordinary reruns.

## Inputs, outputs and failures

| Input | Contract |
| --- | --- |
| `command` | Required, no default; exactly `export`, `validate`, `plan` or `apply`. |
| `token` | Unused for validate; required and nonempty for export/plan/apply. Supply it explicitly; there is no ambient or implicit `github.token` fallback. |
| `plan-reporting` | `detailed` by default, or `status-only` for plan only. |
| `prune` | Optional comma-separated deletion scopes for plan/apply; default empty. Current source also supports `actions-variables`, `agents-variables`, `environment-variables`, `actions-policies`, `environments`, `collaborators`, `branch-protection-rules`, `autolinks` and `deploy-keys`; see [pruning](#explicit-collection-pruning). |

Validate checks local configuration without API authentication. The supported Actions setup uses one GitHub App for export, plan and apply, with one explicit installation-token input per Action call. The caller may mint different short-lived, repository/permission-scoped tokens from that same App for different jobs. Export and plan are nonmutating, but complete observation can require a write grant; see [authentication and App permissions](#authentication-and-app-permissions). The Action does not mint a token or certify its actual grants; collaborator prune additionally requires positive management-path evidence using that same credential; there is no ambient-token or personal-credential fallback.

There are no inputs for arbitrary arguments, configuration path, target, CLI version or download URL. Nonempty `prune` is rejected for export and validate before installation. The Action's checkout metadata and explicit export/plan/apply token requirements also apply to configurations with no managed fields.

| Success output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit (`HEAD`), not the settings file hash, workflow SHA or Action SHA. |
| `config-path` | `.github/settings.yml`, relative to the checkout root, after successful export only; empty for other commands. It does not identify a committed file or transfer it to another job. |
| `target` | Configuration checkout's resolved GitHub `origin`, in `OWNER/REPO` form. |
| `cli-version` | Verified installed CLI release selected by the Action package. |
| `cli-sha256` | SHA-256 of the verified installed binary. |
| `plan-outcome` | `changed`, `no-difference` or `unmanaged` after a successful status-only plan and Summary write; empty for export, validate, apply and detailed plans. |

Use `steps.<id>.outputs.<name>` only after successful completion. Outputs are published after the command and, for plan, successful Summary rendering, writing and ephemeral report cleanup. Differences, no differences and no managed fields are successful plan outcomes.

Missing command/token, unsupported prerequisites, install/metadata/output failures and CLI/configuration/API errors fail the Action. A nonzero plan status is preserved; otherwise Summary status determines success. Failed calls have no successful output contract. Plan Summary displays escaped settings values and categorical failure diagnostics, and distinguishes workflow, configuration, Action and CLI identities. Export/validate/apply report through the Action step's outcome and logs; they do not render a plan Summary.

### Export a local configuration

Use `command: export` with an explicit installation token in a trusted checkout.
Choose a reviewed Action package that implements export; the immutable v0.27.0
pin shown below does not gain this capability from current source changes.
The Action uses the same [automatic export](commands.md#export-behavior) as the
CLI and returns `steps.<id>.outputs.config-path` only after success.
`config-sha` still identifies checkout HEAD, rather than the generated file.

Export attempts independent supported reads. Unavailable families produce bounded
info and are omitted initially or retain compatible valid existing intent. A
successful export does not certify that every family was readable. Invalid
existing YAML, symlinks, unsafe destinations, concurrent edits and local I/O errors
retain the shared CLI failure boundaries. No settings file is required for first
export, but a resolvable checkout commit and direct origin are always required.

The generated file can contain plaintext non-secret Variables. Export does not
commit, push, upload, cache or render its contents in logs or Summary. It disables
inherited `GH_DEBUG` during acquisition and discards command stdout; normal export
diagnostics remain on stderr. The path output leaves publication to the caller.
Other code in the same trusted job can still read the local file and credential.

Failed info or required normalization-warning delivery follows the shared CLI
failure contract: no configuration is saved and the Action has no successful
outputs. Readable observations do not override that diagnostic failure.

Export does not accept `prune` or `plan-reporting: status-only`, and creates no
plan report or Summary. A failed first-file write can leave a local file, and
result-output failure can occur after a completed export. Failed calls have no
successful-output contract and do not roll back local changes.

### Private plan reporting

For plan, `plan-reporting: status-only` displays only a validated outcome and bounded managed count; it omits policy values, names, actors and raw diagnostics. Setup checks the installed binary's private-reporting capability before authenticated work and reports a fixed error for an unsupported package. This optional public Action capability is independent of this repository's operational workflows.

Status-only failures use fixed diagnostics and preserve failure status. Detailed reports stay in the installer's private temporary directory with mode 0600 and are removed by an `always()` cleanup step, including after ordinary failures. No report is uploaded or cached. Forced runner termination can prevent cleanup; use a fresh supported hosted runner, and never arrange for other steps to cache/upload its temporary data. Metadata success outputs still identify the caller's checkout and installed package; the caller must supply a trusted target/revision and must not inject confidential data into public identities or caller logs.

This mode does not make a write-capable token read-only or protect it from other code in the shared job. Use trusted workflow/Action code, handle candidate configuration strictly as data, and keep credential acquisition and lifecycle under the caller's approved boundary. A no-difference result is an observation, not a lock or authority for a later apply.

## Published package and consumer pins

The examples pin the immutable [v0.27.0 Action/CLI package](https://github.com/daiksud/gh-settings/releases/tag/v0.27.0) by its full source SHA `6dbdbb3ab11cbc0ef257fef089d6915d31d003d6`. The Action installs that exact release, not arbitrary current source code or the latest release; future pin changes require separately reviewed adoption. Published assets and `SHA256SUMS` are available on the release page.

Installation downloads the selected `gh-settings-linux-amd64` asset and `SHA256SUMS` from fixed upstream HTTPS release paths. It requires one valid checksum for the exact asset, verifies before execution and checks exact `gh-settings <version>` output. There is no latest fallback or caller version/URL override. The binary and checksum come from the same GitHub release; the Action SHA fixes the selected version, not an independent binary digest.

### Labels and permissions

The selected published CLI determines supported [label behavior](settings.md#labels).
**Safety:** The pinned v0.27.0 CLI uses ID-selected labels and repository
Rulesets. Unlisted objects are left untouched, and omitted or explicitly empty
collections never request implicit deletion. An entry without an ID is
create-only. Updating a listed Ruleset still sends supported whole-object
values and defaults, which may alter required reviews, checks, or bypasses.
Review the [Rulesets contract](settings.md#rulesets), observed plan, and
pinned binary before approving apply.

Listing needs Issues or Pull requests read, except public resources can be listed without those grants. Label mutations need either permission at write level; Administration write alone is insufficient. The default example App token still requests only Administration write for repository fields/topics. For label mutations, separately approve the App/installation permission and a matching token request scoped to the containing repository. Installing or pinning the Action does not grant permissions.

The pinned v0.27.0 label apply is ID-based and never deletes unlisted labels. An ID-less collision fails with ID guidance. The CLI disables ID writeback whenever `GITHUB_ACTIONS=true`; created label/Ruleset IDs remain available in output or error diagnostics for manual adoption. See the [labels contract](settings.md#labels).

### Actions event policies

Current-source CLI and Action planning/apply support
[Actions event policies](settings.md#actions-event-policies). Check the selected
published package before adding this key; this source change does not update a
caller's pin. Fine-grained/App tokens need **Administration write even for
list/GET and plan**. Treat the plan credential as write-capable and protect its
path accordingly; a read-only Administration token is insufficient.

Only repository-owned event/actor policies with supported workflow targeting are
managed. `target_workflows` declares include/exclude patterns; omission preserves
existing targeting on update and uses All workflows on create. Unlisted
objects remain unchanged, and there is no delete. GitHub Actions mode suppresses
local created-ID writeback; record confirmed IDs from output/recovery diagnostics
before retrying. A failed write/readback can have an uncertain effect, and
creation is never automatically retried. Local fixtures cover active/disabled
metadata transitions; actual enforcement and workflow execution were not tested.
Disabled-policy storage readback does not establish event enforcement. Full
reports identify selected policy objects; status-only Summary hides their names
and values, while the intermediate report still follows the protected-report
lifecycle.

### Repository variables and plaintext reporting

Current-source CLI and Action planning/apply accept separate [Actions repository variables](settings.md#actions-repository-variables) and [Agents repository variables](settings.md#agents-repository-variables); this does not assert support in the pinned published package. Values must be non-sensitive. Detailed plan artifacts, logs and Summary can contain their plaintext values; status-only Summary hides variable names and values but its intermediate report still contains them under the existing protected-report lifecycle. The same name in both families retains separate values and resource labels. Do not store credentials in variables. No Secrets, Environment or organization variable endpoint is used, and no App grant or token-request permission is added by this implementation. Supply only the **Variables** or **Agent variables** permissions needed by the respective selected declaration.

### Actions OIDC subject template

Current source supports the [active OIDC template](settings.md#actions-oidc-subject-template)
through shared export, plan and apply use cases. Detailed reports show the complete
active mode and ordered claim keys; status-only output hides them. Apply publishes
success outputs only after a matching no-op or verified update. Select a package
that includes this capability; the examples' existing pin is unchanged. Changing
the subject requires coordination with the caller's cloud trust configuration.

### Copilot MCP tools

Current-source planning/apply support the independently declared [MCP tools toggle](settings.md#copilot-code-review-mcp-tools). Check support in the selected published package before using the key; this source change does not update the caller's pin. Full reports show native booleans at the fixed UI path; status-only Summary hides the control and values. The toggle stores configuration only, without managing MCP servers, secrets, grants, review entitlement, execution or billing. Repository administration and endpoint authorization remain caller prerequisites; exact fine-grained/App-token grants were not independently established. Apply verifies both the requested toggle and preservation of acquired sibling settings.

### Copilot review effort

Current-source planning/apply support the independently declared [Review effort level](settings.md#copilot-code-review-effort-level), including Max when the repository reports it as available. Confirm support in the selected published package before adding the key; this source change does not update a caller's pin. Full reports use string choices at the UI path; status-only output hides them. Default means the API's observable configured-null state and is not proof of a cleared hidden override. Unavailable or unsupported effort observations fail rather than falling back. The existing Copilot endpoint credential and review-entitlement boundaries below apply equally; no permission, license or review is created.

### Copilot custom instructions

Current-source planning/apply support the [Use custom instructions toggle](settings.md#copilot-code-review-custom-instructions). Check that the selected published package supports the key before adding it to a declaration; this source addition does not update the caller's pin. The API stores only configuration, not Copilot access or review activation. No instructions file, review request, workflow/token grant or license is created. Exact App-token permission grants and instruction consumption were not verified by the owner-authenticated storage probe. Full reports show native booleans; the existing status-only lifecycle hides the control and values in its published Summary.

## Personal collaborators

Current-source export/plan/apply supports the
[personal collaborator contract](settings.md#personal-collaborators). Confirm
support in the selected immutable package; this change does not update its pin.
Detailed plan Summary distinguishes initial invitations, pending acceptance and
active access, including matching pending people. Explicit `prune: collaborators`
(or standalone `all`) additionally shows invitation cancellations and grant removals
for a present list. The same explicit installation token must prove target access
and complete admin-only invitation visibility; no second credential or installation
ID is used. See [prune guards and side effects](settings.md#collaborator-pruning).
Status-only publication omits
recipient identities. Action apply never writes user IDs into the checkout;
record the reported verified user IDs through a reviewed declaration update
before relying on scheduled no-reinvitation behavior. An invitation is not proof
of accepted access. No installation grant or recipient acceptance is created.

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
| Personal collaborators (current source) | Complete collaborator inventory (Metadata read and documented caller access) plus Administration read for all pending invitations | Administration write; bodyless initial invitation or explicit prune DELETE, fresh identity/management-path guards and independent complete readback |
| Deploy keys (current source) | Administration read, complete personal-public target and key inventory | Administration write; immutable creation or explicit guarded deletion with independent readback |
| Labels | Issues **or** Pull requests read | The same permission family at write level; Administration alone is insufficient |
| Repository Rulesets | Metadata read for GET **plus write access to the Ruleset for complete bypass visibility** | Administration write, with complete observation and account eligibility |
| Actions repository variables (current source) | Repository Variables read for selected names | Repository Variables write, including fresh read and exact readback |
| Agents repository variables (current source) | Repository Agent variables read for selected names | Repository Agent variables write, including fresh read and exact readback |
| Actions event policies (current source) | Administration **write**, including list/GET, export and plan ([official API](https://docs.github.com/en/rest/actions/policies)) | Administration write, complete preflight, fresh detail guard and independent acknowledgment/readback |
| Actions OIDC active template (current source) | Actions read and complete personal-public target identity | Actions write, known immutable companion replay and independent readback |
| Actions execution, workflow permissions, retention, contributor fork approval | Administration read | Administration write |
| Actions cache retention and private sharing access | Administration read plus repository applicability metadata | Administration write plus independent readback |
| Actions cache size eviction limit | Actions read | Administration write plus Actions read for preflight/readback |
| Secret scanning/push protection | Complete repository security-block visibility; Metadata read alone is insufficient | Administration write plus applicable security authority/entitlement and complete readback |
| Private vulnerability reporting | Metadata read and supported public applicability | Administration write and applicable repository authority |
| Immutable release policy | Administration read; target GET 404 means disabled under the required-permissions precondition, with owner enforcement unknown | Administration write; fresh owner-enforcement preflight and readback |
| Pages lifecycle/configuration | Pages read | Pages write plus Administration write and applicable repository authority |
| Classic branch protection (current source) | Complete GraphQL rule/actor visibility, including raw companions | Authorized GraphQL create/update/delete and independent full readback; exact installation grant sufficiency was not live-tested, and REST grant tables alone do not prove it |
| Named Environments (unreleased source only) | Actions read, including enabled custom-protection visibility; App additions also need Administration read for available integrations; parent pruning also needs Environments read for variable/secret metadata | Administration write plus complete supported observation/readback and prune absence confirmation |
| PR sample's candidate file retrieval | Contents read for filtered Git metadata and the exact settings blob | No mutation; not an apply credential |

GitHub documents installation-token support for the relevant [repository/topic](https://docs.github.com/en/rest/repos/repos), [label](https://docs.github.com/en/rest/issues/labels), [Ruleset](https://docs.github.com/en/rest/repos/rules#get-a-repository-ruleset), [Actions permissions](https://docs.github.com/en/rest/actions/permissions), [cache limits](https://docs.github.com/en/rest/actions/cache), [Pages](https://docs.github.com/en/rest/pages/pages) and [Environment](https://docs.github.com/en/rest/deployments/environments) REST APIs. The Ruleset GET explicitly limits `bypass_actors` to actors with write access: nonmutating plan is **not** proof that a read-only installation token is sufficient. Missing visibility must fail, never become an empty bypass list.

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

This repository's [Environment scope tests](contributing/repository-automation.md#repository-environment-scope-tests) illustrate the scope check and its limits. They run with candidate source in ordinary CI, so they do not supply an independent trusted policy. They are not included in the two consumer examples. Owner review must still cover apply steps, Action revisions and credential handling.

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

## Dependabot settings

Current source supports the two [Dependabot controls](settings.md#dependabot-alerts-and-security-updates) in validate/plan/apply and UI-named detailed Summary differences; the same-version report carries their two native booleans and closed count bounds. The pinned published package is unchanged and does not gain this capability from a configuration edit. Use a package that implements it before adoption. No Action input, token grant or credential source is added.

Complete family reads require Administration read plus applicable actor authorization; writes require Administration write. Under this required-permissions precondition, GET 404 means disabled and adds no permission-risk warning. Pause metadata remains unavailable on 404; known paused state is diagnosed without requesting a resume. DELETE 204 plus matching fresh whole-family readback completes disabled intent, allowing declared dependency order to continue. Explicit 401/403, timeouts, malformed/incomplete reads, unexpected write status and target/companion mismatch fail while retaining prior completed progress. These endpoint-specific semantics do not certify actual grants, live enforcement or update activity. The [configuration reference](settings.md#dependabot-alerts-and-security-updates) owns the full contract.

## Fork workflow policies

[Fork workflow policies](settings.md#fork-pr-workflow-policies), with public contributor approval separate from private fork execution/token/secrets/approval controls. Existing Administration read/write credentials are required; no Action input or grant is added. Summary shows UI choices and boolean differences, including credentials passed to forks. It remains credential-free presentation; apply independently rereads state. Fixtures do not establish live mutation compatibility, workflow execution safety or organization-policy enforcement.

[Pages configuration](settings.md#pages-configuration) through validate/plan/apply and UI-named Summary entries, including a domain-clearing null and inactive workflow source. Pages reads require Pages read; updates require Pages write and Administration write plus the applicable actor authorization. Export uses the same acquired-state contract; caller grants remain prerequisites. Configuration success does not promise deployment/DNS/certificate readiness, and partial update failures retain prior effects. Complete Source intent creates missing sites; explicit deploy-from-branch with exact `branch: none` deletes them. Partial creation intent and deletion conflicts fail. Confirmed creation remains reported if an optional follow-up fails. Site visibility is not managed.

The Summary shares Actions workflow-permission and contributor-approval, Security and Pages display paths and choices with CLI plans. Private plan transport fields retain their internal identities; these are not public YAML field names.

### Persistent pull request limits

The Action accepts the same optional [Pull request limits declaration](settings.md#persistent-pull-request-limits)
as the CLI, under `repository.settings.interaction_limits.pull_request_limits`.
Its token must already authorize the administrator-only creation-cap endpoint.
Detailed plans/Summaries retain the three distinct UI controls and native values;
status-only reports hide them. Omitted controls, temporary restrictions and
exemptions remain unmanaged. A failed post-PATCH confirmation reports possible
applied effects without retrying the write or rolling back earlier operations.

## Explicit collection pruning

Current source accepts optional comma-separated `prune` for `command: plan` or
`apply`, using the same [CLI prune contract](settings.md#collection-pruning). It
adds deletions to other declared work and defaults to no pruning. `export` and `validate`
reject a nonempty prune input. Omitted collections stay unmanaged; explicit []
can delete all objects in the selected scope. Label deletion loses Issue/PR
assignments, including undeclared Dependabot labels. Detailed plan Summary shows
these consequences; status-only retains its existing value-hiding boundary.

The Action checks installed CLI prune support before remote work and fails with
an older published package instead of silently ignoring the input. Source changes
do not add this capability to an existing published Action/CLI. Actions Policies
use `actions-policies`; parent deletion uses `environments`; personal access deletion
uses `collaborators`; Autolink deletion uses `autolinks`; classic rule deletion uses `branch-protection-rules`; deploy key deletion uses `deploy-keys`. Standalone `all` selects every currently supported
scope and cannot be mixed with another token.
Environment pruning retains every listed name, including variable-only entries,
and runs only after retained policy/variable work. Parent deletion removes
contained configuration/secrets and fails jobs waiting on protection rules; recreation does not
restore them. Complete deletion snapshots need Actions read and Environments read
for child variables/secret metadata; DELETE needs Administration write. Metadata
and child values stay out of plan/report output, and snapshot transport ignores
inherited `GH_DEBUG`. Status-only also hides parent names and policy. See the
[Environment pruning contract](settings.md#environment-pruning) for custom-rule
blocking, inactive-pattern visibility limits, fresh checks and nontransactional
failure effects.

Current-source Environment protection App selection shares the CLI's
[custom-rule ownership and verification](settings.md#custom-protection-apps).
Detailed plans show App-ID membership; status-only output omits selections and
Environment names. Raw binding evidence stays out of plan reports and transport
debug logs. Apply publishes success outputs only after verification and never
writes binding rule IDs to configuration. This source coverage does not upgrade
the pinned published CLI or establish actual App permissions.

Classic branch-protection entries use the same
[partial-ownership contract](settings.md#classic-branch-protection) in CLI and
Action. Detailed Summary shows requested patterns, checkboxes and selected active
status-check policy, including check names and App-node sources; status-only
hides names and values after validating the same private report. Unowned actor,
PR, latent check and deployment snapshots never enter the report. Needed writes
with selected status-check App sources require same-credential exact App-node
reads before and after mutation; matching rules need no App lookup. A failed
postwrite check suppresses success outputs and leaves uncertain effects.
Explicit classic prune
shows supported before-policy and whole-rule removal consequences, including
protections outside the supported controls. Omission remains unmanaged; []
selects all acquired rules under its scope or standalone `all`. Complete frozen
candidates, exact drift checks and independent complete absence verification use
the [same deletion contract](settings.md#classic-branch-protection-pruning). This source addition
does not change an immutable published Action or CLI, and it adds no live mutation
to the repository workflows. Classic creation IDs are reported for recovery;
Actions does not rewrite the configuration file.

## Repository AI Scan

Current source supports the [AI Scan for repositories toggle](settings.md#ai-scan-for-repositories)
independently of PR scanning in export, plan and apply. Verify the selected
immutable package supports it. Apply changes configuration and verifies the PR
companion; it does not request a scan. Detailed reports show the boolean, while
status-only publication hides it. Actual scan usage and billing remain separate.

## Deploy keys in current source

The Action shares the [Deploy keys contract](settings.md#deploy-keys), including
canonical public-key identity, exact matching no-ops and explicit `deploy-keys`
pruning. Detailed plans show title, fingerprint and read/write grant; status-only
hides those values. Transport ignores `GH_DEBUG` for private snapshot data.
Creation reports a verified ID for follow-up binding and leaves the caller's YAML
unchanged. Repeating an IDless declaration collides instead of adopting it. Failed
confirmation withholds success outputs, preserves recovery evidence and stops
later deletion. These source capabilities do not alter a published package pin.

## Autolink permissions and identity recording

Current source supports [Autolink references](settings.md#autolink-references) through the same export/plan/apply contract. App installation tokens require repository Administration read for acquisition and write for creation/deletion. Nothing grants or broadens permissions automatically. `prune: autolinks`, comma combinations and standalone `all` use fixed candidates for a present collection. Ordinary apply preserves unlisted references.

Autolinks are immutable ID-bound objects: changes require explicit pruning and later creation. The Action does not write creation IDs, commit or push. Record confirmed IDs through the [creation-ID procedure](contributing/repository-automation.md#repository-settings-changes) before repeating creation; unbound collisions fail. Detailed plans and Summary may show non-secret prefix/URL/format, while status-only hides them. Export saves local YAML without logging its contents; API debug bodies are suppressed. Package support still depends on the selected immutable release.
