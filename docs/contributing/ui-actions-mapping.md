---
type: Reference
title: Prepared Actions UI mappings
description: Internal Actions controls, API translation and activation handoff.
sources:
  - id: actions-ui
    resource: https://github.com/daiksud/gh-settings/settings/actions
  - id: actions-settings
    resource: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository
---

## Preparation boundary

This Actions mapping was prepared for [#484](https://github.com/daiksud/gh-settings/issues/484) and is now composed into current-source YAML by [#479](https://github.com/daiksud/gh-settings/issues/479). The [public reference](../settings.md#actions-execution-policy) owns accepted choices. [#480](https://github.com/daiksud/gh-settings/issues/480) normalization/default warnings and [#481](https://github.com/daiksud/gh-settings/issues/481) complete reporting remain unfinished. Native internal reports remain separate from UI intent; there is no compatibility or migration layer.

The [internal UI model](../../internal/settings/uimodel/actions.go) has typed control choices without transport enums or serialization tags. [Declaration conversion](../../internal/github/ui_actions.go) supplies existing validated adapter inputs. Observation helpers consume acquired values directly, retaining raw latent policy and retention caps outside the UI model. They never call declaration normalization. Overlay helpers preserve unrelated incoming UI fields; they copy newly acquired values rather than claiming a deep copy of the whole incoming tree.

## Inventory and paths

English controls were inspected on 2026-10-28 in Actions / General. Public controls were inspected on gh-settings. Private-only controls were inspected in an authorized private repository; its identity and current values are intentionally excluded from this public reference. Conditional controls were revealed without saving, and unsaved edits were discarded. This establishes wording and visible choices, not effective live policy changes or permission coverage.[^actions-ui][^actions-settings]

The UI prefix `A` below means `repository.settings.actions.general`. Labels are converted to snake_case. A repository owner's changing name is represented by the stable word `owner` in radio choices.

| UI label | UI path below `A` | Internal API member |
| --- | --- | --- |
| Actions permissions | `actions_permissions` | `enabled` and, when enabled, `allowed_actions` |
| Require actions to be pinned to a full-length commit SHA | `require_actions_to_be_pinned_to_a_full_length_commit_sha` | `sha_pinning_required` |
| Allow actions created by GitHub | `allow_actions_created_by_github` | Selected-actions `github_owned_allowed` |
| Allow actions by Marketplace verified creators | `allow_actions_by_marketplace_verified_creators` | Selected-actions `verified_allowed` |
| Allow or block specified actions and reusable workflows | `allow_or_block_specified_actions_and_reusable_workflows` | Selected-actions `patterns_allowed` |
| Check, workflow run, status, artifact and log retention | `check_workflow_run_status_artifact_and_log_retention` | Retention `days` |
| Approval for running fork pull request workflows from contributors | `approval_for_running_fork_pull_request_workflows_from_contributors` | Public fork `approval_policy` |
| Workflow permissions | `workflow_permissions` | Workflow `default_workflow_permissions` |
| Allow GitHub Actions to create and approve pull requests | `allow_github_actions_to_create_and_approve_pull_requests` | Workflow `can_approve_pull_request_reviews` |
| Fork pull request workflows / Run workflows from fork pull requests | `fork_pull_request_workflows.run_workflows_from_fork_pull_requests` | Private fork `run_workflows_from_fork_pull_requests` |
| Send write tokens to workflows from fork pull requests | `fork_pull_request_workflows.send_write_tokens_to_workflows_from_fork_pull_requests` | Private fork `send_write_tokens_to_workflows` |
| Send secrets and variables to workflows from fork pull requests | `fork_pull_request_workflows.send_secrets_and_variables_to_workflows_from_fork_pull_requests` | Private fork `send_secrets_and_variables` |
| Require approval for fork pull request workflows | `fork_pull_request_workflows.require_approval_for_fork_pull_request_workflows` | Private fork `require_approval_for_fork_pr_workflows` |

Cache retention/size, sharing access, runners and other unsupported controls remain outside this family. The existing endpoint clients retain their narrow acquisition scopes; observation conversion does not expand API access or follow `selected_actions_url`.

## Finite choices

| UI Actions permission choice, snake_case | Internal translation |
| --- | --- |
| `allow_all_actions_and_reusable_workflows` | `enabled=true`, `allowed_actions=all` |
| `disable_actions` | `enabled=false`; no desired `allowed_actions` |
| `allow_owner_actions_and_reusable_workflows` | `enabled=true`, `allowed_actions=local_only` |
| `allow_owner_and_select_non_owner_actions_and_reusable_workflows` | `enabled=true`, `allowed_actions=selected` |

The last two labels display the actual owner in GitHub. The selected radio reveals the three selected-actions controls. Explicitly choosing another radio while declaring a selected companion is rejected. An omitted radio leaves the existing state-dependent preflight responsible for applicability. Transitions from a nonselected policy to selected require all three companions; enabling disabled execution happens only after the selected policy is written. Pattern entries retain order, repetition, negation and explicit empty lists.

| UI Workflow permission choice, snake_case | Internal translation |
| --- | --- |
| `read_and_write_permissions` | `write` |
| `read_repository_contents_and_packages_permissions` | `read` |

| UI Public fork approval choice, snake_case | Internal translation |
| --- | --- |
| `require_approval_for_first_time_contributors_who_are_new_to_github` | `first_time_contributors_new_to_github` |
| `require_approval_for_first_time_contributors` | `first_time_contributors` |
| `require_approval_for_all_external_contributors` | `all_external_contributors` |

API tokens are not accepted as UI choices. Complete out-of-UI workflow permission and public-fork approval strings now use their defined fallbacks with raw provenance and warnings under #480; other unknown acquired API choices still fail pending the remaining work. Named observation-default candidates are `allow_all_actions_and_reusable_workflows`, `read_repository_contents_and_packages_permissions` and `require_approval_for_first_time_contributors`. The official settings guide identifies first-time contributors as the default public-fork approval policy.[^actions-settings] The workflow and public-fork fallbacks are active; the Actions execution constant remains handoff metadata. Public-fork observations retain exact raw strings in their opaque observation; raw equality governs no-op and fresh write verification. None supplies defaults for omitted declarations, missing responses or denied acquisition. There is no invented retention or private-fork default.

## Presence, hidden values and applicability

Every desired scalar remains optional. Explicit false, empty lists and minimum retention values remain present. Observations require complete acquired fields and acquisition markers; an unavailable endpoint cannot become false or empty. Disabling Actions preserves raw `allowed_actions` and any acquired selected policy outside the visible UI model. Unrequested selected companions remain unacquired. The visible radio becomes `disable_actions`; latent values are never replayed as new desired UI controls.

Public fork approval retains the existing public-repository gate. Private fork controls retain the private-repository gate. Private observations always retain four raw acquired booleans, but show only the execution checkbox when disabled. Explicit disabled execution plus a hidden dependent desired control is rejected. Companion-only intent with omitted execution defers to the existing runtime prerequisite check. A necessary write requires explicit execution; enabling requires all four choices, including credential and approval choices. No omitted required transport parameter is invented.

Retention displays the acquired configured `days`, even when it exceeds the acquired organization cap. `maximum_allowed_days` is adapter metadata, not desired YAML or a substitute configured value. Desired/export validation remains responsible for the cap; the new helper preserves raw observations without validating them as desired state. Writes still send only `days` and verify readback. Bounds remain 1–400 with the acquired tighter cap applied by existing preflight.

## Validation and activation handoff

Independent [permission fixtures](../../internal/github/ui_actions_execution_test.go), [selected fixtures](../../internal/github/ui_actions_selected_test.go), [workflow fixtures](../../internal/github/ui_actions_workflow_test.go), [retention fixtures](../../internal/github/ui_actions_retention_test.go), [public fork fixtures](../../internal/github/ui_actions_public_fork_test.go), [private fork fixtures](../../internal/github/ui_actions_private_fork_test.go) and [composition fixtures](../../internal/github/ui_actions_composition_test.go) specify expected values without deriving expectations from production mappings. They exercise exact payloads, independent raw readback, hidden/raw acquisition, unchanged required companions, state prerequisites and acquisition failures.

Current root composition wires the family through the one UI codec, desired conversion and complete export projection while retaining native application sequencing/private reports. Existing acquisition/provenance and cap metadata stay outside desired intent for #480/#481. Full UI reporting is not completed here. This activation adds no feature control, live setting change or normalization-derived observed change.

[^actions-ui]: GitHub, Actions / General, read-only UI inspection, 2026-10-28.
[^actions-settings]: GitHub Docs, managing GitHub Actions settings for a repository, consulted 2026-10-28.
