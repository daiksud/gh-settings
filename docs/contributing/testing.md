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
| Installed process behavior | [Acceptance tests](../../tests/acceptance/) | The actual supplied binary, isolated Git/files, local API requests, stdout/stderr and exit codes |
| Release notes | [Release-notes helper tests](../../scripts/release-notes/main_test.go) | Temporary Git history, published predecessor selection, exact frozen ranges, classification, and Markdown rendering |
| Release publication logic | [Release shell tests](../../tests/release.sh), [release gate tests](../../tests/github-actions/release_test.go) | Mock publication/recovery, note generation before remote changes, stored-note preservation, and declared source/version/artifact/gate relationships |

[Feature specifications](../features/) map behavior to ordinary Go or shell tests. No separate BDD runner interprets those Markdown files. Keep detailed parser/type matrices at their owner and use representative acceptance cases to prove composition, rather than duplicating every low-level case.

Markdown checks use the pinned rumdl from [mise.toml](../../mise.toml) and [.rumdl.toml](../../.rumdl.toml). MD051 checks local heading fragments and MD057 checks relative file links; external HTTP availability is not a CI dependency.

## Local process acceptance

Build the product once, then supply that existing executable to the tagged acceptance suite:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The harness requires an explicit binary path and Unix-socket support. It uses go-gh's `http_unix_socket` configuration in an isolated temporary `GH_CONFIG_DIR`, temporary Git repositories, and a local HTTP fixture. Dummy tokens never authenticate with GitHub. Fixtures are cleaned up without adding product hooks or test-only transport options.

Process cases cover export → edit → validate → plan → apply, native false/empty values, read-side null normalization, checkout selection, declared-field comparison, exact requests, file preservation, diagnostics, and failure paths. Plan is read-only. Repository/topics apply has no preliminary GET; labels apply reads fresh collection state after those direct writes. Invalid input and empty CLI plan/apply make no API requests. Explicitly empty topics or labels collections are managed, so they are not the empty-declaration case. [Topics acceptance](../../tests/acceptance/topics_test.go) covers the topics resource through the built product.

### Merge response acceptance

[Merge transport tests](../../internal/github/merge_methods_test.go), [PR option tests](../../internal/github/pr_options_test.go), and [squash](../../internal/github/squash_commit_test.go)/[normal merge enum tests](../../internal/github/merge_commit_test.go) own response completeness: each required field must be present with the correct type and a supported enum value. Any GraphQL errors discard even complete data. [Observation tests](../../internal/github/observation_test.go) also reject missing fields that were not declared; expected responses use literal fixtures independent of production field definitions.

[Merge acceptance](../../tests/acceptance/merge_methods_test.go) retains representative permission, incomplete-response, and partial-data/error failures, including rejection of an earlier REST comparison with empty stdout and nonzero status. Its failed export preserves the existing destination. Detailed null/type/enum matrices remain at the adapter boundary; successful plan cases and complete declared-only PATCH cases remain at the process boundary.

### Labels resource acceptance

[Labels acceptance](../../tests/acceptance/labels_test.go) runs the current-source binary through omission, clearing, case-insensitive identity, omitted attributes, create/update/delete ordering, no-change reads, invalid input, and failures that stop later operations. It checks rejected GET/POST/PATCH/DELETE redirects during apply, complete pagination during export, and preservation of the existing file when a later page fails. Typed label reports and a separate credential-free Summary retain attribute presence, escaped names, and the warning that deletes remove issue/PR associations.

[Application labels tests](../../internal/app/labels_test.go) separately own repository → topics → labels-read → labels-write sequencing, fresh comparison, already matching collections with no mutation, earlier-write preservation, and rejection of an earlier partial plan when a later labels read/comparison fails. [Adapter tests](../../internal/github/labels_test.go) own exact request payloads and escaped name segments; full validation/identity matrices stay in the model and codec tests.

These checks use local API responses and dummy credentials. They verify current-source behavior without establishing live label permissions or support in the published Action/CLI pair.

### Metadata, reports, and Summary

[Metadata acceptance](../../tests/acceptance/metadata_test.go) checks the supplied executable's actual version/digest and a separate committed consumer identity without settings, credentials, or API requests. Detailed origin syntax and isolation matrices remain in workspace tests. Mock metadata and owned test-executable adapters prove wiring or failure handling, rather than the installed product's identity.

The Linux-only [`TestMetadataUsesRunningImage`](../../internal/actions/metadata_linux_test.go) launches an owned copy of the test executable and replaces only that child's pathname before requesting metadata. The digest must identify the original running image; the parent test runner is not replaced. Native checks on other platforms cover an unchanged executable path, and cross-compilation alone does not run this Linux regression.

[Plan-report acceptance](../../tests/acceptance/plan_report_test.go) inspects native typed JSON and categorical failures through the real interface. Label operations occupy their own typed report entries and count as one managed collection, independently of the number of operations. [Summary acceptance](../../tests/acceptance/summary_test.go) runs credential-free outside a checkout, checking escaped values, separate identity links, append behavior, and zero API requests. Detailed report/state/storage cases, including label reports and the maximum managed count, remain in Actions owner tests. Report/Summary output preserves a failed plan's nonzero status and never becomes apply input.

## CI validation

