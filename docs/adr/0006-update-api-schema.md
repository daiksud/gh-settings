---
type: ADR
title: Follow the update API request schema
description: Record the decision to align supported configuration fields with GitHub update request fields.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
---

## Status

Accepted — agreed, not implemented.

## Context

The current five settings are writable parameters of GitHub's `Update a repository` endpoint.[^github-update-repository] Matching their names and value types to that endpoint keeps the supported fields aligned with the API while excluding response-only metadata.

## Decision

Configuration field names, value types, and enum values follow the corresponding GitHub update API request schema.

## Consequences

The supported fields are exactly `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki`. No other update parameters or read-only response metadata are included. This decision does not choose the YAML document's wrapper or nesting shape; that remains open in the [settings contract](../settings.md#configuration-semantics).

Schema alignment does not mean supporting every API parameter. Only `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki` are in the current design. Read-only response metadata is excluded.

[Declared-field ownership](0002-managed-fields.md) remains a separate decision: omitted fields stay unmanaged, with no default injection. Null handling, the YAML wrapper shape, and future composition across multiple update endpoints remain unresolved. This decision does not add labels, rulesets, or other settings to the current scope.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-12 for the five supported body parameters and their names and types.
