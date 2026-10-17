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

Current source implements export, validation, planning and direct apply. Export saves `.github/settings.yml` silently at the checkout root. That configuration targets only its containing repository. This page defines the five repository fields, topics and labels resources, and versioned YAML contract; see [commands](commands.md) for current behavior and [the published package boundary](github-actions.md#published-package-and-consumer-pins) for released consumers. Future settings remain outside this scope.

## Supported fields

The initial design supports exactly five basic fields under `settings.repository`. Their keys and types follow the corresponding GitHub repository update request body schema; the YAML grouping is a separate configuration decision.[^github-update-repository]

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The three booleans take `true` or `false`. There are no enum-valued fields in the initial design. `validate` rejects values outside these string and boolean types, including explicit null, as decided in [ADR 0019](adr/0019-reject-explicit-null.md). Read-only response metadata and all other repository fields are excluded, even when GitHub exposes them as writable parameters. API defaults do not authorize sending omitted fields.

## Topics

Current source adds `settings.topics.names` without changing `schemaVersion: 1`. Topics support is published in [v0.8.0](https://github.com/daiksud/gh-settings/releases/tag/v0.8.0), which this repository now adopts in its PR/main workflows. Older releases do not support this group.

```yaml
schemaVersion: 1
settings:
  topics:
    names:
      - github-cli
      - settings-as-code
```

`names` follows the [replace-topics request body](https://docs.github.com/en/rest/repos/repos#replace-all-repository-topics). It is an array of unique lowercase strings using letters, numbers and hyphens, each 1–50 characters long, with at most 20 entries. Uppercase names are rejected locally rather than relying on server lowercasing. Null, scalars, non-string elements and duplicate names are invalid. Aliases and merge keys retain the existing YAML semantics. [GitHub's naming rules](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/classifying-your-repository-with-topics) apply to public and private repositories.

| Declaration | Meaning |
| --- | --- |
| Omitted `topics`, or `topics: {}` | Topics unmanaged; no topics read during plan or topics write during apply. |
| `names: [github-cli]` | Manage the complete topic set; replace any other remote topics. |
| `names: []` | Explicitly clear all topics. |
| `names: null` | Invalid; never interpreted as omission or clearing. |

Names are compared as a set. Export, comparisons and PUT payloads use sorted lists without changing the configuration file. An empty observed list is exported as `names: []`; exporting promotes that observed set to a managed declaration. Topic names are public metadata, even on private repositories.

Export reads `GET /repos/{owner}/{repo}`, then `GET /repos/{owner}/{repo}/topics`, followed by all labels pages; all reads and complete serialization finish before overwriting the file. Missing, null, invalid or unreadable topic names fail export instead of becoming an empty set. Plan reads only declared resources in repository-then-topics order. Topic reads need Metadata read (public resources may be read without authentication); writes need Administration write. Existing PR read-token isolation and target-scoped apply credentials are unchanged. No paid-plan prerequisite is documented for topics.

Apply sends a repository PATCH only for declared repository fields, followed by `PUT /repos/{owner}/{repo}/topics` with only `{"names":[...]}` when topic names are declared. It does not GET first. An explicit clear sends `{"names":[]}`, never null. Both write clients reject redirects. On failure, later operations stop and earlier successful updates remain applied; there is no transaction, retry or rollback.

## Labels

Current source adds `settings.labels`, an array of mappings with required `name` and optional `color`/`description` strings. Published v0.8.0 does not support Labels. See the [Labels feature contract](features/labels.feature.md) for schema, API mapping and tests.

Omitting `labels` leaves it unmanaged. A provided list is authoritative: remote labels absent from the list are deleted; `labels: []` deletes all labels. Deleting a label removes its issue/PR associations. A changed name is create plus delete, with no rename inference. Review the explicit create/update/delete plan before apply.

Omitted color/description preserves existing values, or uses GitHub defaults for creation. Empty description clears it; null is invalid. Colors are six hexadecimal characters without `#` and normalized lowercase; descriptions permit at most 100 characters. Names use case-insensitive identity without trimming, and ambiguous names fail. Export reads every page before writing and saves the complete sorted label collection. Apply reconciles a fresh label read after repository PATCH and topics PUT; creates/updates precede deletes, and any failure stops later operations without rollback. Labels listing needs Issues or Pull requests read, while mutations need either permission at write level; Administration write alone is insufficient. This change does not alter credentials or released workflow pins.

## Configuration semantics

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. The source `apply` command sends only declared supported fields. Null is not omission; it fails local validation for these five fields under [ADR 0019](adr/0019-reject-explicit-null.md).

Following [ADR 0020](adr/0020-group-settings-by-resource.md), the YAML document contains a `settings` mapping and a `repository` mapping within it. [ADR 0021](adr/0021-require-schema-version.md) requires root `schemaVersion: 1`, an integer file-format version independent of the CLI release. The five keys inside `repository` match the `Update a repository` request body fields.[^github-update-repository] The following configuration declares one managed field:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
```

Its update request body contains only `{"has_issues": false}`. The version and outer YAML mappings are not sent as API fields. The other four supported fields stay absent and unmanaged; no defaults are injected. A document containing only `schemaVersion: 1`, or also empty `settings` and `repository` mappings, manages no fields. Apply succeeds without an API request when no fields are managed.

The `validate` command distinguishes these declarations:

| YAML under `settings.repository` | Meaning |
| --- | --- |
| Field omitted | Unmanaged; `apply` leaves it out of the request. |
| `has_issues: false` | A declared boolean value, not omission. |
| `description: ""` | A declared empty string, not omission. `description: ''` is equivalent. |
| `description: null` or `has_issues: null` | Invalid for these string and boolean fields. A bare `description:` is also YAML null and invalid. |

The same type rules apply to `homepage`, `has_projects`, and `has_wiki`. Quoted `'null'` is a string value, not YAML null. Validation rejects unknown or duplicate keys and wrong types; it accepts partial declarations and empty mappings. These rules apply to the current five fields only, without defining null behavior for future settings.

Fields at the YAML root, missing or unsupported versions, and unsupported groups fail validation. The required version and grouping are defined by [ADR 0020](adr/0020-group-settings-by-resource.md) and [ADR 0021](adr/0021-require-schema-version.md); no automatic migration or compatibility parsing is provided.

`export` normalizes GET `description` and `homepage` nulls to empty strings, preserving existing empty strings too. It serializes those values as quoted YAML strings (`""`), not empty YAML values or null. This read-side conversion does not make explicit YAML null valid input. The published read schema permits null for these fields, whereas the update body defines optional strings without `nullable`; export follows that writable representation.[^github-repository-openapi] This does not assert that the server rejects every null update, and no live update was performed to verify clearing. Missing response keys and null booleans are errors, not nulls or defaults.

Root keys other than `schemaVersion` and `settings`, and settings groups other than `repository`, `topics` and `labels`, are rejected.

Rulesets, Pages, and Actions settings remain unsupported. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are outside repository scope. Unresolved ruleset collection behavior remains in the [future direction](concept.md#future-direction).

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.
