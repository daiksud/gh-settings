# Testing and CI

Tests verify current product and workflow contracts at their owning boundaries. Use [development](development.md#development-checks) for tool setup and commands, [architecture](architecture.md) for ownership, and [repository automation](repository-automation.md) for PR state and operational workflows.

## Test ownership

| Concern | Owning checks | Scope |
| --- | --- | --- |
| Declaration presence and comparison | [Repository model](../../internal/settings/repository_test.go), [comparison](../../internal/settings/comparison_test.go), [topics](../../internal/settings/topics_test.go), [labels](../../internal/settings/labels_test.go) | Omitted vs explicit values, managed counts, native values, topic sets, label identity/attribute presence, and stable operation ordering |
| YAML contract | [Codec](../../internal/workspace/config_test.go), [topics codec](../../internal/workspace/topics_test.go), [labels codec](../../internal/workspace/labels_test.go) | Types, strict rejection, aliases/merges, omission, and round trips |
| Files and target resolution | [Configuration files](../../internal/workspace/config_file_test.go), [repository resolution](../../internal/workspace/repository_test.go) | Fixed paths, file behavior, direct-origin formats, scope and isolation |
| GitHub transport | [Repository reads](../../internal/github/repository_test.go), [writes](../../internal/github/apply_test.go), [topics](../../internal/github/topics_test.go), [labels](../../internal/github/labels_test.go) | Complete/paginated reads, null/type normalization, exact declared-only requests, escaped label paths, failures and redirects |
| Application sequencing | [Application tests](../../internal/app/), [labels cases](../../internal/app/labels_test.go) | Validation/failure order, empty local no-ops, selective resource access, export composition, fresh labels reads, and partial writes |
| CLI and Actions presentation | [CLI tests](../../internal/cli/), [Actions tests](../../internal/actions/), [label reports](../../internal/actions/labels_test.go) | Dispatch, diagnostics/status, typed field/label reports, metadata, escaped rendering, and Summary storage |
| Composite and caller wiring | [Composite](../../tests/github-actions/composite_test.go), [consumer](../../tests/github-actions/consumer_test.go), [operational caller](../../tests/github-actions/operational_test.go) | Actual YAML composition, token exposure, output/status ordering, and caller-owned boundaries |
| Independent PR gates | [Message workflow tests](../../tests/github-actions/pr_message_test.go), [code-result tests](../../tests/github-actions/pr_code_gate_test.go) | Metadata/code event separation, required-message activation, failed/skipped/cancelled jobs, stale revisions, API failures, and status publication |
| Installed process behavior | [Acceptance tests](../../tests/acceptance/) | The actual supplied binary, isolated Git/files, local API requests, stdout/stderr and exit codes |
| Release metadata | [Metadata helper tests](../../scripts/release-metadata/main_test.go) | Temporary Git history, version precedence/preparation, fixed published boundaries, exact native-note requests, breaking-change descriptions, and retry identity |
| Release publication logic | [Release shell tests](../../tests/release.sh), [release gate tests](../../tests/github-actions/release_test.go), [local Action tests](../../tests/github-actions/release_actions_test.go) | Mock publication/recovery, canonical orchestration, manifested artifact identity/tamper rejection, stored-note preservation, and declared source/version/gate relationships |

[Feature specifications](../features/) map behavior to ordinary Go or shell tests. No separate BDD runner interprets those Markdown files. Keep detailed parser/type matrices at their owner and use representative acceptance cases to prove composition, rather than duplicating every low-level case.

Markdown checks use the pinned rumdl from [mise.toml](../../mise.toml) and [.rumdl.toml](../../.rumdl.toml). MD051 checks local heading fragments and MD057 checks relative file links; external HTTP availability is not a CI dependency.

Current-source [archive lifecycle model](../../internal/settings/archive_test.go), [API](../../internal/github/archive_test.go), [application](../../internal/app/archive_test.go), and [built-process acceptance](../../tests/acceptance/archive_test.go) tests verify typed state, target identity, narrow updates, ordered transitions, partial failures, and export opt-in. They use local fixtures only; live lifecycle permissions and recovery are not established.

## Local process acceptance

Build the product once, then supply that existing executable to the tagged acceptance suite:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The harness requires an explicit binary path and Unix-socket support. It uses go-gh's `http_unix_socket` configuration in an isolated temporary `GH_CONFIG_DIR`, temporary Git repositories, and a local HTTP fixture. Dummy tokens never authenticate with GitHub. Fixtures are cleaned up without adding product hooks or test-only transport options.

Process cases cover export → edit → validate → plan → apply, native false/empty values, read-side null normalization, checkout selection, declared-field comparison, exact requests, file preservation, diagnostics, and failure paths. Plan is read-only. Repository/topics apply has no preliminary GET; nonempty label selections read fresh collection state after those direct writes. Invalid input and empty CLI plan/apply make no API requests. Empty topics remain an authoritative managed set; empty labels select no objects. [Topics acceptance](../../tests/acceptance/topics_test.go) covers the topics resource through the built product.

### Merge response acceptance

[Merge transport tests](../../internal/github/merge_methods_test.go), [PR option tests](../../internal/github/pr_options_test.go), and [squash](../../internal/github/squash_commit_test.go)/[normal merge enum tests](../../internal/github/merge_commit_test.go) own response completeness: each required field must be present with the correct type and a supported enum value. Any GraphQL errors discard even complete data. [Observation tests](../../internal/github/observation_test.go) also reject missing fields that were not declared; expected responses use literal fixtures independent of production field definitions.

[Merge acceptance](../../tests/acceptance/merge_methods_test.go) retains representative permission, incomplete-response, and partial-data/error failures, including rejection of an earlier REST comparison with empty stdout and nonzero status. Its failed export preserves the existing destination. Detailed null/type/enum matrices remain at the adapter boundary; successful plan cases and complete declared-only PATCH cases remain at the process boundary.

### Labels resource acceptance

[Labels acceptance](../../tests/acceptance/labels_test.go) runs the current-source binary through empty selections, create-only collisions, ID-selected updates and renames, description defaults, unmanaged objects, invalid input, complete pagination and failed reads. Typed reports and a separate credential-free Summary expose complete supported selected objects, escaped names and ID identity, without implicit deletions. [ID writeback acceptance](../../tests/acceptance/label_ids_test.go) checks repeat apply after recording IDs, explicit opt-out, GitHub Actions suppression, concurrent local edits and partial remote creation failure with recoverable IDs.

[Application labels tests](../../internal/app/labels_test.go) separately own repository → topics → labels-read → labels-write sequencing, fresh comparison, already matching collections with no mutation, earlier-write preservation, and rejection of an earlier partial plan when a later labels read/comparison fails. [Adapter tests](../../internal/github/labels_test.go) own exact request payloads and escaped name segments; full validation/identity matrices stay in the model and codec tests.

These checks use local API responses and dummy credentials. They verify current-source behavior without establishing live label permissions or support in the published Action/CLI pair.

### Metadata, reports, and Summary

[Metadata acceptance](../../tests/acceptance/metadata_test.go) checks the supplied executable's actual version/digest and a separate committed consumer identity without settings, credentials, or API requests. Detailed origin syntax and isolation matrices remain in workspace tests. Mock metadata and owned test-executable adapters prove wiring or failure handling, rather than the installed product's identity.

The Linux-only [`TestMetadataUsesRunningImage`](../../internal/actions/metadata_linux_test.go) launches an owned copy of the test executable and replaces only that child's pathname before requesting metadata. The digest must identify the original running image; the parent test runner is not replaced. Native checks on other platforms cover an unchanged executable path, and cross-compilation alone does not run this Linux regression.

[Plan-report acceptance](../../tests/acceptance/plan_report_test.go) inspects native typed JSON and categorical failures through the real interface. Label operations occupy their own typed report entries and count as one managed collection, independently of the number of operations. [Rulesets acceptance](../../tests/acceptance/rulesets_test.go) verifies the built CLI, report and credential-free Summary expose complete supported ID-selected objects and default-derived changes while leaving unlisted objects untouched. [Rulesets ID acceptance](../../tests/acceptance/ruleset_ids_test.go) covers writeback/repeat apply, opt-out/Actions suppression and partial-creation recovery. [Summary acceptance](../../tests/acceptance/summary_test.go) checks escaped values, separate identity links, append behavior, and zero API requests. Detailed report/state/storage cases, including label reports and the maximum managed count, remain in Actions owner tests. Report/Summary output preserves a failed plan's nonzero status and never becomes apply input.

## CI validation

The [PR code](../../.github/workflows/ci-pr.yml) and [main](../../.github/workflows/ci-push.yml) workflows retain the product stages below. The separate [PRLint workflow](../../.github/workflows/prlint.yml) validates title/body edits without repeating product work. PR Draft/Ready transitions and approval automation are described in [repository automation](repository-automation.md#pr-lifecycle); [PR validation](pr-validation.md) documents the independent code/message checks and required-context boundary.

| Stage | Work and handoff |
| --- | --- |
| `pr-message` (PR only) | Validate the current title and optional body together as the squash message; run [policy regression cases](../../tests/github-actions/commitlint.sh) and reject invalid Conventional Commits |
| Commit stage | Formatting, modules, vet, race tests, workflow/shell checks, release/installer fixtures, Markdown/local links, and one product build |
| Acceptance stage (process) | Download the same run's manifested product, verify source SHA, target, binary name, digest and version, then execute the tagged process suite |
| Acceptance stage (public Action) | Verify the identical candidate on Ubuntu 24.04 x64 and exercise the published Action interface against a local fixture without rebuilding the product |
| `pass` | Require successful matrix resolution, commit, and all acceptance legs; PR message validation is required separately. Metadata-only edits do not publish a new code result; failed, cancelled, or skipped prerequisites cannot produce success |

The shared [CI commit composite](../../.github/actions/ci-commit/action.yml) calls the existing [commit-stage](../../.github/actions/commit-stage/action.yml) checks and packages one Linux amd64 candidate with a manifest containing the checked-out source SHA, target, binary name, binary SHA-256, and committed version. PR validation builds the merge source; push validation builds the main revision. Both workflows delegate target selection to [CI metadata](../../.github/actions/ci-metadata/action.yml).

Each acceptance matrix leg downloads the same run's artifact with digest mismatch treated as an error and checks out the exact producer source separately into `.action`. The [CI acceptance composite](../../.github/actions/ci-acceptance/action.yml) validates manifest identity against the checkout and bytes before choosing the process or public-Action acceptance path. The [process acceptance composite](../../.github/actions/acceptance-stage/action.yml) additionally verifies the embedded version and runs the tagged process suite with test caching disabled. No acceptance leg rebuilds the product.

All runners install required tools from pinned mise configuration. The public-Action leg uses Ubuntu 24.04 x64 and Go only to compile its local API fixture; commit and process checks use `ubuntu-slim`. Tool caches are disabled, artifacts are retained for one day, and CI candidate artifacts are not promoted to privileged release or settings workflows.

### Hosted candidate composite

The public-Action acceptance leg delegates its scenarios to the existing [candidate composite](../../.github/actions/candidate-smoke/action.yml), passing the verified source, version, digest, and binary path. It creates a separate consumer that tracks only `.github/settings.yml`, without a root `action.yml` or `action/` helper directory. The process leg instead runs the built executable through the local API acceptance tests.

[Fixture preparation](../../tests/github-actions/prepare-candidate-download.sh) verifies the transferred candidate's checksum and Action checkout's source/version. A bounded [curl stand-in](../../tests/github-actions/candidate-curl.sh) serves only the unchanged installer's two fixed versioned asset/checksum URLs from those verified bytes, without network forwarding. Actual `uses: ./.action` expansion exercises installation, checksum/version validation, metadata, and settings operations. No product rebuild, caller version/download override, committed version rewrite, or old-release fallback is used.

The fixture supplies current-source bytes instead of downloading a published artifact, so candidate results do not establish published-artifact identity. The [published package and consumer pins reference](../github-actions.md#published-package-and-consumer-pins) owns supported-version and adoption guidance. Label-specific current-source behavior is covered by the owner/process tests above; the hosted API cases below remain representative repository-field cases.

Credential-free validate must report all four outputs matching the independent consumer commit/origin and candidate version/checksum. Hosted logs record the distinct Action and consumer roots/SHAs. Missing or tampered artifacts, mismatched identities, and fixture/assertion failures fail the aggregate gate.

### Candidate API and Summary cases

Plan and direct apply use the same verified composite/binary with separate literal dummy read/write tokens. The [local API server](../../tests/github-actions/plan-fixture/main.go) requires Unix-socket transport, isolates GitHub CLI configuration, and records safe method/path observations. It accepts only the expected target/method/token combinations. Closed loopback HTTP/HTTPS proxies prevent ordinary HTTP transport from reaching an external service.

The [candidate checks](../../tests/github-actions/candidate-plan.sh) exercise these representative repository-field cases:

| Operation/case | API observation | Required result | Settings review |
| --- | --- | --- | --- |
| Plan changed string/false | One GET | Success and four matching outputs | Native false and escaped HTML/multiline diff |
| Plan local read403 | One GET | Failure and four empty outputs | Categorical read failure |
| Plan invalid boolean | Zero requests | Failure and four empty outputs | Categorical validation failure |
| Apply empty string/false | One PATCH, no GET | Success and four matching outputs | None added |
| Apply local write403 | One attempted PATCH, no GET | Failure and four empty outputs | None added |
| Apply invalid boolean | Zero requests | Failure and four empty outputs | None added |
| Apply empty `settings: {}` | Zero requests | Success and four matching outputs | None added |

Apply requests are independently checked as the complete received native JSON object `{"description":"","has_issues":false}`, with every other field omitted. The server records the decoded request instead of manufacturing expected payloads. A failed write attempt does not establish the remote outcome.

Every case preserves consumer configuration bytes and commit. The fixture must stay live even for zero-request cases, and an `always()` step stops/removes it. Expected failures use step-level `continue-on-error` only alongside mandatory assertions of the observed failure and empty outputs. Empty apply still meets the public composite's explicit-token and metadata requirements.

For plan, a finite before/after snapshot identifies exactly one new nonempty Settings review. Assertions inspect the bytes written by the embedded Go step, excluding old files and scrubbed upload copies; they do not substitute another renderer or override the Summary destination. All apply cases require zero new reviews. Hosted file rendering establishes neither GitHub Summary publication nor a live settings update.

### Composed callers and examples

Local [composite](../../tests/github-actions/composite_test.go) and [caller](../../tests/github-actions/consumer_test.go) fixtures use current candidate package bytes, explicit typed reports, mock downloads/CLI behavior, and owned test-executable adapters. The actual Go Summary adapter is reused without rebuilding the product. Checks cover token-free setup/metadata/Summary, explicit execution tokens, original plan-status precedence, and four outputs only after required work. The [PR data-preparation fixture](../../tests/github-actions/pr_candidate_example_test.go) executes the sample's actual inline script with real filtered local Git objects and dummy API transport: malicious executable/YAML content is not executed, symlinks/identity/size/read failures fail, and credentials are neither logged nor persisted. It does not establish live installation permissions, private-fork visibility, effective protection or token revocation.

Examples and active main callers use the same single-job apply pattern: three `uses` steps for checkout, token creation, and apply. [Operational tests](../../tests/github-actions/operational_test.go) separately verify this repository's Environment and credential bindings. Approval and token lifecycle stubs are simulations; these tests do not load pinned release-tag bytes or establish remote protection/revocation behavior.

## Release and publication evidence

The [release-metadata tests](../../scripts/release-metadata/main_test.go) run with the ordinary Go suite and pinned Go toolchain. They use temporary Git history and an injected API boundary to verify version precedence, preparation, publication-order selection, complete frozen ranges, retry identity, native-note request parameters, and retained breaking-change footers. The [publisher fixtures](../../tests/release.sh) check generation failures before remote state and preservation of existing notes and artifacts during retries. These tests neither publish a release nor claim to execute GitHub's native renderer.

The [release runbook](releases.md) owns dispatch and recovery. Its build matrix transfers each native product with a manifest; the acceptance matrix reuses the bounded public Action fixture with the frozen source, calculated/committed version, and manifested Linux product. Actual token-free validate must succeed before publication. Source/helper tests inspect orchestration and wrapper behavior without dispatching a release, and therefore are not hosted release evidence.

| Evidence | What it establishes | Limit |
| --- | --- | --- |
| Owner and composed tests | Local behavior, actual YAML wiring, and defined failure/status boundaries | Mocks do not establish platform enforcement or installed released bytes |
| Built-product acceptance | The supplied executable runs real entries against isolated files/Git and a local API | Dummy credentials/responses do not prove live GitHub behavior |
| Hosted candidate composite | Actual `uses` expansion, same-source/artifact integration, outputs, Summary bytes and representative direct PATCH behavior | Candidate fixture downloads are not published assets |
| Release acceptance gate | An authorized release run passed both native targets and actual candidate validate using the manifested products | Source/fixture tests alone do not prove that a release ran |
| Post-publication verification | The published tag and actual downloaded assets match both accepted manifests, and the published root Action reports all four identities | Failure after publication does not permit changing immutable tags/assets |

Inspect the relevant workflow run for hosted or publication evidence. A successful local fixture, Summary file, or workflow source check cannot establish actual Environment approval, branch policy, token revocation, or GitHub Summary publication.
