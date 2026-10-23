---
type: ADR
title: Manage custom property values only with complete schema evidence
description: Separate declared repository values from organization schema and inherited defaults.
---

## Status

Accepted for the bounded current-source implementation of #354. Published v0.17.0 rejects the group and export option. Default-backed, required, explicitly required, enterprise-sourced, URL and regex-constrained properties remain unadopted; #354 remains open for those decisions.

## Context

Repository property values have a documented string/array/null write union. Null unsets a value, while omission preserves it. Organization constraints determine valid types, selections and editing rights. The value response does not identify whether a value was explicitly assigned or inherited from a default. Treating an effective default as explicit intent would break unset and round trips.

The REST schema also does not document the string regex constraint exposed by GraphQL. Missing constraints, hidden data and unrelated endpoint success cannot establish permission or absence of constraints.

## Decision

Adopt `settings.custom_properties` as a map that owns only named values. Preserve null, empty string and empty array as distinct intents. An empty map owns nothing. Keep schema creation, editing and deletion outside this resource.

Require complete organization schema and repository value collections plus stable repository/organization identities. Adopt only organization-sourced, repository-editable properties with explicitly observed false required/require-explicit flags and a null default. Require complete selection choices. Supplement each string schema with the documented GraphQL regex field and matching organization/property identity; reject unavailable constraints or nonempty regex patterns. Do not infer unconstrained behavior from an omitted field.

Use explicit export opt-in. Export every observed schema name, including null for an unset value, only if the whole export is supported. Plan validates selected names against schema and compares native values. Apply uses fresh observations, sends declared names only, and verifies values, retained unowned values, selected schema and identity afterward. The final pre-write reread rejects intervening observation changes; it is not an atomic lock and cannot eliminate a race at the name-addressed endpoint. No schema mutation, hidden retry or rollback is introduced.

## Consequences

Repositories whose requested constraints cannot be established fail safely. Ordinary export and declarations without this resource retain their permissions. A full opt-in export can fail on an unsupported organization property even when only other properties would be editable. Selected supported names can be managed without owning other names, but complete value reads and string-constraint visibility are still required.

The [configuration reference](../settings.md#custom-property-values-unreleased-source) owns the public syntax; the [behavior specification](../features/custom-properties.feature.md) maps tests and official contracts. Local fixtures prove request, failure and presentation behavior only. No live property/schema changes, credentials, grants or release are part of this implementation.
