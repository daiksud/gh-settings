---
type: ADR
title: Validate PR squash messages as Conventional Commits
description: Record the PR squash message format independently of version calculation.
sources:
  - id: conventional-commits
    resource: https://www.conventionalcommits.org/en/v1.0.0/
    title: Conventional Commits 1.0.0
---

# ADR 0016. Validate PR squash messages as Conventional Commits

## Status

Accepted. The `commitlint` check enforces the commit and squash-message format on all PRs.

## Context

Future release automation needs structured change descriptions. Conventional Commits defines a type, optional scope, and description in the title line.[^conventional-commits] The repository uses squash merges and retains the PR title and body as the final commit message.

## Decision

Validate each commit between the event base and head, then the PR title and body together as the proposed squash commit. Use the repository's `@commitlint/config-conventional` preset and its standard warning behavior. Reject HTML comment openers anywhere in the PR body so template instructions and hidden generated text stay out of the squash commit.

## Consequences

The independent `commitlint` check runs on PR creation, reopening, pushes and edits, including drafts and Dependabot updates. Authors can check messages locally using the [development guide](../setup.md#pull-request-messages). Reviewers confirm the final squash message before merge. Either `!` in the header or a `BREAKING CHANGE:` or `BREAKING-CHANGE:` footer can signal a breaking change. [ADR 0017](0017-pr-based-semver.md) records version calculation separately.

[^conventional-commits]: [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/), checked on 2026-10-12 for title structure and breaking-change notation.
