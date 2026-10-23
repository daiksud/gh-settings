---
type: ADR
title: Observe organization policy before managing private repository forks
description: Adopt a bounded repository allow_forking boolean with explicit export and applicability checks.
---

## Status

Accepted for the bounded #330 implementation in unreleased source. Published v0.17.0 does not support this field or export option. This decision does not authorize live forking-policy changes, organization policy updates, broader credentials or a release.

## Context

The repository API's `allow_forking` boolean is not a universal fork prohibition. GitHub's [repository policy guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/managing-the-forking-policy-for-your-repository) limits the setting to private organization repositories and requires organization permission before either repository choice. Public/personal repositories and hidden policy cannot be normalized to false.

The [organization REST response](https://docs.github.com/en/rest/orgs/orgs#get-an-organization) documents `members_can_fork_private_repositories`. Full details depend on organization-owner visibility; public endpoint success or a token's required-permissions header does not prove that visibility. Classic credentials require `admin:org` for full organization details. Existing credentials must already expose the actual field; the implementation never grants access to obtain it.

## Decision

Adopt `settings.repository.allow_forking` only for completely observed active, non-fork, private organization repositories whose matching organization reports `members_can_fork_private_repositories: true`. Require positive repository/organization IDs, canonical target names, explicit visibility/private/fork/archive booleans and a present repository policy boolean. Missing/null/denied/redirected data or disabled organization policy fails both true and false declarations. Internal repositories, fork-network semantics and lifecycle composition remain unadopted.

Omission is unmanaged; false is explicit repository prohibition. Null and non-booleans are invalid. Ordinary export remains unchanged; `export --forking` opts into this additional observation. Reject composition with `archived` and `--archive` locally. Plan is read-only. Apply runs after other resources, reacquires target/policy before writing, skips matches, and sends only `allow_forking`. Complete PATCH response and independent readback must confirm target and requested state, including continuing organization applicability.

## Consequences

Readback verifies the repository setting, not permission for every actor to create a fork or absence/deletion of existing forks. Organization/enterprise restrictions can still reject a mutation. The tool never creates/deletes forks, changes visibility or organization policy, or modifies fork-workflow token/secrets controls. API name addressing leaves a race window; fresh checks are not an atomic identity lock. Failed pre-write observation makes no policy write; an attempted PATCH or failed readback has uncertain effects. Earlier successful resource writes remain applied, with no automatic retry or rollback.

The [behavior specification](../features/private-forking.feature.md) maps local fixture coverage. No live policy mutation or credential applicability was tested. Future broader support requires explicit observation and lifecycle contracts rather than treating the current bounded adoption as proof.
