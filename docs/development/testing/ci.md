# CI and public Action verification

Use the [local testing guide](../testing.md) to choose a package or run the native CLI. This page maps PR and main checks to their workflows, distinguishes isolated fixtures from runs on GitHub, and identifies the candidate artifacts each check exercises.

## PR screening and main inspection

The [PR workflow](../../../.github/workflows/pr.yml), named `pr`, runs Screening
only: source lint, Go race and offline tests, and a native Linux build.
The PR build reports the Go source default version `dev`.
Its single `screening` job uses `ubuntu-slim` and reports its result directly.
Default `pull_request` activities run it when a PR is opened, updated with
commits, or reopened. Title/body edits and draft/readiness transitions do not
rerun it, and the workflow leaves the PR state unchanged.
The separate [PRLint workflow](../../../.github/workflows/prlint.yml) checks
title/body edits and proposed squash messages.

The [main workflow](../../../.github/workflows/ci-push.yml) runs full Screening
and Inspection. It resolves the release version for the exact `main` SHA,
builds Linux amd64 and macOS arm64 natively using the shared
[build](../../../.github/actions/build/action.yml) Action, then packages and
uploads each product in separate workflow steps. It tests those same
binaries and the public Action through
[release-acceptance](../../../.github/actions/release-acceptance/action.yml).
Each artifact has a source/version/target/SHA-256 manifest and is retained
for seven days for use by the manual release workflow.

The candidate public Action inspection runs every `TestBuilt*` test in
`tests/github-actions` with the `acceptance` build tag and the supplied binary.
New tests using that prefix must run on the hosted Linux runner, consume that
binary rather than rebuild the product, and use isolated local fixtures and
dummy credentials without live GitHub access. The workflow selects the prefix
so a new supplied-product fixture does not need a separate allowlist entry.

Main CI concurrency is grouped by source SHA. A later merge therefore keeps
earlier commits' build and acceptance runs active; only duplicate runs for the
same SHA share a cancelling group. Implementation can continue while those runs
execute. Track each merged commit through its own aggregate result, and repair
failed CI before continuing implementation. Runner availability can still delay
individual jobs; a queued or cancelled job is not successful verification.

PR Screening and native build jobs resolve Go from [go.mod](../../../go.mod) with
`actions/setup-go`, and Bun from `packageManager` in
[package.json](../../../package.json) with `oven-sh/setup-bun`. Dependencies install
from [bun.lock](../../../bun.lock) with `bun install --frozen-lockfile`.
Screening calls [lint](../../../.github/actions/lint/action.yml),
[unit-test](../../../.github/actions/unit-test/action.yml) and
[build](../../../.github/actions/build/action.yml). It does not upload artifacts.
Actionlint and ShellCheck are not part of the check suite. Structural workflow,
permission, dependency-inventory and shell fixture tests continue in Go/Python.

The [full CI/release dependency inventory](../../../.github/actions/lock-workspace-dependencies/action.yml)
remains for main and release. PR Screening's external dependencies are read
from its runtime steps directly.

### Go cache in CI

Setup-go caches module downloads and compilation data using
[go.sum](../../../go.sum). A cache hit never skips validation, race-detected tests
or the native build. Release artifacts and manifests are separately verified;
they are not restored from the Go cache. Frozen acceptance harnesses and
publication helpers select Go from their own source checkout's go.mod.

### Hosted candidate composite

[Supplied-product Ruleset coverage checks](../../../tests/github-actions/ruleset_coverage_acceptance_test.go)
exercise detailed/status-only plans, exact threshold/default payloads, independent
readback, repeated no-ops, failure propagation and unchanged source YAML. The
[built CLI lifecycle](../../../tests/acceptance/ruleset_coverage_test.go) additionally
covers automatic export/retention, offline validation, removal, ID recovery and
explicit pruning through isolated API fixtures.

[Supplied-product workflow permission checks](../../../tests/github-actions/composite_workflow_permissions_acceptance_test.go)
exercise the actual public Action with matching and changed declarations, repeated
apply, minimal PUT payloads and failed independent readback. They verify success
outputs only after completion; API responses remain isolated fixtures.

[Supplied-product Actions execution checks](../../../tests/github-actions/actions_execution_acceptance_test.go)
run the actual composite through matching and changed selected-policy apply,
delayed activation, repeated no-ops and uncertain selected-write/readback failures.
They verify exact ordered requests and success-only outputs against isolated APIs.

The public-Action acceptance leg delegates its scenarios to the existing [candidate composite](../../../.github/actions/e2e-test/action.yml), passing the verified source, version, digest, and binary path. It creates a separate consumer that tracks only `.github/settings.yml`, without a root `action.yml` or `action/` helper directory. The process leg instead runs the built executable through the local API acceptance tests.

[Fixture preparation](../../../tests/github-actions/prepare-candidate-download.sh) verifies the transferred candidate's checksum, executable-reported version, and Action checkout source. A bounded [curl stand-in](../../../tests/github-actions/candidate-curl.sh) serves only the unchanged installer's two fixed versioned asset/checksum URLs from those verified bytes, without network forwarding. Actual `uses: ./.action` expansion exercises installation, checksum/version validation, metadata, and settings operations. No product rebuild, public caller version/download override, committed version rewrite, or old-release fallback is used.

Export scenarios additionally begin without a settings file and verify guarded file production, success-only config-path, unchanged HEAD/index, mutation-free acquisition and absence of plaintext-variable canaries from logs, Summary and outputs. They do not publish the generated file. The supplied-product checks execute the actual composite shell steps locally with the transferred binary; the subsequent `uses: ./.action` cases separately exercise hosted expansion.

