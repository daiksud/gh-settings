---
type: ADR
title: Require an explicit version for initial releases
description: Keep version selection manual in the first release workflow.
---

## Status

Accepted — implemented by the manual release workflow.

## Context

The project needs a release path before automatic version selection has been designed. The [release specification](../releases.md) starts the initial workflow manually.

## Decision

Require an explicit version input for each initial release run.

## Consequences

The supplied version is embedded in the release binaries and identifies the proposed tag and GitHub Release. A conflicting existing identity stops the run under [ADR 0013](0013-immutable-release-identity.md). The exact version and tag spelling remain implementation details. Future automatic selection is a separate decision in [ADR 0017](0017-pr-based-semver.md).
