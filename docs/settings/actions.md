---
type: Reference
title: "Actions settings"
description: "Actions settings declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Actions execution policy

All paths below are relative to `repository.settings.actions.general.actions_permissions`:

| Field | Choice / type |
| --- | --- |
| `policy` | `allow_all_actions_and_reusable_workflows`, `disable_actions`, `allow_owner_actions_and_reusable_workflows`, `allow_owner_and_select_non_owner_actions_and_reusable_workflows` |
| `require_actions_to_be_pinned_to_a_full_length_commit_sha` | boolean |
| `allow_actions_created_by_github` | boolean; selected-policy companion |
| `allow_actions_by_marketplace_verified_creators` | boolean; selected-policy companion |
| `allow_or_block_specified_actions_and_reusable_workflows` | ordered string array; selected-policy companion |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        actions_permissions:
          policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows
          require_actions_to_be_pinned_to_a_full_length_commit_sha: true
          allow_actions_created_by_github: true
          allow_actions_by_marketplace_verified_creators: false
          allow_or_block_specified_actions_and_reusable_workflows: ['my-org/*', '!my-org/blocked@*']
```

Selecting an allow policy explicitly enables Actions and selects its policy; `policy: disable_actions` declares only disabled execution, leaving latent allowed policy unmanaged. There are no separate public API-shaped enabled/allowed fields. Omitting `policy` does not implicitly enable Actions. SHA pinning is independent; reusable workflows can still use tags.

Companions can accompany the selected non-owner policy or an omitted policy, not another explicit policy. An omitted policy requires an already selected observed policy for companion updates. Transition from a nonselected policy requires all three companions: inactive values cannot be acquired or invented. Plan shows them as unavailable before the transition. Existing selected policy permits partial declarations merged with complete fresh companions.

Patterns are at most 1000 nonempty strings. An explicit array replaces the entire sequence, and `[]` clears it. Case, order, duplicates, wildcards and negative entries are preserved; the tool does not emulate matching. Removing a negative entry may permit more actions. GitHub documents pattern applicability limits; configuring a list alone does not prove enforcement on private repositories.

Ordinary `export` includes the execution policy, SHA control and visible acquired companions in the same section. Disabled/other policy hides selected companions; complete raw acquisition is still required. Failed or incomplete acquisition retains this family’s prior declaration with info, without erasing independently acquired Actions siblings. Correctly typed unknown execution policies follow the observation policy below.

Apply reads complete fresh state and skips matching intent. A needed base PUT includes the endpoint-required enabled value, using the fresh companion when omitted, and only changed optional fields. A needed selected PUT preserves omitted companions from the complete active read. Policy changes precede selected writes. When enabling a disabled repository with selection, it selects policy while disabled and installs companions only as needed, then enables. Independent readback verifies the original complete requested state and preserved raw companions, including initially matching declared values. Repeated matching apply performs no PUT.

Operations are not transactional: an already enabled target can temporarily use remembered/default selection between writes. Earlier policy changes survive later refusal; failed writes/readback have uncertain effects. There is no rollback, saved-plan execution, automatic retry, lock or permission expansion. Existing Administration read/write and organization/enterprise policy remain prerequisites; policy changes can stop future workflows. See [execution policy](../features/actions-execution-policy.feature.md).

## Check, run, status, artifact and log retention

Use integer `repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention` (1–400 days). It translates to the existing retention endpoint. Ordinary `export` attempts retention automatically. The server's maximum is read-only metadata and is never replayed as intent. Existing repository visibility, range and freshly observed cap checks apply; a plan/readback cannot invent missing cap information or prove licensing. Apply verifies the requested value and preserves failure/uncertainty reporting. See [retention](../features/retention.feature.md).

## Actions cache policy

The optional `cache` section under `repository.settings.actions.general` manages two independent integer controls:

| Field below `cache` | Value | Export option |
| --- | --- | --- |
| `cache_retention` | 1–365 days; public repositories are limited to 90 | `--cache-retention` |
| `cache_size_eviction_limit` | 1–10000 GB | `--cache-storage` |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        cache:
          cache_retention: 7
          cache_size_eviction_limit: 10
```

