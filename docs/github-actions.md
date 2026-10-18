# Use the public settings Action

## Public Action and supported checkout

`daiksud/gh-settings` validates, plans or applies the checked-out repository's `.github/settings.yml`. The Action installs its selected CLI release and exposes normal `with` inputs and step outputs.[^composite-metadata] Consumers provide the checkout, credentials and approval policy; they do not install the CLI or copy Action internals. This guide covers the pinned v0.9.0 package for repository fields, topics and [labels](#labels-and-permissions).

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

The examples use the published [v0.9.0 Action/CLI package](https://github.com/daiksud/gh-settings/releases/tag/v0.9.0), pinned to the full Action SHA `07bfc905e0b52c433ce666b8c0c16978909e02c5`. Its selected CLI version is fixed by that package; use a reviewed full SHA when adopting or updating the workflows. Release assets and `SHA256SUMS` are available on the release page.

Installation downloads the selected `gh-settings-linux-amd64` asset and `SHA256SUMS` from fixed upstream HTTPS release paths. It requires one valid checksum for the exact asset, verifies before execution and checks exact `gh-settings <version>` output. There is no latest fallback or caller version/URL override. The binary and checksum come from the same GitHub release; the Action SHA fixes the selected version, not an independent binary digest.

Older pins retain their released behavior. Maintainers document immutable package history and known limitations in the [release guide](contributing/releases.md#historical-release-limitations).

### Labels and permissions

[Labels support](settings.md#labels) is published in v0.9.0 and included in the pinned package. Older v0.8.0 packages reject `settings.labels`. Checking out newer Action source does not build that source; the selected release determines CLI capabilities.

Listing needs Issues or Pull requests read, except public resources can be listed without those grants. Label mutations need either permission at write level; Administration write alone is insufficient. The default example App token still requests only Administration write for repository fields/topics. For label mutations, separately approve the App/installation permission and a matching token request scoped to the containing repository. Do not assume installing or pinning v0.9.0 grants access.

This repository imports its existing public labels with zero planned operations while keeping its Administration/Metadata-only App token unchanged. A zero-difference apply reads Labels but performs no label writes. Any future label mutation needs a separately approved permission expansion; public listing and zero-difference adoption do not establish write access. If observed labels drift during adoption, refresh the inventory rather than overwriting live state.

Labels apply reads fresh membership and performs only needed CRUD operations, with authoritative deletion of undeclared labels and their issue/PR associations. Review the [labels contract](settings.md#labels) before adoption.

## Consumer files and compatibility

The [copyable PR/main examples](../examples/github-actions/README.md) need two workflow files plus `.github/settings.yml`. No caller script, root `action.yml` or copied `action/` directory is required.

The PR workflow validates and plans with a read-only token and produces a review Summary. Main uses one job with three steps: checkout, App token creation and direct apply. It checks out the event's exact configuration SHA. Main does not consume a PR artifact, depend on the PR run, or produce another plan before approval.

## Ordered owner setup

The caller owns Environment selection/approval, App token minting/revocation and concurrency. The Action's token input becomes `GH_TOKEN` only for the plan/apply command step. Installation, validation, Summary and metadata/output steps do not receive that input. Composite steps still share the caller job, so all code in a credential-bearing job must be trusted and privileged credentials must stay out of PR code.

Configure supported Environment controls before adopting the main workflow; availability depends on the GitHub plan and repository visibility.[^environment-controls]

1. Install a dedicated selected-repositories App on the target with Administration read/write and automatic Metadata read. The pinned official token Action requests Administration write for only the containing repository using explicit `owner` and `repositories` inputs; retain its default post-job revocation.[^app-token-action]
2. Protect the example Environment `settings-apply` with the owner reviewer, self-approval allowed, administrator bypass disabled and only a Branch `main` deployment rule. After configuring protection, store Environment variable `SETTINGS_APP_CLIENT_ID` and secret `SETTINGS_APP_PRIVATE_KEY`. An Environment name alone is not a workflow allowlist.
3. Adopt the [two workflows](../examples/github-actions/README.md#copy-the-workflows) and root configuration through review. Retain exact-event checkout, no persisted credentials, main event/ref guards and concurrency. The entire main job waits for approval before checkout or token issuance.

The examples do not create the App or configure the Environment. Confirm actual approval, branch/tag restrictions and token cleanup in GitHub; workflow YAML alone does not establish those external controls.

## Review a pending run

Review the PR plan Summary and the configuration at the main run's exact commit before approving `settings-apply`. The PR revision and resulting main commit can differ. Main does not depend on PR success or rerun plan before approval; its checkout uses `github.sha`, never a moving main branch.

A plan is informational. The pinned package sends declared repository fields and topics directly; Labels alone reads fresh state before reconciliation. Neither consumes a saved plan. Intervening changes to managed fields can be overwritten; omitted fields stay outside the request. Review the [apply and partial-failure behavior](commands.md#apply-behavior) before approving changes.

The main job mints its token before the settings Action installs the CLI and validates the configuration. Installation or validation failure stops apply; the token Action still performs its normal post-job cleanup.

PR concurrency cancels superseded runs. Main uses a repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. These examples provide no custom queue, stale-commit check, drift/freshness guard or reapproval controller.

## Failure and recovery

| Situation | Recovery |
| --- | --- |
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
