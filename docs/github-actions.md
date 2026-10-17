---
type: Guide
title: Use the public settings Action
description: Invoke the composite Action and separately adopt owner-approved consumer workflows.
sources:
  - id: composite-metadata
    resource: https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax
  - id: environment-controls
    resource: https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments
  - id: app-token-action
    resource: https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510
---

## Public Action and supported checkout

The root [composite Action](../action.yml) implements the [ADR 0023 contract](adr/0023-public-composite-action.md) and [behavior specification](features/actions.feature.md#public-composite-action-contract). Current source selects **published v0.7.0** and invokes one verified Go executable for metadata, settings operations and typed plan/Summary presentation. Shell retains installation bootstrap and Actions wiring; no Python runtime or command wrapper is required by this source. Workflow/configuration, Action and CLI identities remain distinct. Composite inputs/outputs use normal `with` and step-output bindings.[^composite-metadata]

Only a fresh GitHub-hosted `ubuntu-24.04` x64 job is supported, with Bash, curl, GNU sha256sum and Git. macOS, Windows, ARM, containers and self-hosted runners are outside this contract. The caller checks out its configuration repository at `GITHUB_WORKSPACE`, without a checkout `path` or persisted credentials, and commits `.github/settings.yml` at that root. A resolvable checkout commit and checkout-scoped GitHub.com `origin` are required even for validate's output metadata. The Action repository is not the target; `GH_REPO` does not override the CLI's origin resolution. See the [configuration semantics](settings.md#configuration-semantics).

The Action uses its own packaged installation bootstrap and verified absolute CLI in a per-invocation runner-temp directory; consumers do not install an extension or copy Action internals. Validate needs no settings authentication/API access, but the Action still downloads the public CLI asset. Workflow, configuration and Action revisions must remain available for ordinary reruns.

## Inputs, outputs and failures

| Input | Contract |
| --- | --- |
| `command` | Required, no default; exactly `validate`, `plan` or `apply`. |
| `token` | No default; unused for validate, required and nonempty for plan/apply. Supply it explicitly; no ambient or implicit `github.token` fallback. |

Validate checks local configuration without credentials. Plan validates first, then reads settings with an explicit read-only caller token, for example `token: ${{ github.token }}` in the read-only PR job. Apply directly invokes the existing CLI using the caller's approved, target-scoped App installation token with Administration write. The Action does not verify the token's actual grants or create it. No export, arbitrary arguments, configuration-path, target, version or download-URL input is supported.

| Success output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit (`HEAD`), not the file hash, workflow SHA or Action SHA. |
| `target` | Configuration checkout's resolved GitHub `origin`, in `OWNER/REPO` form. |
| `cli-version` | Verified installed CLI release selected by the Action package. |
| `cli-sha256` | SHA-256 of the verified installed binary. |

Use `steps.<id>.outputs.<name>` only after successful completion. Outputs are published after the command and, for plan, successful Summary rendering/writing. Missing command/token, unsupported prerequisites, install/metadata/output failures and CLI/configuration/API errors fail the Action. CLI failure status is preserved; differences, no differences and no managed fields all exit successfully. Failed calls have no successful output contract. Plan Summary quotes settings/error text and separates workflow, configuration, Action and CLI identities; validate/apply do not render a plan Summary. The main example uses the Action step's outcome and logs to report apply results.

Current source selects **published v0.7.0** in [`action/cli-version`](../action/cli-version) for the verified executable’s Go metadata and typed plan/Summary entries, sharing checkout/target resolution and planning with settings operations. Candidate CI verifies integration using the same source binary and local artifact bytes. The examples and active consumer workflows use published v0.7.0. Current source shares the CLI requirement for a directly configured GitHub.com origin: HTTPS, `git@github.com:owner/repo.git` or full SSH. Rewrite-only aliases and username-less SCP fail metadata before settings execution or success outputs; direct-origin identity survives transport mirror rewrites. Released v0.6.1 keeps its original compatibility. Authenticated plan writes an ephemeral typed report, then a separate credential-free step renders/writes its Summary. A nonzero plan status is preserved; otherwise Summary failure determines failure. Validate/apply also invoke the verified CLI directly; shell retains installation bootstrap and necessary Actions wiring. No command wrapper, terminal-text classification or Python renderer remains in current source, while released v0.6.1 keeps its original implementation. A reviewed full SHA pins that file and therefore the CLI version. Future version-tag calls must agree with the file; release publication rejects a different requested version. Installation downloads only that version's `gh-settings-linux-amd64` and `SHA256SUMS` from fixed upstream HTTPS paths, requires one valid checksum for the exact asset, verifies before execution, and checks exact `gh-settings <version>` output. The SHA pin fixes the version; it does not commit the binary digest. The checksum trust source is the same GitHub release, including its integrity over HTTPS. There is no latest fallback or caller version/URL override.

## Published package and consumer pins

The examples and README snippet use the published [v0.7.0 package](https://github.com/daiksud/gh-settings/releases/tag/v0.7.0), released on 2026-10-17, at `e9edb03f65ac943cd0ebf6a9556c18c485719e66`. It installs CLI v0.7.0 with Linux amd64 SHA-256 `c0ab75712dc7648a6ba9416c468ec9e5f43fd864ab85376fac84aeba3c86a9b0` and uses the Go metadata and typed plan/Summary entries without Python.

The former [v0.6.1](https://github.com/daiksud/gh-settings/releases/tag/v0.6.1) package is retained for historical compatibility but is no longer used by this repository's active workflows. Existing tags/assets stay immutable; see [historical release limitations](releases.md#historical-release-limitations) for older versions.

## Consumer files and compatibility

The [PR/main examples](../examples/github-actions/README.md) need only these two workflow files plus the repository's `.github/settings.yml`:

| Example source | Consumer destination / purpose |
| --- | --- |
| `examples/github-actions/settings-pr.yml` | `.github/workflows/settings-pr.yml`: PR validate/plan with an explicit read token. |
| `examples/github-actions/settings-main.yml` | `.github/workflows/settings-main.yml`: one approved job with checkout, App token creation and direct apply. |

No caller script, root `action.yml` or copied `action/` directory is required. Both examples use the same v0.7.0 Action pin. Main checks out the event's exact configuration SHA and passes the caller-created token to `command: apply`; it has no prepare job, handoff outputs or separate apply Summary step. The PR plan provides review information without becoming an input or dependency of main apply.

The active [PR](../.github/workflows/settings-pr.yml) and [main](../.github/workflows/settings-main.yml) workflows now use the same v0.7.0 Action pin and single-job apply pattern. They retain this repository's existing `gh-settings` Environment and `ANSWER26_*` App credentials rather than the sample resource names. The legacy [operational handoff verifier](../.github/scripts/verify-settings-handoff.sh) is not invoked.

The active PR workflow watches `examples/github-actions/**` as in the PR example. The active main workflow matches the main example and watches only `.github/settings.yml` and its own workflow path.

### Active repository Environment guard

The [`settings-environment-guard`](../.github/workflows/settings-environment-guard.yml) uses `pull_request_target` so its implementation comes from trusted `main`. It checks out the proposed PR merge revision as **data only** and runs the main-branch YAML-aware [scope test](../tests/github-actions/settings_environment_scope_test.go); no PR code executes in this guard, and its token is read-only. The check also runs on pushes to `main`. It permits `gh-settings` only as a literal in `.github/workflows/settings-main.yml`, job `apply`, and rejects other references, dynamic names, job YAML merges/aliases, and all job-level reusable-workflow calls that could select an Environment outside this repository's policy.

The ordinary Go suite in [`ci-pr`](../.github/workflows/ci-pr.yml) also runs this test and contributes to the [existing required `pass` check](https://github.com/daiksud/gh-settings/rules/24911852). This catches accidental mistakes, but it is **not an unforgeable merge gate**: PR code can change its own tests and CI. Even requiring the trusted guard's `check` job from the GitHub Actions app is insufficient: a PR can add another job named `check`, and GitHub's required status checks do not bind the identity to a specific workflow file. <https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules> documents this limitation.

**Do not disable `gh-settings` Required reviewers based only on this workflow or a required GitHub Actions `check` status.** A replacement that PR-authored workflows cannot spoof must first be configured and enforced externally: for example, an organization/enterprise **required-workflow Ruleset**, or a dedicated GitHub App producing its own required check using credentials unavailable to PR code. Required workflows are configured at the organization/enterprise level, so they are not directly available in this user-owned repository. The connector cannot configure either solution or edit GitHub Environment protections. Continue to require the current PR review and `pass` check, and keep `gh-settings` restricted to the **`main` deployment branch** and its existing `ANSWER26_*` credentials. Remove Required reviewers at [Settings → Environments](https://github.com/daiksud/gh-settings/settings/environments) **only after** a non-spoofable required policy has been established and verified.

The trusted guard uses `pull_request_target`. For public repositories, GitHub's [default event policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target) is scheduled to block that event starting **2026-11-02**, unless an applicable Actions event policy permits it. Verify policy eligibility and that the guard actually runs before relying on it. The guard cannot protect against other repositories, untrusted branches allowed to deploy, or a deliberately modified source policy without an external enforcement boundary.

## Ordered owner setup

The caller owns Environment selection/approval, App token minting/revocation and concurrency. The Action's token input becomes `GH_TOKEN` only for the plan/apply command step; installation, validation, Summary and metadata/output steps do not receive that input. Composite steps share the caller job, so all code in a credential-bearing job must be trusted and privileged credentials must stay out of PR code.

For a new installation, configure the target's supported Environment controls before adopting the workflows; availability depends on the GitHub plan and repository visibility.[^environment-controls]

1. Install a dedicated selected-repositories App on the target with Administration read/write and automatic Metadata read. The caller's pinned official token Action requests Administration write for only the containing repository through explicit `owner` and `repositories` inputs, retaining default post-job revocation.[^app-token-action]
2. Protect `settings-apply` with the owner reviewer, self-approval allowed, administrator bypass disabled and only a Branch `main` rule. Store Environment variable `SETTINGS_APP_CLIENT_ID` and secret `SETTINGS_APP_PRIVATE_KEY` after protection is configured; an Environment name alone is not a workflow allowlist.
3. Adopt the two YAML files and root configuration through review, retaining exact-event checkout, no persisted credentials, main event/ref guards and concurrency. The entire main job waits for approval before checkout or token issuance. Review the PR plan and exact main configuration before approving it; main creates no additional pre-approval plan.

The examples perform no App/Environment setup themselves. Actual waiting/rejection, branch/tag enforcement and revocation depend on the configured GitHub controls and are not established by local test fixtures.

This repository's operational callers preserve Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, `ANSWER26_PRIVATE_KEY` and their existing declaration. The main job checks out `github.sha` and applies directly. The [active PR guard](#active-repository-environment-guard) replaces its deployment approval after GitHub's reviewer rule is removed; the sample Environment `settings-apply` retains its own approval policy.

## Review a pending run

For the simplified example, review the PR plan Summary and the configuration at the main run's exact commit before approving `settings-apply`. The PR revision and resulting main commit may differ. Main does not depend on PR success or rerun plan before approval, so confirm the approved configuration explicitly; its checkout uses `github.sha`, never a moving main branch.

For this repository's active workflow, review the PR plan Summary, Action pin and intended configuration **before merging the PR**. Once the Environment reviewer requirement is removed, no separate `gh-settings` deployment approval is needed; the main job still does not run another plan or depend on PR artifacts.

A plan is informational. No differences and no managed fields are successful outcomes. Direct apply sends declared values without reading current settings or consuming a saved plan; intervening changes to managed fields can be overwritten, while omitted fields remain outside the request. The simplified main job mints its token before the settings Action installs the CLI and validates configuration; installation or validation failure stops apply and the token Action still performs normal post-job cleanup.

PR concurrency cancels superseded runs. Main uses a repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. Sample workflows retain their documented approval behavior; the active Environment runs without a second approval once its Required reviewers rule is disabled. No custom queue, drift/freshness guard, stale-commit check or reapproval controller is added.

## Failure and recovery

| Situation | Maintainer action |
| --- | --- |
| Missing/invalid config | Correct the declaration and rerun through normal review; validation fails before API access. |
| CLI download/checksum/version error | Inspect the fixed release identity and reviewed pin. Do not execute an unchecked binary or switch to source/latest. |
| Plan read/API permission error | Inspect ordinary read-token access and the target origin. Do not provide administration credentials to PR/prepare. |
| Rejected sample Environment approval | The sample's main job cannot start. The active repository instead uses required PR checks after GitHub's reviewer rule is removed. |
| Missing App key/client ID or token error | Verify the protected Environment and dedicated installation scope. Have the owner use secure entry; no privileged token fallback. |
| Failed/cancelled apply or output failure | Read target settings before deciding what to do: an update may already have succeeded. The CLI does not promise atomicity or automatic rollback. |
| Token cleanup failure | Inspect the official Action's post-job logs and dedicated installation. Treat revocation as unverified; do not expose the token while investigating. |
| Disable further executions | Disable the adopted settings workflow in Actions and cancel pending/running executions as appropriate; review any update already sent. Disabling does not undo settings. Credential revocation/protection changes remain owner security operations. |

## References

- [GitHub contexts](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts)
- [Environment setup and supported controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
- [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510)

[^composite-metadata]: [GitHub composite metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax). Inputs and mapped step outputs; runtime checks remain Action-owned.
[^environment-controls]: [GitHub Environment controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments). Check availability and protection rules for the actual target before owner setup.
[^app-token-action]: [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510). Explicit target/permission inputs and default post-job revocation; actual revocation needs runtime evidence.