These controls are independent of check/workflow/artifact/log retention and Actions execution policy. Omitted controls are neither read nor changed by plan/apply; selecting one export option does not acquire the other. Empty `cache: {}` declares nothing. Nulls, strings, fractions and out-of-range integers are invalid.

A cache size eviction limit above 10 GB can incur additional charges. It is an eviction threshold, **not a spending budget**. Retention does not guarantee that caches survive until that age: eviction or manual deletion can remove them sooner. gh-settings manages repository limits only, not cache entries, usage, billing budgets, or organization/enterprise policy.

Each selected control requires a complete current read before its PUT and an independent readback afterward. Matching values skip PUT. GitHub can reject a value because of a tighter parent policy; gh-settings does not clamp it or change that policy. Missing, denied or malformed reads fail rather than supply defaults. A successful HTTP response with different readback fails verification and may have changed remote state. Later failure leaves earlier successful writes applied, without rollback or retry. See [behavior and API boundaries](../features/actions-general-controls.feature.md).

## Fork PR workflow policies

Public-fork approval is `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`:

- `require_approval_for_first_time_contributors_who_are_new_to_github`
- `require_approval_for_first_time_contributors`
- `require_approval_for_all_external_contributors`

A complete successful public-fork policy GET with an out-of-UI string projects to `require_approval_for_first_time_contributors` (native `first_time_contributors`). Original unknown, empty or whitespace strings remain exact provenance. Missing/null members, wrong types, unsupported visibility and failed reads remain errors. Export warns and saves only the UI choice; plan retains one write-required decision even when desired intent equals the fallback. Omitted intent remains unmanaged.

Apply checks exact raw values before skipping a write and in fresh readback after PUT. A retained unknown value fails verification with possible write effects; a canonical exact match remains a no-op. Each successful managed apply read can warn: the initial application read, updater preflight and a normalized readback. Pure plan readers emit no stderr; detailed plan/Summary carry the decision warning, while status-only Summary suppresses values.

Ordinary `export` observes this public-repository control. Visibility, complete observations, narrow payloads, readback, redirect rejection and partial-failure reporting remain required. Upper policy may refuse or ignore changes; no retry, rollback or credential expansion occurs. See [fork workflows](../features/fork-workflows.feature.md).

## Default workflow permissions

These fields belong to `repository.settings.actions.general.workflow_permissions`:

| Field | Choice / type |
| --- | --- |
| `default_permissions` | `read_and_write_permissions` or `read_repository_contents_and_packages_permissions` |
| `allow_github_actions_to_create_and_approve_pull_requests` | boolean |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        workflow_permissions:
          default_permissions: read_repository_contents_and_packages_permissions
          allow_github_actions_to_create_and_approve_pull_requests: false
```

Each is independent and optional. Ordinary `export` includes both after complete acquisition, independently of execution policy. Omission is unmanaged, false remains explicit, and unknown choices/null/wrong types fail locally. Apply compares a complete fresh observation, skips matching intent and sends only changed declared workflow-policy members. Independent readback verifies the original declaration, including initially matching values, and preserves omitted raw companions. A matching repeat apply sends no update. Direct apply remains nontransactional: organization refusal can leave an earlier execution-policy update applied, and a failed PUT/readback can have uncertain effects. This declaration does not create tokens or waive a PR's approval requirements. See [workflow permissions](../features/workflow-permissions.feature.md).

A complete successful workflow-policy GET with a correctly typed permission string outside the UI choices uses `read_repository_contents_and_packages_permissions` and emits a warning naming the YAML field, raw token and fallback. Missing/null members, wrong types and failed reads remain acquisition errors. Export saves the UI fallback without changing GitHub. Plan retains an actionable write-required comparison even if the desired choice equals that fallback; it does not claim independently observed zero drift. Apply sends only declared members and confirms the requested permission against a fresh raw readback. An unchanged out-of-UI token fails verification with possible write effects retained. An omitted permission remains unmanaged and is verified as an unchanged raw companion during approval-only writes, without replaying or printing the fallback.

## Existing runner custom labels

Use `repository.settings.actions.runners` to manage custom labels on existing
repository self-hosted runners in a personally owned public repository:

```yaml
runners:
  - id: 23
    custom_labels: [gpu, accelerated]
  - id: 24
    custom_labels: []
