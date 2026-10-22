---
type: ADR
title: Require an explicit version for initial releases
description: Keep version selection manual in the first release workflow.
---

## Status

Superseded for version selection by [ADR 0017](0017-pr-based-semver.md). The workflow remains manually dispatched; fixed source, shared Action/CLI version, and immutable release identity remain required.

## Context

The project needed a release path before automatic version selection was designed. The initial workflow started manually with an explicitly selected version.

## Decision

Require an explicit version input for each initial release run.

## Consequences

The supplied version was embedded in the release binaries and identified the proposed tag and GitHub Release. A conflicting existing identity stopped the run under [ADR 0013](0013-immutable-release-identity.md). Automatic selection now replaces the operator-supplied version without removing those checks; the [release guide](../contributing/releases.md) describes the current procedure.
