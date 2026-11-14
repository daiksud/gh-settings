# Existing webhook configuration

The [configuration contract](../settings.md#existing-webhook-configuration) owns
public declaration and evidence limits.

- Existing positive IDs select optional Payload URL/Content type; omission and an
  empty family make no plan/apply requests. There is no lifecycle/prune ownership.
- Strict literal shape and URL validation precede remote work. Public URLs retain
  exact bytes and never trigger destination requests.
- Export requires the entire paginated inventory and coherent complete parent and
  config evidence. Unknown/inaccessible state omits or retains the whole family;
  complete empty inventory exports an explicit empty list.
- Plan is read-only. Apply prepares all selected IDs before this family's first
  write, then acquires fresh companions per hook. A match skips PATCH. A needed
  write requires a bound unarchived personal-public repository and existing hook.
- Dedicated config PATCH sends only changed selected members. Secret is discarded
  from observations and never replayed or compared. Parent Active/events and SSL
  remain private and are verified unchanged against the fresh snapshot.
- Complete HTTP 200 ACK is followed by independent parent/config/target GETs.
  A mismatching selected-already-matching value, omitted control, event set,
  creation identity or target fails confirmation. Each hook completes before the
  next. Prewrite failure and uncertain effects remain distinct; no rollback occurs.
- Detailed plans show selected destinations for review. Status-only hides IDs and
  values. Closed reports carry no private companions or raw API diagnostics.
- CLI and actual supplied-product Action fixtures prove local request sequences,
  omission, partial progress, debug privacy, no-op repeats and unchanged YAML.
  They make no live delivery, receiver authentication or secret-equality claim.

## Verification mapping

`internal/workspace/ui_webhooks_test.go` owns literal codec evidence;
`internal/settings/webhooks_test.go` owns declaration/preservation/report values;
`internal/github/webhooks_test.go` owns API and fresh reconciliation;
`internal/app/webhooks_test.go` owns sequencing/retention/progress;
`internal/actions/webhooks_test.go` owns report/Summary boundaries.
Built-process cases are in `tests/acceptance/webhooks_test.go`; the actual public
Action is exercised by `tests/github-actions/composite_webhooks_acceptance_test.go`.
