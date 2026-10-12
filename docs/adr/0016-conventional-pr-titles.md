---
type: ADR
title: Use strict Conventional Commits PR titles
description: Record the future PR title format independently of version calculation.
sources:
  - id: conventional-commits
    resource: https://www.conventionalcommits.org/en/v1.0.0/
    title: Conventional Commits 1.0.0
---

## Status

Accepted — future work, not implemented.

## Context

Future release automation needs structured change descriptions. Conventional Commits defines a type, optional scope, and description in the title line.[^conventional-commits] The repository currently has no release automation that enforces a PR title format.

## Decision

Require PR titles to conform strictly to the Conventional Commits format when future automatic versioning is introduced.

## Consequences

Title validation and its exact integration point remain future work. This decision does not require `!` in every title or settle how every breaking change will be signaled. [ADR 0017](0017-pr-based-semver.md) records version calculation separately.

[^conventional-commits]: [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/), checked on 2026-10-12 for title structure and breaking-change notation.