The [PR](../../.github/workflows/ci-pr.yml) and [main](../../.github/workflows/ci-push.yml) workflows use the following validation stages. Only the PR workflow validates PR message metadata. PR Draft/Ready transitions and approval automation are described in [repository automation](repository-automation.md#pr-lifecycle).

| Stage | Work and handoff |
| --- | --- |
| Commitlint (PR only) | Validate the PR title and optional body together as the squash message; run [policy regression cases](../../tests/github-actions/commitlint.sh) and reject invalid Conventional Commits |
| Commit stage | Formatting, modules, vet, race tests, workflow/shell checks, release/installer fixtures, Markdown/local links, and one product build |
| Acceptance stage | Download that run's product artifact, verify digest and embedded version, then execute the tagged process suite |
| Composite smoke | Use the same artifact with the producer's exact Action source and a separate configuration-only consumer |
| `pass` | Require commitlint, acceptance, and composite smoke for PRs; require acceptance and composite smoke for main. Failures, cancellation, or skipped validation do not become success |

[Commit stage](../../.github/actions/commit-stage/action.yml) exposes its actual checkout SHA, committed CLI version, and binary SHA-256. PR validation builds the merge source, and downstream jobs check out that exact producer SHA. Ordinary CI uses [`action/cli-version`](../../action/cli-version); a release supplies its explicit version and checks equality separately.

Both downstream jobs download the same run's artifact with digest mismatch treated as an error. [Acceptance stage](../../.github/actions/acceptance-stage/action.yml) verifies SHA-256 and the embedded version before launch. It compiles the acceptance harness separately with test-result caching disabled; it neither rebuilds the product nor reruns the ordinary Go suite.

CI installs tools from the same pinned mise configuration. Acceptance installs only Go; composite smoke also installs Go solely to compile the standard-library local API fixture, before the download stand-in enters PATH. Tool caches are disabled. Artifacts are retained for one day, and PR artifacts are not reused by privileged release or settings workflows.

### Hosted candidate composite

The `composite-smoke` job runs on the public Action's supported Ubuntu 24.04 x64 runner; commit/acceptance jobs use `ubuntu-slim`. It checks out exact producer source into `.action` and creates a separate consumer that tracks only `.github/settings.yml`. The consumer has no root `action.yml` or `action/` helper directory.

[Fixture preparation](../../tests/github-actions/prepare-candidate-download.sh) verifies the transferred candidate's checksum and the Action checkout's source/version. A bounded [curl stand-in](../../tests/github-actions/candidate-curl.sh) serves only the unchanged installer's two fixed versioned asset/checksum URLs from those verified bytes, with no network forwarding. Actual `uses: ./.action` expansion exercises installation, checksum/version validation, metadata, and settings operations. No product rebuild, caller version/download override, committed version rewrite, or old-release fallback is used.

The fixture supplies current-source bytes even while the committed CLI version still identifies an older published package. Candidate results therefore do not prove that newly implemented labels support is available through the public installer or existing caller pins. Label-specific current-source behavior is covered by the owner/process tests above; the hosted API cases below remain representative repository-field cases.

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

Local [composite](../../tests/github-actions/composite_test.go) and [caller](../../tests/github-actions/consumer_test.go) fixtures use current candidate package bytes, explicit typed reports, mock downloads/CLI behavior, and owned test-executable adapters. The actual Go Summary adapter is reused without rebuilding the product. Checks cover token-free setup/metadata/Summary, explicit execution tokens, original plan-status precedence, and four outputs only after required work.

Examples and active main callers use the same single-job apply pattern: three `uses` steps for checkout, token creation, and apply. [Operational tests](../../tests/github-actions/operational_test.go) separately verify this repository's Environment and credential bindings. Approval and token lifecycle stubs are simulations; these tests do not load pinned release-tag bytes or establish remote protection/revocation behavior.

## Release and publication evidence

The [release-notes helper tests](../../scripts/release-notes/main_test.go) run with the ordinary Go suite and pinned Go toolchain. They use local Git repositories and release-list fixtures to verify publication-order selection, full-history requirements, the exact previous-release-to-frozen-source range, and categorized rendering from complete commit messages. The [publisher fixtures](../../tests/release.sh) separately check generation before a new remote tag or release and preservation of existing notes during retries. Neither test layer publishes a release.

The [release runbook](releases.md) owns dispatch and recovery. Its pre-publication candidate job reuses the bounded fixture with the frozen source, requested/committed version, and existing Linux release artifact. Actual token-free validate must succeed before publication; source/helper tests inspect that required gate without dispatching a release.

| Evidence | What it establishes | Limit |
| --- | --- | --- |
| Owner and composed tests | Local behavior, actual YAML wiring, and defined failure/status boundaries | Mocks do not establish platform enforcement or installed released bytes |
| Built-product acceptance | The supplied executable runs real entries against isolated files/Git and a local API | Dummy credentials/responses do not prove live GitHub behavior |
| Hosted candidate composite | Actual `uses` expansion, same-source/artifact integration, outputs, Summary bytes and representative direct PATCH behavior | Candidate fixture downloads are not published assets |
| Release candidate gate | An authorized release run passed actual candidate validate alongside native acceptance | Source/fixture tests alone do not prove that a release ran |
| Post-publication smoke | The published tag source/version and real downloaded CLI match the accepted release identities | Failure after publication does not permit changing immutable tags/assets |

Inspect the relevant workflow run for hosted or publication evidence. A successful local fixture, Summary file, or workflow source check cannot establish actual Environment approval, branch policy, token revocation, or GitHub Summary publication.
