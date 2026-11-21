# Testing and CI

Start with the contract affected by your change. Use an owning package test for individual rules and a built-product test when the change crosses YAML, API, CLI or Action boundaries. Find the relevant [behavior scenarios](features/README.md) and [development commands](setup.md#development-checks).

| What changed | How to verify it |
| --- | --- |
| A setting, its API mapping or its output | [Find its owner tests](testing/settings.md), then [exercise the CLI](#local-process-acceptance) or [run the supplied-product Action tests](testing/ci.md#composed-callers-and-examples). |
| The CLI or Action contract across several settings | [Compare CLI and Action evidence](#personal-public-contract-reconciliation). |
| A workflow, job boundary, credential or artifact | Check the [PR and main CI jobs](testing/ci.md) and the affected [repository automation](repository-automation.md). |
| Documentation, YAML examples or the permissions matrix | Run the [documentation checks](../../CONTRIBUTING.md#documentation-checks). |
| Version calculation or publication | Use the [release validation guide](release-validation.md) and distinguish [local, hosted and published evidence](#release-and-publication-evidence). |

## Test ownership

| Concern | Owning checks | Scope |
| --- | --- | --- |
| Declaration presence and comparison | [Repository model](../../internal/settings/repository_test.go), [comparison](../../internal/settings/comparison_test.go), [topics](../../internal/settings/topics_test.go), [labels](../../internal/settings/labels_test.go) | Omitted vs explicit values, managed counts, native values, topic sets, label identity/attribute presence, and stable operation ordering |
| YAML contract | [UI config](../../internal/workspace/ui_config_test.go), [YAML safety](../../internal/workspace/ui_yaml_safety_test.go), [Labels UI codec](../../internal/workspace/ui_labels_test.go) | Types, strict rejection, aliases/merges, omission, and round trips |
| Files and target resolution | [Configuration files](../../internal/workspace/config_file_test.go), [repository resolution](../../internal/workspace/repository_test.go) | Fixed paths, file behavior, direct-origin formats, scope and isolation |
| GitHub transport | [Repository reads](../../internal/github/repository_test.go), [writes](../../internal/github/apply_test.go), [topics](../../internal/github/topics_test.go), [labels](../../internal/github/labels_test.go) | Complete/paginated reads, null/type normalization, exact declared-only requests, escaped label paths, failures and redirects |
| Application sequencing | [Application tests](../../internal/app), [labels cases](../../internal/app/labels_test.go) | Validation/failure order, empty local no-ops, selective resource access, export composition, fresh labels reads, and partial writes |
| CLI and Actions presentation | [CLI tests](../../internal/cli), [Actions tests](../../internal/actions), [label reports](../../internal/actions/labels_test.go) | Dispatch, diagnostics/status, typed field/label reports, metadata, escaped rendering, and Summary storage |
| Composite and caller wiring | [Composite](../../tests/github-actions/composite_test.go), [consumer](../../tests/github-actions/consumer_test.go), [operational caller](../../tests/github-actions/operational_test.go) | Actual YAML composition, token exposure, output/status ordering, and caller-owned boundaries |
| Independent PR gates | [Message workflow tests](../../tests/github-actions/pr_message_test.go), [screening workflow tests](../../tests/github-actions/pr_code_gate_test.go) | Default PR activities, metadata/code event separation, direct screening steps, read-only execution, and independent message validation |
| Installed process behavior | [Acceptance tests](../../tests/acceptance) | The actual supplied binary, isolated Git/files, local API requests, stdout/stderr and exit codes |
| Version calculation | [Python tests](../../.github/scripts/test_release_version.py) | Temporary Git history, SemVer precedence, fixed published boundaries, invalid catalogs, and immutable retry selection |
| Release notes | [Go metadata tests](../../scripts/release-metadata/main_test.go) | Committed category precedence, formal footers, escaping, fixed Git ranges, and changed-baseline failures |
| Release publication logic | [Release shell tests](../../tests/release.sh), [release gate tests](../../tests/github-actions/release_test.go), [local Action tests](../../tests/github-actions/release_actions_test.go) | Mock publication/recovery, canonical orchestration, manifested artifact identity/tamper rejection, stored-note preservation, and declared source/version/gate relationships |

[Feature specifications](features/README.md) map behavior to ordinary Go or shell tests. No separate BDD runner interprets those Markdown files. Keep detailed parser/type matrices at their owner and use representative acceptance cases to prove composition, rather than duplicating every low-level case.

Markdown checks use the package-managed rumdl from [package.json](../../package.json) and [.rumdl.toml](../../.rumdl.toml). MD051 checks local heading fragments and MD057 checks relative file links; external HTTP availability is not a CI dependency.

## Local process acceptance

Build the product once, then supply that existing executable to the tagged acceptance suite:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The harness requires an explicit binary path and Unix-socket support. It uses go-gh's `http_unix_socket` configuration in an isolated temporary `GH_CONFIG_DIR`, temporary Git repositories, and a local HTTP fixture. Dummy tokens never authenticate with GitHub. Fixtures are cleaned up without adding product hooks or test-only transport options.

Each CLI subprocess has a finite 60-second harness budget, including the direct
file-descriptor failure cases. This allows headroom for full export's independent
family reads on hosted runners; it does not change product API deadlines or retry
failed commands. The shared executor always returns a diagnostic result with
captured stdout, stderr and process exit error, separately from a setup or context
error. [Harness tests](../../tests/acceptance/process_test.go) verify setup failure,
an expired deadline, cancellation after the child signals that both streams were
written, and ordinary success/failure without requiring the product binary.

## Personal-public contract reconciliation

For a contract verification pass, reconcile the existing owners below before
adding tests. These are representative checks, not an exhaustive platform matrix.
Record the actual source SHA, supplied binary identity, workflow run and observed
results in the tracking Issue or PR. Keep final-candidate selection and published
artifact verification separate from current-source fixture results.

| Contract boundary | CLI evidence | Public Action evidence |
| --- | --- | --- |
| Export, validate, plan and apply composition; native false/empty values | [Export/edit lifecycle](../../tests/acceptance/apply_test.go), [export/validate](../../tests/acceptance/export_validate_test.go) and [topics lifecycle](../../tests/acceptance/topics_test.go) | [Supplied-product export](../../tests/github-actions/composite_export_acceptance_test.go), [webhook lifecycle](../../tests/github-actions/composite_webhooks_acceptance_test.go) and [hosted operation wiring](../../.github/actions/e2e-test/action.yml) |
| Omission and unavailable observations without invented state | [Independent export acquisition](../../tests/acceptance/repository_export_isolation_test.go) and [CodeQL configuration](../../tests/acceptance/codeql_configuration_test.go) | [First/repeated export](../../tests/github-actions/composite_export_acceptance_test.go) and [unavailable webhook retention](../../tests/github-actions/composite_webhooks_acceptance_test.go); shared acquisition owners retain detailed family cases |
| Repeat no-op and independent readback | [Topics](../../tests/acceptance/topics_test.go), [labels](../../tests/acceptance/label_readback_test.go) and [Rulesets](../../tests/acceptance/ruleset_readback_test.go) | [Runner labels](../../tests/github-actions/runners_acceptance_test.go) and [webhooks](../../tests/github-actions/composite_webhooks_acceptance_test.go) |
| Selected prune scopes, omitted siblings and standalone `all` | [Prune specification and owner mappings](features/prune.feature.md), including [Environment process checks](../../tests/acceptance/environment_prune_test.go) | [Environment prune](../../tests/github-actions/environment_prune_acceptance_test.go), [runner prune](../../tests/github-actions/runner_prune_acceptance_test.go) and [mixed webhook prune](../../tests/github-actions/composite_webhook_prune_mixed_acceptance_test.go) |
| Partial failure, earlier progress and absent success outputs | [Label ID recovery](../../tests/acceptance/label_ids_test.go) and [webhook mixed prune](../../tests/acceptance/webhook_prune_mixed_test.go) | [Apply progress diagnostics](../../tests/github-actions/apply_progress_acceptance_test.go) and [mixed webhook prune](../../tests/github-actions/composite_webhook_prune_mixed_acceptance_test.go) |

The [same-artifact acceptance workflow](../../.github/actions/release-acceptance/action.yml)
runs the native process suite and the supplied-product `TestBuilt` Action checks;
the hosted lane additionally expands the actual root composite. Those API fixtures
use dummy credentials, not live target settings. Bounded live observations need
their own target, authorization, method and limits; they do not certify every
setting, effective enforcement or App permission. Live webhook delivery is not a
v1 completion requirement, and organization-only features are outside this pass.

Before claiming released CLI/Action parity, check the selected immutable package,
its installed CLI and the [consumer pins](../guides/github-actions.md#published-package-and-consumer-pins),
not only source metadata. The selected package determines whether it supports the declaration and command under test.
Do not mark final-candidate or publication checks complete until those specific
artifacts have actually been selected and verified. The [evidence categories](#release-and-publication-evidence) explain what
each check can establish.

## CI validation

Pull requests run screening and a native Linux build. Main CI builds and tests the Linux and macOS candidates from the merged source. Read the [workflow and artifact checks](testing/ci.md) to identify the applicable job and what its result establishes.

## Release and publication evidence

Use [release validation](release-validation.md) for local version, artifact, workflow and publisher checks. The [maintainer release guide](releases.md) owns dispatch and recovery.

| Evidence | What it establishes | Limit |
| --- | --- | --- |
| Owner and composed tests | Local behavior, actual YAML wiring, and defined failure/status boundaries | Mocks do not establish platform enforcement or installed released bytes |
| Built-product acceptance | The supplied executable runs real entries against isolated files/Git and a local API | Dummy credentials/responses do not prove live GitHub behavior |
| Hosted candidate composite | Actual `uses` expansion, same-source/artifact integration, outputs, Summary bytes and representative direct PATCH behavior | Candidate fixture downloads are not published assets |
| Release acceptance gate | An authorized release run passed both native targets and actual candidate validate using the manifested products | Source/fixture tests alone do not prove that a release ran |
| Post-publication verification | The published tag and actual downloaded assets match both accepted manifests, and the published root Action reports all four identities | Failure after publication does not permit changing immutable tags/assets |

Inspect the relevant workflow run for hosted or publication evidence. A successful local fixture, Summary file, or workflow source check cannot establish actual Environment approval, branch policy, token revocation, or GitHub Summary publication.
