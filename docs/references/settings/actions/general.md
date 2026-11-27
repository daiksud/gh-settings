---
type: Reference
title: "General Actions settings"
---

# General Actions settings

Choose the control under **Settings → Actions → General**. Execution policy, workflow permissions, public-fork approval, workflow retention and cache limits can be declared independently.

## Actions execution policy

All paths below are relative to `repository.settings.actions.general.actions_permissions`:

| Field | Choice / type |
| --- | --- |
| `policy` | `allow_all_actions_and_reusable_workflows`, `disable_actions`, `allow_owner_actions_and_reusable_workflows`, `allow_owner_and_select_non_owner_actions_and_reusable_workflows` |
| `allow_actions_created_by_github` | boolean; selected-policy companion |
| `allow_actions_by_marketplace_verified_creators` | boolean; selected-policy companion |
| `allow_or_block_specified_actions_and_reusable_workflows` | ordered string array; selected-policy companion |
| `require_actions_to_be_pinned_to_a_full_length_commit_sha` | boolean |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        actions_permissions:
          policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows
          allow_actions_created_by_github: true
          allow_actions_by_marketplace_verified_creators: false
          allow_or_block_specified_actions_and_reusable_workflows: ['my-org/*', '!my-org/blocked@*']
          require_actions_to_be_pinned_to_a_full_length_commit_sha: true
