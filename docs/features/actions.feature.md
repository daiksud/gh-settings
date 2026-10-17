---
type: Feature
title: Owner-approved settings automation
description: Specify inert workflow examples and the public composite Action around the existing CLI.
---

## Feature: Review settings before approved direct apply

### Rule: Examples are inert until separately adopted

- The two workflow examples live under `examples/github-actions/`; consumers copy no support scripts.
- Merging examples does not enable settings workflows, install an App, enter secrets, or apply settings.
- Every job uses a fresh GitHub-hosted Linux runner and reviewed full-SHA Action pins.
- Both examples pin published v0.7.0. The Action verifies its fixed CLI against the same-release SHA256SUMS and exact version before settings execution; no caller verifier is required.

### Rule: PRs have only read access

- On a PR, check out the containing repository's PR revision without persisted credentials.
- Validate the checkout-root `.github/settings.yml` before read-only plan.
- Valid configuration and successful plan exit successfully whether differences exist or not.
- Missing/invalid configuration fails before API access; authentication/API failures fail visibly.
- Fork changes have no App credential, Environment, apply, or privileged fallback path.
- Per-PR concurrency cancels superseded executions.

### Rule: Main apply uses one approved job

- Only `push` on Branch `main` changing the configuration or adopted main workflow is eligible; retain explicit event/ref guards.
- The workflow has exactly one job and three `uses` steps: checkout, the official App token Action, and `daiksud/gh-settings` with `command: apply`.
- No caller `run` step, external script, prepare job, handoff output or separate apply Summary step is needed.
- Check out the triggering `github.sha` without persisted credentials; a newer main commit cannot change this run's configuration input.
- Review the PR plan and exact main configuration before approving the job. Main does not depend on the PR result, create a pre-approval plan or consume a saved plan.
- Repository-scoped concurrency does not cancel the running job; GitHub may replace a pending run.

### Rule: The Summary separates identities from settings data

- First run shows `github.actor` and `github.triggering_actor`; reruns show the original actor separately from the rerun initiator and incremented attempt.
- Include event, run URL/attempt, workflow path/ref and exact workflow-source SHA/link.
- Include target, configuration SHA and exact configuration-file link, CLI version/checksum, and observed before → desired after output.
- Workflow-source SHA can differ from configuration SHA; each link uses the displayed identity.
- Distinguish changed settings, no differences, no managed fields, and errors.
- Render quotes, newlines, Markdown fences, HTML, and workflow-command-like values as data that cannot forge review context.
- State that direct apply can overwrite intervening changes to declared remote fields.
- The owner reviews complete PR plan content and the exact main configuration before approval. Successful planning alone does not prove Summary publication; no publication polling is required.

### Rule: Owner approval gates credentials and apply

- The single main job waits at `settings-apply` before checkout, token creation or apply. It has no prepare dependency.
- Required reviewer is the owner; self-approval is allowed, administrator bypass is disabled, and only Branch `main` is eligible with no Tag rule.
- While waiting, or after rejected approval, no App token is minted and no apply runs.
- The Environment key is available only inside the approved job; the official token Action narrows Administration write to the one target repository and retains normal revocation.
- The token is supplied only to the apply step, never checkout, bootstrap, Summary, global environment, artifact, or job/workflow output.
- Apply uses the triggering configuration commit. The settings Action installs and verifies its CLI after token minting, then validates configuration before a settings API request.
- The Action step outcome and logs report apply success or failure. Missing credentials, installation or CLI failure fails visibly, and default token cleanup still runs after minting. Output failure may follow a successful update; no rollback is promised.
- Ordinary reruns use GitHub's approval behavior without custom freshness, drift, or reapproval checks.

### Verification boundary

Focused offline checks and composed consumer coverage run in required CI, alongside hosted validate smoke and built-product acceptance. Markdown/local-link checks validate document structure; they do not prove runtime behavior. Actual Environment waiting/rejection, branch/tag enforcement, and token revocation require separately authorized GitHub verification; offline fixtures cannot establish those platform behaviors.

### Public composite Action contract

