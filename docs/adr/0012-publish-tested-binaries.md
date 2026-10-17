---
type: ADR
title: Publish the exact tested binaries
description: Build each release target once and preserve binary identity through publication.
---

## Status

Accepted — implemented by the manual release workflow.

## Context

Rebuilding a product after acceptance can change its bytes, even when the source commit and version match. The existing [main CI](../../.github/workflows/ci-push.yml) builds a validation artifact for its own run, not a multi-platform release artifact.

## Decision

Publish only the exact release binaries that passed acceptance.

## Consequences

The release pipeline builds each platform binary once, then transfers and verifies its identity, including SHA-256, through commit stage, acceptance stage, and publication. Existing composite stages may be reused or refactored, but they must not introduce a second product build for a target. Main CI artifacts are not substitutes for release-built artifacts. Compatible execution and transfer details remain to be designed in the [release specification](../contributing/releases.md).
