---
type: ADR
title: Distribute GitHub CLI compatible raw executables
description: Select precompiled extension asset format for the initial platforms.
sources:
  - id: github-cli-extensions
    resource: https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions
    title: Creating GitHub CLI extensions
---

## Status

Accepted — implemented by the manual release workflow.

## Context

The extension needs downloadable executables that GitHub CLI can select for each supported platform. GitHub's guide specifies a `gh-` name and OS-architecture suffix for precompiled release assets.[^github-cli-extensions]

## Decision

Publish each initially supported platform as a raw executable asset named `gh-settings-<GOOS>-<GOARCH>`.

## Consequences

For the initial macOS Apple Silicon and Linux x86_64 targets, the names are `gh-settings-darwin-arm64` and `gh-settings-linux-amd64`. The [release specification](../releases.md) also requires checksums and release notes. Archive formats, more platforms, and use of GoReleaser are not selected by this decision.

[^github-cli-extensions]: [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions), checked on 2026-10-12 for the precompiled extension asset convention.
