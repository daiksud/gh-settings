# Architecture decision records

These records explain why a design was accepted and what tradeoffs it introduced. They are contributor reference material; their original context and rollout notes may describe an earlier implementation. Read the [architecture guide](../contributing/architecture.md) for the current design and the [user documentation](../README.md#for-users) for supported behavior.

Keep decision numbers and paths stable. Record a superseding decision and update status links when an accepted decision changes, rather than rewriting its historical context. Proposed implementation work belongs in issues; current development and release instructions belong in the [contributor guides](../README.md#for-contributors).

| ADR | Decision |
| --- | --- |
| 0001 | [Deliver as a GitHub CLI extension](0001-github-cli-extension.md) |
| 0002 | [Manage only declared fields](0002-managed-fields.md) |
| 0003 | [Apply declared values directly](0003-direct-apply.md) |
| 0004 | [Validate strictly before mutation](0004-strict-validation.md) |
| 0005 | [Do not roll back automatically](0005-no-automatic-rollback.md) |
| 0006 | [Follow the update API request schema](0006-update-api-schema.md) |
| 0007 | [Implement in Go](0007-go-implementation.md) |
| 0008 | [Use the official GitHub CLI Go library](0008-official-go-gh-library.md) |
| 0009 | [Use the YAML organization's Go YAML v4 library](0009-yaml-v4-library.md) |
| 0010 | [Require an explicit version for initial releases](0010-explicit-release-version.md) |
| 0011 | [Pin each release to one main commit](0011-pin-release-source.md) |
| 0012 | [Publish the exact tested binaries](0012-publish-tested-binaries.md) |
| 0013 | [Never overwrite an existing release tag](0013-immutable-release-identity.md) |
| 0014 | [Publish only a complete draft release](0014-draft-release-publication.md) |
| 0015 | [Distribute GitHub CLI compatible raw executables](0015-raw-extension-binaries.md) |
| 0016 | [Validate PR squash messages as Conventional Commits](0016-conventional-pr-titles.md) |
| 0017 | [Derive future SemVer bumps from merged PRs](0017-pr-based-semver.md) |
| 0018 | [Never overwrite published release assets](0018-preserve-published-assets.md) |
| 0019 | [Reject explicit null for the first five settings](0019-reject-explicit-null.md) |
| 0020 | [Group repository settings within settings](0020-group-settings-by-resource.md) |
| 0021 | [Require a configuration schema version](0021-require-schema-version.md) |
| 0022 | [Owner-approved settings workflows](0022-owner-approved-actions.md) |
| 0023 | [Public composite Action contract](0023-public-composite-action.md) |
| 0024 | [Require focused PRs and explicit breaking-change reviews](0024-code-review-policy.md) |
| 0025 | [Reconcile repository Rulesets as declarative collections](0025-declarative-repository-rulesets.md) |
| 0026 | [Require explicit, complete security observations](0026-explicit-security-observations.md) |
| 0027 | [Isolate repository lifecycle transitions](0027-repository-lifecycle.md) |
| 0028 | [Manage existing Pages configuration with explicit lifecycle boundaries](0028-declarative-pages-settings.md) |
| 0029 | [Separate Environment protection ownership from lifecycle](0029-environment-ownership-and-protection.md) |
| 0031 | [Separate direct repository grants from effective access](0031-repository-access-ownership.md) |
