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
- The example Action verifies the fixed released CLI against its same-release SHA256SUMS before execution. The caller verifies the committed expected CLI version and digest before App token issuance; prepare and apply use the same Action pin from the same configuration commit.

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

Focused offline checks and composed consumer coverage run in required CI, alongside hosted validate smoke and built-product acceptance. Markdown/local-link checks validate document structure; they do not prove runtime behavior. Actual Environment waiting/rejection, branch/tag enforcement, and token revocation require separately authorized GitHub verification; offline fixtures cannot establish those platform behaviors.

### Public composite Action contract

Consumers can validate, inspect, or directly apply their checked-out repository settings through a reviewed Action package without copying helper files. The implemented root Action follows [ADR 0023](../adr/0023-public-composite-action.md); the [usage guide](../github-actions.md) identifies reviewed full-SHA calls and supported consumer files. The preceding workflow-example rules remain applicable to those examples.

### Rule: A release selects the same Action and CLI version

- Invoke the root Action as `uses: daiksud/gh-settings@<approved full commit SHA>`; use a reviewed revision containing the implemented package.
- Published `v0.6.0` retains the earlier installer using CLI `v0.5.0`; changing main does not repair that immutable tag. Preserve it and its assets; a corrected release requires separate authorization.
- The Action revision selects packaged support code. Published `v0.5.0` is a CLI release with no root `action.yml`; keep that immutable tag and its assets unchanged.
- The committed `action/cli-version` selects an explicit CLI release. Download only its `gh-settings-linux-amd64` asset and `SHA256SUMS` through fixed upstream HTTPS paths; require one well-formed checksum for that asset, verify it before execution, and check the exact reported version. A SHA pin fixes the version; the checksum is trusted from that GitHub release. Never fall back to latest or accept an arbitrary URL.
- A strict version-tag invocation must agree with the committed CLI version or fail before download. SHA-pinned and local invocations read the committed version. Future release publication requires that same version; it cannot silently retain an older CLI.
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
| `cli-version` | Verified installed release version selected by the committed package version. |
| `cli-sha256` | SHA-256 of the verified installed CLI binary. |

- Emit these outputs only after command success and, for plan, successful Summary render/write. Composite outputs map to actual step outputs.
- Every successful command includes metadata resolution for these outputs; inability to resolve the checkout commit or target fails the Action even when standalone CLI validation or an empty Plan/Apply declaration needs no origin or credentials. Plan/apply still require their explicit token input for empty declarations; these independent audit requirements do not change with a CLI no-op.
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

### Rule: Source metadata identifies the consumer and executing binary

The source-only `gh settings action metadata` interface emits the existing `config-sha`, `target`, `cli-version` and `cli-sha256` keys as single-line `key=value` records. It resolves the current consumer checkout, including subdirectories, through the shared Go checkout/target rules. The executable or Action source directory is not the target. No settings file, token or API access is needed. The CLI version is the injected version; the digest identifies the actual executing binary. Missing checkout/commit/target/version, CR/LF in values and output errors fail visibly. This same-version interface adds no old-CLI fallback. The current source migration selects explicitly unreleased CLI v0.7.0 and binds composite setup directly to this verified executable entry from the consumer working directory. The released v0.6.1 Action/CLI pins remain unchanged. Setup receives no caller token; metadata failure must stop settings execution and successful result publication. Python Summary adoption remains separate.

