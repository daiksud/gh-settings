---
type: Feature
title: Owner-approved settings automation
description: Specify inert workflow examples and the public composite Action around the existing CLI.
---

## Feature: Review settings before approved direct apply

### Rule: Examples are inert until separately adopted

- Workflow examples and support files live under `examples/github-actions/`.
- Merging examples does not enable settings workflows, install an App, enter secrets, or apply settings.
- Every job uses a fresh GitHub-hosted Linux runner and reviewed full-SHA Action pins.
- The CLI is a fixed released binary verified against its committed SHA-256 before execution; prepare and apply use the same pin from the same configuration commit.

### Rule: PRs have only read access

- On a PR, check out the containing repository's PR revision without persisted credentials.
- Validate the checkout-root `.github/settings.yml` before read-only plan.
- Valid configuration and successful plan exit successfully whether differences exist or not.
- Missing/invalid configuration fails before API access; authentication/API failures fail visibly.
- Fork changes have no App credential, Environment, apply, or privileged fallback path.
- Per-PR concurrency cancels superseded executions.

### Rule: Main prepare establishes review context

- Only `push` on Branch `main` is eligible; PR, other branch, tag, manual, PR-target, and workflow-run contexts cannot unlock apply.
- Check out the triggering configuration SHA rather than a moving main branch.
- Validate, plan with the read token, then render/write the Summary before prepare succeeds.
- Invalid input, API failure, or detectable Summary render/write failure blocks apply.
- Expose only required non-secret handoff values, including the configuration SHA and CLI identity; do not save a plan artifact.
- A newer main commit cannot alter this run's configuration input.
- Repository-scoped concurrency does not cancel the running pipeline; GitHub may replace a pending run.

### Rule: The Summary separates identities from settings data

- First run shows `github.actor` and `github.triggering_actor`; reruns show the original actor separately from the rerun initiator and incremented attempt.
- Include event, run URL/attempt, workflow path/ref and exact workflow-source SHA/link.
- Include target, configuration SHA and exact configuration-file link, CLI version/checksum, and observed before → desired after output.
- Workflow-source SHA can differ from configuration SHA; each link uses the displayed identity.
- Distinguish changed settings, no differences, no managed fields, and errors.
- Render quotes, newlines, Markdown fences, HTML, and workflow-command-like values as data that cannot forge review context.
- State that direct apply can overwrite intervening changes to declared remote fields.
- The owner must not approve missing/incomplete review content. Successful prepare alone does not prove Summary publication; no publication polling is required.

### Rule: Owner approval gates credentials and apply

- Apply needs successful prepare and the same main guard, then waits at `settings-apply`.
- Required reviewer is the owner; self-approval is allowed, administrator bypass is disabled, and only Branch `main` is eligible with no Tag rule.
- While waiting, or after rejected approval, no App token is minted and no apply runs.
- The Environment key is available only inside the approved job; the official token Action narrows Administration write to the one target repository and retains normal revocation.
- The token is supplied only to the apply step, never checkout, bootstrap, Summary, global environment, artifact, or job/workflow output.
- Apply uses prepare's exact configuration commit and verified CLI identity.
- A successful apply reports success; missing credentials or CLI failure fails visibly. Output failure may follow a successful update; no rollback is promised.
- Ordinary reruns use GitHub's approval behavior without custom freshness, drift, or reapproval checks.

### Verification boundary

Each implementation PR adds its focused offline checks; composed coverage follows before adoption. Markdown/local-link checks validate this contract. Actual Environment waiting/rejection, branch/tag enforcement, and token revocation require separately authorized GitHub verification; offline fixtures cannot establish those platform behaviors.

### Public composite Action contract

Consumers can validate, inspect, or directly apply their checked-out repository settings through a reviewed Action package without copying helper files. This is the accepted contract for staged implementation in [ADR 0023](../adr/0023-public-composite-action.md), not a claim that the root Action already exists. The preceding workflow-example rules remain applicable to those examples.

### Rule: Action revision and CLI release are independent

- Invoke the root Action as `uses: daiksud/gh-settings@<approved full commit SHA>` after its implementation.
- The Action revision selects packaged support code. Published `v0.5.0` is a CLI release with no root `action.yml`; keep that immutable tag and its assets unchanged.
- Install only CLI `v0.5.0`, asset `gh-settings-linux-amd64`, SHA-256 `b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4`, and verify exact reported version `gh-settings v0.5.0` before settings execution.
- Support only a fresh GitHub-hosted Ubuntu 24.04 x64 job. The caller checks out the configuration repository at `GITHUB_WORKSPACE` without persisted credentials.
- Read only checkout-root `.github/settings.yml`; use the existing CLI's checkout-scoped GitHub `origin` target semantics, not the Action repository or an assumed `github.repository` target.
- Action-owned installation, execution, and Summary helpers come from the Action package, independently of caller helper files. Execute only the verified absolute CLI path in a per-invocation runner-temp location.

