---
type: ADR
title: Deliver as a GitHub CLI extension
description: Record the agreed decision to deliver as a GitHub CLI extension.
---

## Status

Accepted — implemented. See the [current architecture](../contributing/architecture.md); the context below records the original decision.

## Context

Maintainers need a command-line interface for the initial repository settings workflow.

## Decision

Deliver the initial tool as a GitHub CLI extension invoked as `gh settings`.

## Consequences

The initial delivery target is the GitHub CLI extension interface. Packaging remains unresolved. [ADR 0007](0007-go-implementation.md) selects Go independently of this delivery decision. The later public GitHub Action delivery is recorded separately in [ADR 0023](0023-public-composite-action.md).
