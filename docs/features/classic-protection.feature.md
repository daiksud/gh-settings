# Classic branch-protection patterns and independent controls

The [configuration contract](../settings.md#classic-branch-protection) owns public
fields, pattern semantics, identity, preservation, creation defaults and recovery.
Ordinary reconciliation manages conversation resolution, signed commits, linear
history, branch locking, conditional fork syncing, administrator enforcement, PR-review requirements, required
deployment environments and the Allow deletions checkbox, separately from Rulesets. Explicit
[classic prune](prune.feature.md#classic-branch-protection-deletion) additionally
removes whole unlisted rules. Check editing and force-push
actor editing remain separate work.

## Behavioral boundaries

- Complete GraphQL repository rule inventory covers patterns with zero matching
  branches. Target and rule identities must agree. Both inventory and each nested
  actor allowance connection require complete pagination; errors with partial data,
  missing/null required values and repeated cursors cannot become empty policy.
- Positive numeric IDs select existing rules; opaque node IDs remain acquired
  metadata. Exact ID-less pattern collisions fail. Existing duplicate patterns may
  remain ID-distinct; newly conflicting requests fail before family writes.
- Creation executes in declaration order and sends declared controls only. An
  existing update reads its latest state, verifies original identity/pattern,
  skips matches and writes changed declared members. Linear-history enablement
  needs a current squash/rebase method without changing one implicitly.
- Independent readback checks original requested state and all known omitted
  companions, retaining raw nullable values and App/actor identities. The queried
  schema is bounded: future unselected GraphQL members cannot be detected.
- Prewrite failures prove no attempted mutation; failed acknowledgment/readback
  leaves uncertain effects. Earlier confirmed writes and created-ID records remain.
  No automatic retry, deletion/recreation, rollback or enforcement proof is implied.
- Export is a supported-control projection. An unavailable family retains prior
  compatible intent with info; local validation/output/save failures remain fatal.
  Private report/summary behavior and guarded ID writeback use existing boundaries.

- Lock branch supports explicit lock and unlock, including false-valued export.
  Omitted lock state and fork-sync values are independently preserved.
  Unexpected fork-sync changes fail after the write with uncertain effects.
  Stored lock and deletion flags do not imply deletion is permitted while locked.

- Administrator enforcement maps true/false directly to `isAdminEnforced`, with
  omission preserving the acquired value. Both creation and update verify the
  requested value independently; incomplete acknowledgment or collateral changes
  remain failures. No bypass actors or unrelated protections are implicitly edited.
- Required deployments form a complete active group: omission/{} is unmanaged,
  disabled forbids a list and enabled requires an explicit exact-name set. Empty
  selection is permitted locally based on the unsaved UI, without a server or
  enforcement guarantee. Export sorts active names and hides inactive storage.
- A matching rule skips Environment reads. A needed write with selected nonempty
  active references binds a complete identity-only inventory, permits nullable
  numeric IDs and verifies selected node IDs after independent rule readback.
  Missing references stop before mutation; post-write replacement is uncertain.
- Unselected raw deployment flag, nullness, order and inactive names are preserved.
  Selecting disabled mode owns the flag without promising inactive storage. Other
  raw companions remain protected; Classic prune retains all raw deployment data.
- An enabled reference to a fixed Environment deletion candidate fails global
  preflight before unrelated writes or unarchive. References do not silently retain
  Environments. Classic still precedes Environment creation; no reference is created.

- PR reviews form a complete active group. Omission/{} remains unmanaged; disabled
  forbids children; enabled requires approval choice and all review checkboxes.
  Approval false requests count zero, while true requires a count from 1 through 6.
- Create and activation send all five fields explicitly; active updates send only
  changes. Disabled export hides latent members, which remain exact private
  preservation/prune evidence when the group is omitted.
- Plan and writer-fresh checks block deactivation with dismissal restrictions,
  dismissal actors or PR bypass actors. Matching disabled state and count-to-zero
  stay allowed. Selected deactivation makes no inactive-child retention promise;
  other controls and all unowned actor identities must survive independently.

## Verification mapping

- Literal YAML booleans, omission, null/type/alias rejection and exact patterns:
  `internal/workspace/ui_classic_protection_test.go`, `ui_classic_lock_test.go`, `ui_classic_admin_test.go` and
  `ui_classic_deployments_test.go`; `ui_classic_reviews_test.go` covers the complete PR group.
- Identity/collision selection, creation order, partial equality, nullable raw
  companions and supported export: `internal/settings/classic_protection_test.go`
  and `classic_deployments_test.go` and `classic_reviews_test.go`.
- Literal GraphQL inputs, complete 101-rule/101-actor pagination, unowned companion
  drift, fresh no-ops, create recovery and failure boundaries:
  `internal/github/classic_protection_test.go`, `classic_lock_test.go`, `classic_admin_test.go` and
  `classic_deployments_test.go`; `classic_reviews_test.go` checks activation, count-zero,
  guarded deactivation and raw preservation.
- Family preflight, declaration order, unarchive, actual progress and export
  retention and Environment-prune conflicts: `internal/app/classic_protection_test.go`
  and `classic_deployments_test.go`.
- Closed report, escaped Summary and status-only privacy:
  `internal/actions/classic_protection_test.go`.
- Supplied executable export/plan/apply/repeat, guarded numeric-ID recording,
  wildcard rules, private Action output and unavailable export:
  `tests/acceptance/classic_protection_test.go`.
- Actual supplied Action lock/admin enable/disable/no-op, private plan and companion drift:
  `tests/github-actions/classic_lock_acceptance_test.go`.
- Actual supplied Action deployment enable/disable/empty/no-op, private output and
  reference replacement failure: `tests/github-actions/classic_deployments_acceptance_test.go`.

- Actual supplied Action PR activation/zero/disable/no-op, guarded rejection, private
  plans and readback drift: `tests/github-actions/classic_reviews_acceptance_test.go`.

The PR-review and required-deployment literal regressions fail on their unchanged
sources because the respective group key is unknown. Automated verification uses local fixtures
and dummy credentials. Official schema and read-only empty personal-public
inventories establish the endpoint, not populated storage behavior, mutation
permission or effective enforcement. No live branch protection was changed.

## Conditional fork syncing

Both Boolean values require explicit `lock_branch: true` and a confirmed personal
public fork. Export shows the value only when locked and applicable. Otherwise
the projection is omitted while exact raw state remains available for preservation
and whole-rule pruning. Lock-only and unlock requests never implicitly own sync.

Required typed `isFork` is bound across every inventory/actor page, fresh selection,
creation prerequisites, ACK, readback and private prune target. Changed or missing
evidence stops work; postwrite failure keeps uncertain effects and recovery separate
from confirmed success. Reports validate the explicit lock prerequisite when the
conditional field is present, but private acquisition owns applicability proof.

`TestClassicForkSyncLiteralCodec` fails on the original unsupported YAML key.
The `classic_fork_sync_test.go` files in settings, github and app cover active and
inactive projections, both no-ops, exact payloads, target drift, omission, report
closure, prune/unarchive, recovery and independent export retention. CLI fork
lifecycle cases verify export/validate/plan/apply/repeat and final roundtrip.
`TestBuiltCompositeClassicForkSync` uses the supplied public Action for both values,
matching state, private plans, readback failure and unchanged checkout. Official
UI documentation supplies evidence; no authenticated fork form, fork creation,
upstream synchronization or live protection change was performed.
