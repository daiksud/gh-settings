---
type: ADR
title: Group repository settings within settings
description: Place API-aligned repository fields under settings.repository in configuration.
---

## Status

Accepted — implemented by export and validate. Supersedes the root-placement consequence of [ADR 0006](0006-update-api-schema.md). [ADR 0021](0021-require-schema-version.md) subsequently defines the first root metadata key.

## Context

The repository update API accepts the five current fields directly in its request body. ADR 0006 therefore placed them at the YAML root. Configuration also needs a stable container for settings from other resources without claiming that those resources share one API request body.

## Decision

Use `settings` as the root settings container and `settings.repository` for the five supported repository fields. Keys and types within `repository` continue to follow the repository update request schema. Future groups such as `settings.topics` or `settings.labels` may be designed separately; they are not supported by this decision. Reserve the YAML root for possible future metadata without defining any metadata now.

## Consequences

`export` emits the nested structure, and `validate` rejects the earlier flat structure and unknown keys or groups. Existing flat configuration must move its five fields under `settings.repository`. Omitted mappings and empty mappings manage no fields; explicit false and empty strings remain declared values. The outer `settings` and `repository` keys are configuration grouping and are not PATCH body fields. No other settings, commands, or migration mechanism are added.
