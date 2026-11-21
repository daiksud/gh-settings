---
type: Reference
title: Supported repositories and settings
description: The v1 target, explicit exclusions, and the difference between product scope and current support.
---

# Supported repositories and settings

Use the [settings index](README.md) to check which fields current source accepts. A released CLI or pinned Action supports the fields implemented in that package. The [coverage ledger](../../development/settings-coverage.md) records the target UI, API evidence and implementation status separately.

## Feature adoption scope

The [v1 goal](https://github.com/daiksud/gh-settings/issues/668) targets **active, personally owned public repositories**. A setting is eligible if the owner can configure it in that repository's GitHub Web UI. Account plan, Preview status, feature execution entitlement, subscriptions and billing are separate from configuration availability. An API field alone does not establish a UI control.

Supporting a setting end to end requires the applicable official APIs to observe and update it. Evidence for only one direction leaves the other unconfirmed. The caller must already have the necessary [permissions](permissions.md). A family may define a meaning for HTTP 404 under that precondition. Explicit denials, incomplete observations and failed verification remain errors for the affected read or write.

## Repositories and controls outside the target

| Area | Boundary |
| --- | --- |
| Already-archived repositories | Ordinary export, plan and apply are outside v1 support. Limited reads and matching operations can still succeed. The explicit [archive and unarchive transition](general.md#repository-archive-lifecycle) has its own contract. |
| Autolink references | The [GitHub UI guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/configuring-autolinks-to-reference-external-resources) lists paid-plan availability, excluding Free. This specific v1 exclusion retains the [existing v0 contract](legacy-autolinks.md). |
| Classic branch protection | [Rulesets](rulesets.md) provide v1 branch protection. The existing [v0 classic controls](legacy-classic-protection.md) remain implemented. Omitting a classic declaration leaves live rules unmanaged. |
| Temporary interaction restrictions | Their finite duration restores normal access automatically. They are excluded by product policy despite the available read, set and remove APIs. [Persistent pull request limits](moderation.md) are a separate setting. |
| Organization or private-only controls | Organization settings, teams, organization-only custom properties, inherited organization Rulesets and private-repository-only controls fall outside the personal-public target. |
| Repository identity, feature files and secrets | Repository rename, visibility, transfer and deletion, authoring GitHub feature configuration files, and managing secret values are excluded. gh-settings manages its own declaration file. |

## Implementation boundaries

The public [collaborator contract](collaborators.md) covers direct users and invitations. [Environment protection Apps](environments.md#custom-protection-apps) can be selected from existing bindings and available Apps. Installing an App or increasing its grants is outside that operation.

Some eligible controls still lack an implementation. Additional configurable custom Ruleset rules are adopted future work where the UI and applicable APIs permit them. Pages visibility and Environment rename are unsupported. The ledger explains what evidence is missing for particular controls. A planned field becomes usable only when the selected package accepts it.
