---
type: ADR
title: Do not roll back automatically
description: Record the agreed decision against automatic rollback.
---

## Status

Accepted — implemented. See the [current architecture](../contributing/architecture.md); the context below records the original decision.

## Context

Permission and API failures need an explicit failure contract without promising restoration of prior settings.

## Decision

Do not perform automatic rollback after an apply failure.

## Consequences

Report permission and API failures with a nonzero exit status. Do not promise transactional behavior or that a failed operation restores prior settings. Maintainers must inspect failures and decide how to recover. This decision does not determine validation policy or require an apply-time read.
