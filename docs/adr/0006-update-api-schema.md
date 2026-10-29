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

The public YAML key and enum spelling decision is superseded by [Issue #479](https://github.com/daiksud/gh-settings/issues/479). The [current configuration reference](../settings.md) defines UI-oriented names and choices; API conversion remains internal. Declared-field ownership remains a separate decision. The original adoption status follows for historical context.

Accepted for field keys and types. The YAML root-placement consequence below was superseded by [ADR 0020](0020-group-settings-by-resource.md). The current-source [PR access extension](../features/pull-request-access.feature.md) explicitly maps its two YAML fields to GraphQL inputs and defines mixed REST/GraphQL write ordering; the five-field historical scope below remains unchanged.

## Context

Matching API field names alone leaves room for a different nesting structure in configuration. The current five settings are body parameters of GitHub's `Update a repository` endpoint.[^github-update-repository] The declaration should follow the corresponding update request schema rather than introduce an independent model.

## Decision

YAML keys and data structures follow the corresponding GitHub update API request schema, including field types and any enum values.

## Consequences

For the current five-field scope, supported fields appear at the YAML root, matching the repository update request body object; there is no additional `repository` wrapper. See the [settings contract](../settings.md#configuration-semantics) for the proposed configuration example.

Schema alignment does not mean supporting every API parameter. Only `description`, `homepage`, `has_issues`, `has_projects`, and `has_wiki` are in the current design. Read-only response metadata is excluded.

[Declared-field ownership](0002-managed-fields.md) remains a separate decision: omitted fields stay absent and unmanaged, with no default injection. [ADR 0019](0019-reject-explicit-null.md) separately decides null handling for the initial five fields. Future composition across multiple update endpoints remains unresolved. This decision does not add labels, rulesets, or other settings to the current scope.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five supported body parameters and request structure.
