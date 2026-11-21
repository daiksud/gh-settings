# Feature: Review settings before approved direct apply

## Rule: Examples are inert until separately adopted

- The two workflow examples live under `examples/github-actions/`; consumers copy no support scripts.
- Merging examples does not enable settings workflows, install an App, enter secrets, or apply settings.
- Every job uses a fresh GitHub-hosted Linux runner and reviewed full-SHA Action pins.
- Both examples use the reviewed full-SHA pin recorded in the [consumer guide](../../guides/github-actions.md#published-package-and-consumer-pins). The Action verifies its fixed CLI against the same-release SHA256SUMS and exact version before settings execution; no caller verifier is required.

### Scenario: The examples are present only under the examples directory

- Given the owner keeps both sample workflow files under `examples/github-actions/`
- When a settings change merges into the repository
- Then those example files start no job and apply no settings

## Rule: PR plan executes trusted code and consumes only candidate data

- Use the trusted base workflow on `pull_request_target`; never execute PR code with its credentials.
- The same App as main mints a target/permission-scoped short-lived token inside the protected `settings-plan` job; no personal credential or ambient fallback is required.
- Obtain only candidate commit/tree metadata and the fixed settings blob. Validate event SHA/number, file mode, blob identity/size and fixed destination; disable hooks, inherited Git credentials/programs and lazy object fetches. No PR code checkout, build, dependency install or local Action execution occurs.
- Validate the prepared root `.github/settings.yml` before nonmutating plan. Its configuration SHA identifies the candidate head, not the trusted workflow revision.
- Valid configuration and successful plan exit successfully whether differences exist or not.
- Missing/invalid configuration fails before API access; authentication/API failures fail visibly.
- Fork content remains untrusted data; it is never given executing code authority. Inaccessible candidate data fails rather than falling back to another credential, adopting another revision or applying settings.
- Plan does not mutate, but complete reads may need write authority (notably Ruleset bypass visibility); the supported App/permission boundary must be established per feature. Fixtures do not prove all live App integrations.
- Per-PR concurrency cancels superseded executions.

### Scenario: The PR head has moved

- Given a PR plan was triggered for one commit and the PR now points to another
- When the trusted workflow checks the candidate
- Then the preparation fails before the settings plan runs
- And neither revision supplies code to execute with the plan credential

## Rule: Main apply uses one approved job

- Only `push` on Branch `main` changing the configuration or adopted main workflow is eligible; retain explicit event/ref guards.
- The workflow has exactly one job and three `uses` steps: checkout, the official App token Action, and `daiksud/gh-settings` with `command: apply`.
- No caller `run` step, external script, prepare job, handoff output or separate apply Summary step is needed.
- Check out the triggering `github.sha` without persisted credentials; a newer main commit cannot change this run's configuration input.
- Review the PR plan and exact main configuration before approving the job. Main does not depend on the PR result, create a pre-approval plan or consume a saved plan.
- Repository-scoped concurrency does not cancel the running job; GitHub may replace a pending run.

### Scenario: A newer main commit arrives before approval

- Given a settings change triggered a main apply and another commit subsequently reaches main
- When the owner approves the original job and it starts
- Then checkout selects the commit that triggered that run

## Rule: The Summary separates identities from settings data

- First run shows `github.actor` and `github.triggering_actor`; reruns show the original actor separately from the rerun initiator and incremented attempt.
- Include event, run URL/attempt, workflow path/ref and exact workflow-source SHA/link.
- Include target, configuration SHA and exact configuration-file link, CLI version/checksum, and observed before → desired after output.
- Workflow-source SHA can differ from configuration SHA; each link uses the displayed identity.
- Distinguish changed settings, no differences, no managed fields, and errors.
- Render quotes, newlines, Markdown fences, HTML, and workflow-command-like values as data that cannot forge review context.
- State that direct apply can overwrite intervening changes to declared remote fields.
- The owner reviews complete PR plan content and the exact main configuration before approval. Successful planning alone does not prove Summary publication; no publication polling is required.

### Scenario: A PR plan uses a different workflow source and settings revision

- Given the trusted workflow runs from the base repository while the settings file comes from a PR commit
- When a detailed plan succeeds
- Then the Summary links the workflow and configuration to their respective revisions

## Rule: Owner approval gates credentials and apply

- The single main job waits at `settings-apply` before checkout, token creation or apply. It has no prepare dependency.
- Required reviewer is the owner; self-approval is allowed, administrator bypass is disabled, and only Branch `main` is eligible with no Tag rule.
- While waiting, or after rejected approval, no App token is minted and no apply runs.
- The Environment key is available only inside the approved job; the official token Action narrows Administration write to the one target repository and retains normal revocation.
- The token is supplied only to the apply step, never checkout, bootstrap, Summary, global environment, artifact, or job/workflow output.
- Apply uses the triggering configuration commit. The settings Action installs and verifies its CLI after token minting, then validates configuration before a settings API request.
- The Action step outcome and logs report apply success or failure. Missing credentials, installation or CLI failure fails visibly, and default token cleanup still runs after minting. Output failure may follow a successful update; no rollback is promised.
- Ordinary reruns use GitHub's approval behavior without custom freshness, drift, or reapproval checks.

### Scenario: An eligible main job waits for review

- Given the owner has configured required reviewers on the apply Environment
- When an eligible main run is awaiting approval
- Then the apply job has not checked out the repository or minted its token
- And rejecting the request prevents the job from applying settings

## Verification boundary

Focused offline checks and composed consumer coverage run in required CI, alongside hosted validate smoke and built-product acceptance. Markdown/local-link checks validate document structure; they do not prove runtime behavior. Actual Environment waiting/rejection, branch/tag enforcement, and token revocation require separately authorized GitHub verification; offline fixtures cannot establish those platform behaviors.

## Public composite Action contract

Consumers can export, validate, inspect, or directly apply their checked-out repository settings through a reviewed Action package without copying helper files. The implemented root Action follows [ADR 0023](../adr/0023-public-composite-action.md); the [usage guide](../../guides/github-actions.md) identifies reviewed full-SHA calls and supported consumer files. The preceding workflow-example rules remain applicable to those examples.

## Rule: A release selects the same Action and CLI version

- Invoke the root Action as `uses: daiksud/gh-settings@<approved full commit SHA>`; use a reviewed revision containing the implemented package.
- The Action revision selects packaged support code and its matching released CLI; see the [consumer pins](../../guides/github-actions.md#published-package-and-consumer-pins). Source changes never rewrite immutable tags or assets.
- A published Action tag selects that same CLI release. Full-SHA pins must resolve exactly one matching stable Git tag; do not infer a new release from current main. Download only its `gh-settings-linux-amd64` and `SHA256SUMS` through fixed HTTPS paths; verify the checksum and exact reported binary version. Never fall back to latest or accept an arbitrary URL.
- Untagged or ambiguously tagged SHA references fail before download. Local candidate smoke invokes the composite without a public Action ref and explicitly supplies the accepted product version; the committed `action/cli-version` remains only a development-candidate default. A published Action never silently retains an older CLI.
- Support only a fresh GitHub-hosted Ubuntu 24.04 x64 job. The caller checks out the configuration repository at `GITHUB_WORKSPACE` without persisted credentials.
- Use only checkout-root `.github/settings.yml`, creating or refreshing it for export; use the existing CLI's checkout-scoped GitHub `origin` target semantics, not the Action repository or an assumed `github.repository` target. Current source requires that selected origin to directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or full SSH. Transport mirror rewrites cannot change the direct identity; rewrite-only aliases and username-less SCP fail metadata before settings execution or successful outputs. Local/include/worktree selection and ignoring `GH_REPO` and system/global/command substitute origins remain shared CLI rules.
- Action-owned installation bootstrap comes from the Action package, independently of caller helper files. The composite directly invokes export/validate/apply and Go metadata/plan/Summary entries through the verified absolute CLI path in a per-invocation runner-temp location. Shell retains necessary Actions wiring.

### Scenario: A full SHA has two matching stable release tags

- Given two distinct stable version tags resolve to the selected Action commit
- When the caller invokes the Action pinned to that SHA
- Then installation fails before downloading a CLI or running a settings command

## Rule: Inputs select one bounded command

| Input | Requirement | Meaning |
| --- | --- | --- |
| `command` | Required, no default | Exact enum `export`, `validate`, `plan`, or `apply`; reject omitted or unknown values at runtime. |
| `token` | Optional for `validate`; required and nonempty for `export`/`plan`/`apply`, no default | Explicit caller token with permissions appropriate to that command. Never fall back to `github.token` or ambient authentication. |
| `plan-reporting` | Default `detailed`; `status-only` is plan-only | Export creates no Summary and rejects status-only. |
| `prune` | Default empty; nonempty is plan/apply-only | Export and validate reject deletion scopes before installation. |

- `validate` performs local CLI validation without authentication or API calls, even if a token input was supplied.
- `plan` requires an explicit caller token with complete observation grants, validates first, and runs read-only CLI plan only after validation succeeds. Differences, no differences, and no managed fields are all successful outcomes.
- `export`, `plan` and `apply` use one caller-created GitHub App, with potentially different short-lived target/permission-scoped installation tokens per job. Apply needs each resource's write/observation permissions, not just a universal Administration grant. Invoke existing direct CLI apply; a previous plan is not required or consumed.
- No arbitrary command/arguments, configuration-path, target, version, asset, or download-URL input is supported.

### Scenario: Validate without a credential

- Given a valid configuration checkout on a supported runner
- And no token input or ambient authentication
- When the caller selects `command: validate`
- Then validation succeeds without GitHub API access
- And all four non-secret success outputs identify that checkout and verified CLI

### Scenario: Invalid configuration blocks plan

- Given an explicit App installation token and invalid `.github/settings.yml`
- When the caller selects `command: plan`
- Then validation fails before comparison or API execution
- And the Action fails without success outputs

## Rule: Export creates a local file without publication

- Use shared CLI automatic export with an explicit token and a committed checkout;
  a missing `.github/settings.yml` or `.github` directory permits first export.
- Retain shared invalid-YAML, symlink, concurrent-edit and I/O failure boundaries.
  Independent unavailable observations remain bounded info, not fabricated values.
- Unset inherited `GH_DEBUG`, discard export stdout and expose no file contents
  through logs, public outputs, plan reports or Summary. Keep ordinary diagnostics.
- Return the fixed config-path only after success; config-sha remains checkout HEAD.
- Do not commit, push, upload or cache the file. It may contain plaintext Variables;
  other code in the caller's trusted job can read it. Publication is caller-owned.
- CLI/output failures fail the Action and withhold successful results. A failed
  first-file write or later output write may leave local changes; no rollback.
- Failed info or required normalization-warning delivery stops the shared CLI
  before saving and withholds successful Action outputs.
- Verify actual candidate composite first export and re-export, token boundaries,
  variable-value canaries, exact outputs and representative local failures.

### Scenario: First export in a committed checkout

- Given a committed checkout has no settings file and the caller supplies an installation token
- When the caller selects `command: export` successfully
- Then `config-path` identifies the new file in that checkout and `config-sha` identifies its original commit
- And the Action makes no commit and publishes no settings file

## Rule: Only successful completion exposes result outputs

### Scenario: A validated plan cannot publish its Summary

- Given the settings comparison succeeded and the Summary cannot be written
- When the Action finishes the plan step
- Then the step fails and supplies no successful result outputs

| Output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit SHA (`HEAD`), not a configuration-file hash, workflow SHA, or Action revision. |
| `config-path` | Fixed `.github/settings.yml` relative to the supported checkout root after export; empty for other commands. No file publication or transfer is implied. |
| `plan-outcome` | Successful status-only plan outcome; empty for export and other reporting modes/commands. |
| `target` | Resolved configuration checkout's GitHub repository in `OWNER/REPO` form, consistent with CLI `origin` semantics. |
| `cli-version` | Verified installed CLI version selected by the published Action reference. |
| `cli-sha256` | SHA-256 of the verified installed CLI binary. |

- Emit these outputs only after command success and, for plan, successful Summary render/write. Composite outputs map to actual step outputs.
- Every successful command includes metadata resolution for these outputs; inability to resolve the checkout commit or target fails the Action even when standalone CLI validation or an empty Plan/Apply declaration needs no origin or credentials. Plan/apply still require their explicit token input for empty declarations; these independent audit requirements do not change with a CLI no-op.
- Missing/unknown command, missing required token, unsupported execution prerequisites, installation/download/checksum/version failure, configuration/CLI/API failure, metadata/output failure, and detectable plan Summary failure fail visibly. Do not run settings commands after installation failure; stop plan after validation failure.
- Preserve CLI failure status and ordinary plan exit 0 when differences exist. Error review content must not imply success. Failed runs have no successful output contract; no token is an output.
- Output or Summary failure may occur after a remote update; no automatic rollback or atomicity is promised.

## Rule: Plan Summary separates execution identities and untrusted data

- The Action owns plan Summary rendering/writing and distinguishes successful and failed planning. It uses typed reports and categorical Summary diagnostics; original CLI stderr remains in command logs.
- Include original and rerun actors, event, run URL/attempt, workflow path/ref and exact workflow revision, target, configuration commit and exact file link, distinct Action identity/ref, and CLI version/SHA-256.
- Obtain Action identity/ref from supported composite context; never relabel workflow SHA as Action SHA. Identify local CI invocation honestly when there is no remote pin.
- Current source distinguishes differences, no differences, no managed settings and failures from the same-version typed report, without classifying terminal sentences.
- Escape untrusted values, including quotes, multiline text, Markdown, HTML, and workflow-command-like content, so they cannot forge Summary markup, links, or execution context.
- State that the observed diff is informational and direct apply may overwrite intervening remote changes. A later apply is not bound to a saved plan.
- Detectable render/write failure fails plan before success outputs. Successful writing does not prove GitHub published the Summary; the owner must not approve missing/incomplete content. No publication polling is required.

### Scenario: A description contains markup

- Given the observed or declared description contains an HTML tag
- When the Action renders a detailed plan Summary
- Then the tag is displayed as settings data and cannot create an HTML element in the review

## Rule: The caller owns approval and credentials

- The caller owns checkout, event/ref guards, Environment selection/approval, App token minting/revocation, and concurrency. The Action does not perform those operations or introduce custom approval/freshness machinery.
- Bind `inputs.token` as `GH_TOKEN` only on the export/plan/apply command execution step. Validate, installer, Summary, and metadata/output steps must not receive that input as `GH_TOKEN`; never write it to `GITHUB_ENV`, files, logs, artifacts, or outputs.
- This is scoped environment exposure, not token process isolation. Composite steps share the caller job; all code in a credential-bearing job must be trusted. The caller must keep privileged credentials away from untrusted PR code.
- Preserve owner-approved main direct apply and repository-scoped concurrency without cancellation of running apply. PR plan remains nonmutating with trusted executing code, data-only candidate preparation and no alternate-credential fallback. Strongly recommend Environment branch restrictions together with enforced protection of workflow/credential-consuming paths; a permitted ref does not identify a workflow.
- These are the generic consumer/example requirements. This repository's Environment, App bindings, and scope guard are documented separately in [repository automation](../repository-automation.md).

### Scenario: A caller omits the token from a plan

- Given the checkout has valid settings and the caller omits the token input
- When the caller selects `command: plan`
- Then the Action fails before installation instead of using an ambient credential

## Rule: Identify the checkout and installed CLI

### Scenario: A workflow validates a separate consumer checkout

- Given a committed consumer checkout with a direct GitHub.com origin and a valid settings file
- And the pinned Action installs its verified CLI outside that checkout
- When the caller runs validate without an API credential
- Then the success outputs identify the consumer's commit and repository, the installed CLI version and the digest of its executing binary
- And validation makes no settings API request

### Scenario: The consumer checkout cannot supply its own target

- Given the checkout has only a global origin, a rewrite-only alias or an unsupported URL
- When the caller runs validate
- Then the Action fails before settings execution and publishes no success outputs

The [identity checks](../../../tests/acceptance/metadata_test.go) exercise direct and indirect origins. The [composite checks](../../../tests/github-actions/composite_test.go) keep the consumer target separate from the Action checkout. On Linux the [running-image test](../../../internal/actions/metadata_linux_test.go) verifies that replacing the launch pathname cannot change the reported digest. See [Actions internals](../actions-internals.md) for the private metadata, plan and Summary interfaces.

## Rule: Publish plan results after Summary completion

### Scenario: A changed plan appends a review to the job Summary

- Given the declared repository description and Issues setting differ from the current values
- And the caller already has content in its job Summary
- When the Action completes a detailed plan
- Then it appends the observed and declared string and boolean values, including explicit false
- And the review identifies the run and attempt, actors, workflow source, configuration commit, target, Action and verified CLI
- And the workflow and configuration links identify their respective repositories and revisions
- And supplied HTML, Markdown and multiline values remain quoted data
- And rendering requires no settings credential and does not change the repository or the declaration

### Scenario: Planning or Summary publication fails

- Given the caller supplies an invalid declaration or the target read fails
- When the Action renders the failure for review
- Then invalid configuration fails before API access and a failed read remains distinguishable from a validation failure
- And the original plan error leaves the step unsuccessful even if a diagnostic Summary was written
- And a required report, rendering, append or cleanup failure also prevents successful plan outputs

A successful plan can report changes, a match or no managed settings. A detailed Summary includes the direct-apply warning. A later apply reads the declaration and current state independently of the earlier plan. Local checks can inspect the bytes written to a Summary file. Only a hosted run can show how GitHub handled that file.

## Rule: Private plan reporting does not publish observed values

Current source supports opt-in `plan-reporting: status-only`. Detailed mode remains the default; the caller still owns the trusted job, identities and token grants. Private mode is plan-only, checks installed capability before authentication and validates the complete report before reducing output to outcome/counts. This package capability is independent of this repository's operational workflows.

### Scenario: A private plan reports status without hidden policy

- Given observed settings, Ruleset actors or API error text contain confidential values
- When the caller runs plan with `plan-reporting: status-only`
- Then the command logs and Summary hide the values and raw diagnostics
- And a successful plan exposes `plan-outcome` after Summary writing and report cleanup
- And an ordinary failure removes the report and retains the failing status

Forced termination can interrupt cleanup.

Verified by `TestPrivatePlanHidesErrors`, `TestPrivateSummaryHidesCompleteRulesetPolicy`, `TestPrivateSummaryRejectsUnusableReportsWithoutDisclosure`, `TestPrivateCompositeOutcomesAndCleanup`, `TestPrivateCompositeFailures`, `TestPrivateCompositeInputBoundary` and the real candidate private scenarios. These tests do not prove a published release or safe arbitrary caller code.

## Rule: Render native topic lists safely

### Scenario: Topic values appear as sorted JSON arrays

- Given the topic list changes from `["old"]` to `["docs", "go"]`
- When the Action renders a detailed Summary
- Then the review displays both sorted lists with each topic escaped as data

Invalid, null, duplicate, unsorted, unsafe or equal topic arrays fail validation before the Summary is written. Other setting types keep their native representation.

Test mapping: [TestTopicsSummaryNativeAndJSONLists](../../../internal/actions/summary_test.go), [TestTopicsExportAndActionReport](../../../tests/acceptance/topics_test.go).

## Rule: Present typed label operations

### Scenario: The plan changes the color of a label

- Given the declaration selects label ID 42 and requests a different color
- When the Action renders a detailed plan
- Then the Summary identifies the label update and shows the old and new colors

The same-version report/Summary presents typed label operations. The [Labels specification](labels.feature.md) owns label scenarios and test mappings, and [Actions internals](../actions-internals.md#source-actions-plan-report) owns the report schema. See the [package selection guide](../../guides/github-actions.md#published-package-and-consumer-pins) for the CLI version installed by a published Action.

## Public contract verification boundary

The [testing guide](../testing.md#ci-validation) owns fixture setup, the hosted candidate case matrix, and the evidence produced by each CI stage. The checks associated with this specification are:

| Contract | Checks |
| --- | --- |
| Installer prerequisites, selected version, and checksum validation | [Installer fixtures](../../../tests/github-actions/action-install.sh) and [release gate tests](../../../tests/github-actions/release_test.go) |
| Consumer and executing-binary identity | [Actions owner tests](../../../internal/actions/run_test.go) and [metadata acceptance](../../../tests/acceptance/metadata_test.go) |
| Typed plan reports and credential-free Summary | [Report](../../../internal/actions/plan_test.go) and [Summary](../../../internal/actions/summary_test.go) owner tests; [plan-report](../../../tests/acceptance/plan_report_test.go) and [Summary](../../../tests/acceptance/summary_test.go) acceptance |
| Root composite status/output/token wiring | [Composite tests](../../../tests/github-actions/composite_test.go) |
| Example and active caller composition | [Consumer tests](../../../tests/github-actions/consumer_test.go) and [operational caller tests](../../../tests/github-actions/operational_test.go) |
| Actual same-source hosted export, validate, plan, direct apply, and failure paths | [Candidate assertions](../../../tests/github-actions/candidate-plan.sh) run during [main CI](../../../.github/workflows/ci-push.yml) Inspection and [release](../../../.github/workflows/release.yml) Inspection when artifacts are rebuilt |

Offline approval/token lifecycle checks use simulations. Candidate integration and Summary-file assertions do not establish released-download identity, Environment enforcement, actual token revocation, GitHub Summary publication, or a live settings update; see the [evidence table](../testing.md#release-and-publication-evidence). Reviewers check usage documentation against the contract and example files; Markdown CI verifies links and structure, not semantic equivalence.
