---
type: Reference
title: Repository settings reference
description: Define the current-source UI-oriented declaration, ownership, validation and API boundaries.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: github-labels
    resource: https://docs.github.com/en/rest/issues/labels
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
  - id: github-actions-policies
    resource: https://docs.github.com/en/rest/actions/policies
  - id: ui-inventory
    resource: ../contributing/ui-repository-mapping.md
---

Use this index to find the declaration and behavior contract for each setting.
The [support policy](scope.md#feature-adoption-scope) separates v1 support from
existing v0 implementation and evidence limits. The selected published CLI or
Action defines the usable package contract.

Start with [YAML structure and value semantics](configuration.md),
[collection ownership and pruning](pruning.md), and the
[GitHub App permission matrix](permissions.md). The matrix separates acquisition
from apply/prune and records compound grants and credential limitations.

## Repository home

- [About: Description, Website and Topics](about.md)
- [Labels](labels.md)

## Settings in sidebar order

| Setting group | Reference |
| --- | --- |
| General | [General settings](general.md) |
| Collaborators | [Personal repository collaborators](collaborators.md) |
| Moderation / Interaction limits | [Moderation settings](moderation.md) |
| Rulesets | [Ruleset configuration](rulesets.md) |
| Actions | [Actions settings](actions.md) |
| Webhooks | [Repository webhooks](webhooks.md) |
| Copilot code review | [Copilot code review settings](copilot.md) |
| Environments | [Environment settings](environments.md) |
| Pages | [Pages settings](pages.md) |
| Advanced Security | [Advanced Security settings](security.md) |
| Deploy keys | [Repository deploy keys](deploy-keys.md) |
| Secrets and variables / Agents | [Agents variables](agents.md) |

Actions variables remain in the [Actions reference](actions.md#actions-repository-variables).
Navigation follows the applicable Settings UI; the YAML field contract remains
[explicitly defined](configuration.md#supported-fields). An index entry is not a
claim that every control on that page is implemented.

## Existing v0 contracts outside v1

- [Autolink references](legacy-autolinks.md): excluded under the confirmed Free-plan scope.
- [Classic branch protection](legacy-classic-protection.md): excluded by product choice; use Rulesets in v1.

These references preserve existing implementation and evidence. They do not
expand v1 scope or authorize migration, deletion, access changes or live apply.
Already-archived targets and other exclusions remain in the [support policy](scope.md#feature-adoption-scope).
