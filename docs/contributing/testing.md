# Testing and CI

Tests verify current product and workflow contracts at their owning boundaries. Use [development](development.md#development-checks) for tool setup and commands, [architecture](architecture.md) for ownership, and [repository automation](repository-automation.md) for PR checks and operational workflows.

## Test ownership

| Concern | Owning checks | Scope |
| --- | --- | --- |
| Declaration presence and comparison | [Repository model](../../internal/settings/repository_test.go), [comparison](../../internal/settings/comparison_test.go), [topics](../../internal/settings/topics_test.go), [labels](../../internal/settings/labels_test.go) | Omitted vs explicit values, managed counts, native values, topic sets, label identity/attribute presence, and stable operation ordering |
| YAML contract | [UI config](../../internal/workspace/ui_config_test.go), [YAML safety](../../internal/workspace/ui_yaml_safety_test.go), [Labels UI codec](../../internal/workspace/ui_labels_test.go) | Types, strict rejection, aliases/merges, omission, and round trips |
| Files and target resolution | [Configuration files](../../internal/workspace/config_file_test.go), [repository resolution](../../internal/workspace/repository_test.go) | Fixed paths, file behavior, direct-origin formats, scope and isolation |
| GitHub transport | [Repository reads](../../internal/github/repository_test.go), [writes](../../internal/github/apply_test.go), [topics](../../internal/github/topics_test.go), [labels](../../internal/github/labels_test.go) | Complete/paginated reads, null/type normalization, exact declared-only requests, escaped label paths, failures and redirects |
| Application sequencing | [Application tests](../../internal/app/), [labels cases](../../internal/app/labels_test.go) | Validation/failure order, empty local no-ops, selective resource access, export composition, fresh labels reads, and partial writes |
| CLI and Actions presentation | [CLI tests](../../internal/cli/), [Actions tests](../../internal/actions/), [label reports](../../internal/actions/labels_test.go) | Dispatch, diagnostics/status, typed field/label reports, metadata, escaped rendering, and Summary storage |
| Composite and caller wiring | [Composite](../../tests/github-actions/composite_test.go), [consumer](../../tests/github-actions/consumer_test.go), [operational caller](../../tests/github-actions/operational_test.go) | Actual YAML composition, token exposure, output/status ordering, and caller-owned boundaries |
| Independent PR gates | [Message workflow tests](../../tests/github-actions/pr_message_test.go), [screening workflow tests](../../tests/github-actions/pr_code_gate_test.go) | Default PR activities, metadata/code event separation, direct screening steps, read-only execution, and independent message validation |
| Installed process behavior | [Acceptance tests](../../tests/acceptance/) | The actual supplied binary, isolated Git/files, local API requests, stdout/stderr and exit codes |
| Version calculation | [Python tests](../../.github/scripts/test_release_version.py) | Temporary Git history, SemVer precedence, fixed published boundaries, invalid catalogs, and immutable retry selection |
| Release notes | [Go metadata tests](../../scripts/release-metadata/main_test.go) | Committed category precedence, formal footers, escaping, fixed Git ranges, and changed-baseline failures |
| Release publication logic | [Release shell tests](../../tests/release.sh), [release gate tests](../../tests/github-actions/release_test.go), [local Action tests](../../tests/github-actions/release_actions_test.go) | Mock publication/recovery, canonical orchestration, manifested artifact identity/tamper rejection, stored-note preservation, and declared source/version/gate relationships |

[Actions event policy scenarios](../features/actions-event-policies.feature.md)
map strict codec, domain reconciliation, complete API acquisition/mutation,
application sequencing, typed reports and snapshot-ID recording to their owners.
[Process acceptance](../../tests/acceptance/actions_event_policies_test.go)
checks the built CLI's independent export selection, create/update/repeat apply,
partial ID recovery and destination preservation. The [actor-policy process
fixture](../../tests/acceptance/actions_actor_policies_test.go) also checks actor-only
creation, combined policies, omitted-rule/target preservation, checked-empty versus
explicit removal, complete private reports, status-only privacy, unsupported actor
isolation and recoverable creation failure. These local fixtures use dummy
credentials and cover active/disabled metadata transitions; actual enforcement
and workflow execution were not tested.

[Feature specifications](../features/) map behavior to ordinary Go or shell tests. No separate BDD runner interprets those Markdown files. Keep detailed parser/type matrices at their owner and use representative acceptance cases to prove composition, rather than duplicating every low-level case.

Markdown checks use the pinned rumdl from [mise.toml](../../mise.toml) and [.rumdl.toml](../../.rumdl.toml). MD051 checks local heading fragments and MD057 checks relative file links; external HTTP availability is not a CI dependency.

## Internal UI repository preparation

The [prepared About/General/Labels mapping](ui-repository-mapping.md) was prepared in #482 and is now consumed by the current UI codec/root composition. These tests independently specify mapping behavior without invoking live writes:

| Test owner | Independent expected behavior |
| --- | --- |
| [About](../../internal/github/ui_about_test.go) | Omission vs empty strings/list; Website to `homepage`; raw null-string/topic reads; exact empty PATCH/PUT payloads; owned values and invalid topics |
| [General scalars](../../internal/github/ui_general_test.go) | Singleton mappings for each true/false control and branch; reverse projection; invalid branch; exact mixed REST payload; complete raw GraphQL observation |
| [Commit menus](../../internal/github/ui_commit_message_test.go) | Seven literal UI/API pairs and exact payloads; absent vs incomplete pairs; API-only choices and invalid/disabled declarations |
| [Creation policies](../../internal/github/ui_creation_policy_test.go) | Both policies and choices; unknown tokens; exact GraphQL input with native access/sponsorship booleans and verified target identity |
| [Labels](../../internal/github/ui_labels_test.go) | UI color prefix; complete raw list conversion; omission/empty; missing observation; selected-ID rename leaving unlisted objects alone; exact update/create bodies and ID validation |

[Conditional desired-policy fixtures](../../internal/github/ui_creation_policy_visibility_test.go) reject explicitly disabled feature/policy combinations while retaining omitted/true gates. [Conditional observation fixtures](../../internal/github/ui_general_conditional_observation_test.go) exercise independent mixed gates for both feature policies and merge menus, six owned raw values, absence and strict validation before hiding. Exact composed GraphQL tests retain visible policy intent and an independent false Sponsorships control.

The new fixtures use literal expectations rather than production mapping definitions. Existing repository, merge-method, access-policy and label owner tests continue to cover complete/malformed/denied reads, pagination, identity checks, redirects and partial failures. Existing archive tests retain lifecycle guards and sequencing. Current codec/root/process tests below separately exercise accepted UI YAML activation in #479. These mapping unit fixtures alone do not establish public composition. Normalization/warning and presentation coverage is described below.

## Internal UI Ruleset preparation

The [Ruleset mapping](ui-ruleset-mapping.md) records the supported UI controls and complete-acquisition boundaries. Current UI activation reuses that mapping.

| Fixture owner | Independent contract checked |
| --- | --- |
| [Root](../../internal/github/ui_ruleset_root_test.go) | Required enforcement/one target, sentinels, short/full-ref patterns, empty criteria, blank rejection, ownership and out-of-UI values |
| [Bypass](../../internal/github/ui_ruleset_bypass_test.go) | Three verified role IDs, dynamic actor IDs, three UI modes, marker validation and target restrictions |
| [Simple rules](../../internal/github/ui_ruleset_rules_test.go) | Four singleton false/true checkbox mappings and complete observed absence |
| [PR settings](../../internal/github/ui_ruleset_pr_test.go) | Singleton flags, finite count choices, merge methods and complete read projection |
| [Checks](../../internal/github/ui_ruleset_checks_test.go) | Singleton flags, empty collections, Any source vs App identity, raw null source and owned values |
| [Analysis](../../internal/github/ui_ruleset_analysis_test.go) | All 20 scanning threshold combinations, four quality choices and required desired values |
| [Collection/transport](../../internal/github/ui_ruleset_collection_test.go) | Acquired IDs, sorting, no partial projection, raw eight-rule reads, exact no-ID POST/selected-ID PUT and independent readback |

[Unsupported Ruleset acceptance](../../tests/acceptance/ruleset_unsupported_controls_test.go) rejects removed declarations before remote work, including false/empty values, and verifies that unsupported observed enforcement, rule types and review-dismissal parameters block export, update and explicit deletion without writes or configuration replacement. These local cases preserve supported PR behavior and do not change live protection.

Expected payloads and choices are literal fixtures, independent of production maps. Existing Ruleset owner tests retain denied/malformed/incomplete reads, pagination, inherited filtering and unsupported Push rejection, identity/target failures and partial-write coverage. Local conversion/transport tests do not establish live policy changes, effective enforcement, new YAML acceptance or normalization warnings.

Current-source [archive lifecycle model](../../internal/settings/archive_test.go), [API](../../internal/github/archive_test.go), [application](../../internal/app/archive_test.go), and [built-process acceptance](../../tests/acceptance/archive_test.go) tests verify typed state, target identity, narrow updates, ordered transitions, partial failures, and automatic export. They use local fixtures only; live lifecycle permissions and recovery are not established.

## Local process acceptance

Build the product once, then supply that existing executable to the tagged acceptance suite:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The harness requires an explicit binary path and Unix-socket support. It uses go-gh's `http_unix_socket` configuration in an isolated temporary `GH_CONFIG_DIR`, temporary Git repositories, and a local HTTP fixture. Dummy tokens never authenticate with GitHub. Fixtures are cleaned up without adding product hooks or test-only transport options.

Process cases cover export → edit → validate → plan → apply, native false/empty values, read-side null normalization, checkout selection, declared-field comparison, exact requests, file preservation, diagnostics, and failure paths. Plan is read-only. Repository REST apply reads fresh state, skips matches and independently verifies changed intent. Topics apply reads complete membership, skips matching sets, and verifies changed sets through an independent GET; nonempty label selections read fresh collection state afterward. Invalid input and empty CLI plan/apply make no API requests. Empty topics remain an authoritative managed set; empty labels select no objects. [Topics acceptance](../../tests/acceptance/topics_test.go) covers matching/reordered sets, explicit clearing, repeated apply, invalid or unavailable reads, rejected PUT, denied/mismatching readback, later-resource stopping and earlier repository progress through the built product. [Topics adapter tests](../../internal/github/topics_test.go) additionally verify complete response validation and redirects. [Topics application tests](../../internal/app/topics_test.go) distinguish actual writes, unchanged work, prewrite failures and unarchive progress, and preserve later Labels/final archive sequencing.

### Merge response acceptance

[Merge transport tests](../../internal/github/merge_methods_test.go), [PR option tests](../../internal/github/pr_options_test.go), and [squash](../../internal/github/squash_commit_test.go)/[normal merge enum tests](../../internal/github/merge_commit_test.go) own response completeness: each required field must be present with the correct type and a supported enum value. Any GraphQL errors discard even complete data. [Observation tests](../../internal/github/observation_test.go) also reject missing fields that were not declared; expected responses use literal fixtures independent of production field definitions.

[Merge acceptance](../../tests/acceptance/merge_methods_test.go) retains representative permission, incomplete-response, and partial-data/error failures, including rejection of an earlier REST comparison with empty stdout and nonzero status. For export, unavailable merge settings emit bounded info and retain compatible prior intent while readable siblings are saved. The [About/General export regressions](../../tests/acceptance/repository_export_isolation_test.go) exercise partial acquisition and local validation. Detailed null/type/enum matrices remain at the adapter boundary; successful plan cases and complete declared-only PATCH cases remain at the process boundary.

### Labels resource acceptance

[Labels acceptance](../../tests/acceptance/labels_test.go) runs the current-source binary through empty selections, create-only collisions, ID-selected updates and renames, description defaults, unmanaged objects, invalid input, complete pagination and failed reads. Typed reports and a separate credential-free Summary expose complete supported selected objects, escaped names and ID identity, without implicit deletions. [ID writeback acceptance](../../tests/acceptance/label_ids_test.go) checks repeat apply after recording IDs, explicit opt-out, GitHub Actions suppression, concurrent local edits and partial remote creation failure with recoverable IDs.

[Application labels tests](../../internal/app/labels_test.go) separately own repository → topics → labels-read → labels-write sequencing, fresh comparison, already matching collections with no mutation, earlier-write preservation, and rejection of an earlier partial plan when a later labels read/comparison fails. [Adapter tests](../../internal/github/labels_test.go) own exact request payloads and escaped name segments; full validation/identity matrices stay in the model and codec tests.

These checks use local API responses and dummy credentials. They verify current-source behavior without establishing live label permissions or support in the published Action/CLI pair.

### Metadata, reports, and Summary

[Metadata acceptance](../../tests/acceptance/metadata_test.go) checks the supplied executable's actual version/digest and a separate committed consumer identity without settings, credentials, or API requests. Detailed origin syntax and isolation matrices remain in workspace tests. Mock metadata and owned test-executable adapters prove wiring or failure handling, rather than the installed product's identity.

The Linux-only [`TestMetadataUsesRunningImage`](../../internal/actions/metadata_linux_test.go) launches an owned copy of the test executable and replaces only that child's pathname before requesting metadata. The digest must identify the original running image; the parent test runner is not replaced. Native checks on other platforms cover an unchanged executable path, and cross-compilation alone does not run this Linux regression.

[Plan-report acceptance](../../tests/acceptance/plan_report_test.go) inspects native typed JSON and categorical failures through the real interface. Label operations occupy their own typed report entries and count as one managed collection, independently of the number of operations. [Rulesets acceptance](../../tests/acceptance/rulesets_test.go) verifies the built CLI, report and credential-free Summary expose complete supported ID-selected objects and default-derived changes while leaving unlisted objects untouched. [Rulesets ID acceptance](../../tests/acceptance/ruleset_ids_test.go) covers writeback/repeat apply, opt-out/Actions suppression and partial-creation recovery. [Summary acceptance](../../tests/acceptance/summary_test.go) checks escaped values, separate identity links, append behavior, and zero API requests. Detailed report/state/storage cases, including label reports and the maximum managed count, remain in Actions owner tests. Report/Summary output preserves a failed plan's nonzero status and never becomes apply input.

## CI validation

The [PR workflow](../../.github/workflows/pr.yml), named `pr`, runs Screening
only: source lint, Go race and offline tests, and a native Linux build.
Its single `screening` job uses `ubuntu-slim` and reports its result directly.
Default `pull_request` activities run it when a PR is opened, updated with
commits, or reopened. Title/body edits and draft/readiness transitions do not
rerun it, and the workflow leaves the PR state unchanged.
The separate [PRLint workflow](../../.github/workflows/prlint.yml) checks
title/body edits and proposed squash messages.

The [main workflow](../../.github/workflows/ci-push.yml) runs full Screening
and Inspection. It resolves the release version for the exact `main` SHA,
builds Linux amd64 and macOS arm64 natively using the shared
[build](../../.github/actions/build/action.yml) Action, then packages and
uploads each product in separate workflow steps. It tests those same
binaries and the public Action through
[release-acceptance](../../.github/actions/release-acceptance/action.yml).
Each artifact has a source/version/target/SHA-256 manifest and is retained
for seven days for use by the manual release workflow.

Both main and release build jobs install the pinned `mise.toml` toolchain
with `mise-action` cache enabled. They use one setup step on Linux and macOS,
and the same build Action. PR Screening retains `setup-go` for Go compilation
and module caching and installs only its non-Go lint tools with mise.

After checkout, PR Screening installs the pinned tools, including Go, with
`mise-action`, then lets `actions/setup-go` cache the existing Go toolchain's
module downloads and compilation results. It then calls
[lint](../../.github/actions/lint/action.yml),
[unit-test](../../.github/actions/unit-test/action.yml), and
[build](../../.github/actions/build/action.yml) in separate steps. A failed lint, unit-test, or build step fails
`screening`. PR CI does not upload artifacts or run process/public-Action
Inspection. The Go cache restores compilation and module data, never binary
artifacts; a cache hit does not skip source checks.

PR Screening uses no inert Action inventory. Its `mise-action` and
`actions/setup-go` steps supply the external Action pins in
[actions.lock](../../.github/workflows/actions.lock).
The [full CI/release inventory](../../.github/actions/lock-workspace-dependencies/action.yml)
remains for the main and release workflows.

### Go cache in CI

PR Screening and native build jobs in `ci-push` and `release` use the same
two-step setup. `mise-action` installs tools declared in
[mise.toml](../../mise.toml), including Go, using its installation cache.
Then `actions/setup-go` finds that Go on PATH without specifying a version or
version file, and caches Go module downloads and compilation results using
[go.sum](../../go.sum). Go is not installed a second time.

No custom cache key, environment fingerprint or restore/save Action is needed.
A cache hit never skips module validation, `go vet`, race-detected tests or
the native build. Release artifacts and manifests are not cached as Go
build outputs, and these caches must not contain secrets.

### Hosted candidate composite

The public-Action acceptance leg delegates its scenarios to the existing [candidate composite](../../.github/actions/e2e-test/action.yml), passing the verified source, version, digest, and binary path. It creates a separate consumer that tracks only `.github/settings.yml`, without a root `action.yml` or `action/` helper directory. The process leg instead runs the built executable through the local API acceptance tests.

[Fixture preparation](../../tests/github-actions/prepare-candidate-download.sh) verifies the transferred candidate's checksum, executable-reported version, and Action checkout source. A bounded [curl stand-in](../../tests/github-actions/candidate-curl.sh) serves only the unchanged installer's two fixed versioned asset/checksum URLs from those verified bytes, without network forwarding. Actual `uses: ./.action` expansion exercises installation, checksum/version validation, metadata, and settings operations. No product rebuild, public caller version/download override, committed version rewrite, or old-release fallback is used.

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
| Apply empty string/false | GET, one PATCH, independent GET | Success and four matching outputs | None added |
| Apply local write403 | GET, one attempted PATCH | Failure and four empty outputs | None added |
| Apply invalid boolean | Zero requests | Failure and four empty outputs | None added |
| Apply empty `settings: {}` | Zero requests | Success and four matching outputs | None added |

Apply requests are independently checked as the complete received native JSON object `{"description":"","has_issues":false}`, with every other field omitted. The server records the decoded request instead of manufacturing expected payloads. A failed write attempt does not establish the remote outcome.

Every case preserves consumer configuration bytes and commit. The fixture must stay live even for zero-request cases, and an `always()` step stops/removes it. Expected failures use step-level `continue-on-error` only alongside mandatory assertions of the observed failure and empty outputs. Empty apply still meets the public composite's explicit-token and metadata requirements.

For plan, a finite before/after snapshot identifies exactly one new nonempty Settings review. Assertions inspect the bytes written by the embedded Go step, excluding old files and scrubbed upload copies; they do not substitute another renderer or override the Summary destination. All apply cases require zero new reviews. Hosted file rendering establishes neither GitHub Summary publication nor a live settings update.

### Composed callers and examples

Local [composite](../../tests/github-actions/composite_test.go) and [caller](../../tests/github-actions/consumer_test.go) fixtures use current candidate package bytes, explicit typed reports, mock downloads/CLI behavior, and owned test-executable adapters. The actual Go Summary adapter is reused without rebuilding the product. Checks cover token-free setup/metadata/Summary, explicit execution tokens, original plan-status precedence, and four outputs only after required work. The [PR data-preparation fixture](../../tests/github-actions/pr_candidate_example_test.go) executes the sample's actual inline script with real filtered local Git objects and dummy API transport: malicious executable/YAML content is not executed, symlinks/identity/size/read failures fail, and credentials are neither logged nor persisted. It does not establish live installation permissions, private-fork visibility, effective protection or token revocation.

Examples and active main callers use the same single-job apply pattern: three `uses` steps for checkout, token creation, and apply. [Operational tests](../../tests/github-actions/operational_test.go) separately verify this repository's Environment and credential bindings. Approval and token lifecycle stubs are simulations; these tests do not load pinned release-tag bytes or establish remote protection/revocation behavior.

## Release and publication evidence

The [shared Python tests](../../.github/scripts/test_release_version.py) exercise version precedence, publication-order selection, complete frozen Git ranges and retry identity. The shared [unit-test Action](../../.github/actions/unit-test/action.yml) runs them in both PR Screening and main CI, alongside the release CI Python tests and offline recovery fixtures. There is no dedicated Python-test workflow. Run them locally when changing the resolver or its tests:

```sh
python3 -B -m unittest discover -s .github/scripts -p 'test_release_version.py' -q
```

Python test commands disable bytecode generation so imports do not leave untracked
`__pycache__` files before the native build. The
[clean-build regression](../../tests/github-actions/unit_test_action_test.go)
executes both unittest commands with importing fixtures and verifies the compiled
binary's revision and `vcs.modified=false` metadata.

The [Go metadata tests](../../scripts/release-metadata/main_test.go) cover categories and formal compatibility markers from committed messages, escaped text, and fixed Git boundaries. The [publisher fixtures](../../tests/release.sh) compare the complete generated body from fixture commits, reject PR metadata and native note-generation requests, and check generation failures before remote state and preservation of existing notes and artifacts during retries. These tests neither publish a release nor establish live GitHub rendering.

The [release runbook](releases.md) owns dispatch and recovery. Its build matrix transfers each native product with a manifest; the acceptance matrix reuses the bounded public Action fixture with the frozen source, calculated release version, and manifested Linux product. Actual token-free validate must succeed before publication. Source/helper tests inspect orchestration and wrapper behavior without dispatching a release, and therefore are not hosted release evidence.

| Evidence | What it establishes | Limit |
| --- | --- | --- |
| Owner and composed tests | Local behavior, actual YAML wiring, and defined failure/status boundaries | Mocks do not establish platform enforcement or installed released bytes |
| Built-product acceptance | The supplied executable runs real entries against isolated files/Git and a local API | Dummy credentials/responses do not prove live GitHub behavior |
| Hosted candidate composite | Actual `uses` expansion, same-source/artifact integration, outputs, Summary bytes and representative direct PATCH behavior | Candidate fixture downloads are not published assets |
| Release acceptance gate | An authorized release run passed both native targets and actual candidate validate using the manifested products | Source/fixture tests alone do not prove that a release ran |
| Post-publication verification | The published tag and actual downloaded assets match both accepted manifests, and the published root Action reports all four identities | Failure after publication does not permit changing immutable tags/assets |

Inspect the relevant workflow run for hosted or publication evidence. A successful local fixture, Summary file, or workflow source check cannot establish actual Environment approval, branch policy, token revocation, or GitHub Summary publication.

## Prepared Actions UI mapping

[Actions mapping fixtures](../../internal/github/ui_actions_composition_test.go) connect literal UI declarations to existing exact request/readback behavior. The [mapping reference](ui-actions-mapping.md) links the owning enum, singleton, acquisition, retention and fork tests. Current public codec/process tests now cover UI activation with independently expected native payloads.

## Prepared Security UI mapping

[Security mapping fixtures](../../internal/github/ui_security_composition_test.go) connect literal UI booleans with existing acquired raw status, narrow PATCH and readback behavior. The [mapping reference](ui-security-mapping.md) identifies reporting and immutability endpoint fixtures, presence, dependency, applicability and owner-known metadata checks. Current codec/root/process tests now cover public UI activation; complete UI reporting remains separate.

## Prepared Pages UI mapping

[Pages mapping fixtures](../../internal/github/ui_pages_composition_test.go) connect literal UI declarations, acquired raw state and exact existing-site updates/readback. The [mapping reference](ui-pages-mapping.md) identifies source/folder, nullable-domain, ownership and metadata checks. Current public/process and lifecycle-limit coverage remains with the existing Pages tests.

## Prepared Environment UI mapping

[Environment mapping fixtures](../../internal/github/ui_environments_composition_test.go) connect literal UI declarations to complete raw reads, exact name-addressed PUT and separately owned typed pattern CRUD. The [mapping reference](ui-environments-mapping.md) links builtin, principal, finite-choice, acquisition/metadata and API-controlled omission tests. Current UI schema/process tests exercise the mapping while preserving API-controlled omissions and raw private report semantics.

## Current UI YAML activation

| Owner | Contract exercised |
| --- | --- |
| [Current codec](../../internal/workspace/ui_config_test.go), [strict YAML safety](../../internal/workspace/ui_yaml_safety_test.go) and family `ui_*_test.go` files | Exact marker, obsolete/API aliases rejected, independent literal inventories, null/false/zero/empty/omitted values, aliases/merges and zero partial decoding |
| [Desired composition](../../internal/github/ui_declaration_test.go) | Six-family reuse, independent native expectations, semantic failure returns zero whole declaration |
| [Export composition](../../internal/github/ui_export_test.go) | Actual acquired data only, pure policy projections, hidden/raw distinctions, no metadata replay, stable Labels and no partial UI output |
| [Root composition](../../checkout_config_test.go) | Deferred bound saver before reads, UI/native conversion, current-schema snapshot ID recording, no-op documents and failure-before-recorder behavior |
| [Ruleset HTTP fixture](../../internal/github/ui_config_rulesets_test.go) | Literal UI YAML through decode/conversion/reconcile to exact independently authored POST/PUT and ID-bound paths |
| [Process suite](../../tests/acceptance/) | The supplied binary exports/validates/plans/applies every adopted family against local API fixtures; independent UI baselines and exact raw payloads preserve ownership/failure semantics |
| [Candidate Action fixture](../../tests/github-actions/candidate-plan.sh) and [composite tests](../../tests/github-actions/composite_test.go) | Current literal UI consumer configs with unchanged installer/token/identity/escaping boundaries; actual hosted `uses` expansion remains separate evidence |

Superseded workspace parser/type files were removed after their safety/presence cases moved to the current codec owners. They are not a compatibility test suite or alternate parser. Raw domain, transport, snapshot/file/Git and private report tests remain at their existing owners. The reporting work tracked in #481 was delivered by [PR #538](https://github.com/daiksud/gh-settings/pull/538); pinned released self-configuration and immutable published assets retain their own contracts.

## Environment ref-type observations

[Reader tests](../../internal/github/environment_ref_type_read_test.go) own complete pagination, raw identity retention and genuine projected collisions. [Update tests](../../internal/github/environment_ref_type_update_test.go) and [strict creation tests](../../internal/github/environment_ref_type_update_failure_test.go) exercise declared intent, canonical companion IDs, omitted/empty lists, raw deletion confirmation and independent final readback. Detailed malformed/identity matrices stay at their decoder, reconciliation and deletion owners.

[Executable export](../../tests/acceptance/normalization_environment_ref_type_test.go) verifies exact diagnostics, literal UI policy, latent privacy and destination preservation for collisions or incomplete acquisition. [Executable command composition](../../tests/acceptance/normalization_environment_ref_type_commands_test.go) verifies fallback-equal write intent, private report evidence, detailed/status-only Summary, literal ordered mutations, canonical repetition and honest final-readback failure. All use isolated local APIs and dummy credentials; none establishes live mutation or a published artifact's behavior.

### Ruleset bypass-mode normalization

[Mapping](../../internal/settings/ruleset_bypass_mapping_test.go), [owned state](../../internal/settings/ruleset_bypass_observation_test.go), [closed changes](../../internal/settings/ruleset_bypass_change_test.go) and [reconciliation](../../internal/settings/ruleset_bypass_reconcile_test.go) cover exact actor identity, nil IDs, multiple raw records, fallback equality, removal and independent scalar evidence. The [pure projector](../../internal/github/normalization_ruleset_bypass_test.go) and [complete reader](../../internal/github/ruleset_bypass_read_test.go) reject incomplete/malformed/ambiguous acquisition without partial output. Original acknowledgment and fresh GET use [strict confirmation](../../internal/github/ruleset_bypass_verification_test.go); [pre-write](../../internal/github/ruleset_bypass_warning_test.go) and [fresh](../../internal/github/ruleset_bypass_readback_warning_test.go) diagnostics preserve context without supplying successful verification values.

[Export](../../internal/app/normalization_ruleset_bypass_export_test.go), [presentation](../../internal/presentation/ruleset_bypass_test.go) and [private reports](../../internal/actions/normalization_ruleset_bypass_test.go) verify ordering, owned callbacks, ordinary silence, malformed/forged metadata, removal and privacy. The [built-executable sequence](../../tests/acceptance/normalization_ruleset_bypass_test.go) fails against the preserved pre-feature binary and verifies current export, plan, reports, Summary, literal updates, ignored raw drift, canonical repetition, actual removal, unlisted-ID privacy and file preservation. These local fixtures use dummy credentials and establish no live grants, protection changes or release behavior.

### Ruleset Allowed merge methods normalization

[Mapping](../../internal/settings/ruleset_merge_methods_mapping_test.go), [owned state](../../internal/settings/ruleset_merge_methods_observation_test.go), [closed changes](../../internal/settings/ruleset_merge_methods_change_test.go) and [reconciliation](../../internal/settings/ruleset_merge_methods_reconcile_test.go) cover typed raw lists, deep ownership, invalid evidence beside valid companions, fallback equality, declaration defaults and PR-rule removal. The [pure projector](../../internal/github/normalization_ruleset_merge_methods_test.go) rejects null elements and duplicates before whole-list projection; the [complete reader](../../internal/github/ruleset_merge_methods_read_test.go) requires complete policy and pagination with no partial observation. [Strict confirmation](../../internal/github/ruleset_merge_methods_verification_test.go) and [fresh diagnostics](../../internal/github/ruleset_merge_methods_readback_warning_test.go) keep raw verification independent from warnings.

[Export](../../internal/app/normalization_ruleset_merge_methods_export_test.go), [presentation](../../internal/presentation/ruleset_merge_methods_test.go) and [private reports](../../internal/actions/normalization_ruleset_merge_methods_test.go) verify scoped owned callbacks, raw-order/named-choice display, typed closed lists, removal and privacy. The [built-executable sequence](../../tests/acceptance/normalization_ruleset_merge_methods_test.go) fails against the preserved pre-feature binary and passes export, plan, report/Summary, literal updates, rejected raw drift, canonical repetition, actual removal, unlisted privacy and destination-preserving errors against the candidate. All use isolated local APIs and dummy credentials; they establish no live settings change or published artifact behavior.

### Ruleset scanning Alerts normalization

[Domain tests](../../internal/settings/ruleset_scanning_alerts_observation_test.go) cover shared UI mapping, exact tool identity, owned raw/effective state, invalid evidence, closed changes, selected reconciliation and removal/replacement. [Reader tests](../../internal/github/ruleset_scanning_alerts_read_test.go) cover the pure projector, complete acquisition, all-or-zero failure and independent companion evidence. [Strict confirmation tests](../../internal/github/ruleset_scanning_alerts_verification_test.go) verify original acknowledgment and fresh GET, scoped warnings and honest uncertain effects.

[Report/presentation tests](../../internal/actions/normalization_ruleset_scanning_alerts_test.go) exercise empty raw strings, closed metadata, exact ordering, escaping, ownership and status-only privacy; [export tests](../../internal/app/normalization_ruleset_scanning_alerts_export_test.go) cover the independently optional sink and acquisition/save boundaries. The [built-executable regression](../../tests/acceptance/normalization_ruleset_scanning_alerts_test.go) fails against the preserved pre-feature binary and checks current export/plan/report/Summary/apply, literal payloads, raw drift, canonical repetition and destination preservation. No test changes live settings or establishes a published artifact's behavior.

### Ruleset scanning Security alerts normalization

[Domain tests](../../internal/settings/ruleset_scanning_security_alerts_observation_test.go) cover the five-choice mapping, named fallback, exact identities, owned raw values, independent companion records, closed changes and selected intent/removal. [Reader tests](../../internal/github/ruleset_scanning_security_alerts_read_test.go) cover complete typed acquisition, simultaneous unknown thresholds and all-or-zero failure. [Strict confirmation tests](../../internal/github/ruleset_scanning_security_alerts_verification_test.go) retain independent original acknowledgment/fresh GET and contextual warnings.

[Report/presentation tests](../../internal/actions/normalization_ruleset_scanning_security_alerts_test.go) validate separate records together, explicit empty strings, ownership, escaping and privacy. [Export tests](../../internal/app/normalization_ruleset_scanning_security_alerts_export_test.go) cover independent sinks and save boundaries. The [built-executable regression](../../tests/acceptance/normalization_ruleset_scanning_security_alerts_test.go) fails against the pre-feature binary and verifies current commands, exact mutations, strict failure, no-op repetition and both thresholds on one tool. Local fixture evidence does not establish live protection, hosted CI or a released artifact's behavior.

## Cross-workflow release gate tests

The offline [release CI tests](../../.github/scripts/test_release_ci.py) cover exact-source main-run selection, latest-run failure and cancellation blocking, artifact provenance, missing/expired artifacts, archive SHA-256 mismatch, version mismatch, binary corruption and fallback. The [release orchestration tests](../../tests/github-actions/release_test.go) check the job dependency and publication boundaries; the Python tests exercise successful reuse, corrupted artifacts, failed CI and fallback outcomes. The tests do not dispatch or publish a hosted release; manual production release verification remains separately authorized.

## Actions cache and sharing controls

The [built-product fixture](../../tests/acceptance/actions_general_controls_test.go) verifies the independent cache-retention, cache-storage and private-sharing export options through actual CLI processes. It checks exact one-member REST writes, fresh readback, omission and existing-flag isolation, matching repeated apply, UI plan/report/Summary and destination preservation on unavailable reads. The [behavior specification](../features/actions-general-controls.feature.md) records the supported bounds, permissions and failure contract. These local fixtures establish neither live mutation nor actual installation permissions or billing behavior.
