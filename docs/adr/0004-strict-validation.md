---
type: ADR
title: Validate strictly before mutation
description: Record the agreed decision to validate strictly before mutation.
---

## Status

Accepted — agreed, not implemented.

## Context

Silently accepting unknown keys or invalid values could make a declaration appear to manage settings that it cannot update.

## Decision

Reject unknown configuration keys and invalid values before API mutation.

## Consequences

Typos and unsupported settings are errors. Local validation cannot guarantee API success. The [settings page](../settings.md) defines supported fields; [ADR 0019](0019-reject-explicit-null.md) separately decides null handling for the initial five fields. [ADR 0006](0006-update-api-schema.md) defines the schema mapping independently of validation policy. This policy can change independently of direct apply and rollback behavior.
