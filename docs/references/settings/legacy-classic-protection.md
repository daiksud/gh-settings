---
type: Reference
title: "Classic protection: existing v0 contract"
description: "Classic protection: existing v0 contract declaration, ownership and API contracts."
---

# Classic branch protection

Use [Rulesets](rulesets.md) for v1 branch protection. This page covers the classic controls that v0 can export and reconcile on an existing repository. [Support scope](scope.md#feature-adoption-scope)

`repository.settings.branch_protection_rules` manages exact classic rule patterns
and six independent checkboxes, conditional fork syncing, and complete PR-review,
required status-check, deployment and force-push groups. It is separate
from Rulesets and does not clone all branch protection. Current source supports personally owned public repositories;
GitHub documents public availability, including Free, in its
[protected branch guide](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches).

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    branch_protection_rules:
      - id: 123456
        branch_name_pattern: 'release/**'
        require_conversation_resolution_before_merging: true
        require_signed_commits: true
        require_linear_history: true
        allow_deletions: false
        lock_branch: false
        do_not_allow_bypassing_the_above_settings: true
        require_a_pull_request_before_merging:
          enabled: true
          require_approvals: true
          required_number_of_approvals_before_merging: 2
          dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false
          require_review_from_code_owners: false
          require_approval_of_the_most_recent_reviewable_push: true
        require_deployments_to_succeed_before_merging:
          enabled: true
          environments: [staging, production]
```

## Select a rule

`id` is an optional positive numeric GraphQL `databaseId`. Without it an entry is
create-only: an existing exact pattern is a collision, even when its controls
match. With ID, a missing or replaced rule fails instead of creating or adopting.
An ID-selected pattern change updates that identity; it never deletes and
recreates the rule. A rename to another rule's exact pattern is rejected. Existing
same-pattern rules remain independently selectable by ID. Duplicate IDs and
conflicting requested patterns fail before this family's writes.

The pattern is required, nonblank and free of control characters. Its exact case,
spaces and slash syntax are preserved. GitHub owns pattern matching and syntax;
`*` does not cross `/`, and `qa/*` differs from `qa/**/*`. A matching branch need
not exist. Only one classic rule applies to a branch, and creation age affects
priority. New rules execute in YAML order; existing rules retain identity and
creation age. Export sorts by numeric ID for stable presentation, without claiming
that this is priority order. See GitHub's
[pattern and priority guide](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule).

Each checkbox is optional. Omission preserves its latest observed value; explicit
`false` requests false. Null, native GraphQL aliases and unlisted controls are
rejected. Without explicit classic prune, omitted collections and `[]` manage no
objects, and unlisted rules are preserved.
[Classic rule pruning](#classic-branch-protection-pruning) adds whole-rule deletion
through a command option; there is no YAML deletion marker or reordering operation.

Plan compares only the requested pattern and controls. Apply first acquires all
selected identities and rejects collisions. Each existing rule is read again by
its acquired opaque node ID before comparison; matching intent sends no mutation.
A changed request sends only changed declared inputs. Enabling linear history
requires fresh squash or rebase availability, after any earlier repository updates;
it does not implicitly enable a merge method. Creation sends the exact pattern
and declared controls; other defaults are controlled by GitHub.

## Lock and administrator enforcement

Lock branch makes matching branches read-only. Selecting lock alone sends no
fork-sync value; an omitted sync setting must survive updates unchanged.
GitHub documents that locked branches cannot be deleted, even when deletion is
otherwise allowed. This checkbox does not rewrite the separate deletion setting
or promise that it will take effect while locked. Readback verifies stored
settings, not an attempted push, deletion or effective enforcement.

`do_not_allow_bypassing_the_above_settings` matches the UI checkbox: true applies
this rule's protections to administrators; false removes this flag's enforcement.
It maps directly to GraphQL `isAdminEnforced`. Omission preserves the acquired
boolean, and neither direction changes actor allowance lists, other rule controls
or repository Rulesets. Overlapping policies can still affect effective access.
The unsaved target form was inspected on 2026-11-14. The
[maintained provider's direct read/write mapping](https://github.com/integrations/terraform-provider-github/blob/1b9c5c13eb840a72c99d088f3bf3b5e00b950c04/github/resource_github_branch_protection.go)
corroborates this polarity despite the ambiguous GraphQL description. This is
mapping evidence, not a live test of administrator bypass or effective enforcement.

## Pull request reviews

`require_a_pull_request_before_merging` owns one complete active policy. Omission
and `{}` leave it unmanaged; `enabled: false` forbids all child fields. With
`enabled: true`, explicitly set `require_approvals` and all three review checkboxes
shown above. Approval true requires an explicit count from 1 through 6; approval
false forbids that count and requests native count 0. Do not omit active children
or use null/native aliases. No default approval or parent enablement is inferred.
Code Owner, stale-dismissal and latest-push choices remain independent of generic
approval count. Stored configuration does not establish applicable Code Owners,
review eligibility or effective merge enforcement.

Creation and disabled-to-enabled transitions send all five native policy fields,
including false booleans and count 0. Within an already-enabled policy, only changed
fields are sent. Disabling sends only `requiresApprovingReviews: false`, without
promising storage of inactive count or review booleans. Export hides inactive
children; active count 0 exports approval false, while 1 through 6 exports approval
true with its count. Unknown active counts are unavailable rather than clamped.
Omission preserves every raw flag, count nullness and review boolean exactly.

Plan rejects enabled-to-disabled transitions with `restrictsReviewDismissals`,
review-dismissal actors or PR bypass actors. Apply repeats this check on the fresh
rule before mutation. This is a conservative boundary for unowned dependent
protections, not evidence that GitHub was observed deleting them. Matching disabled
state remains a no-op even with latent actors. Changing an active approval count
to zero is allowed while those unowned settings are independently preserved. The
client never adds or removes bypass, review-dismissal or push actors.

The personal-public unsaved form inspected on 2026-11-14 offered approval counts
1 through 6 and left the other review checkboxes visible when Require approvals
was unchecked. The
[maintained provider's independent parent/count mapping](https://github.com/integrations/terraform-provider-github/blob/1b9c5c13eb840a72c99d088f3bf3b5e00b950c04/github/util_v4_branch_protection.go#L155-L204)
and the official GraphQL read/create/update fields corroborate PR-required with
zero generic approvals. No live rule or merge enforcement was tested.

## Required deployments

`require_deployments_to_succeed_before_merging` is a complete active group.
Omission and `{}` leave it unmanaged. `enabled: true` requires an explicit
`environments` list, including `[]`; `enabled: false` forbids that list. Names are
an exact-spelling set: duplicates, blank names, controls and null values fail.
No trimming, case folding or Environment creation is performed. Export sorts
active names and emits only `enabled: false` for disabled groups. The unsaved
personal-public form offered an empty selection on 2026-11-14; this is not proof
that the server accepts it or that it enforces useful protection.

A fresh matching rule sends no mutation and makes no Environment request. Before
a needed mutation with an explicitly enabled nonempty group, the writer completely
paginates identity-only Environment observations and resolves each exact name to
its node ID. This also applies when another declared control caused the write.
The [GraphQL Environment schema](https://docs.github.com/en/graphql/reference/deployments#environment)
permits a null numeric database ID; the node ID remains the identity anchor.
After independent rule readback, selected names and node IDs are checked again.
Missing, ambiguous or foreign references fail before mutation; failed or changed
post-write observations stop later work with uncertain effects. New unselected
Environments do not cause drift. Disabling, omission and an empty active set do
not acquire Environment identities. No-op equality does not establish Environment
existence or deployment success.

Deployment-reference acquisition failures use bounded request, identity and page
categories. Upstream GraphQL/HTTP/transport text and response-controlled member
names are excluded from CLI and Action diagnostics. This leaves pre-write failure
and post-write uncertainty, earlier progress and stop behavior intact.

Referenced Environments must already exist at the Classic write stage, which
precedes Environment creation and updates. Create a missing Environment in a
separate apply. The client does not create, rename or deploy to a reference.
Exact spelling is a conservative selection rule, not a claim about GitHub name
matching. Replacement before fresh acquisition and later remote races are not
ruled out. When Environment prune is selected, an explicitly enabled reference
to a fixed deletion candidate fails global preflight before any writes or
unarchive, even for a matching rule. Retain that Environment explicitly or change
the requirement; references are never silently added to retained names. Conflict
detection conservatively uses Environment deletion's existing name equivalence;
reference selection remains exact. This check does not adopt omitted requirements or certify their viability after an
intentional Environment deletion.

Omitting the deployment group preserves its raw flag, nullness, list order and
inactive names exactly. Selecting it owns the complete active configuration;
disabling sends only `requiresDeployments: false` and makes no promise about
hidden inactive-list storage. Complete known PR, check/App binding, deployment,
fork-sync, force-push and actor-allowance state is acquired as preservation
evidence. Omitted groups stay unowned. Null stays distinct from zero or empty; complete disabled-group raw values are not rewritten.
Missing, malformed, partial, foreign or ambiguous observations fail. All nested
actor connections are paginated. After mutation acknowledgment, an independent
query verifies original requested controls and all omitted known companions.
Collateral changes or failed readback stop later work with uncertain effects;
earlier verified writes remain applied. No retry, rollback or atomic lock is
promised. Stored-value readback does not establish effective branch enforcement.

## Fork syncing

`allow_fork_syncing` is an optional Boolean for a personally owned public fork.
Selecting either `true` or `false` requires explicit `lock_branch: true` in the
same rule. Omission preserves the native value; null and non-booleans fail. This
does not implicitly lock a branch, fetch or merge upstream changes, create a fork,
or change permissions:

```yaml
branch_protection_rules:
  - id: 123456
    branch_name_pattern: main
    lock_branch: true
    allow_fork_syncing: false
```

The official [creation guide](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule#creating-a-branch-protection-rule)
documents Allow fork syncing under Lock branch. The GraphQL
`lockAllowsFetchAndMerge` Boolean has direct true/false polarity. This is documented
UI evidence; no authenticated fork form or live syncing was tested.

Plan and apply reject a selected value on a confirmed non-fork. Complete typed
`isFork` evidence is required throughout inventory, actor pages, fresh selection,
acknowledgment and independent readback. Missing evidence is unavailable, never
assumed false. A changed fork status fails target verification, including private
whole-rule prune rechecks and final absence confirmation.

Export includes the observed value only for locked fork rules, together with
`lock_branch: true`. Unlocked rules and non-forks omit this inactive projection;
their exact native Boolean remains private preservation/prune evidence. Missing
active projection fails acquisition instead of producing incomplete YAML. Existing
updates send only changed declared values; creation sends both selected lock and
sync values. A matching rule makes no mutation. Selecting lock alone never sends
fork syncing, and unlocking with sync omitted still verifies that its raw value
survived. Unexpected changes remain uncertain effects after mutation, not success.

## Force pushes

`allow_force_pushes` owns a complete active group. Omission or `{}` leaves it
unmanaged. `enabled: false` forbids children. `enabled: true` requires
`who_can_force_push: everyone` or `specified_actors`. Everyone forbids `actors`;
specified actors require a nonempty authoritative set:

```yaml
allow_force_pushes:
  enabled: true
  who_can_force_push: specified_actors
  actors:
    - type: user
      id: "<opaque User node ID>"
    - type: app
      id: "<opaque App node ID>"
```

Each actor requires exactly `type: user|app` and an opaque nonblank string ID.
Nulls, unknown members/types, case variants, controls and duplicate IDs fail.
Logins, numeric IDs and installation IDs are not substitutes. Actor ordering is
insignificant; export sorts identities. No App installation or access grant is
performed. User and ChatGPT Codex Connector App selection were observed on the
unsaved personal-public form on 2026-11-14. No rule was saved or force push tested.
The [pinned provider mapping](https://github.com/integrations/terraform-provider-github/blob/1b9c5c13eb840a72c99d088f3bf3b5e00b950c04/github/util_v4_branch_protection.go)
corroborates the native mapping alongside the official GraphQL schema.

Everyone maps to `allowsForcePushes: true`. Specified actors map to false plus the
complete `bypassForcePushActorIds` set. False with no actors means disabled;
false alone does not disable existing allowances. Explicit disabling sends false
and an empty actor list together. Selecting a specified list replaces the complete
membership, including previously acquired Team allowances. Everyone changes only
the boolean and preserves all latent actor identities; omission preserves both
native controls. Unexpected latent changes fail verification, rather than implying
that GitHub always retains inactive storage.

Before any needed rule mutation with specified actors selected, the client checks
every desired node ID and User/App type, even if the actor set matches and another
flag caused the write. Only a wholly matching rule skips these extra lookups.
The scalar mutation acknowledgment does not prove actor membership; independent
complete paginated readback verifies actors and omitted raw state. Other protections
and repository access permissions still determine effective force-push access.

Team is a known acquired identity, but its personal-target selection remains
unconfirmed and it is not accepted in desired YAML. Other Classic updates and
whole-rule prune can preserve or inspect that known state. Detailed plans may show
an active Team in the acquired before-policy, never as a desired input. An active
specified set containing Team makes automatic Classic export unavailable, retaining
compatible prior intent instead of dropping it. Everyone can export normally while
latent Team identities remain private. Unknown actor kinds or incomplete reads fail.

## Creation IDs and export

Only independently confirmed creations receive guarded numeric-ID writeback,
using the original unique pattern. The existing file/symlink/concurrent-edit
checks, `--no-writeback` and GitHub Actions suppression apply. If confirmation or
local recording fails, retain the reported numeric or opaque recovery identity
and inspect the remote rule before retrying; never guess or decode node IDs.

Automatic export projects every readable rule's ID, pattern, six checkboxes, applicable fork syncing and
active PR-review, status-check, deployment and force-push groups. It does not export or adopt the other controls.
Unavailable acquisition retains compatible prior declarations with bounded info; it never fabricates an
empty or partial collection. The complete known query follows the inspected
[GraphQL branch-protection schema](https://docs.github.com/en/graphql/reference/branches).
GraphQL cannot reveal future unselected schema fields. Additional actor controls remain separate work;
this is not complete classic UI coverage. Queries and mutations use existing
[GraphQL authentication](https://docs.github.com/en/graphql/guides/forming-calls-with-graphql)
without granting permissions. The REST concrete-branch endpoint is not used for
wildcard rules, and its permission table does not prove every GraphQL App grant.
See the [behavior and verification specification](../../development/features/classic-protection.feature.md).

## Required status checks

The optional `require_status_checks_to_pass_before_merging` group owns the complete
active policy when selected:

```yaml
require_status_checks_to_pass_before_merging:
  enabled: true
  require_branches_to_be_up_to_date_before_merging: false
  status_checks:
    - name: screening
      accept_updates_from: APP_opaque_node_ID
    - name: legacy
      accept_updates_from: any
```

Omission or `{}` leaves the group unmanaged. A nonempty group requires `enabled`.
With `true`, the strictness boolean and complete `status_checks` list are required;
`[]` is valid, but does not establish useful check enforcement. With `false`, child
controls are forbidden: this requests deactivation only. Null, wrong types, unknown
members and native GraphQL aliases are rejected.

Check names retain exact case and bytes and must be nonblank and free of control
characters. Names must be unique, even when sources differ; this is a conservative
product boundary. Distinct names may share one App. `accept_updates_from` is either
the exact string `any` or an opaque GraphQL App node ID. Numeric App/installation
IDs, slugs and inferred providers are not substitutes. The writer always sends
`appId`; GitHub's [input contract][classic-check-input] otherwise infers the most
recent source. gh-settings does not discover recently run checks, grant access or
require App installation.

For an active observed policy, both native check lists must be present and
nonnull, names must be unique, and the legacy context list must contain exactly
the same names as the App-bound list. This coherence check is a supported-observation
restriction, not a guarantee about every GitHub rule. A malformed, missing or
partial observation makes the whole classic family unavailable. An explicit null
App means `any`; a missing App member is unknown. Inactive export contains only
`enabled: false`. Its typed latent lists, including nullness, order and source
bindings, remain private preservation evidence when the group is omitted.

Creation or enabling sends all three writable members, including explicit false
strictness and an empty list. Already active updates send changed members only;
check order does not cause a write. Disabling sends only the false gate and permits GitHub to reset latent child values. The legacy context alias
is never sent. Omitted groups must retain their exact raw state across other
updates; selecting an active group owns its complete semantic policy.

A matching fresh rule sends no mutation or App lookup. Before a needed write
with a selected active group, each distinct non-`any` source must resolve through
the same credential to the exact [App node][classic-check-app]. This applies even
when only another declared checkbox changes. After the complete mutation
acknowledgment, independent rule/actor readback verifies the entire original
request and omitted state, then the same App identities are checked again.
Prewrite failures stop before mutation; failed acknowledgment or subsequent
verification leaves uncertain effects and creation IDs are recovery information
only. Readback proves stored settings, not check execution or effective protection.

## Classic branch-protection pruning

`--prune branch-protection-rules` and standalone `all` add unlisted whole-rule
removal when `repository.settings.branch_protection_rules` is present. An omitted
collection stays unmanaged. `[]` selects all initially acquired rules; a nonempty
list retains declared numeric IDs and newly created rules. All retained rules must
be listed. ID-less exact-pattern collisions, missing/foreign IDs and conflicting
renames fail before any remote write, including unrelated settings or unarchive.
Ordinary create/update retains partial-control ownership and complete active
PR-review, status-check, deployment and force-push ownership.

Deletion removes the entire rule, including unowned protections. An overlapping
rule may then take effect; recreation cannot restore creation-age priority. The plan shows
this consequence and each complete supported before-policy, without raw private
companions. It does not delete branches. Inspect the complete plan before apply.

Before any remote write, acquire the complete rule/actor inventory and freeze each
unlisted candidate's repository ID, numeric ID, opaque node ID, exact pattern,
six flags, fork applicability and conditional syncing, active PR-review/status-check/deployment/force-push projections and all known raw companions,
including inactive PR-review, status-check, deployment and force-push values. Missing, denied, partial,
unsupported or inconsistent evidence blocks the whole preflight. A retained rule's unreadable
state can also block this complete inventory. Before this family's work, recheck
the initial candidates: disappearance, replacement or state drift stops execution.
Newly observed unlisted rules remain untouched, including when the initial
candidate set was empty. They never expand the frozen selection. Only this
invocation's confirmed unarchive adjusts the acquired archive state; unrelated
repository merge-method changes do not constitute deletion-policy drift.

All declared classic creates/updates and required creation-ID writeback precede
fixed deletions. Immediately before each deletion, independently read its selected
node and complete actor state and require an exact snapshot match. The official
[GraphQL deletion contract](https://docs.github.com/en/graphql/reference/branches#deletebranchprotectionrule)
accepts the acquired opaque ID. Its nullable payload has no deleted-ID echo:
require an error-free `DeleteBranchProtectionRulePayload` acknowledgment, then an
independent complete repository inventory proving both old node and numeric ID
absent. A distinct retained same-pattern rule is allowed. A null node alone is not
absence confirmation.

Mutation errors, partial GraphQL errors, malformed acknowledgments and incomplete
or mismatching readback stop later work with uncertain effects and the fixed
recovery IDs. Earlier confirmed writes remain applied. No retry, rollback,
recreation or deletion-ID writeback occurs. The final read/delete race remains,
and future unselected GraphQL fields cannot be detected. Private companion
snapshots stay out of reports and inherited `GH_DEBUG` logs. No live deletion or
credential-grant sufficiency was established by the local fixture tests.

[classic-check-input]: https://docs.github.com/en/graphql/reference/branches#requiredstatuscheckinput

[classic-check-app]: https://docs.github.com/en/graphql/reference/apps#app

[Settings index](README.md) · [GitHub App permissions](permissions.md)
