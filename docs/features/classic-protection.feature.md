# Classic branch-protection patterns and independent controls

The [configuration contract](../settings.md#classic-branch-protection) owns public
fields, pattern semantics, identity, preservation, creation defaults and recovery.
Current source manages conversation resolution, signed commits, linear history,
branch locking, administrator enforcement and deletions, separately from Rulesets.
It adds no classic prune,
PR/check/deployment editing, fork-sync or force-push actor editing.

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
  Omitted lock state and the unowned fork-sync flag are independently preserved.
  Unexpected fork-sync changes fail after the write with uncertain effects.
  Stored lock and deletion flags do not imply deletion is permitted while locked.

- Administrator enforcement maps true/false directly to `isAdminEnforced`, with
  omission preserving the acquired value. Both creation and update verify the
  requested value independently; incomplete acknowledgment or collateral changes
  remain failures. No bypass actors or unrelated protections are implicitly edited.

## Verification mapping

- Literal YAML booleans, omission, null/type/alias rejection and exact patterns:
  `internal/workspace/ui_classic_protection_test.go` and `ui_classic_lock_test.go` and `ui_classic_admin_test.go`.
- Identity/collision selection, creation order, partial equality, nullable raw
  companions and supported export: `internal/settings/classic_protection_test.go`.
- Literal GraphQL inputs, complete 101-rule/101-actor pagination, unowned companion
  drift, fresh no-ops, create recovery and failure boundaries:
  `internal/github/classic_protection_test.go` and `classic_lock_test.go` and `classic_admin_test.go`.
- Family preflight, declaration order, unarchive, actual progress and export
  retention: `internal/app/classic_protection_test.go`.
- Closed report, escaped Summary and status-only privacy:
  `internal/actions/classic_protection_test.go`.
- Supplied executable export/plan/apply/repeat, guarded numeric-ID recording,
  wildcard rules, private Action output and unavailable export:
  `tests/acceptance/classic_protection_test.go`.
- Actual supplied Action lock/admin enable/disable/no-op, private plan and companion drift:
  `tests/github-actions/classic_lock_acceptance_test.go`.

The literal YAML acceptance case fails against the pre-change executable because
`branch_protection_rules` is unknown. Automated verification uses local fixtures
and dummy credentials. Official schema and read-only empty personal-public
inventories establish the endpoint, not populated storage behavior, mutation
permission or effective enforcement. No live branch protection was changed.