### Rule: Two inputs select one bounded command

| Input | Requirement | Meaning |
| --- | --- | --- |
| `command` | Required, no default | Exact enum `validate`, `plan`, or `apply`; reject omitted or unknown values at runtime. |
| `token` | Optional for `validate`; required and nonempty for `plan`/`apply`, no default | Explicit caller token with permissions appropriate to that command. Never fall back to `github.token` or ambient authentication. |

- `validate` performs local CLI validation without authentication or API calls, even if a token input was supplied.
- `plan` requires a read-only caller token, validates first, and runs read-only CLI plan only after validation succeeds. Differences, no differences, and no managed fields are all successful outcomes.
- `apply` requires a caller-created GitHub App installation token scoped to the approved target repository with Administration write. Invoke existing direct CLI apply; a previous plan is not required or consumed.
- No `export`, arbitrary command/arguments, configuration-path, target, version, asset, or download-URL input is supported.

#### Scenario: Validate without a credential

- Given a valid configuration checkout on a supported runner
- And no token input or ambient authentication
- When the caller selects `command: validate`
- Then validation succeeds without GitHub API access
- And all four non-secret success outputs identify that checkout and verified CLI

#### Scenario: Invalid configuration blocks plan

- Given an explicit read-only token and invalid `.github/settings.yml`
- When the caller selects `command: plan`
- Then validation fails before plan or API execution
- And the Action fails without success outputs

### Rule: Only successful completion exposes four non-secret outputs

| Output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit SHA (`HEAD`), not a configuration-file hash, workflow SHA, or Action revision. |
| `target` | Resolved configuration checkout's GitHub repository in `OWNER/REPO` form, consistent with CLI `origin` semantics. |
| `cli-version` | Verified installed release version, `v0.5.0`. |
| `cli-sha256` | Verified CLI binary SHA-256, `b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4`. |

- Emit these outputs only after command success and, for plan, successful Summary render/write. Composite outputs map to actual step outputs.
- A validate success includes metadata resolution for these outputs; inability to resolve the checkout commit or target fails the Action even though local CLI validation itself needs no origin or credentials.
- Missing/unknown command, missing required token, unsupported execution prerequisites, installation/download/checksum/version failure, configuration/CLI/API failure, metadata/output failure, and detectable plan Summary failure fail visibly. Do not run settings commands after installation failure; stop plan after validation failure.
- Preserve CLI failure status and ordinary plan exit 0 when differences exist. Error review content must not imply success. Failed runs have no successful output contract; no token is an output.
- Output or Summary failure may occur after a remote update; no automatic rollback or atomicity is promised.

### Rule: Plan Summary separates execution identities and untrusted data

- The Action owns plan Summary rendering/writing, including command status and CLI error output when available.
- Include original and rerun actors, event, run URL/attempt, workflow path/ref and exact workflow revision, target, configuration commit and exact file link, distinct Action identity/ref, and CLI version/SHA-256.
- Obtain Action identity/ref from supported composite context; never relabel workflow SHA as Action SHA. Identify local CI invocation honestly when there is no remote pin.
- Present existing CLI text without a new diff schema; distinguish differences, no differences, no managed settings, and failures.
- Escape untrusted values, including quotes, multiline text, Markdown, HTML, and workflow-command-like content, so they cannot forge Summary markup, links, or execution context.
- State that the observed diff is informational and direct apply may overwrite intervening remote changes. A later apply is not bound to a saved plan.
- Detectable render/write failure fails plan before success outputs. Successful writing does not prove GitHub published the Summary; the owner must not approve missing/incomplete content. No publication polling is required.

### Rule: The caller owns approval and credentials

- The caller owns checkout, event/ref guards, Environment selection/approval, App token minting/revocation, and concurrency. The Action does not perform those operations or introduce custom approval/freshness machinery.
- Bind `inputs.token` as `GH_TOKEN` only on the plan/apply command execution step. Validate, installer, Summary, and metadata/output steps must not receive that input as `GH_TOKEN`; never write it to `GITHUB_ENV`, files, logs, artifacts, or outputs.
- This is scoped environment exposure, not token process isolation. Composite steps share the caller job; all code in a credential-bearing job must be trusted. The caller must keep privileged credentials away from untrusted PR code.
- Preserve owner-approved main direct apply and repository-scoped concurrency without cancellation of running apply. PR plan remains read-only with no privileged fallback.
- Active workflows keep Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, and `ANSWER26_PRIVATE_KEY`. Inert examples keep their existing `settings-apply` contract until separately adopted. This specification performs no operational migration, credential operation, release, or live apply.

### Public contract verification boundary

This documentation change uses Markdown/local-link checks and comparison with existing CLI commands, features, the fixed release asset, and official composite metadata syntax. Installer, command, renderer, wiring, hosted validate, and consumer integration checks belong to their focused implementation Issues in ADR 0023's order. Offline checks do not prove Environment enforcement, token revocation, Summary publication, or a live update.
