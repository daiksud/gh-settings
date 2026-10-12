---
type: ADR
title: Pin each release to one main commit
description: Prevent a moving main branch from changing the release source mid-run.
---

## Status

Accepted — implemented by the manual release workflow.

## Context

`main` can advance while a release is building and validating. Reading it again before tagging could select code different from the tested code.

## Decision

Record the latest `main` commit when a release workflow starts and use that exact commit throughout the run.

## Consequences

Every source checkout, platform build, test, tag, and release operation refers to the recorded commit. A newer `main` commit waits for another release. Pinning source does not prove binary identity; [ADR 0012](0012-publish-tested-binaries.md) requires publication of the tested bytes.
