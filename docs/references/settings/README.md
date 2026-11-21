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
    resource: ../../development/ui-repository-mapping.md
---

# Repository settings reference

Find the field that corresponds to the control you want to manage. Start with the [YAML format and ownership rules](configuration.md) for the required version and a complete declaration. To choose credentials or review a deletion, use the [permission matrix](permissions.md) or the [prune scopes](pruning.md).

## Repository home

| Location | Settings |
| --- | --- |
| [About](about.md) | Description, website and the complete set of topics. |
| [Labels](labels.md) | Names, colors and descriptions of selected labels. |

## Repository Settings

| GitHub page | Settings |
| --- | --- |
| [General](general.md) | Features, default branch, merge methods and commit messages, release immutability and archive state. |
| [Collaborators](collaborators.md) | Direct collaborators, pending invitations and removal. |
| [Moderation](moderation.md) | Persistent pull request limits and their exemption list. |
| [Rulesets](rulesets.md) | Repository-owned rulesets, bypass actors and supported rules. |
| [Actions](actions/README.md) | General controls, self-hosted runners, workflow policies and OIDC subject claims. |
| [Webhooks](webhooks.md) | Existing hooks, creation and explicit whole-hook deletion. |
| [Copilot code review](copilot.md) | Custom instructions, MCP tools and review effort. |
| [Environments](environments.md) | Protection rules, deployment branches and tags, protection Apps and variables. |
| [Pages](pages.md) | Publishing source, custom domain, HTTPS and site lifecycle. |
| [Advanced Security](security.md) | Private vulnerability reporting, Dependabot, CodeQL, AI Scan and Secret Protection. |
| [Deploy keys](deploy-keys.md) | Public keys, write access and removal. |
| Secrets and variables | [Actions variables](actions/variables.md) and [Agents variables](agents.md). Values are plaintext. |

## Existing v0 contracts outside v1

[Autolink references](legacy-autolinks.md) and [classic branch protection](legacy-classic-protection.md) have existing v0 implementations. Read the [support policy](scope.md#feature-adoption-scope) for the v1 target and its exceptions. The [selected CLI or Action release](../../guides/github-actions.md#published-package-and-consumer-pins) determines which fields an installed package can use.