```

Selecting an allow policy explicitly enables Actions and selects its policy; `policy: disable_actions` declares only disabled execution, leaving latent allowed policy unmanaged. There are no separate public API-shaped enabled/allowed fields. Omitting `policy` does not implicitly enable Actions. SHA pinning is independent; reusable workflows can still use tags.

Companions can accompany the selected non-owner policy or an omitted policy, not another explicit policy. An omitted policy requires an already selected observed policy for companion updates. Transition from a nonselected policy requires all three companions: inactive values cannot be acquired or invented. Plan shows them as unavailable before the transition. Existing selected policy permits partial declarations merged with complete fresh companions.

Patterns are at most 1000 nonempty strings. An explicit array replaces the entire sequence, and `[]` clears it. Case, order, duplicates, wildcards and negative entries are preserved; the tool does not emulate matching. Removing a negative entry may permit more actions. GitHub documents pattern applicability limits; configuring a list alone does not prove enforcement on private repositories.

Ordinary `export` includes the execution policy, SHA control and visible acquired companions in the same section. Disabled/other policy hides selected companions; complete raw acquisition is still required. Failed or incomplete acquisition retains this family’s prior declaration with info, without erasing independently acquired Actions siblings. Correctly typed unknown execution policies follow the observation policy below.

Apply reads complete fresh state and skips matching intent. A needed base PUT includes the endpoint-required enabled value, using the fresh companion when omitted, and only changed optional fields. A needed selected PUT preserves omitted companions from the complete active read. Policy changes precede selected writes. When enabling a disabled repository with selection, it selects policy while disabled and installs companions only as needed, then enables. Independent readback verifies the original complete requested state and preserved raw companions, including initially matching declared values. Repeated matching apply performs no PUT.

Operations are not transactional: an already enabled target can temporarily use remembered/default selection between writes. Earlier policy changes survive later refusal; failed writes/readback have uncertain effects. There is no rollback, saved-plan execution, automatic retry, lock or permission expansion. Existing Administration read/write and organization/enterprise policy remain prerequisites; policy changes can stop future workflows. See [execution policy](../../../development/features/actions-execution-policy.feature.md).

### Execution-policy observation normalization

A complete, correctly typed unknown `allowed_actions` string (including empty or whitespace) retains its exact raw value and projects to the named `allow_all_actions_and_reusable_workflows` fallback. Both acquired booleans remain unchanged. With execution enabled, export warns and emits that fallback; a declared policy keeps one UI decision actionable even at fallback equality. Disable owns one native field; enabling owns two, or five with all three selected companions. Managed counts reflect native fields internally.

With execution disabled, the visible choice is the known `disable_actions`, so the latent unknown policy produces no projection warning. Export omits latent choices, and a matching Disable plan has no differences. Enabling requires an explicit canonical policy; selected intent still needs all three companions and configures them before enabling. Private radio reports retain raw provenance for closed report validation/presentation but detailed output does not disclose hidden policy; status-only output suppresses all policy values.

A needed base update sends only changed declared policy/SHA members with the endpoint-required enabled value. Disable and SHA-only updates preserve the exact raw policy; SHA-only updates do not warn about omitted policy intent. Fresh raw readback must confirm declared canonical policy or preserved raw policy, both booleans and any managed selected companions. An unchanged unknown string cannot confirm a declared fallback, and a different unknown token cannot confirm preservation. Failed verification retains uncertainty about writes and earlier completed resource work. Direct repeated apply skips raw matches, including an already disabled target with unknown latent policy. Missing/null/wrong types and failed acquisition never become fallbacks.

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

Each is independent and optional. Ordinary `export` includes both after complete acquisition, independently of execution policy. Omission is unmanaged, false remains explicit, and unknown choices/null/wrong types fail locally. Apply compares a complete fresh observation, skips matching intent and sends only changed declared workflow-policy members. Independent readback verifies the original declaration, including initially matching values, and preserves omitted raw companions. A matching repeat apply sends no update. Direct apply remains nontransactional: organization refusal can leave an earlier execution-policy update applied, and a failed PUT/readback can have uncertain effects. This declaration does not create tokens or waive a PR's approval requirements. See [workflow permissions](../../../development/features/workflow-permissions.feature.md).

A complete successful workflow-policy GET with a correctly typed permission string outside the UI choices uses `read_repository_contents_and_packages_permissions` and emits a warning naming the YAML field, raw token and fallback. Missing/null members, wrong types and failed reads remain acquisition errors. Export saves the UI fallback without changing GitHub. Plan retains an actionable write-required comparison even if the desired choice equals that fallback; it does not claim independently observed zero drift. Apply sends only declared members and confirms the requested permission against a fresh raw readback. An unchanged out-of-UI token fails verification with possible write effects retained. An omitted permission remains unmanaged and is verified as an unchanged raw companion during approval-only writes, without replaying or printing the fallback.

## Fork PR workflow policies

Public-fork approval is `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`:

- `require_approval_for_first_time_contributors_who_are_new_to_github`
- `require_approval_for_first_time_contributors`
- `require_approval_for_all_external_contributors`

A complete successful public-fork policy GET with an out-of-UI string projects to `require_approval_for_first_time_contributors` (native `first_time_contributors`). Original unknown, empty or whitespace strings remain exact provenance. Missing/null members, wrong types, unsupported visibility and failed reads remain errors. Export warns and saves only the UI choice; plan retains one write-required decision even when desired intent equals the fallback. Omitted intent remains unmanaged.

Apply checks exact raw values before skipping a write and in fresh readback after PUT. A retained unknown value fails verification with possible write effects; a canonical exact match remains a no-op. Each successful managed apply read can warn: the initial application read, updater preflight and a normalized readback. Pure plan readers emit no stderr; detailed plan/Summary carry the decision warning, while status-only Summary suppresses values.

Ordinary `export` observes this public-repository control. Visibility, complete observations, narrow payloads, readback, redirect rejection and partial-failure reporting remain required. Upper policy may refuse or ignore changes; no retry, rollback or credential expansion occurs. See [fork workflows](../../../development/features/fork-workflows.feature.md).

## Check, run, status, artifact and log retention

Use integer `repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention` (1–400 days). It translates to the existing retention endpoint. Ordinary `export` attempts retention automatically. The server's maximum is read-only metadata and is never replayed as intent. Existing repository visibility, range and freshly observed cap checks apply; a plan/readback cannot invent missing cap information or prove licensing. Apply verifies the requested value and preserves failure/uncertainty reporting. See [retention](../../../development/features/retention.feature.md).

## Actions cache policy

The optional `cache` section under `repository.settings.actions.general` manages two independent integer controls:

| Field below `cache` | Value |
| --- | --- |
| `cache_retention` | 1–365 days, up to 90 for public repositories |
| `cache_size_eviction_limit` | 1–10000 GB |

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

These controls are independent of check/workflow/artifact/log retention and Actions execution policy. Export attempts both cache settings automatically. Plan and apply skip each omitted control. Empty `cache: {}` declares nothing. Nulls, strings, fractions and out-of-range integers are invalid.

A cache size eviction limit above 10 GB can incur additional charges. It is an eviction threshold, **not a spending budget**. Retention does not guarantee that caches survive until that age: eviction or manual deletion can remove them sooner. gh-settings manages repository limits only, not cache entries, usage, billing budgets, or organization/enterprise policy.

Each selected control requires a complete current read before its PUT and an independent readback afterward. Matching values skip PUT. GitHub can reject a value because of a tighter parent policy; gh-settings does not clamp it or change that policy. Missing, denied or malformed reads fail rather than supply defaults. A successful HTTP response with different readback fails verification and may have changed remote state. Later failure leaves earlier successful writes applied, without rollback or retry. See [behavior and API boundaries](../../../development/features/actions-general-controls.feature.md).

## Excluded private Actions controls

Private-repository sharing and fork run, token and secret controls are outside the public-repository target. Validation rejects `general.access` and `general.fork_pull_request_workflows`, including empty mappings. To choose when a public fork needs approval, use [public contributor approval](#fork-pr-workflow-policies). See GitHub's [private Actions sharing documentation](https://docs.github.com/en/actions/how-tos/reuse-automations/share-with-your-organization) and [fork workflow settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository).

[All Actions settings](README.md) · [GitHub App permissions](../permissions.md#actions)
