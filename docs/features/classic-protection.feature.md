# Classic branch-protection patterns and independent controls

The [configuration contract](../settings.md#classic-branch-protection) owns public
fields, pattern semantics, identity, preservation, creation defaults and recovery.
Ordinary reconciliation manages conversation resolution, signed commits, linear
history, branch locking, administrator enforcement, required deployment environments, force-push permissions and the Allow deletions
checkbox, separately from Rulesets. Explicit
[classic prune](prune.feature.md#classic-branch-protection-deletion) additionally
removes whole unlisted rules. PR/check editing,
fork-sync and other actor controls remain separate work.

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

## Verification mapping

- Literal YAML booleans, omission, null/type/alias rejection and exact patterns:
  `internal/workspace/ui_classic_protection_test.go`, `ui_classic_lock_test.go`, `ui_classic_admin_test.go` and
  `ui_classic_deployments_test.go`.
- Identity/collision selection, creation order, partial equality, nullable raw
  companions and supported export: `internal/settings/classic_protection_test.go`
  and `classic_deployments_test.go`.
- Literal GraphQL inputs, complete 101-rule/101-actor pagination, unowned companion
  drift, fresh no-ops, create recovery and failure boundaries:
  `internal/github/classic_protection_test.go`, `classic_lock_test.go`, `classic_admin_test.go` and
  `classic_deployments_test.go`.
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

The required-deployment literal regression fails on unchanged source because
its group key is unknown. Automated verification uses local fixtures
and dummy credentials. Official schema and read-only empty personal-public
inventories establish the endpoint, not populated storage behavior, mutation
permission or effective enforcement. No live branch protection was changed.

## Force-push actor ownership

`allow_force_pushes` distinguishes disabled, Everyone and a nonempty specified
User/App set. Native false with nonempty allowances is specified, not disabled.
Everyone preserves hidden actor identities; explicit disabled clears both native
controls. Unmanaged groups preserve all known state, including Team. Acquired
Before accepts known Team while desired After/YAML rejects it. Active Team sets
make automatic export unavailable rather than silently losing members.

Every needed rule mutation with selected specified actors requires exact node-ID
and type resolution, including unrelated-flag changes with matching actors. Only
whole-rule no-op skips these lookups. Scalar ACK does not fabricate an actor
projection; independent complete paginated readback verifies requested membership
and raw preservation. Creation errors keep recovery separate from confirmed IDs.

The literal baseline is `TestClassicForcePushLiteralCodec`. Model cases in
`internal/settings/classic_force_push_test.go` cover projection, preservation and
closed Before/After contracts. `internal/github/classic_force_push_test.go` covers
all transitions, paginated actors, bounded identity queries, no-op, partial ACK and
readback failures. App tests retain prior export on active Team and preserve it
through other updates. CLI lifecycle force cases and the actual supplied
`TestBuiltCompositeClassicForcePush` exercise User/App selection, Everyone,
disabling, repeat no-op, private plans, lookup failures, uncertain readback and
unchanged source. No live push, setting mutation or App installation is claimed.
