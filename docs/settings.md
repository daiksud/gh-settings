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

`gh settings` currently implements read-only export; validation, planning, and mutation remain proposed. Configuration lives in `.github/settings.yml` and targets only the repository containing it. This page defines the five fields available to export and the agreed configuration contract for future commands. See the [commands](commands.md) for the proposed interface and the [concept](concept.md) for future direction and unresolved decisions.

## Supported fields

The initial design supports exactly five basic fields under `settings.repository`. Their keys and types follow the corresponding GitHub repository update request body schema; the YAML grouping is a separate configuration decision.[^github-update-repository]

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The three booleans take `true` or `false`. There are no enum-valued fields in the initial design. Explicit null handling in future YAML input remains unresolved. Read-only response metadata and all other settings are excluded, even when GitHub exposes them as writable parameters. API defaults do not authorize sending omitted fields.

## Configuration semantics

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. Apply sends only declared supported fields; it does not send omitted fields. Null is not omission; explicit null handling in future YAML input remains unresolved.

Following [ADR 0020](adr/0020-group-settings-by-resource.md), the YAML document contains a `settings` mapping and a `repository` mapping within it. [ADR 0021](adr/0021-require-schema-version.md) requires root `schemaVersion: 1`, an integer file-format version independent of the CLI release. The five keys inside `repository` match the `Update a repository` request body fields.[^github-update-repository] The following proposed configuration declares one managed field:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
```

Its future update request body would contain only `{"has_issues": false}`. The version and outer YAML mappings are not sent as API fields. The other four supported fields stay absent and unmanaged; no defaults are injected. A document containing only `schemaVersion: 1`, or also empty `settings` and `repository` mappings, manages no fields. Sending an update is not implemented.

Earlier releases of `export` wrote the five fields directly at the YAML root. To use this version, add `schemaVersion: 1` and move those fields under `settings.repository` as shown above. The planned `validate` command will reject old flat documents and missing or unsupported versions. No automatic migration or compatibility parsing is provided.

Export normalizes GET `description` and `homepage` nulls to empty strings, preserving existing empty strings too. Empty strings are serialized as YAML strings, not empty YAML values or null. The published read schema permits null for these fields, whereas the update body defines optional strings without `nullable`; export follows that writable representation.[^github-repository-openapi] This does not assert that the server rejects every null update, and no live update was performed to verify clearing. Missing response keys are errors, not nulls or defaults.

Explicit null handling in YAML input for future commands remains unresolved. Root keys other than `schemaVersion` and `settings`, and other groups within `settings`, are not supported by the agreed format. How future settings from multiple update endpoints would compose remains unresolved; this does not broaden the current five-field schema.

Labels, rulesets, Pages, and Actions settings are excluded from the initial design. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are outside repository scope. Confirmed future label semantics and unresolved ruleset collection behavior remain in the [future direction](concept.md#future-direction).

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.
