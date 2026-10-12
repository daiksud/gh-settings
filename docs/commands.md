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

`gh settings export` is implemented; the other commands remain proposed. The agreed initial interface has exactly four commands: `export`, `validate`, `plan`, and `apply`. They share the five fields defined on the [settings page](settings.md). Export reads the current checkout's repository and writes configuration to standard output; it does not require or read `.github/settings.yml`. The proposed mutation workflow targets the repository containing that configuration. See the [concept](concept.md) for decision reasoning, future direction, and unresolved choices.

Saved plan artifacts and drift protection are outside the initial scope. GitHub Action delivery is deferred.

## Interface and implementation status

The `export` example is implemented. `validate`, `plan`, and `apply` remain proposed and are rejected by the current CLI. No target override or output-format flags are implemented. Additional flags and output formats remain undecided. The [settings page](settings.md#configuration-semantics) defines the agreed root mapping for the current five-field scope.

```text
gh settings export
gh settings validate
gh settings plan
gh settings apply
```

| Command | Contract |
| --- | --- |
| `export` (implemented) | Read current supported settings and write YAML configuration to standard output. Exclude unsupported fields and read-only metadata. |
| `validate` | Check configuration against the supported schema. Unknown keys and invalid values fail validation. |
| `plan` | Read current settings and show differences for managed fields only. The result is informational. |
| `apply` | Validate configuration, then send declared supported fields directly to the repository update API. Omitted fields are not sent. |

The update endpoint is `PATCH /repos/{owner}/{repo}`.[^github-update-repository] The owner and repository identify the repository containing the configuration; this does not introduce a configurable list of targets.

`apply` does not fetch current settings, compute differences, compare against a saved plan, or preserve external changes to managed values. If a managed value changes remotely after `plan`, `apply` still sends the YAML value. The observed plan diff can therefore differ from changes at apply time. Omitting a field leaves it outside the request.

For example, declaring only `has_issues: false` means that apply sends that supported field with the declared value and sends none of the other four fields. The [configuration example](settings.md#configuration-semantics) places this field at the YAML root, following the update request body schema.

## Export behavior

Export resolves the current Git checkout's `origin` remote, including from subdirectories. The current implementation accepts GitHub.com HTTPS and SSH URLs and ignores `GH_REPO`. Missing or unsupported origins fail before the API read. GitHub API authentication uses the existing go-gh/gh integration. The command normalizes GitHub description/homepage nulls to empty strings; it distinguishes missing response keys from null and rejects missing keys or null booleans. The command preserves false and empty string values and does not write configuration files or change repository settings. Read, decoding, and output failures return a nonzero status; read and decoding failures leave standard output empty. See the [behavior specification and test mapping](features/export.feature.md).

## Failures and limits for proposed mutation commands

Unknown keys and invalid values fail before API mutation. Permission and API failures are reported with a nonzero exit status. GitHub may reject a request even when configuration passes local validation; successful validation does not guarantee successful mutation. The endpoint documents failure responses including permission and validation failures.[^github-update-repository]

There is no automatic rollback and no transactional guarantee. A failure does not promise restoration of prior settings. The [selected go-gh library](adr/0008-official-go-gh-library.md) provides GitHub CLI authentication integration; remaining authentication details are undecided, and the tool must report access failures rather than claim that all declared settings can always be applied.

## Acceptance boundaries for future implementation

- Export includes only the five supported settings as configuration.
- Validation rejects unsupported keys and values of the wrong type before mutation.
- Plan compares only declared fields with current settings and describes an observation, not a binding execution plan.
- Apply sends declared supported values without reading current settings, including when a remote managed value has changed since plan.
- Omitted fields are absent from the update request.
- Permission or API failure produces a nonzero exit status without automatic rollback.

Export acceptance conditions are covered by the [implemented scenarios and Go tests](features/export.feature.md); the remaining conditions are intended for future implementation. The [settings page](settings.md#configuration-semantics) records the agreed structure and unresolved null handling for future YAML input. Future composition across multiple update endpoints remains undecided.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.
