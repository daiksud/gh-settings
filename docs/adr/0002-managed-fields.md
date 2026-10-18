---
type: ADR
title: Manage only declared fields
description: Record the agreed decision to manage only declared fields.
---

## Status

Accepted — implemented. See the [current architecture](../contributing/architecture.md); the context below records the original decision.

## Context

Maintainers may want to adopt settings as code for only part of a repository, leaving other values outside tool ownership.

## Decision

Treat fields written in YAML as managed and authoritative; omitted fields at any nesting depth are unmanaged.

## Consequences

Declared values express the desired values. Omitted fields are not sent by apply. For retained labels, omitted attributes preserve their current values; collection membership has its own [resource-specific semantics](../settings.md#labels). Null must not be inferred from omission. [ADR 0019](0019-reject-explicit-null.md) separately decides explicit null handling for the initial five fields.
