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

`gh settings` is unimplemented. The agreed initial interface has exactly four commands: `export`, `validate`, `plan`, and `apply`. They operate on the containing repository's `.github/settings.yml` using the five fields defined on the [settings page](settings.md). See the [concept](concept.md) for decision reasoning, future direction, and unresolved choices.

Saved plan artifacts and drift protection are outside the initial scope. GitHub Action delivery is deferred.

## Proposed interface

These examples illustrate the proposed interface only. They do not describe working software or an installation procedure. Flags, output formats, and the exact YAML wrapper shape remain undecided.

```text
gh settings export
gh settings validate
gh settings plan
gh settings apply
```

| Command | Contract |
| --- | --- |
| `export` | Read current supported settings and write them as configuration. Exclude unsupported fields and read-only metadata. |
| `validate` | Check configuration against the supported schema. Unknown keys and invalid values fail validation. |
| `plan` | Read current settings and show differences for managed fields only. The result is informational. |
| `apply` | Validate configuration, then send declared supported fields directly to the repository update API. Omitted fields are not sent. |

The update endpoint is `PATCH /repos/{owner}/{repo}`.[^github-update-repository] The owner and repository identify the repository containing the configuration; this does not introduce a configurable list of targets.

`apply` does not fetch current settings, compute differences, compare against a saved plan, or preserve external changes to managed values. If a managed value changes remotely after `plan`, `apply` still sends the YAML value. The observed plan diff can therefore differ from changes at apply time. Omitting a field leaves it outside the request.

For example, declaring only `has_issues: false` means that apply sends that supported field with the declared value and sends none of the other four fields. This is a field-level example, not a decision about the full YAML wrapper shape.

## Failures and limits

Unknown keys and invalid values fail before API mutation. Permission and API failures are reported with a nonzero exit status. GitHub may reject a request even when configuration passes local validation; successful validation does not guarantee successful mutation. The endpoint documents failure responses including permission and validation failures.[^github-update-repository]

There is no automatic rollback and no transactional guarantee. A failure does not promise restoration of prior settings. Authentication details remain undecided; the tool must report access failures rather than claim that all declared settings can always be applied.

## Acceptance boundaries for future implementation

- Export includes only the five supported settings as configuration.
- Validation rejects unsupported keys and values of the wrong type before mutation.
- Plan compares only declared fields with current settings and describes an observation, not a binding execution plan.
- Apply sends declared supported values without reading current settings, including when a remote managed value has changed since plan.
- Omitted fields are absent from the update request.
- Permission or API failure produces a nonzero exit status without automatic rollback.

These are intended acceptance conditions, not tests executed by this documentation change. The [settings page](settings.md#configuration-semantics) records the unresolved null handling and YAML shape needed before a complete validator or export format can be implemented.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.