The [public adapter tests](../../internal/actions/run_test.go) cover identity/output failures; `TestActionMetadata` in the [actual candidate acceptance suite](../../tests/acceptance/metadata_test.go) covers the normal consumer and executing-binary identities without API access. The [source interface](../commands.md#source-actions-metadata) describes output and failure semantics.

On the supported Linux runner, the digest remains bound to the running image after its launch pathname is replaced or deleted. `TestMetadataUsesRunningImage` in the [Linux regression](../../internal/actions/metadata_linux_test.go) verifies replacement of an owned child copy. Native development checks on other platforms assume an unchanged executable path and add no supported Action platform.

#### Scenario: A candidate reports its consumer identity

- Given a committed consumer checkout with a GitHub origin and no settings file or credentials
- And the candidate executable resides outside that checkout
- When metadata is requested from a consumer subdirectory
- Then the four records identify the consumer commit and origin, injected version and actual candidate digest
- And no API request occurs

#### Scenario: The unreleased composite uses verified Go metadata

- Given the source composite installs and verifies its committed unreleased CLI version
- And the consumer checkout is separate from the Action package
- When validate runs without caller credentials
- Then setup invokes that verified executable's `action metadata` entry from the consumer directory
- And the four successful audit outputs identify that consumer and candidate executable
- And metadata failure stops before settings execution or successful output publication

### Rule: Source planning writes ephemeral typed presentation data

The source-only `gh settings action plan REPORT_PATH` entry invokes shared application planning and writes one JSON report to a caller-owned runner-temporary file whose parent already exists. New report files use mode 0600. The same CLI/Action version owns the format: explicit outcome, available target, managed count and ordered changes with native string/bool before/after values. Changed, no-difference and unmanaged results are distinct without parsing terminal sentences. Invalid configuration and repository-read failure remain distinguishable failures with minimum safe categorical diagnostics; raw authentication errors, tokens, environment dumps and request headers are not serialized.

Success is silent on stdout and exits zero. Planning or report-storage failure exits nonzero, including when a failure report was written; writing a report never turns a failed plan into success. Validation precedes API access, and unmanaged planning retains its local no-op. Target resolution has an explicit `target-failure` outcome without an unresolved target; read failures retain the available target. Failure reports have no comparison, so their zero managed count does not mean unmanaged. Apply never reads this report: it is ephemeral presentation data, not saved-plan execution. Composite adoption is separate; no multi-version protocol is defined.

#### Scenario: A changed report retains native values

- Given declared repository description and issues settings differ from the observed values
- When the source plan entry writes its report
- Then the outcome is `changed` with the target, two managed fields and their ordered changes
- And descriptions remain JSON strings and issues values remain JSON booleans, including false
- And planning reads once, changes no configuration and returns success without terminal plan text

### Rule: Source Summary renders typed review data without credentials

The source-only `gh settings action summary REPORT_PATH SUMMARY_PATH` entry reads the same-version typed plan report and explicit nonsecret `SETTINGS_*` run metadata, then appends review content to the supplied Summary path. It does not read terminal plan sentences, configure authentication, call an API or dump the environment. Required context includes original and initiating/rerun actors, event, run/attempt, workflow path/ref and exact revision, configuration revision, target, distinct Action identity/ref and CLI version/checksum. Workflow and configuration links use their own repository and revision; local Action use has no invented remote pin.

Changed values retain native string/bool meaning. No-difference, unmanaged and planning failures are distinct from the typed outcome, and a failed plan remains nonzero after successful Summary writing. Missing/malformed report or required metadata and detectable writing failures return nonzero. Input loading, validation and rendering precede opening the append destination. Untrusted multiline metadata and values remain quoted data, including blank lines, Markdown, HTML and workflow-command-like text. The Summary states that direct apply may overwrite intervening changes and that rendering/writing does not prove GitHub publication. No publication polling or saved-plan apply is added; composite adoption and Python deletion remain separate.

#### Scenario: A changed Summary preserves review identities

- Given a changed typed report with native description and issues values
- And explicit workflow, configuration, Action and CLI identities with distinct workflow and target repositories
- And an existing caller-owned Summary
- When the source Summary entry runs without credentials
- Then it appends all required review context and observed/desired values without replacing prior content
- And links use the displayed identities, success is silent, no API request occurs and unrelated environment data is absent

### Public contract verification boundary

Required CI includes installer/command/renderer fixtures, root composite and actual consumer YAML composition checks, same-run candidate composite validate, and real CLI built-product acceptance. Offline consumer approval/token lifecycle and CLI/download responses are simulated; offline consumer fixtures copy the current candidate package with a mocked candidate-version CLI and published caller Action identity, replacing only the copied verifier expected version and dummy checksum. They do not load the pinned tag bytes.

Hosted smoke uses the CLI built once with the committed candidate version and the Action from that producer's exact source SHA, including PR merge source. A bounded test-only download fixture verifies the transferred artifact/source/version and serves only the unchanged installer's fixed two URLs; missing/tampered artifacts and unexpected requests fail. Actual `uses` expansion validates a separate configuration-only checkout and compares all four outputs to this run's identities. There is no product rebuild, production input override, committed version rewrite or old-release fallback. Candidate-fixture evidence does not verify real release downloads, publication or immutable released assets. Production-identity checks and active public-Action caller tests remain. Usage documentation is checked against the actual contract, snippets, example copies and helper references. Offline checks do not prove Environment enforcement, token revocation, Summary publication, or a live update. See the [CI evidence boundary](../development.md#ci-validation).
