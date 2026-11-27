---
type: Reference
title: "Workflow event and actor policies"
---

# Actions event policies

`repository.settings.actions.policies` manages repository-owned policies in
**Settings → Actions → Policies**, with **Target workflows** inclusion/exclusion
patterns followed by **Workflow execution protections** with **Restrict actors** and **Restrict events**. These are separate objects from the
[Actions execution policy](general.md#actions-execution-policy). Without prune, omission or
`policies: []` selects nothing, makes no policy API request and deletes nothing.
Unlisted policies remain untouched; removing an entry stops managing it.
Explicit `--prune actions-policies` adds deletions for a present list as described
below.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      policies:
        - policy_name: Restrict workflow events
          enforcement_status: disabled
          target_workflows:
            include: ['.github/workflows/ci*.yml']
            exclude: ['.github/workflows/ci-experimental.yml']
          workflow_execution_protections:
            restrict_events:
              allowed_events: [pull_request, push, workflow_dispatch]
```

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive integer REST ID of an existing policy owned by the target repository; omission means create-only |
| `policy_name` | Required nonempty/non-whitespace string without control characters; identity and collisions use exact spelling |
| `enforcement_status` | Required `disabled` or `active`; neither is inferred |
| `target_workflows` | Optional object with required `include` and `exclude` string arrays; declares the complete workflow selection |
| `workflow_execution_protections.restrict_actors` | Optional object with `allowed_actors`, or `enabled: false` to turn the rule off |
| `workflow_execution_protections.restrict_actors.allowed_actors` | Required when the actor rule is enabled; array of unique actor identities, including `[]` for checked with no allowed actors |
| `workflow_execution_protections.restrict_events` | Optional object with `allowed_events`, or `enabled: false` to turn the rule off |
| `workflow_execution_protections.restrict_events.allowed_events` | Required when the event rule is enabled; nonempty array of unique supported event strings, compared as a set and exported sorted |

Each allowed actor selects exactly one identity: `{user: 155234749}` for a
positive REST user ID, or `{role: write}`, `{role: maintain}` or
`{role: repository_admin}`. These roles map to the verified `RepositoryRole`
IDs 4, 2 and 5. User IDs identify existing accounts; they are not login strings.
No access is granted by adding an allowed actor. Other actor kinds and role IDs,
including Bot, App, installation and team identities, are unsupported.

A restriction object without `enabled`, or with `enabled: true`, requires its
allowed array. `enabled: false` forbids that array and removes that rule.
An empty actor array keeps **Restrict actors checked with no allowed actors**;
it is not the same as turning the rule off. Empty event arrays remain invalid.
Nulls, wrong types, unknown keys, duplicate identities and mixed actor identities
fail locally.

On creation, omitted restrictions are off and at least one rule must be enabled.
On ID-selected updates, an omitted restriction preserves the freshly acquired
rule, including its absence. Metadata-only updates may omit both restrictions.
Explicitly disabling the final rule fails before writing. The complete effective
policy must contain at least one supported restriction. For example:

```yaml
workflow_execution_protections:
  restrict_actors:
    allowed_actors:
      - role: repository_admin
      - user: 155234749
  restrict_events:
    enabled: false
```

This changes only the supported restriction selections on that policy; omitted
workflow targeting still follows the preservation contract below.

Supported events are `branch_protection_rule`, `check_run`, `check_suite`,
`create`, `delete`, `deployment`, `deployment_status`, `discussion`,
`discussion_comment`, `fork`, `gollum`, `image_version`, `issue_comment`, `issues`,
`label`, `merge_group`, `milestone`, `page_build`, `project`, `project_card`,
`project_column`, `public`, `pull_request`, `pull_request_review`,
`pull_request_review_comment`, `pull_request_target`, `push`, `registry_package`,
`release`, `repository_dispatch`, `schedule`, `status`, `watch`, `workflow_call`,
`workflow_dispatch` and `workflow_run`.

Duplicate IDs or exact names, missing required fields, nulls, unknown keys and
unsupported event/enforcement values fail locally. A no-ID entry never adopts
an existing name: a collision fails and identifies the existing ID. An ID must
exist in this repository; missing or foreign IDs never fall back to creation.
An ID-bound declaration replaces the complete supported event object and may
rename it to a noncolliding name. Renames cannot rely on another planned rename
freeing the name. There is no delete declaration or name-based update.

`target_workflows` corresponds to **Include by pattern** and **Exclude by
pattern** in the UI. Both arrays are required when the object is supplied, and
at least one must contain a pattern. Empty `include` means every workflow except
the exclusions. `~ALL` represents **All workflows** and is valid only as the sole
include entry, never in exclusions. For example, `include: [~ALL]` with
`exclude: []` explicitly resets targeting to All workflows. An exclude-only
selection uses `include: []` and one or more exclusion patterns.

Patterns must be nonempty/non-whitespace strings without control characters;
duplicates within either array are rejected. Case and exact pattern spelling
are preserved. Comparisons treat each array as a set and exports sort it. All
workflows normalize to `[~ALL]` with empty exclusions; exclude-only targeting
uses an empty include array. The tool passes patterns to GitHub and does not
emulate workflow matching.

Omitting `target_workflows` creates an All workflows policy, but preserves an
existing policy's freshly acquired targeting on update. It does not reset or
broaden an existing selection. To change targeting, declare both arrays. Remote
omitted/null/empty `conditions`, or `workflow_path` with empty exclusions and
empty includes or `[~ALL]`, represent All workflows. A present `workflow_path`
requires both typed arrays. Unknown conditions, unsupported actor identities,
Preview rules, unknown rules/fields and unsupported event/enforcement values are not
lossily converted. A complete unsupported object's ID/name is retained so
unlisted objects stay unmanaged without prune; selecting it for update, as a
prune candidate, or including it in a full collection export fails. Incomplete or inconsistent reads always fail.
Currently unsupported actor types and Preview lockfile controls require target
UI/API assessment and implementation; Preview status alone is not an exclusion.
[OIDC subject configuration](oidc.md#actions-oidc-subject-template) has separate support.
Workflow and other feature configuration-file authoring remain excluded.

`--prune actions-policies` adds deletion of unlisted repository-owned IDs only
when `policies` is present; `[]` selects every current policy as a candidate.
An omitted list remains unmanaged. Deleting a policy removes its workflow
restriction, so future workflow execution may be permitted. All candidates must
be completely supported; unknown/Preview rules, unadopted actor types or targeting,
wrong ownership, incomplete reads and missing permissions fail before any write.
Apply fixes the candidate set, performs creates/updates first, rechecks exact
identity/state before each DELETE, requires HTTP 204 and confirms ID absence in
a complete fresh inventory. Earlier successful changes survive a later failure.
There is no policy `delete` YAML field.

Plan acquires the complete repository collection and details before showing
creates/ID-bound updates and explicitly requested prune deletions. Apply freshly acquires and preflights the whole
collection before its first policy write, skips matches and rechecks each
selected update's detail immediately before mutation. Drift or newly unsupported
state fails rather than overwriting it. Create uses POST and update uses PUT at
the [official repository Actions policies API](https://docs.github.com/en/rest/actions/policies).
Changed explicit targeting is sent as `conditions.workflow_path`; omitted
targeting uses All workflows on create and preserves the preflight-verified
selection on update. Unchanged targeting need not be resent. Complete write acknowledgment and
an independent final GET must both match the requested identity, name,
enforcement, event/actor rule presence and identity sets, and supported targeting.

Failures stop later policy writes; earlier successful changes remain applied.
A write/verification failure may leave the remote effect uncertain. Known new
IDs remain available for recovery, including acknowledgment/readback failures;
uncertain creation is never automatically retried. There is no transaction,
rollback or concurrency lock. Inspect remote state and record the correct ID
before retrying. Local apply records confirmed creations using
[guarded ID writeback](../labels.md#apply-id-writeback), with `policy_name` as selector;
`--no-writeback` and `GITHUB_ACTIONS=true` suppress it.

Ordinary `export` independently observes the full repository collection,
including IDs and explicit canonical `target_workflows`, sorted by exact policy
name with sorted events, actor identities and patterns. Export explicitly emits
`enabled: false` for absent restriction rules, so a snapshot does not accidentally
preserve an existing rule. All workflows exports as `include: [~ALL]`
and `exclude: []`, so a saved snapshot owns its complete selection. All pages and details must be complete and representable before adoption;
a failure emits info and retains the prior collection while other reads continue.
Fine-grained and GitHub App tokens require **Administration write even for
list/GET, export and plan**, as well as writes, according to the official API.
The client never grants permissions or substitutes empty state for denied reads.

The Free/personal/public UI and disabled-policy API storage, including
include/exclude patterns, omitted-target preservation, explicit All workflows
reset, User and the three RepositoryRole identities, actor-only/combined rules,
checked-empty actors and unchecked-rule removal, were verified;
local fixtures cover active/disabled metadata transitions. Actual enforcement
and workflow execution were not tested. Storage readback does not establish
workflow blocking or a particular token's grants. See the
[behavior specification](../../../development/features/actions-event-policies.feature.md) and
[event-policy adoption evidence](https://github.com/daiksud/gh-settings/issues/647)
and [workflow-targeting evidence](https://github.com/daiksud/gh-settings/issues/649).

[All Actions settings](README.md) · [GitHub App permissions](../permissions.md#actions)
