---
type: ADR
title: Follow the update API request schema
description: Record the decision to follow GitHub update request keys and data structures in YAML.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
---

## Status

Accepted for field keys and types. The YAML root-placement consequence below was superseded by [ADR 0020](0020-group-settings-by-resource.md).

## Context

Matching API field names alone leaves room for a different nesting structure in configuration. The current five settings are body parameters of GitHub's `Update a repository` endpoint.[^github-update-repository] The declaration should follow the corresponding update request schema rather than introduce an independent model.

## Decision

YAML keys and data structures follow the corresponding GitHub update API request schema, including field types and any enum values.

## Consequences

For the current five-field scope, supported fields appear at the YAML root, matching the repository update request body object; there is no additional `repository` wrapper. See the [settings contract](../settings.md#configuration-semantics) for the proposed configuration example.

Schema alignment does not mean supporting every API parameter. Only `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki` are in the current design. Read-only response metadata is excluded.

[Declared-field ownership](0002-managed-fields.md) remains a separate decision: omitted fields stay absent and unmanaged, with no default injection. Null handling and future composition across multiple update endpoints remain unresolved. This decision does not add labels, rulesets, or other settings to the current scope.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five supported body parameters and request structure.
