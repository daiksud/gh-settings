---
type: Specification
title: Settings MVP
description: Define the five supported settings and four proposed commands.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
---

## Scope and status

The MVP is an unimplemented GitHub CLI extension, `gh settings`, operating on `.github/settings.yml` in the repository containing it. It manages repository settings only. See the [concept](concept.md) for ownership principles, future direction, and unresolved choices.

Exactly five basic fields are supported. These names and types match the body parameters of GitHub's repository update API.[^github-update-repository]

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The three booleans take `true` or `false`. There are no enum-valued fields in this MVP. Null handling remains unresolved. Read-only response metadata and all other settings are excluded, even when GitHub exposes them as writable parameters. API defaults do not authorize sending omitted fields.

Labels, rulesets, Pages, Actions settings, Organization settings, multi-repository management, saved plan artifacts, and drift protection are not part of the MVP. GitHub Action delivery is deferred.

## Proposed commands

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

These are intended acceptance conditions, not tests executed by this documentation change. Null handling and the exact YAML shape need decisions before a complete validator or export format can be implemented.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters, update endpoint, and failure responses. This reference does not expand the supported scope.