Consumers can validate, inspect, or directly apply their checked-out repository settings through a reviewed Action package without copying helper files. The implemented root Action follows [ADR 0023](../adr/0023-public-composite-action.md); the [usage guide](../github-actions.md) identifies reviewed full-SHA calls and supported consumer files. The preceding workflow-example rules remain applicable to those examples.

### Rule: A release selects the same Action and CLI version

- Invoke the root Action as `uses: daiksud/gh-settings@<approved full commit SHA>`; use a reviewed revision containing the implemented package.
- The Action revision selects packaged support code. Current source selects published v0.7.1 while examples and active workflows retain reviewed v0.7.0 pins; see the [consumer pins](../github-actions.md#published-package-and-consumer-pins). [Older release limitations](../releases.md#historical-release-limitations) remain historical; source changes never rewrite immutable tags or assets.
- The committed `action/cli-version` selects an explicit CLI release. Download only its `gh-settings-linux-amd64` asset and `SHA256SUMS` through fixed upstream HTTPS paths; require one well-formed checksum for that asset, verify it before execution, and check the exact reported version. A SHA pin fixes the version; the checksum is trusted from that GitHub release. Never fall back to latest or accept an arbitrary URL.
- A strict version-tag invocation must agree with the committed CLI version or fail before download. SHA-pinned and local invocations read the committed version. Future release publication requires that same version; it cannot silently retain an older CLI.
- Support only a fresh GitHub-hosted Ubuntu 24.04 x64 job. The caller checks out the configuration repository at `GITHUB_WORKSPACE` without persisted credentials.
- Read only checkout-root `.github/settings.yml`; use the existing CLI's checkout-scoped GitHub `origin` target semantics, not the Action repository or an assumed `github.repository` target. Current source requires that selected origin to directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or full SSH. Transport mirror rewrites cannot change the direct identity; rewrite-only aliases and username-less SCP fail metadata before settings execution or successful outputs. Local/include/worktree selection and ignoring `GH_REPO` and system/global/command substitute origins remain shared CLI rules. Released v0.6.1 retains its original compatibility.
- Action-owned installation bootstrap comes from the Action package, independently of caller helper files. Published v0.7.0 directly invokes validate/apply and Go metadata/plan/Summary entries through the verified absolute CLI path in a per-invocation runner-temp location. Shell retains necessary Actions wiring; released v0.6.1 retains its original helpers.

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
- Then validation fails before comparison or API execution
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

- The Action owns plan Summary rendering/writing and distinguishes successful and failed planning. Published v0.7.0 uses typed reports and safe categorical Summary diagnostics; original CLI stderr remains in command logs. Released v0.6.1 retains its original quoted CLI-text presentation.
- Include original and rerun actors, event, run URL/attempt, workflow path/ref and exact workflow revision, target, configuration commit and exact file link, distinct Action identity/ref, and CLI version/SHA-256.
- Obtain Action identity/ref from supported composite context; never relabel workflow SHA as Action SHA. Identify local CI invocation honestly when there is no remote pin.
- Current source distinguishes differences, no differences, no managed settings and failures from the same-version typed report, without classifying terminal sentences.
- Escape untrusted values, including quotes, multiline text, Markdown, HTML, and workflow-command-like content, so they cannot forge Summary markup, links, or execution context.
- State that the observed diff is informational and direct apply may overwrite intervening remote changes. A later apply is not bound to a saved plan.
- Detectable render/write failure fails plan before success outputs. Successful writing does not prove GitHub published the Summary; the owner must not approve missing/incomplete content. No publication polling is required.

### Rule: The caller owns approval and credentials

- The caller owns checkout, event/ref guards, Environment selection/approval, App token minting/revocation, and concurrency. The Action does not perform those operations or introduce custom approval/freshness machinery.
- Bind `inputs.token` as `GH_TOKEN` only on the plan/apply command execution step. Validate, installer, Summary, and metadata/output steps must not receive that input as `GH_TOKEN`; never write it to `GITHUB_ENV`, files, logs, artifacts, or outputs.
- This is scoped environment exposure, not token process isolation. Composite steps share the caller job; all code in a credential-bearing job must be trusted. The caller must keep privileged credentials away from untrusted PR code.
- Preserve owner-approved main direct apply and repository-scoped concurrency without cancellation of running apply. PR plan remains read-only with no privileged fallback.
- Active workflows retain Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, and `ANSWER26_PRIVATE_KEY`, and use the single-job main apply pattern. A [trusted-source PR Environment guard](../github-actions.md#active-repository-environment-guard) detects unauthorized declarations and rejects reusable-workflow delegation. It must not replace the deployment reviewer gate until separately enforced by a non-spoofable required workflow or dedicated App check, with `main` still the only eligible deployment branch. The reserved status `gh-settings` is unique in the repository's intended workflow files, but a same-named job in PR-controlled GitHub Actions still cannot provide an unforgeable merge gate. Inert examples retain the `settings-apply` Environment approval contract. This specification requires no credential operation, release, or live apply.

### Rule: Source metadata identifies the consumer and executing binary

The same-version `gh settings action metadata` interface emits the existing `config-sha`, `target`, `cli-version` and `cli-sha256` keys as single-line `key=value` records. It resolves the current consumer checkout, including subdirectories, through the shared Go checkout/target rules. The executable or Action source directory is not the target. No settings file, token or API access is needed. The CLI version is the injected version; the digest identifies the actual executing binary. Missing checkout/commit/target/version, CR/LF in values and output errors fail visibly. This same-version interface adds no old-CLI fallback. The current source selects published CLI v0.7.1 and binds composite setup directly to this verified executable entry from the consumer working directory. The active workflows use the published v0.7.0 Action/CLI pins. Setup receives no caller token; metadata failure must stop settings execution and successful result publication. Current source also adopts typed Go plan/Summary presentation; released v0.6.1 keeps its original helpers.

The [public adapter tests](../../internal/actions/run_test.go) cover identity/output failures, including CR/LF, empty target and writer protection; `TestActionMetadata` in the [actual candidate acceptance suite](../../tests/acceptance/metadata_test.go) covers the normal consumer and executing-binary identities without API access. `TestActionMetadataRejectsIndirectOrigin` covers both removed indirect forms in committed consumer checkouts, without credentials, identity output or API access. The [source interface](../commands.md#source-actions-metadata) describes output and failure semantics. `TestRootCompositeRejectsGlobalOnlyOrigin` in the [composed tests](../../tests/github-actions/composite_test.go) invokes the real Go metadata adapter through an owned test-executable child: a committed consumer with only an unrelated global origin fails before settings/API/success outputs even when the separate Action checkout has a valid direct origin and commit. The failure precedes binary hashing; normal mocked identity records are separate from actual candidate digest evidence. Detailed format/scope rules remain at the workspace owner, without a composite URL matrix.

On the supported Linux runner, the digest remains bound to the running image after its launch pathname is replaced or deleted. `TestMetadataUsesRunningImage` in the [Linux regression](../../internal/actions/metadata_linux_test.go) verifies replacement of an owned child copy. Native development checks on other platforms assume an unchanged executable path and add no supported Action platform.

#### Scenario: A candidate reports its consumer identity

- Given a committed consumer checkout with a GitHub origin and no settings file or credentials
- And the candidate executable resides outside that checkout
- When metadata is requested from a consumer subdirectory
- Then the four records identify the consumer commit and origin, injected version and actual candidate digest
- And no API request occurs

#### Scenario: Source metadata rejects an indirect origin

- Given a committed consumer checkout whose origin is a rewrite-only alias or username-less SCP form
- When the current candidate is asked for Action metadata without credentials
- Then it exits 1 with an origin diagnostic and no identity output
- And no API request occurs

#### Scenario: The composite uses verified Go metadata

- Given the source composite installs and verifies its committed CLI version
- And the consumer checkout is separate from the Action package
- When validate runs without caller credentials
- Then setup invokes that verified executable's `action metadata` entry from the consumer directory
- And the four successful audit outputs identify that consumer and candidate executable
- And metadata failure stops before settings execution or successful output publication

### Rule: Source planning writes ephemeral typed presentation data

The same-version `gh settings action plan REPORT_PATH` entry invokes shared application planning and writes one JSON report to a caller-owned runner-temporary file whose parent already exists. New report files use mode 0600. The same CLI/Action version owns the format: explicit outcome, available target, managed count and ordered changes with native string/bool/topic-list before/after values. Changed, no-difference and unmanaged results are distinct without parsing terminal sentences. Invalid configuration and repository-read failure remain distinguishable failures with minimum safe categorical diagnostics; raw authentication errors, tokens, environment dumps and request headers are not serialized.

Success is silent on stdout and exits zero. Planning or report-storage failure exits nonzero, including when a failure report was written; writing a report never turns a failed plan into success. Validation precedes API access, and unmanaged planning retains its local no-op. Target resolution has an explicit `target-failure` outcome without an unresolved target; read failures retain the available target. Failure reports have no comparison, so their zero managed count does not mean unmanaged. Apply never reads this report: it is ephemeral presentation data, not saved-plan execution. The current composite invokes this entry directly with its explicit read token; shared application planning owns validation before API access. No separate CLI validation call or multi-version protocol is needed.

#### Scenario: A changed report retains native values

- Given declared repository description and issues settings differ from the observed values
- When the source plan entry writes its report
- Then the outcome is `changed` with the target, two managed fields and their ordered changes
- And descriptions remain JSON strings and issues values remain JSON booleans, including false
- And planning reads once, changes no configuration and returns success without terminal plan text

### Rule: Source Summary renders typed review data without credentials

The same-version `gh settings action summary REPORT_PATH SUMMARY_PATH` entry reads the same-version typed plan report and explicit nonsecret `SETTINGS_*` run metadata, then appends review content to the supplied Summary path. It does not read terminal plan sentences, configure authentication, call an API or dump the environment. Required context includes original and initiating/rerun actors, event, run/attempt, workflow path/ref and exact revision, configuration revision, target, distinct Action identity/ref and CLI version/checksum. Workflow and configuration links use their own repository and revision; local Action use has no invented remote pin.

Changed values retain native string/bool/topic-list meaning. No-difference, unmanaged and planning failures are distinct from the typed outcome, and a failed plan remains nonzero after successful Summary writing. Missing/malformed report or required metadata and detectable writing failures return nonzero. Input loading, validation and rendering precede opening the append destination. Untrusted multiline metadata and values remain quoted data, including blank lines, Markdown, HTML and workflow-command-like text. The Summary states that direct apply may overwrite intervening changes and that rendering/writing does not prove GitHub publication. The current composite invokes this entry in a separate credential-free step after authenticated planning, using the same ephemeral report and verified executable. Original nonzero plan status wins over rendering status; otherwise a Summary failure determines failure. The four public outputs follow all required work. Python rendering and terminal-text classification are removed, without publication polling or saved-plan apply.

#### Scenario: A changed Summary preserves review identities

- Given a changed typed report with native description and issues values
- And explicit workflow, configuration, Action and CLI identities with distinct workflow and target repositories
- And an existing caller-owned Summary
- When the source Summary entry runs without credentials
- Then it appends all required review context and observed/desired values without replacing prior content
- And links use the displayed identities, success is silent, no API request occurs and unrelated environment data is absent

#### Scenario: Composite plan separates authenticated execution from presentation

- Given the composite has verified its same-version CLI and consumer metadata
- And the caller supplies an explicit read token
- When the authenticated step writes a typed plan report
- Then a separate credential-free step renders that report with explicit nonsecret run metadata
- And changed, unchanged, unmanaged and failed outcomes retain their intended review meaning
- And a failed plan remains failed even after diagnostic rendering
- And report, metadata, render, write or status-handoff failure prevents successful public outputs

### Public contract verification boundary

Required CI includes installer fixtures, Go report/Summary owner tests, root composite and consumer YAML composition checks, same-run candidate composite validate and local plan, and real CLI built-product acceptance. Offline consumer approval/token lifecycle and CLI/download responses are simulated; consumer fixtures copy the current candidate package with a mocked candidate-version CLI and each caller's declared Action identity. Both example and operational main callers are checked with the single-job apply fixture; the old operational handoff verifier is no longer invoked. Their mock plan producer emits explicit typed fixture data, and presentation calls the actual Go Summary adapter through an owned test-executable child. They do not load the pinned tag bytes or rebuild the product.

The real composite plan cases use the required [Unix-socket API fixture](../../tests/github-actions/plan-fixture/main.go), isolated GitHub CLI configuration and literal dummy read token. The [runner checks](../../tests/github-actions/candidate-plan.sh) observe actual outcomes and outputs, unchanged committed configuration, and a finite capture of the actual embedded-step Summary bytes:

| Candidate case | API observation | Action outcome and outputs | Summary observation |
| --- | --- | --- | --- |
| Changed string and boolean | Exactly one repository GET | Success and four expected identities | Native false and escaped HTML/multiline diff |
| Local read403 | Exactly one repository GET | Failure and four empty outputs | Categorical repository-read failure |
| Committed invalid boolean | Zero requests with a live required fixture | Failure and four empty outputs | Categorical configuration-validation failure |

The same composite also invokes real direct apply with a separate literal dummy write token. Its received PATCH is decoded and recorded, then independently compared as a complete native JSON object; no expected payload is manufactured by the server.

| Candidate apply case | API observation | Action outcome and outputs | Summary observation |
| --- | --- | --- | --- |
| Declared empty string and false | Exactly one PATCH, no GET; only `description: ""` and `has_issues: false` | Success and four expected identities | No new Settings review |
| Local write403 | Exactly one identical attempted PATCH, no GET | Failure and four empty outputs | No new Settings review |
| Committed invalid boolean | Zero requests and no captured PATCH with a live fixture | Failure and four empty outputs | No new Settings review |
| Empty `settings: {}` declaration | Zero requests and no captured PATCH with a live fixture | Success and four expected identities | No new Settings review |

All apply cases preserve configuration bytes and commit. A write error does not establish the remote outcome. Empty apply retains the public composite's explicit token and metadata requirements. Existing composed tests own token-free setup/metadata/rendering and explicit execution-token boundaries; bounded caller fixtures retain their labeled approval/token-revocation simulations.

Unexpected method, target or dummy-token use is rejected. Missing local fixture, unavailable Summary bytes or failed assertions cannot become skipped success. No-difference/unmanaged details remain in owner tests. This is actual candidate execution and rendering evidence, not Summary publication, real credential lifecycle or live API proof.

Hosted smoke uses the CLI built once with the committed candidate version and the Action from that producer's exact source SHA, including PR merge source. A bounded test-only download fixture verifies the transferred artifact/source/version and serves only the unchanged installer's fixed two URLs; missing/tampered artifacts and unexpected requests fail. Actual `uses` expansion validates a separate configuration-only checkout and compares all four outputs to this run's identities. There is no product rebuild, production input override, committed version rewrite or old-release fallback. Candidate-fixture evidence does not verify real release downloads, publication or immutable released assets. Production-identity checks and active public-Action caller tests remain. Usage documentation is checked against the actual contract, snippets, example copies and helper references. Offline checks do not prove Environment enforcement, token revocation, Summary publication, or a live update. See the [CI evidence boundary](../development.md#ci-validation).

### Rule: Render native topic lists safely

- Given a same-version report contains a topics.names change.
- When the credential-free Summary step runs.
- Then sorted string arrays render as escaped JSON lists.
- And invalid, null, duplicate, unsorted, unsafe or equal arrays fail before writing.
- And existing repository fields retain their native types.

Test mapping: [TestTopicsSummaryNativeAndJSONLists](../../internal/actions/summary_test.go), [TestTopicsExportAndActionReport](../../tests/acceptance/topics_test.go).

### Rule: Present typed label operations

The unreleased [Labels contract](labels.feature.md) extends this command with authoritative membership, omitted attribute preservation, typed create/update/delete operations and its own process/owner tests. Repository/topics behavior above is retained; Labels apply alone reads current state before reconciling. Deleted labels lose issue/PR associations. Published v0.8.0 does not include this extension.
