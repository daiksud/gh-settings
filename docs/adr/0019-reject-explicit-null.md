---
type: ADR
title: Reject explicit null for the first five settings
description: Distinguish null from omission while validating the initial repository update fields.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
    title: GitHub REST API — Update a repository
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
    title: GitHub repository update request schema
  - id: openapi-nullable
    resource: https://spec.openapis.org/oas/v3.0.3.html#schema-object
    title: OpenAPI 3.0.3 Schema Object
---

## Status

Accepted — implemented by local validation for the first five fields.

## Context

The `PATCH /repos/{owner}/{repo}` request schema defines `description` and `homepage` as strings and `has_issues`, `has_projects`, and `has_wiki` as booleans. None declares `nullable: true`; OpenAPI 3.0.3 defaults `nullable` to false. The five request fields are optional, so omission and a present null have different meanings. GitHub's repository GET response separately permits null for `description` and `homepage`. The update API's prose does not describe live null-update behavior.

## Decision

For these five settings, `validate` rejects an explicit YAML null as outside the corresponding update request field type. It accepts omitted fields, explicit false, and empty strings according to their distinct meanings. This is a decision for the current field schema, not a blanket rule for future API endpoints or settings.

## Consequences

An unquoted empty YAML scalar and `null` fail validation for these fields; a quoted empty string and a quoted `"null"` are strings. This local rule does not claim that every null request is rejected by the live GitHub server, and no repository settings were changed to probe it. Future fields must be checked against their own update request schemas. A schema-valid declaration still may fail when GitHub checks permissions or repository state.
