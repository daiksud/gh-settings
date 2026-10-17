---
type: ADR
title: Use the official GitHub CLI Go library
description: Select go-gh for GitHub API access and GitHub CLI authentication integration.
sources:
  - id: github-cli-extensions
    resource: https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions
    title: Creating GitHub CLI extensions
  - id: go-gh-readme
    resource: https://github.com/cli/go-gh/blob/aadf36735579aa6ad2343b2e1b0f5387c22c71f0/README.md
    title: GitHub CLI go-gh README
  - id: go-gh-rest-client
    resource: https://github.com/cli/go-gh/blob/aadf36735579aa6ad2343b2e1b0f5387c22c71f0/pkg/api/rest_client.go
    title: GitHub CLI go-gh REST client
---

## Status

Accepted — implemented. See the [current architecture](../contributing/architecture.md); the context below records the original decision.

## Context

The maintainer requested official libraries. GitHub's extension guide recommends `go-gh` for Go extensions.[^github-cli-extensions] The library lives in the GitHub CLI project's `cli/go-gh` repository; its current module path is `github.com/cli/go-gh/v2`, and its API clients use GitHub CLI authentication conventions.[^go-gh-readme] This establishes the official GitHub CLI project provenance and fit for the extension.

## Decision

Use `github.com/cli/go-gh/v2` for GitHub API access and integration with GitHub CLI authentication.

## Consequences

The REST client resolves host and credentials from the `gh` environment and configuration and exposes GET and PATCH methods.[^go-gh-rest-client] The library can support export and plan reads, and a direct apply PATCH without a preceding current-settings fetch. Library selection does not change the [command contracts](../commands.md) or [configuration schema](../settings.md).

The library's repository helper respects `GH_REPO`.[^go-gh-readme] Adopting it does not authorize a target override: the tool still targets only the repository containing `.github/settings.yml`. Detailed target resolution and credential/error presentation remain to be designed within that boundary. Remaining authentication details and the deferred Action's authentication design are not settled by this choice.

This is GitHub CLI project-maintained tooling. A GitHub API client hosted by another organization, such as `google/go-github`, must not be described as GitHub-official solely because it accesses GitHub. Prefer standard library facilities where sufficient; no command framework is selected here. The YAML dependency is selected separately in [ADR 0009](0009-yaml-v4-library.md). The dependency version will be pinned during implementation; this PR adds no dependencies or implementation.

[^github-cli-extensions]: [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions), checked on 2026-10-11 for its go-gh recommendation.
[^go-gh-readme]: [GitHub CLI go-gh README](https://github.com/cli/go-gh/blob/aadf36735579aa6ad2343b2e1b0f5387c22c71f0/README.md), checked on 2026-10-11 for purpose, v2 imports, authentication conventions, and `GH_REPO` behavior. The source SHA records evidence, not a selected dependency version.
[^go-gh-rest-client]: [GitHub CLI go-gh REST client](https://github.com/cli/go-gh/blob/aadf36735579aa6ad2343b2e1b0f5387c22c71f0/pkg/api/rest_client.go), checked on 2026-10-11 for configuration resolution and direct GET/PATCH methods.
