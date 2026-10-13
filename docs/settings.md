---
type: Specification
title: Repository settings
description: Define supported settings and configuration ownership semantics.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
    title: GitHub repository read and update schemas
---

## Scope and status

The published v0.2.0 release and current source implement read-only export and local validation with a versioned configuration format. Planning and mutation remain proposed. Configuration lives in `.github/settings.yml` and targets only the repository containing it. This page defines the five fields available to export and the versioned configuration contract. See the [commands](commands.md) for the proposed interface and the [concept](concept.md) for future direction and unresolved decisions.

## Supported fields

The initial design supports exactly five basic fields under `settings.repository`. Their keys and types follow the corresponding GitHub repository update request body schema; the YAML grouping is a separate configuration decision.[^github-update-repository]

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The three booleans take `true` or `false`. There are no enum-valued fields in the initial design. `validate` rejects values outside these string and boolean types, including explicit null, as decided in [ADR 0019](adr/0019-reject-explicit-null.md). Read-only response metadata and all other settings are excluded, even when GitHub exposes them as writable parameters. API defaults do not authorize sending omitted fields.

## Configuration semantics

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. The proposed `apply` command would send only declared supported fields; it does not exist in the current CLI. Null is not omission; it fails local validation for these five fields under [ADR 0019](adr/0019-reject-explicit-null.md).

Following [ADR 0020](adr/0020-group-settings-by-resource.md), the YAML document contains a `settings` mapping and a `repository` mapping within it. [ADR 0021](adr/0021-require-schema-version.md) requires root `schemaVersion: 1`, an integer file-format version independent of the CLI release. The five keys inside `repository` match the `Update a repository` request body fields.[^github-update-repository] The following configuration declares one managed field:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
```

Its future update request body would contain only `{"has_issues": false}`. The version and outer YAML mappings are not sent as API fields. The other four supported fields stay absent and unmanaged; no defaults are injected. A document containing only `schemaVersion: 1`, or also empty `settings` and `repository` mappings, manages no fields. Sending an update is not implemented.

The `validate` command in v0.2.0 and the current source distinguishes these declarations:

| YAML under `settings.repository` | Meaning |
| --- | --- |
| Field omitted | Unmanaged; the proposed `apply` would leave it out of the request. |
| `has_issues: false` | A declared boolean value, not omission. |
| `description: ""` | A declared empty string, not omission. `description: ''` is equivalent. |
| `description: null` or `has_issues: null` | Invalid for these string and boolean fields. A bare `description:` is also YAML null and invalid. |

The same type rules apply to `homepage`, `has_projects`, and `has_wiki`. Quoted `'null'` is a string value, not YAML null. Validation rejects unknown or duplicate keys and wrong types; it accepts partial declarations and empty mappings. These rules apply to the current five fields only, without defining null behavior for future settings.

Releases before v0.2.0 wrote the five fields directly at the YAML root. To validate one of those earlier files with v0.2.0 or the current source, add `schemaVersion: 1` and move those fields under `settings.repository` as shown above; the old flat document fails validation. Missing or unsupported versions also fail validation. No automatic migration or compatibility parsing is provided.

In v0.2.0 and the current source, `export` normalizes GET `description` and `homepage` nulls to empty strings, preserving existing empty strings too. It serializes those values as quoted YAML strings (`""`), not empty YAML values or null. This read-side conversion does not make explicit YAML null valid input. The published read schema permits null for these fields, whereas the update body defines optional strings without `nullable`; export follows that writable representation.[^github-repository-openapi] This does not assert that the server rejects every null update, and no live update was performed to verify clearing. Missing response keys and null booleans are errors, not nulls or defaults.

Root keys other than `schemaVersion` and `settings`, and other groups within `settings`, are not supported yet. How future settings from multiple update endpoints would compose remains unresolved; the grouping does not broaden the current five-field schema.

Labels, rulesets, Pages, and Actions settings are excluded from the initial design. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are outside repository scope. Confirmed future label semantics and unresolved ruleset collection behavior remain in the [future direction](concept.md#future-direction).

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.
