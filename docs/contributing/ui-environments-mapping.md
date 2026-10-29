---
type: Reference
title: Prepared Environment UI mappings
description: Internal Environment controls, separate pattern ownership and activation handoff.
sources:
  - id: environments-ui
    resource: https://github.com/daiksud/gh-settings/settings/environments
  - id: current-contract
    resource: ../settings.md
---

## Preparation boundary

This Environment mapping was prepared for [#487](https://github.com/daiksud/gh-settings/issues/487) and is now composed by current-source UI YAML activation in [#479](https://github.com/daiksud/gh-settings/issues/479). The [public contract](../settings.md#environments) retains API-controlled omissions and separate pattern ownership. [#480](https://github.com/daiksud/gh-settings/issues/480) normalization/warnings and [#481](https://github.com/daiksud/gh-settings/issues/481) complete reporting remain separate. No compatibility alias, migration or new family is added.

The [UI model](../../internal/settings/uimodel/environments.go) has display terminology without transport tags. [Adapter conversions](../../internal/github/ui_environments.go) translate desired values through the existing declaration validator. Acquired conversion uses observation values directly, never the desired constructor. `PreparedEnvironment.ID` and its pattern states retain acquired numeric IDs outside desired UI fields.

## Verified UI controls

English controls were inspected on 2026-10-28 in an existing gh-settings Environment. The Required reviewers and Wait timer groups were revealed in an unsaved form; a reload confirmed their original unchecked state. The deployment mode menu and an empty Add deployment branch or tag rule dialog were inspected without selecting a new policy, adding a rule or saving. Secrets/variables are unrelated to this mapping and no values are included here.[^environments-ui]

The UI prefix `E` means `repository.settings.environments[]`.

| UI location / control | UI path below `E` | Internal API conversion |
| --- | --- | --- |
| Selected Environment name | `name` | Name-addressed resource identity; never numeric environment ID |
| Deployment protection rules / Wait timer; Time to wait (minutes) | `deployment_protection_rules.wait_timer` | Integer `wait_timer`; zero encodes the unchecked group, positive values the displayed minutes |
| Deployment protection rules / Required reviewers | `deployment_protection_rules.required_reviewers` | Typed numeric principal list to API `reviewers`; explicit empty list encodes an unchecked group |
| Required reviewers / people or teams | `deployment_protection_rules.required_reviewers[].person` or `.team` | Exactly one positive ID; person → API `type=User`, team → `type=Team` |
| Deployment protection rules / Prevent self-review | `deployment_protection_rules.prevent_self_review` | `prevent_self_review` |
| Deployment protection rules / Allow administrators to bypass configured protection rules | `deployment_protection_rules.allow_administrators_to_bypass_configured_protection_rules` | `can_admins_bypass` |
| Deployment branches and tags | `deployment_branches_and_tags` | Explicit menu to null or two internal policy booleans |
| Add deployment branch or tag rule / Ref type | `deployment_branch_or_tag_rules[].ref_type` | `branch` / `tag` to pattern endpoint `type` |
| Add deployment branch or tag rule / Name pattern | `deployment_branch_or_tag_rules[].name_pattern` | Literal pattern to endpoint `name` |

The compact timer/list values encode the corresponding checkbox groups without inventing an extra enabled key. The unsaved UI's suggested 15 minutes is not replayed as omitted desired intent. Timer bounds remain 0–43200 minutes. Reviewers retain the existing six-principal maximum, unique typed identity and positive IDs. Prevent self-review with an explicitly empty reviewer list is invalid; an omitted reviewer list stays API-controlled.

## Finite choices and defaults

| Deployment branches and tags choice | Internal API policy |
| --- | --- |
| `no_restriction` | Explicitly declared `deployment_branch_policy=null` |
| `protected_branches_only` | `protected_branches=true`, `custom_branch_policies=false` |
| `selected_branches_and_tags` | `protected_branches=false`, `custom_branch_policies=true` |

An omitted menu leaves policy undeclared, distinct from explicit No restriction. The API flag names do not become public aliases. Ref type choices are the verified UI's Branch / Tag converted to `branch` / `tag`; matching API tokens remain an adapter detail. Name patterns retain case and content, including wildcards; branch and tag with the same name remain different identities.

`DefaultEnvironmentDeploymentChoice=no_restriction` is the active [deployment-mode observation fallback](../settings.md#environment-deployment-mode-observation-normalization); `DefaultEnvironmentRefType=branch` remains preparation metadata for the remaining #480 ref-type work. Neither supplies omitted requests or acquisition failures. Equal typed boolean pairs project No restriction with concrete pair provenance; valid null remains exact, and malformed/missing boolean members still fail. Future ref-type normalization must preserve raw typed pattern identity and provenance: it cannot silently reinterpret an unknown identity as a branch before an apply/delete decision. Principal identity discriminators are validated references, not a user-selectable enum to coerce into a person. Unknown reviewer identities, malformed data and missing/null required fields remain errors. Booleans, valid timer integers, names and patterns need no unknown-enum fallback.

## Identity, presence and ownership

Environment names remain identity, using the existing Unicode simple-fold comparison without trimming or changing display case. Desired duplicates fail; unlisted environments remain untouched. Numeric environment and pattern IDs are acquisition metadata only. They neither select desired environments nor create generated-ID writeback files. Projection sorts by existing folded-name order while retaining display names and owned values.

Omitted built-in fields remain **API-controlled on the name-addressed PUT**. They are not promised unmanaged retention: the adapter sends no observed values for omitted timer, reviewer, self-review, administrator-bypass or branch-policy members, and accepts the API's resulting values for them. The current reconciliation still forces the necessary upsert/readback for requests with such omissions. False, zero, empty reviewers and explicit null policy remain present when selected. An environment name alone is still a name-only request, not implicit defaults.[^current-contract]

Pattern collections have separate ownership. Omission means no pattern CRUD; explicit `[]` owns and clears that environment's selected pattern collection only. Patterns require an explicit Selected branches and tags choice, or an acquired existing environment in that mode when the menu is omitted. Explicit patterns plus No restriction/Protected branches only are invalid. API pattern lists are canonical typed sets; exact duplicates fail. Root `EnvironmentCollectionDeclaration` also preserves omitted selection versus explicit `[]` as pointer presence. Neither empty case implies deleting environments.

Complete acquired custom mode must include its complete pattern list and matching identity metadata. A normalized equal pair with raw custom flag true also acquires every pattern page, keeping that list latent in separate state rather than active effective policy. Explicit acquired empty list is known empty; inactive/unrequested effective-policy patterns remain nil and cannot become clearing intent. Separately acquired latent state does not make those patterns active. Projection validates pattern metadata consistency, owns copied principal/field pointers and returns no partial UI collection on error. Unknown or unavailable observations cannot become absent/default environments.

## Transport and failure boundaries

The existing client inventories names completely before selected detail reads, validates built-in protection data and rejects custom protection apps. Name-addressed PUT refreshes repository identity/archive state and sends only declared built-ins/policy. The independently owned pattern collection never enters that payload. Explicit mode is confirmed against its raw nullable pair in the acknowledgment and fresh read; omitted mode remains API-controlled without preservation or unowned diagnostics. Unknown-mode pattern intent requires explicit Selected, and the intermediate fresh read must prove canonical Selected before CRUD. Mode entry and pattern operations are separate calls: create desired missing entries first, then freshly verify the typed identity/ID of each obsolete entry before deletion. New IDs stay metadata. A failure can leave the mode and earlier pattern changes applied; there is no transaction, rollback or intermediate enforcement guarantee.

Readback verifies requested members without claiming that omitted built-ins retained prior values. Required inventory/detail/custom-protection/pattern reads must be complete. HTTP 404, permission failures, unknown controls or malformed data remain errors, not absence. No environment deletion, renaming, custom protection apps, secrets, variables or new permission/security setting is introduced by these preparation helpers.

## Validation and activation handoff

Independent [builtin/reviewer fixtures](../../internal/github/ui_environments_test.go), [menu/pattern fixtures](../../internal/github/ui_environments_patterns_test.go), [acquired collection fixtures](../../internal/github/ui_environments_observation_test.go) and [literal raw/exact transport fixtures](../../internal/github/ui_environments_composition_test.go) cover all supported controls and choices. They assert omission, explicit false/zero/[], raw API-null policy versus the UI no-restriction choice, copied ownership, typed identities, folded name ordering, matching metadata, exact narrow/full PUT, API-controlled omissions, create-before-remove and partial failure without rollback. Existing [Environment](../../internal/github/environments_test.go), [bypass](../../internal/github/environment_bypass_test.go) and [pattern](../../internal/github/environment_patterns_test.go) matrices retain pagination, unknown/custom protections, identity, permissions and uncertain-effect checks.

Current activation connects UI paths while retaining root/per-field presence and separate pattern ownership. #480 must preserve raw typed identity/provenance when normalizing finite choices; #481 must retain API-controlled omission and partial-operation diagnostics. User documentation describes the active UI contract; API null and raw report policy are not public YAML aliases.

## Active deployment-mode observation evidence

The shared [closed mapping](../../internal/settings/environment_deployment_mapping.go), [bounded carrier](../../internal/settings/environment_deployment_observation.go) and [state/change tests](../../internal/settings/environment_deployment_change_test.go) separate raw equal booleans from effective declared-null policy. The [detail projector](../../internal/github/normalization_environment_deployment.go) returns preliminary state and a separate record, leaving custom-protection and latent pattern acquisition to `readOne` before final observation construction. It does not fabricate acquired empty patterns.

[Complete reader fixtures](../../internal/github/environment_deployment_read_test.go), [real Update fixtures](../../internal/github/environment_deployment_update_test.go) and [executable commands](../../tests/acceptance/normalization_environment_deployment_test.go) cover empty/multiple-page latents, failed later acquisition, fallback-equal raw verification, scope/omissions, unknown-mode pattern prerequisites, ordered CRUD and retained partial effects. [Private reports](../../internal/actions/normalization_environment_deployment_test.go) retain effective before-policy, actual partial requests and status-only privacy. These fixtures establish client behavior, not live settings, entitlement or effective deployment protection. Ref-type normalization and complete #481 reporting remain pending.

[^environments-ui]: GitHub, Environments / Configure environment and conditional protection/pattern forms, unsaved inspection, 2026-10-28.
[^current-contract]: Current repository documentation, Environment settings, consulted 2026-10-28 before public YAML activation; the current link now reflects the activated contract.
