---
type: Reference
title: "Collection ownership and pruning"
description: "Collection ownership and pruning declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Collection pruning

Normal apply creates/updates the declared independent objects and preserves
unlisted objects. `plan --prune labels,rulesets` previews additional deletions;
`apply --prune labels,rulesets` executes them alongside all other declared work.
Use one comma-separated option, not repeated flags. Empty elements, duplicate or
unknown names fail locally. Current source supports `labels`, `rulesets`, `actions-variables`,
`agents-variables`, `environment-variables`, `actions-policies`, `environments`, `collaborators`,
`branch-protection-rules`, `autolinks`, `deploy-keys`, `webhooks` and `runners`. Standalone `all` selects all currently supported scopes; combining it with another
scope is invalid. Each collection still requires explicit presence in YAML.

Omitted lists remain unmanaged. Explicit `[]` selects all current objects in that
scope; a nonempty list prunes objects whose identities are not declared (IDs for
labels/Rulesets/Actions Policies/classic rules/Autolinks/deploy keys/webhooks/runners, canonical names for variables, folded names for Environments, resolved immutable user IDs for collaborators). ID-less label/Ruleset/Actions Policy/classic rule/Autolink entries
remain create-only and never adopt a name collision. Acquire complete inventories
and validate every prune candidate before any remote write, including unrelated
settings writes. Unsupported candidates fail the whole preflight. Create/update
before deletion; keep the prepared deletion set fixed, check drift before DELETE
and verify absence afterward. Completed earlier writes survive a later failure;
readback failure can leave uncertain effects. There is no lock or rollback.

Deleting labels removes their Issue/PR assignments; recreating them does not
restore assignments. Automatically created Dependabot labels are deletion
candidates too if undeclared. Plan displays this consequence. Topics, PR exemption
lists and deployment branch/tag pattern replacement are unchanged. The
[behavior specification](../features/prune.feature.md) records stages and tests.

### Environment pruning

`--prune environments` owns unlisted parent lifecycles only when
`repository.settings.environments` is present. Omission remains unmanaged even
with `all`; `[]` selects every acquired Environment. A nonempty list retains all
listed names, including variable-only entries and `variables: []`. Policy upserts
and all selected Environment variable work finish before parent deletions. New
parents never join the fixed deletion set. No ID is written to YAML and no
`delete` member is accepted.

Before any writes, acquire every candidate's positive parent ID, complete
supported policy, active raw pattern identities, variable inventory and secret
name/creation/update metadata. The dedicated custom-deployment-protection endpoint
must independently confirm a complete empty list. Nonempty, denied, malformed or
normalized unsupported state blocks the whole operation. Retained names do not
need deletion snapshots. Recheck the fixed candidate set before Environment work
and each candidate immediately before DELETE. Account only for this command's
confirmed unarchive; other target/archive or observable child drift stops deletion.
After DELETE 204, a complete fresh inventory must omit both the original ID and
folded name. Same-name recreation fails confirmation. Reads are not an atomic
conditional-delete lock; a race remains between the final check and deletion.

Parent deletion removes contained configuration, including variables and patterns.
[GitHub documents](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments#deleting-an-environment)
secret/protection removal and failure of jobs waiting on protection rules. A later
workflow reference can recreate a name without restoring the deleted resource's
protection. Secret values are never read, exported or backed up; YAML cannot
restore them. Secret metadata is only private drift evidence, not complete
rotation detection, and is excluded from plans/reports. Metadata pages must be
complete, bounded and unambiguous; duplicate members, case aliases, malformed
Unicode and unknown fields are not evidence of equality. Prune snapshot
transport ignores inherited `GH_DEBUG` so child evidence is not copied to debug logs.

[Environment list/detail](https://docs.github.com/en/rest/deployments/environments)
and [custom-protection reads](https://docs.github.com/en/rest/deployments/protection-rules#get-all-deployment-protection-rules-for-an-environment)
require Actions read. Child variable and [secret metadata reads](https://docs.github.com/en/rest/actions/secrets#list-environment-secrets)
require Environments read; deletion requires Administration write. These are
preconditions, not automatic permission grants. Inactive patterns are inaccessible
under the documented custom-mode prerequisite: explicit parent deletion is allowed
with that observation limit, without claiming an acquired empty latent list.
Detailed plans show complete supported before-policy and absence, with deletion
consequences; status-only output hides names and policy. Failed prewrite reads
make no deletion; failed DELETE/absence verification can have uncertain effects.
Earlier verified writes survive later failure. No retry or rollback occurs.
