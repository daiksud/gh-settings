---
type: Specification
title: Repository settings
description: Define supported settings and configuration ownership semantics.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
---

## Scope and status

`gh settings` is an unimplemented GitHub CLI extension. Configuration lives in `.github/settings.yml` and targets only the repository containing it. This page defines the fields supported in the agreed initial design, not implemented functionality. See the [commands](commands.md) for the proposed interface and the [concept](concept.md) for future direction and unresolved decisions.

## Supported fields

The initial design supports exactly five basic fields. These names and types match the body parameters of GitHub's repository update API.[^github-update-repository]

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The three booleans take `true` or `false`. There are no enum-valued fields in the initial design. Null handling remains unresolved. Read-only response metadata and all other settings are excluded, even when GitHub exposes them as writable parameters. API defaults do not authorize sending omitted fields.

## Configuration semantics

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. Apply sends only declared supported fields; it does not send omitted fields. Null is not omission, and null handling remains unresolved.

The exact YAML wrapper shape has not been decided. Field-level examples on the commands page do not define a complete configuration schema. Null handling and the YAML shape need decisions before a complete validator or export format can be implemented.

Labels, rulesets, Pages, and Actions settings are excluded from the initial design. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are outside repository scope. Confirmed future label semantics and unresolved ruleset collection behavior remain in the [future direction](concept.md#future-direction).

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters. This reference does not expand the supported scope.
