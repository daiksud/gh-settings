---
type: ADR
title: Implement in Go
description: Record the agreed implementation language independently of delivery and library choices.
sources:
  - id: github-cli-extensions
    resource: https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions
    title: Creating GitHub CLI extensions
---

## Status

Accepted — agreed, not implemented.

## Context

The maintainer chose Go for the new tool. GitHub's extension guide documents a Go-based precompiled extension path.[^github-cli-extensions] The delivery decision in [ADR 0001](0001-github-cli-extension.md) did not previously settle the implementation language.

## Decision

Implement `gh settings` in Go.

## Consequences

Future implementation will use the Go toolchain. Toolchain version and release packaging remain undecided. This decision does not select a command framework, YAML package, or GitHub API library; [ADR 0008](0008-official-go-gh-library.md) records the API library choice separately.

Prefer the Go standard library where sufficient. No official YAML package has been verified for this design, so the YAML dependency remains undecided. This documentation change adds no Go code, module files, or dependencies.

[^github-cli-extensions]: [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions), checked on 2026-10-11 for the documented Go extension path.
