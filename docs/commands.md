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

The published v0.3.0 release implements `gh settings export`, `gh settings validate`, and read-only `gh settings plan`; export emits the versioned `settings.repository` format. Current source also implements `gh settings apply`; apply has not been released. The agreed initial interface has exactly four commands: `export`, `validate`, `plan`, and `apply`. They share the five fields defined on the [settings page](settings.md). Export reads the current checkout's repository and writes configuration to standard output; it does not require or read `.github/settings.yml`. The mutation workflow targets the repository containing that configuration. See the [concept](concept.md) for decision reasoning, future direction, and unresolved choices.

Saved plan artifacts and drift protection are outside the initial scope. GitHub Action delivery is deferred.

## Interface and implementation status

All four command examples are implemented in current source. Export, validate, and plan are available in v0.3.0; apply has not been released. No target override or output-format flags are implemented. Additional flags and output formats remain undecided. The [settings page](settings.md#configuration-semantics) defines root `schemaVersion: 1` and the `settings.repository` mapping for the current five-field scope.

```text
gh settings export
gh settings validate
gh settings plan
gh settings apply
```

| Command | Contract |
| --- | --- |
| `export` (implemented) | Read current supported settings and write YAML configuration to standard output. Exclude unsupported fields and read-only metadata. |
| `validate` (implemented) | Check `.github/settings.yml` locally against the supported schema. Unknown or duplicate keys and invalid values fail validation. |
| `plan` (implemented) | Read current settings and show differences for managed fields only. The result is informational. |
| `apply` (implemented in source) | Validate configuration, then send declared supported fields directly to the repository update API. Omitted fields are not sent. |

The update endpoint is `PATCH /repos/{owner}/{repo}`.[^github-update-repository] The owner and repository identify the repository containing the configuration; this does not introduce a configurable list of targets.

`apply` does not fetch current settings, compute differences, compare against a saved plan, or preserve external changes to managed values. If a managed value changes remotely after `plan`, `apply` still sends the YAML value. The observed plan diff can therefore differ from changes at apply time. Omitting a field leaves it outside the request.

For example, declaring only `settings.repository.has_issues: false` means that apply sends that supported field with the declared value and sends none of the other four fields. The [configuration example](settings.md#configuration-semantics) shows the YAML mapping; the update request body contains only `has_issues`.

## Export behavior

Export resolves the current Git checkout's configured `origin` remote, including from subdirectories and repository-specific included or worktree configuration. A configured GitHub URL keeps its repository identity when Git `insteadOf` changes its transport destination; an alias rewritten to a GitHub URL remains usable. Global Git configuration cannot supply an origin outside the checkout. The current implementation accepts GitHub.com HTTPS and SSH URLs and ignores `GH_REPO`. Missing or unsupported origins fail before the API read. GitHub API authentication uses the existing go-gh/gh integration. The command normalizes GitHub description/homepage nulls to empty strings; it distinguishes missing response keys from null and rejects missing keys or null booleans. The command preserves false and empty string values and does not write configuration files or change repository settings. Read, decoding, and output failures return a nonzero status; read and decoding failures leave standard output empty. See the [behavior specification and test mapping](features/export.feature.md).

## Validate behavior

Validate reads `.github/settings.yml` at the current Git checkout's root, including when run from a subdirectory. It checks one YAML root mapping with required integer `schemaVersion: 1`, optional `settings.repository`, and the five supported fields. `description` and `homepage` must be strings; `has_issues`, `has_projects`, and `has_wiki` must be booleans. It rejects missing or unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple YAML documents, and values with the wrong shape or type. Empty settings mappings and partial declarations are valid. Omitted fields remain omitted; explicit false and empty strings are valid. Explicit null fails the string/boolean type check under [ADR 0019](adr/0019-reject-explicit-null.md). Alias and merge values are checked after YAML resolves them. Earlier flat YAML files fail validation and need a version plus their fields moved under `settings.repository`. This check does not verify GitHub's server-side behavior for null updates.

Success is silent with exit status zero. Failure reports a diagnostic on standard error and exits nonzero with no standard output. Validate reads no GitHub settings, requires no GitHub origin or authentication, and does not change the file or repository settings. See the [behavior specification and test mapping](features/validate.feature.md).

## Plan behavior

Plan reads and validates the Git root's `.github/settings.yml` before resolving origin, configuring authentication, or reading GitHub. It uses export's current-repository resolution and GET response handling, including null string normalization and go-gh's existing sanitization of some API control characters to caret notation. It compares only declared fields; omitted fields are unmanaged, while false and empty strings are explicit managed values. It reads the current five fields even when none are managed, so origin, authentication, and API errors still fail.

Changed fields appear in fixed order: description, homepage, has_issues, has_projects, has_wiki. Each line uses `settings.repository.FIELD: CURRENT → DESIRED`. Strings are double-quoted using Go string escaping, including quotes, backslashes, newlines, and control characters; booleans use `true` or `false`. Equal managed values are omitted. When all managed values match, output is `No differences for managed repository settings.` When no fields are declared, output is `No managed repository settings.`

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding, and output errors exit 1 and report a diagnostic on standard error. Failures before writing output leave standard output empty. Plan does not edit configuration or mutate repository settings. See the [behavior specification and test mapping](features/plan.feature.md). Its observation does not change the independent direct-update contract for apply.

## Apply behavior

Apply reads and validates the Git root's `.github/settings.yml`, including when run from a subdirectory, using the same schemaVersion 1 parser as validate and plan. Invalid configuration fails before origin resolution, authentication, or API access. When no repository fields are declared, it prints `No managed repository settings.` and exits 0 without resolving origin or configuring an API client.

For declared fields, apply uses export's current-repository resolution, ignores `GH_REPO`, and uses go-gh authentication. It sends one `PATCH /repos/{owner}/{repo}` containing only the declared repository fields, without `schemaVersion` or the outer YAML mappings. False and empty strings are included; omitted fields are absent. It performs no GET, diff reconciliation, or saved-plan check, even if the declared value already matches the remote value. Redirects fail instead of risking a PATCH being converted into GET; update origin when a repository moves.

A successful update prints `Applied repository settings.` and exits 0. File, validation, origin, authentication, API/response, and output failures exit 1 and report a diagnostic on standard error. Failures before output leave standard output empty. An output failure can occur after the PATCH succeeds; retrying sends the declared values again. Apply does not edit the configuration. See the [behavior specification and test mapping](features/apply.feature.md).

## Failures and limits for mutation

Unknown keys and invalid values fail before API mutation. Permission and API failures are reported with a nonzero exit status. GitHub may reject a request even when configuration passes local validation; successful validation does not guarantee successful mutation. The endpoint documents failure responses including permission and validation failures.[^github-update-repository]

There is no automatic rollback and no transactional guarantee. A failure does not promise restoration of prior settings. The [selected go-gh library](adr/0008-official-go-gh-library.md) provides GitHub CLI authentication integration; the tool reports access failures rather than claim that all declared settings can always be applied.

## Acceptance boundaries

- Export includes only the five supported settings as configuration.
- Validation rejects unsupported keys and values of the wrong type before mutation.
- Plan compares only declared fields with current settings and describes an observation, not a binding execution plan.
- Apply sends declared supported values without reading current settings, including when a remote managed value has changed since plan.
- Omitted fields are absent from the update request.
- Permission or API failure produces a nonzero exit status without automatic rollback.

All four commands have implemented behavior specifications and Go tests ([export](features/export.feature.md), [validate](features/validate.feature.md), [plan](features/plan.feature.md), [apply](features/apply.feature.md)). The [settings page](settings.md#configuration-semantics) records the agreed structure and null handling for these five fields. Future composition across multiple update endpoints remains undecided.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.