```

Each entry requires a unique positive numeric runner `id` and an explicit
`custom_labels` list of at most 100 strings. The ID selects an existing runner;
there is no name-based adoption, creation, rename or ID writeback. Omitting the
family leaves it unmanaged. Without prune, `runners: []` selects no runners and
makes no family requests during plan/apply. An entry with `custom_labels: []`
clears that runner's custom labels. Unlisted runners remain untouched unless
`--prune runners` or standalone `--prune all` is explicitly selected. With either
selector, a present list retains the listed IDs and removes unlisted remote
registrations; an explicit empty list selects all acquired registrations. This
changes the effect of `all` for existing present runner declarations. Review the
plan and your list before applying it.

Labels form a case-insensitive set. The client uses Unicode simple folding,
without trimming or Unicode normalization; order and case-only changes are no-ops.
Submitted spelling remains in YAML and the request. Duplicate case identities,
blank or control-containing names, nulls, unknown YAML fields and malformed IDs
fail locally. The nonblank/control rule is a product guard, not an additional
GitHub schema claim. A desired custom label that collides with an acquired
read-only label is rejected before this family's first write.

Export acquires the complete paginated inventory plus each runner's detail and
labels before replacing this family. Known empty inventory exports `runners: []`.
Any incomplete, denied or inconsistent runner/label observation omits the family
on first export or retains its valid prior declaration with bounded info; other
families continue. Missing/null/unknown label type never implies custom or
read-only, and default labels are never inferred from OS. Label IDs are optional
observation evidence, not YAML bindings. Present IDs must be positive; conflicting
known IDs fail. Both reads contribute the strongest available ID evidence.

Without runner pruning, plan reads only selected runner IDs and compares their complete custom sets.
Apply prepares every selected runner before the first label update, then freshly
reads each runner and repository before writing. A match makes no PUT. For an already archived target declared `archived: true`,
non-prune matching requires the independent identity-bound checks described in
the [archive lifecycle](general.md#repository-archive-lifecycle). A needed write on an archived repository fails; an explicitly requested unarchive still
precedes configuration. Replacement sends only the complete declared custom list.
HTTP 200 must contain a complete matching label acknowledgment, followed by
independent repository, same-ID runner and label GETs. Verification preserves
fresh runner name/OS and exact read-only label names/types, including every
previously known read-only ID. Custom-label IDs are not owned. Missing prior IDs
remain unknown, and later missing known read-only IDs fail verification.

For ordinary label updates, operational status, busy, version and other liveness metadata are not drift
conditions. Failed pre-write reads are distinguished from uncertain effects after
a PUT attempt. Earlier verified writes remain applied; later work stops, without
retry or rollback. Detailed plans and Action Summaries show selected IDs and
custom labels; status-only hides them. For ordinary label updates, name/OS/read-only
evidence and raw API errors do not enter reports. Input files are not modified.

### Runner registration pruning

Explicit pruning owns only remote deregistration, corresponding to GitHub's
[Force remove control](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/remove-runners).
It does not stop services, remove host software or credentials, cancel jobs, or
restore a registration. Removing a runner can affect routing and queued or running
work. GitHub provides no atomic busy or version condition for DELETE: a job may
start between the final check and removal.

Before any write, pruning acquires the complete personal-public target and stable
paginated inventory, details and labels, fixes the unlisted IDs, prepares selected
label updates, and requires each candidate to have a valid online/offline state and
`busy: false`. Unknown or incomplete data fails closed. Both online and offline
idle runners may be selected. The complete inventory is checked again before the
family's first write and each fixed-ID DELETE. Additions, disappearance, changed
candidate state or metadata, and changed retained identity or configuration fail
rather than expanding or replacing the candidate set. Retained operational status,
busy and version changes alone do not block an unrelated deletion. Selected label
updates complete first; only independently verified own changes are incorporated
into later checks. A confirmed explicit unarchive updates the expected archive state.

Deletion requires an empty HTTP 204 and a fresh complete inventory confirming the
fixed ID's absence and preservation of retained objects. A 404 or acknowledgment
alone does not prove removal. A naturally disappearing runner requires replanning.
An attempted deletion or failed verification can have uncertain effects; earlier
confirmed changes remain, later work stops, and no retry or rollback is automatic.
Detailed deletion plans show ID, name, OS and custom labels as the before value;
status-only hides these values. Operational state, optional raw label IDs and other
private evidence remain outside reports. Fixtures do not establish live job behavior.

The [official labels guide](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/apply-labels)
confirms the user-repository UI and case-insensitivity. The
[repository runner REST API](https://docs.github.com/en/rest/actions/self-hosted-runners)
provides observation and custom-label replacement. Reads require repository
Administration read and writes require Administration write. Local fixtures do
not prove a populated UI, exact server Unicode comparison, live writes, runner
availability, routing, physical OS or job completion. This support does not
register runners, generate tokens, install software, manage groups or modify
workflows and hosts.

## Actions event policies

`repository.settings.actions.policies` manages repository-owned policies in
**Settings → Actions → Policies**, with **Target workflows** inclusion/exclusion
patterns and the **Restrict events** and **Restrict actors** rules. These are separate objects from the
[Actions execution policy](#actions-execution-policy). Without prune, omission or
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
          restrict_events:
            allowed_events: [pull_request, push, workflow_dispatch]
```

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive integer REST ID of an existing policy owned by the target repository; omission means create-only |
| `policy_name` | Required nonempty/non-whitespace string without control characters; identity and collisions use exact spelling |
| `enforcement_status` | Required `disabled` or `active`; neither is inferred |
| `target_workflows` | Optional object with required `include` and `exclude` string arrays; declares the complete workflow selection |
| `restrict_events` | Optional object with `allowed_events`, or `enabled: false` to turn the rule off |
| `restrict_events.allowed_events` | Required when the event rule is enabled; nonempty array of unique supported event strings, compared as a set and exported sorted |
| `restrict_actors` | Optional object with `allowed_actors`, or `enabled: false` to turn the rule off |
| `restrict_actors.allowed_actors` | Required when the actor rule is enabled; array of unique actor identities, including `[]` for checked with no allowed actors |

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
restrict_events:
  enabled: false