The fixture supplies current-source bytes instead of downloading a published artifact, so candidate results do not establish published-artifact identity. The [published package and consumer pins reference](../../guides/github-actions.md#published-package-and-consumer-pins) owns supported-version and adoption guidance. Label-specific behavior is covered by the [label fixtures](settings.md#labels-resource-acceptance); the hosted API cases below remain representative repository-field cases.

Credential-free validate must report all four outputs matching the independent consumer commit/origin and candidate version/checksum. Hosted logs record the distinct Action and consumer roots/SHAs. Missing or tampered artifacts, mismatched identities, and fixture/assertion failures fail the aggregate gate.

### Candidate API and Summary cases

Plan and direct apply use the same verified composite/binary with separate literal dummy read/write tokens. The [local API server](../../../tests/github-actions/plan-fixture/main.go) requires Unix-socket transport, isolates GitHub CLI configuration, and records safe method/path observations. It accepts only the expected target/method/token combinations. Closed loopback HTTP/HTTPS proxies prevent ordinary HTTP transport from reaching an external service.

The [candidate checks](../../../tests/github-actions/candidate-plan.sh) exercise these representative repository-field cases:

| Operation/case | API observation | Required result | Settings review |
| --- | --- | --- | --- |
| Plan changed string/false | One GET | Success and four matching outputs | Native false and escaped HTML/multiline diff |
| Plan local read403 | One GET | Failure and four empty outputs | Categorical read failure |
| Plan invalid boolean | Zero requests | Failure and four empty outputs | Categorical validation failure |
| Apply empty string/false | GET, one PATCH, independent GET | Success and four matching outputs | None added |
| Apply local write403 | GET, one attempted PATCH | Failure and four empty outputs | None added |
| Apply invalid boolean | Zero requests | Failure and four empty outputs | None added |
| Apply empty `settings: {}` | Zero requests | Success and four matching outputs | None added |
| First export and repeat | Closed REST reads and four read-only GraphQL queries | Local file, five matching outputs and unchanged HEAD/index | None added |
| Export invalid file or symlink | Zero requests | Failure, preserved destination and empty outputs | None added |

Apply requests are independently checked as the complete received native JSON object `{"description":"","has_issues":false}`, with every other field omitted. The server records the decoded request instead of manufacturing expected payloads. A failed write attempt does not establish the remote outcome.

Plan/apply cases preserve consumer configuration bytes and commit. Export creates or refreshes only the local configuration and preserves HEAD/index. Its fixture permits only known observations, including GraphQL queries sent through POST, and deliberately makes several families unavailable. The fixture must stay live even for zero-request cases, and an `always()` step stops/removes it. Expected failures use step-level `continue-on-error` only alongside mandatory assertions of the observed failure and empty outputs. Empty apply still meets the public composite's explicit-token and metadata requirements.

For plan, a finite before/after snapshot identifies exactly one new nonempty Settings review. Assertions inspect the bytes written by the embedded Go step, excluding old files and scrubbed upload copies; they do not substitute another renderer or override the Summary destination. All apply cases require zero new reviews. Hosted file rendering establishes neither GitHub Summary publication nor a live settings update.

### Composed callers and examples

Local [composite](../../../tests/github-actions/composite_test.go) and [caller](../../../tests/github-actions/consumer_test.go) fixtures use current candidate package bytes, explicit typed reports, mock downloads/CLI behavior, and owned test-executable adapters. The actual Go Summary adapter is reused without rebuilding the product. Checks cover token-free setup/metadata/Summary, explicit execution tokens, original plan-status precedence, and identity/command-specific outputs only after required work. Export boundary cases cover explicit tokens, prune/private-mode rejection, no Summary, inherited-debug removal, failure propagation and success-only config-path. The [PR data-preparation fixture](../../../tests/github-actions/pr_candidate_example_test.go) executes the sample's actual inline script with real filtered local Git objects and dummy API transport: malicious executable/YAML content is not executed, symlinks/identity/size/read failures fail, and credentials are neither logged nor persisted. It does not establish live installation permissions, private-fork visibility, effective protection or token revocation.

Examples and active main callers use the same single-job apply pattern: three `uses` steps for checkout, token creation, and apply. [Operational tests](../../../tests/github-actions/operational_test.go) separately verify this repository's Environment and credential bindings. Approval and token lifecycle stubs are simulations; these tests do not load pinned release-tag bytes or establish remote protection/revocation behavior.

The [supplied-product export checks](../../../tests/github-actions/composite_export_acceptance_test.go)
also cover `.github` symlinks, concurrent edits, destination write failure and
output failure after a completed save. They capture real composite logs with
`GH_DEBUG=api` and check the local variable canary never reaches logs, Summary or
outputs. Run them against an already-built executable with a stable release
version (the candidate gate uses its verified producer binary):

```sh
mise exec -- go test -tags=acceptance ./tests/github-actions -run '^TestBuilt' -count=1 -binary /absolute/path/to/gh-settings
```

The supplied-product Environment prune check also runs the real composite's
plan (detailed and status-only) and repeated `all` apply against a local API. It
verifies omitted sibling scopes, complete parent deletion, unchanged YAML,
private child evidence and cleanup without live deletion. The secret metadata
integrity fixtures additionally cover ambiguous pages, a bounded response,
complete pagination and privacy; built CLI and supplied-product Action cases
reject ambiguous fresh evidence before DELETE. Owning tests cover
child drift/custom protection and partial progress; the
[prune specification](../features/prune.feature.md#environment-stage-test-mapping)
records their boundaries.

For failures writing export diagnostics, use the [diagnostic delivery cases](settings.md#export-diagnostic-delivery). For separately authorized checks against a live repository, see the [candidate Topics verification procedure](../repository-automation.md#bounded-candidate-action-verification).
