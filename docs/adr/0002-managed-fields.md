---
type: ADR
title: Manage only declared fields
description: Record the agreed decision to manage only declared fields.
---

## Status

Accepted — agreed, not implemented.

## Context

Maintainers may want to adopt settings as code for only part of a repository, leaving other values outside tool ownership.

## Decision

Treat fields written in YAML as managed and authoritative; omitted fields at any nesting depth are unmanaged.

## Consequences

Declared values express the desired values. Omitted fields are not sent by apply. For future retained labels, omitted attributes preserve their current values; collection membership has its own [future semantics](../concept.md#future-direction). Null must not be inferred from omission. [ADR 0019](0019-reject-explicit-null.md) separately decides explicit null handling for the initial five fields.