restrict_actors:
  allowed_actors:
    - role: repository_admin
    - user: 155234749
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
[OIDC subject configuration](#actions-oidc-subject-template) has separate support.
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
[guarded ID writeback](labels.md#apply-id-writeback), with `policy_name` as selector;
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
[behavior specification](../features/actions-event-policies.feature.md) and
[event-policy adoption evidence](https://github.com/daiksud/gh-settings/issues/647)
and [workflow-targeting evidence](https://github.com/daiksud/gh-settings/issues/649).

## Actions OIDC subject template

Use `repository.settings.actions.oidc.subject_claim` for the **Subject claim**
page of a personally owned public repository:

```yaml
repository:
  settings:
    actions:
      oidc:
        subject_claim:
          use_default_template: false
          subject_claim_template: [repo, context, job_workflow_ref]
          use_immutable_subject_claim: true
```

The group has two independently selectable units: the complete active template
and the optional `use_immutable_subject_claim` boolean. Omission or an empty mapping
leaves it unmanaged. An immutable-only declaration is valid. If either template
field is selected, `use_default_template: false` requires an explicit ordered array;
`true` forbids that array and selects GitHub's default template. An array without
an explicit false mode is invalid. Keys are unique nonempty ASCII alphanumeric
or underscore strings; order and case are preserved. There is no closed claim
name list. Null, unsupported keys and coercions are rejected. An explicit
`[]` is accepted locally because neither the official API schema nor the inspected
form establishes a minimum; GitHub acceptance and exact readback are still required.

Selecting default deactivates the custom template. It does not promise to retain
hidden inactive keys that GitHub ignores in default mode. Changing a subject can
stop cloud authentication: coordinate the corresponding cloud trust configuration
before applying it. This tool does not issue tokens, run workflows, or change
cloud trust, credentials, grants or billing.

`use_immutable_subject_claim` requests the API configuration corresponding to the
UI checkbox, including explicit true or false. It does not certify effective token
format, universal UI editability or a reversible transition. GitHub documents
[mandatory immutable subjects](https://docs.github.com/en/actions/reference/security/oidc#immutable-subject-claims)
for some repositories independently of manual opt-in. gh-settings does not infer
eligibility from creation dates, rename/transfer history or the returned prefix.
The editable legacy control was observed unsaved on 2026-11-14; persisted legacy
opt-out has not been live-verified. GitHub decides whether a particular update is
permitted. A false configuration readback is not proof of a name-only cloud token.

Ordinary export acquires this family independently. Known default exports true
without keys; known custom exports false with its ordered keys. Known immutable
configuration exports its boolean, including false. A missing optional immutable
field remains unknown: export refreshes the readable active template and retains
only valid previous immutable intent, or omits that leaf on first export, with
bounded info. Retained intent is not acquired evidence. Whole-family transport,
identity or malformed-data failure retains compatible prior whole intent. Required
diagnostic delivery must succeed before saving.

Plan compares one managed group. Apply reacquires complete personal-public target
identity and active configuration, then skips known matching selected values.
Selecting immutable requires its known boolean even for comparison; unknown is
neither false nor a proven difference. A matching template-only declaration can
still skip with unknown immutable. A needed PUT requires both complete active
template and known immutable evidence. Immutable-only writes replay the fresh
active mode and exact ordered custom keys; template-only writes replay immutable.
Combined intent supplies both. Omission never silently selects a different mode.
Missing or malformed required state fails before writing. A successful PUT must
return HTTP 201 with a bounded empty acknowledgment, followed by independent GET
confirmation of the original repository/owner identity, archive state, requested
values and replayed active companions. The read-only prefix may legitimately
change and is not an equality guard. A failed write or readback reports uncertain effects;
earlier verified progress remains and later work stops. No rollback or automatic
retry occurs, and fresh reads are not a concurrency lock.

The official [repository OIDC API](https://docs.github.com/en/rest/actions/oidc)
uses Actions read for observation and Actions write for updates. Its read-only
subject prefix is neither YAML intent nor identity evidence; it and raw API bodies
are excluded from diagnostics even with `GH_DEBUG=api`. Current-source support
requires adoption through a package that includes this capability.

## Actions repository variables

`repository.settings.actions.variables` corresponds to **Secrets and variables → Actions → Variables → Repository variables**. Navigation-only sidebar groups are omitted from YAML. This current-source feature manages only explicitly named, non-sensitive repository variables:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      variables:
        - name: BUILD_CHANNEL
          value: stable
        - name: OPTIONAL_LABEL
          value: nightly
```

Both `name` and nonempty string `value` are required. Names contain only ASCII letters, digits and underscores, cannot start with a digit or `GITHUB_` (case insensitive), and are unique ignoring case. Internal identity and export names are uppercase. Missing/null/empty values are invalid; quote values that YAML would interpret as numbers or booleans. Without prune, omission and `[]` select nothing and unlisted variables remain unchanged. `--prune actions-variables` makes a present list authoritative for retention; [] deletes all acquired variables, while omission always remains unmanaged. There is no YAML delete or rename member.

Plan reads each selected name individually and shows creates or value updates. Apply reads fresh state, creates absent names with POST, updates changed values with value-only PATCH, skips equal values, and requires exact individual GET readback after a mutation. Only GET 404 means absent under the caller-permissions precondition. Other failed/incomplete reads fail. Mutations and verification failures can have remote effects; earlier writes remain applied, without rollback, retry or concurrency protection.

Ordinary `export` lists all supported Actions repository variables through complete paginated inventory reads. Unavailable/incomplete/unsupported inventories retain prior valid declarations with info. Actions, Agents and Environment inventory pages reject duplicate members, ambiguous aliases of selected fields, malformed Unicode, trailing data and oversized responses. An unavailable preflight prevents deletion; an invalid readback after deletion leaves its effects uncertain. Values are **plaintext** in YAML, exports, detailed plans and Summary. Actions, Agents and Environment variable API requests suppress inherited `GH_DEBUG` transport logging to keep raw request and response bodies out of debug logs. Secrets and organization variables remain outside scope. Agents and Environment inventories are separate independent families; output does not classify or mask variable values.

GitHub's [Variables REST API](https://docs.github.com/en/rest/actions/variables) requires repository Variables read for reads and write for mutations with fine-grained/App tokens. An empty-value create was rejected with HTTP 422 (`Variable value cannot be empty.`); validation rejects empty desired values before requests. No whitespace-trimming rule is inferred. GitHub enforces its [variable limits](https://docs.github.com/en/actions/reference/workflows-and-actions/variables#limits-for-configuration-variables); server rejections are errors. No permission is granted automatically. Free/personal/public UI eligibility and API persistence were observed separately; this is not a runtime entitlement gate. See the [behavior specification](../features/repository-variables.feature.md).

### Variable pruning

`--prune actions-variables`, `--prune agents-variables` and
`--prune environment-variables` select independent collections. Their
comma-separated combination uses the common [prune contract](pruning.md#collection-pruning).
Pruning requires complete name/value inventories, including every page (API
maximum 30 entries per page) and a stable, exact `total_count`. Duplicate names,
missing/null values, count drift and incomplete/denied pages fail before writes.
Prune acquires all current values in a selected collection, not just declared
names; values remain non-sensitive plaintext and can appear in detailed output.
Named export continues to read only its explicit selections.

All candidates are prepared before any remote write. Each family fresh-reads its
inventory, rejects changed candidates, upserts declared values before deleting
its fixed candidates, rechecks the exact acquired name/value before each DELETE,
and requires HTTP 204 plus fresh complete inventory absence. A concurrent change
can still race these checks; there is no lock, retry or rollback. Earlier writes
remain applied after later failure and uncertain DELETE/readback effects are
reported honestly. Plan and detailed Summary display deletion to `absent`, not
an empty value. Status-only output still hides names/values.

Environment-variable pruning touches only listed Environments with an explicit
`variables` array. `variables: []` prunes that Environment; an omitted array does
not. Existing-parent inventory and deletion are bound to its numeric identity,
so detected same-name parent replacement fails. A missing parent is an error
unless the same declaration explicitly creates its Environment policy first;
then no earlier variables are inferred and unexpected deletion candidates are
rejected. Variable-only intent never creates or deletes an Environment, changes
its policy/patterns or accesses secrets. [Environment-object deletion](pruning.md#environment-pruning)
uses the separate `environments` scope; standalone `all` selects both scopes,
while retaining every listed parent, including variable-only entries.

API evidence: [Actions variable lists/deletion](https://docs.github.com/rest/actions/variables)
and [Agents variable lists/deletion](https://docs.github.com/rest/agents/variables).

## Excluded private Actions controls

As of 2026-11-09, `repository.settings.actions.general.access` and `.fork_pull_request_workflows` are unsupported, including empty mappings and explicit false. The former shares a private repository's actions/workflows; the latter controls private fork workflow execution, tokens, secrets and approval. Neither is a configurable control for the target public repository; the exclusion follows visibility, not account plan. See GitHub's [private Actions sharing documentation](https://docs.github.com/en/actions/how-tos/reuse-automations/share-with-your-organization) and [fork workflow settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository).

The old `--actions-access` and `--private-fork-workflows` export options are rejected before repository/API/file work. No alias or migration is provided. Public contributor approval, Actions execution, workflow permissions, retention and cache controls remain separate. Removing these YAML management paths does not change live settings: their dedicated endpoints are not selected by supported public intent, so no private nested state is read-modify-written. Internal raw identities remain available for validation; an attempted private-only export projection fails instead of silently dropping values.
