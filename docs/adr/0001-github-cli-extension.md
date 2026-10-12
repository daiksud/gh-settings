---
type: ADR
title: Deliver as a GitHub CLI extension
description: Record the agreed decision to deliver as a GitHub CLI extension.
---

## Status

Accepted — agreed, not implemented.

## Context

Maintainers need a command-line interface for the initial repository settings workflow.

## Decision

Deliver the initial tool as a GitHub CLI extension invoked as `gh settings`.

## Consequences

The initial delivery target is the GitHub CLI extension interface. Packaging remains unresolved. The deferred GitHub Action requirement is recorded separately in the [future direction](../concept.md#future-direction).
