---
type: Reference
title: "Collection ownership and pruning"
description: "Collection ownership and pruning declaration, ownership and API contracts."
---

# Collection pruning

Plan and apply normally leave unlisted independent objects alone. Use prune when the declaration should also remove objects missing from a selected collection. For example, preview unlisted labels and Rulesets with `gh settings plan --prune labels,rulesets`. After reviewing the full plan, `gh settings apply --prune labels,rulesets` includes those deletions alongside other declared updates.

## Which objects a scope selects

| Collection in YAML | Effect within a selected prune scope |
| --- | --- |
| Omitted | Preserve every object in the collection. |
| Present and empty (`[]`) | Select every current object in the collection for deletion, subject to that resource's preflight checks. |
| Present with entries | Retain declared identities. Select unlisted objects for deletion. A create-only entry never adopts an existing object just because the names match. |

Use one comma-separated list of scopes, or `all` alone. Empty elements, unknown names, duplicates and mixing `all` with another scope fail validation. The [public Action](../action.md#inputs) takes the same selection in its `prune` input.

## Supported scopes

| Scope | What deletion affects |
| --- | --- |
| [`labels`](labels.md) | Removes labels and their issue and pull request assignments. Undeclared Dependabot-created labels are candidates too. |
| [`rulesets`](rulesets.md#deleting-rulesets) | Removes the entire unlisted Ruleset and its protections. |
| [`actions-policies`](actions/policies.md) | Removes unlisted repository-owned event or actor policies. |
| [`actions-variables`, `agents-variables`, `environment-variables`](actions/variables.md#variable-pruning) | Removes unlisted names in the selected variable family. For Environment variables the parent must be listed and its variable list present. |
| [`environments`](environments.md#environment-pruning) | Removes the whole unlisted Environment, including its configuration and secrets. Listed variable-only parents are retained. |
| [`collaborators`](collaborators.md#collaborator-pruning) | Cancels unlisted invitations and removes direct grants after confirming that the credential can manage them. |
| [`branch-protection-rules`](legacy-classic-protection.md#classic-branch-protection-pruning) | Removes an entire unlisted classic rule, including its unowned protections. |
| [`autolinks`](legacy-autolinks.md#pruning) | Removes unlisted reference IDs. Existing references are immutable. |
| [`deploy-keys`](deploy-keys.md#deploy-key-pruning) | Revokes unlisted deployment keys from this repository. |
| [`webhooks`](webhooks.md#explicit-webhook-pruning) | Removes unlisted whole hooks, including their opaque secrets. |
| [`runners`](actions/runners.md#runner-registration-pruning) | Deregisters unlisted runners and can affect jobs. |

Some fields already own complete sets in ordinary apply, such as topics, a [managed pull request exemption list](moderation.md#pull-request-limit-exemption-list), or an Environment's explicit deployment patterns. See the [configuration semantics](configuration.md#configuration-semantics) and the field's own rules before using an empty list.

## Preflight and partial failures

Apply acquires the complete candidate inventories and checks every selected deletion before any remote write. An unsupported candidate stops the whole operation. Apply performs declared creations and updates before deletions, keeps the deletion set fixed and checks for identity or state changes before deleting each candidate.

Deletion also requires independent confirmation of absence. A rejected or unconfirmed operation stops later work. Earlier writes can remain in place, and a final check cannot lock the resource against concurrent changes. Inspect the current state before retrying. [CLI recovery](../commands.md#recovering-from-a-failed-apply)
