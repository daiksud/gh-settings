---
type: Specification
title: Settings commands
description: Define export, validate, plan, and apply behavior.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
---

## Scope and status

This reference describes current source. Published v0.8.0 includes topics; Labels support is unreleased. Older v0.7.0 consumers retain their five-field behavior. The four settings commands are `export`, `validate`, `plan` and `apply`, sharing the repository fields, topics and labels on the [settings page](settings.md). Export reads the current checkout's repository and writes that checkout's root `.github/settings.yml`; it does not require existing configuration. Mutation targets only the repository containing that configuration. Consumers pinned to v0.6.1 retain that release's behavior; see the [published package and consumer profiles](github-actions.md#published-package-and-consumer-pins). The [concept](concept.md) records design reasoning and future scope.

Saved plan artifacts and drift protection are outside the initial scope. [Inert GitHub Actions examples](github-actions.md) compose these commands; security setup and activation remain separate.

## Interface and implementation status

All four commands below are implemented. There are no target override or output-format flags. The [settings page](settings.md#configuration-semantics) defines root `schemaVersion: 1` and the `settings.repository` and `settings.topics` mappings.

```text
gh settings export
gh settings validate
gh settings plan
gh settings apply
```

| Command | Contract |
| --- | --- |
| `export` | Read current supported settings and silently save YAML to checkout-root `.github/settings.yml`. Exclude unsupported fields and read-only metadata. |
| `validate` | Check `.github/settings.yml` locally against the supported schema. Unknown or duplicate keys and invalid values fail validation. |
| `plan` | Validate, then read current settings and show differences for declared fields only. An empty declaration succeeds locally without origin or credentials. |
| `apply` | Validate, then send declared fields directly to the resource endpoints; Labels reads fresh state for reconciliation. An empty declaration succeeds locally without origin or credentials. |

The repository update endpoint is `PATCH /repos/{owner}/{repo}`.[^github-update-repository] The owner and repository identify the repository containing the configuration; this does not introduce a configurable list of targets.

For repository fields and topics, `apply` does not fetch current settings or compute differences. Labels alone reads current membership for CRUD reconciliation. Neither path compares against a saved plan or preserves external changes to managed values. If a managed value changes remotely after `plan`, `apply` still sends the YAML value. The observed plan diff can therefore differ from changes at apply time. Omitting a field leaves it outside the request.

For example, declaring only `settings.repository.has_issues: false` means that apply sends that supported field with the declared value and sends none of the other four fields. The [configuration example](settings.md#configuration-semantics) shows the YAML mapping; the update request body contains only `has_issues`.

## Topics command behavior

The [topics contract](settings.md#topics) defines validation and omission. Export reads repository settings and topics before overwriting the file. Plan reads only managed resources and prints stable arrays, for example `settings.topics.names: ["old"] → []`. An empty topics mapping is a local no-op when no other fields are managed.

Apply uses repository PATCH then topics PUT, skips undeclared resources, and sends `names: []` only for explicit clearing. The first failure stops execution; earlier successful writes remain applied. There is no preliminary GET, automatic rollback or retry. Existing success/no-op messages are retained. See [topics process acceptance](../tests/acceptance/topics_test.go).

## Labels command behavior

The [Labels contract](features/labels.feature.md) defines authoritative membership: omission is unmanaged, a list deletes undeclared labels, and `[]` deletes all labels and their issue/PR associations. Export reads every page after repository/topics and writes only after complete reads and serialization. Plan shows sorted create/update operations before deletes, with an assignment-loss warning for deletes. Apply reads fresh Labels state after the existing direct repository/topics writes, then performs only needed POST/PATCH/DELETE operations. Omitted label attributes preserve remote values; an empty description clears it. Any failure stops later operations and earlier successful writes remain applied. Saved reports are never apply input.

## Source Actions metadata

The v0.7.0 CLI also provides `gh settings action metadata` for same-version CLI/Action wiring. This is separate from the four settings commands. The published v0.7.0 composite invokes it through the verified executable; the released v0.6.1 composite keeps its original implementation. It resolves the current consumer checkout, including subdirectories, using the shared Go origin rules and ignoring `GH_REPO`. Neither the executable directory nor an Action source directory selects the target; no `GITHUB_WORKSPACE` override mode is added. No settings file, token or API request is needed.

Success emits exactly four single-line `key=value` records: `config-sha` is the checkout's resolved commit, `target` is its origin's owner/repository, `cli-version` is the injected executable version and `cli-sha256` is the SHA-256 of the actual executing binary. No binary-path or version argument selects a different identity. Missing identity or CR/LF in its components fails before output; output errors also return a nonzero status with a stderr diagnostic. Write failures can leave partial output. This interface adds no old-CLI fallback or multi-version protocol. See the [metadata behavior and tests](features/actions.feature.md#rule-source-metadata-identifies-the-consumer-and-executing-binary).

On the supported Linux runner, the digest reads the running image through `/proc/self/exe`, so replacement or deletion of its launch pathname does not select other bytes. Failure to read that reference fails the command. Other platforms use the executable pathname for native development checks and require that path to remain unchanged; this does not extend the Action's supported platforms.

## Source Actions plan report

The v0.7.0 CLI provides `gh settings action plan REPORT_PATH` for same-version Actions presentation. It runs shared application planning once and writes JSON instead of terminal plan text. Supply a caller-owned runner-temporary path with an existing parent; new files use mode 0600. Success is silent and exits zero. Missing arguments or unavailable storage fail visibly; a planning failure remains nonzero even after its report is written. Write failures can leave partial output.

The [Actions-owned report](../internal/actions/plan.go) has these fields:

| Field | Meaning |
| --- | --- |
| `outcome` | `changed`, `no-difference`, `unmanaged`, `invalid-config`, `target-failure` or `read-failure`. |
| `target` | Available origin owner/repository; omitted for unmanaged, validation or target-resolution failure. |
| `managed` | Number of managed fields/collections in a successful comparison (Labels counts once); zero on failure does not mean unmanaged. |
| `changes` | Ordered field/before/after entries; values remain native strings, booleans or topic string arrays. Field paths are `repository.FIELD` or `topics.names` in current source. Empty comparisons/failures have an empty array. |
| `labels` | Optional typed create/update/delete entries with name and applicable before/after color/description objects. Omitted attributes remain absent. Absent when no label operations exist; not a saved execution plan. |
| `diagnostic` | Static categorical failure text; omitted on success. No raw error, token, environment dump or request header is serialized. |

Validation precedes target resolution and API access. Valid unmanaged declarations succeed locally without origin or credentials. Read failures include the already-resolved target; ordinary stderr diagnostics retain their existing text. The report is ephemeral presentation data, never apply input. The published v0.7.0 composite calls this entry with its explicit read token, then passes the report to a separate credential-free Summary step. Shared planning owns validation; no separate CLI validation call, multi-version format or saved-plan execution is introduced. See the [report behavior](features/actions.feature.md#rule-source-planning-writes-ephemeral-typed-presentation-data).

## Source Actions Summary

The v0.7.0 CLI provides `gh settings action summary REPORT_PATH SUMMARY_PATH`. It reads the same-version typed report and explicit nonsecret run metadata, then appends review content to the caller-owned Summary path. It requires no checkout, settings file, credentials or API access and writes no GitHub outputs. The published v0.7.0 composite invokes this verified entry in a separate step with both token variables unset. Original nonzero plan status is preserved; otherwise Summary status determines success. The four public outputs follow both required stages.

The metadata allowlist is `SETTINGS_ACTOR`, `SETTINGS_TRIGGERING_ACTOR`, `SETTINGS_EVENT`, `SETTINGS_RUN_ID`, `SETTINGS_RUN_ATTEMPT`, `SETTINGS_WORKFLOW_REF`, `SETTINGS_WORKFLOW_SHA`, `SETTINGS_CONFIG_SHA`, `SETTINGS_TARGET`, `SETTINGS_CLI_VERSION` and `SETTINGS_CLI_SHA256`. All are required. `SETTINGS_ACTION_REPOSITORY` and `SETTINGS_ACTION_REF` must either both be present or both absent; absence displays an honest local-checkout identity. Other environment values are not inspected or included.

Run ID and attempt must be positive integers, workflow/configuration revisions 40 hex characters and the CLI digest 64 hex characters. Workflow references contain an owner/repository/path and `@refs/` reference; target and optional Action repository contain owner/repository. The file entry requires an explicit non-null `managed` count from zero through seven. The renderer rejects inconsistent report states, counts or targets, mismatched categorical failure diagnostics, unknown or duplicate fields, no-op changes and field/value-type mismatches for the supported fields. Topic arrays must be sorted, unique and valid, and equality is checked without comparing slices as interface values. Empty strings and false remain valid change values. Workflow links use the workflow repository/revision; configuration links use the explicit target/configuration revision. Original and rerun actors, Action identity and CLI identity remain distinct.

The [pure renderer](../internal/actions/summary.go) classifies typed outcomes rather than terminal sentences. It quotes HTML and multiline data, retaining native observed/desired values and the direct-apply overwrite warning. Missing/malformed input or required metadata fails before opening the destination, preserving existing content. A blank-line separator starts the appended section even when prior text has no trailing newline. New files use mode 0600; append, write and close failures return nonzero, without atomicity or rollback guarantees. Success is silent. A valid failed-plan report is rendered and appended but still exits nonzero. Rendering/writing does not prove that GitHub published the Summary; review it before separately authorizing apply. See the [Summary behavior](features/actions.feature.md#rule-source-summary-renders-typed-review-data-without-credentials).

## Export behavior

Export resolves the current Git checkout's configured checkout-scoped `origin` remote, including from subdirectories and repository-specific included or worktree configuration. Published v0.7.0 requires that selected origin to directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or a full SSH URL parsed by go-gh. A direct GitHub URL keeps its repository identity when Git `insteadOf` changes its transport destination. System, global, and command Git configuration cannot supply the origin target, even inside a checkout; only local/worktree origin values are eligible. Rewrite-only aliases and username-less `github.com:owner/repo` are rejected before API use; configure a supported direct URL instead. Consumers pinned to v0.6.1 retain that release's username-less SCP and rewrite-alias compatibility. The v0.7.0 support reduction does not change settings YAML. `GH_REPO` remains ignored. Missing or unsupported origins fail before the API read. GitHub API authentication uses the existing go-gh/gh integration. The command normalizes GitHub description/homepage nulls to empty strings; it distinguishes missing response keys from null and rejects missing keys or null booleans. The command preserves false and empty string values and saves the complete YAML to `.github/settings.yml` at the same checkout root, creating `.github` when missing. Success is silent with exit status zero; it changes no GitHub settings. An existing regular configuration file is always overwritten, without a confirmation or overwrite flag. Symlinks at `.github` or `settings.yml`, including dangling links, and other non-regular destinations are rejected. API reading and YAML serialization finish before opening the destination for writing. Write errors return a nonzero status and can leave the file truncated or partially written; Git handles restoration of tracked configuration. All failure diagnostics go to stderr with no stdout. Read the saved file after success instead of redirecting export stdout. See the [behavior specification and test mapping](features/export.feature.md).

## Validate behavior

Validate reads `.github/settings.yml` at the current Git checkout's root, including when run from a subdirectory. It checks one YAML root mapping with required integer `schemaVersion: 1`, optional `settings.repository`, `settings.topics` and `settings.labels`, and their supported fields. `description` and `homepage` must be strings; `has_issues`, `has_projects`, and `has_wiki` must be booleans. It rejects missing or unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple YAML documents, and values with the wrong shape or type. Empty settings mappings and partial declarations are valid. Omitted fields remain omitted; explicit false and empty strings are valid. Explicit null fails the string/boolean type check under [ADR 0019](adr/0019-reject-explicit-null.md). Alias and merge values are checked after YAML resolves them. Root-level fields outside the versioned settings contract fail validation. This check does not verify GitHub's server-side behavior for null updates.

Success is silent with exit status zero. Failure reports a diagnostic on standard error and exits nonzero with no standard output. Validate reads no GitHub settings, requires no GitHub origin or authentication, and does not change the file or repository settings. See the [behavior specification and test mapping](features/validate.feature.md).

## Plan behavior

Plan reads and validates the Git root's `.github/settings.yml` before resolving origin, configuring authentication, or reading GitHub. A valid version-only document, empty `settings` or empty `settings.repository` is a local no-op: it needs no origin or credentials and makes no API calls. Invalid configuration still fails validation. For declared fields, plan uses export's current-repository resolution and GET response handling, including null string normalization and go-gh's existing sanitization of some API control characters to caret notation. It compares only declared fields; omitted fields are unmanaged, while false and empty strings are explicit managed values.

Changed fields appear in fixed order: description, homepage, has_issues, has_projects, has_wiki. Each line uses `settings.repository.FIELD: CURRENT → DESIRED`. Strings are double-quoted using Go string escaping, including quotes, backslashes, newlines, and control characters; booleans use `true` or `false`. Equal managed values are omitted. When all managed values match, output is `No differences for managed repository settings.` When no fields are declared, output is `No managed repository settings.`

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding, and output errors exit 1 and report a diagnostic on standard error. Failures before writing output leave standard output empty. Plan does not edit configuration or mutate repository settings. See the [behavior specification and test mapping](features/plan.feature.md). Its observation does not change the independent direct-update contract for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) retains its independent target/commit metadata and explicit plan/apply token requirements, including for empty declarations. A standalone CLI no-op does not relax that audit contract.

## Apply behavior

Apply reads and validates the Git root's `.github/settings.yml`, including when run from a subdirectory, using the same schemaVersion 1 parser as validate and plan. Invalid configuration fails before origin resolution, authentication, or API access. When no resources are managed, it prints `No managed repository settings.` and exits 0 without resolving origin or configuring an API client.

For declared fields, apply uses export's current-repository resolution, ignores `GH_REPO`, and uses go-gh authentication. It sends one `PATCH /repos/{owner}/{repo}` containing only the declared repository fields, without `schemaVersion` or the outer YAML mappings. False and empty strings are included; omitted fields are absent. It performs no GET, diff reconciliation, or saved-plan check, even if the declared value already matches the remote value. Redirects fail instead of risking a PATCH being converted into GET; update origin when a repository moves.

A successful update prints `Applied repository settings.` and exits 0. File, validation, origin, authentication, API/response, and output failures exit 1 and report a diagnostic on standard error. Failures before output leave standard output empty. An output failure can occur after the PATCH succeeds; retrying sends the declared values again. Apply does not edit the configuration. See the [behavior specification and test mapping](features/apply.feature.md).

## Failures and limits for mutation

Unknown keys and invalid values fail before API mutation. Permission and API failures are reported with a nonzero exit status. GitHub may reject a request even when configuration passes local validation; successful validation does not guarantee successful mutation. The endpoint documents failure responses including permission and validation failures.[^github-update-repository]

There is no automatic rollback and no transactional guarantee. A failure does not promise restoration of prior settings. The [selected go-gh library](adr/0008-official-go-gh-library.md) provides GitHub CLI authentication integration; the tool reports access failures rather than claim that all declared settings can always be applied.

## Acceptance boundaries

- Export includes only supported repository fields, topic names and label definitions as configuration.
- Validation rejects unsupported keys and values of the wrong type before mutation.
- Plan compares only declared fields with current settings and describes an observation, not a binding execution plan.
- Apply sends declared repository/topic values directly; Labels reconciles fresh state. Remote managed values can change since plan.
- Omitted fields are absent from the update request.
- Permission or API failure produces a nonzero exit status without automatic rollback.

All four commands have implemented behavior specifications and Go tests ([export](features/export.feature.md), [validate](features/validate.feature.md), [plan](features/plan.feature.md), [apply](features/apply.feature.md)). The [settings page](settings.md#configuration-semantics) records the agreed structure and null handling for repository fields and topic names. Mixed-resource apply ordering and partial-failure behavior are defined in the [topics](settings.md#topics) and [Labels](features/labels.feature.md) contracts.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.
