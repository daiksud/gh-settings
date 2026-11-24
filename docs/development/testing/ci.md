# CI and public Action verification

Use the [local testing guide](../testing.md) to choose a package or run the native CLI. This page maps PR and main checks to their workflows, distinguishes isolated fixtures from runs on GitHub, and identifies the candidate artifacts each check exercises.

## PR screening and main inspection

[PR screening](../pr-validation.md#pr-screening) runs lint, race and offline tests, then builds a Linux CLI reporting `dev`. Its build ends in the PR job.

The [main workflow](../../../.github/workflows/push.yml) calls the [version workflow](../../../.github/workflows/version.yml)
for the exact `main` SHA. A potential PATCH can require native comparison builds
before the version is known. The [candidate workflow](../../../.github/workflows/candidate.yml)
runs screening on Linux amd64 and builds macOS arm64 on its native runner.
It uploads each binary and a source/version/target/SHA-256 manifest. Three
inspection lanes use those artifact IDs to test the two native binaries and the
candidate public Action. After all three pass, `deployment` records their
IDs and digests. The binaries and deployment record are retained for seven days.
The [release guide](../releases.md#cross-run-artifact-reuse) describes how a manual
release selects them.

The candidate public Action inspection runs every `TestBuilt*` test in
`tests/github-actions` with the `acceptance` build tag and the supplied binary.
New tests using that prefix must run on the hosted Linux runner, consume that
binary rather than rebuild the product, and use isolated local fixtures and
dummy credentials without live GitHub access. The workflow selects the prefix
so a new supplied-product fixture does not need a separate allowlist entry.

Main CI concurrency is grouped by source SHA. A later merge therefore keeps
earlier commits' build and acceptance runs active; only duplicate runs for the
same SHA share a cancelling group. Implementation can continue while those runs
execute. Track each merged commit through its own deployment result, and repair
failed CI before continuing implementation. Runner availability can still delay
individual jobs; a queued or cancelled job is not successful verification.

### Go cache in CI

The build jobs select Go from [go.mod](../../../go.mod) and use [go.sum](../../../go.sum) as the cache dependency file. Setup-go caches module downloads and compilation data. Tests and native builds run on cache hits. The acceptance jobs verify the transferred artifacts against their manifests and select Go from the frozen source checkout to compile the harness.

### Hosted candidate composite

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

To run all supplied-product Action cases locally, pass an executable built with a stable version embedded:

```sh
mise exec -- go test -tags=acceptance ./tests/github-actions -run '^TestBuilt' -count=1 -binary /absolute/path/to/gh-settings
```

The [setting verification guide](settings.md) maps export, pruning and resource-specific cases to their owning tests. For authorized checks against a live repository, use the [candidate Topics verification procedure](../repository-automation.md#bounded-candidate-action-verification).
