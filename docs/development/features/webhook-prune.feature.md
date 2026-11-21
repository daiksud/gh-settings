# Feature: Remove unlisted webhooks

The [public contract](../../references/settings/webhooks.md#explicit-webhook-pruning) owns scope,
irreversible effects and concurrency limits. These scenarios cover explicit deletion of complete hook objects.

## Rule: Prune only a present collection after complete preflight

### Scenario: An empty list opts in to removing every hook

- Given the repository contains webhook 17 and the declaration contains `webhooks: []`
- When the owner runs plan without prune
- Then webhook 17 is retained
- When the owner runs plan with `--prune webhooks`
- Then the deletion of webhook 17 and the loss of its opaque Secret are shown
- And GitHub still contains the webhook

### Scenario: A candidate changes before deletion

- Given the initial complete inventory selected webhook 17 for deletion
- When apply sees a changed candidate timestamp before sending DELETE
- Then it stops without deleting webhook 17
- And earlier confirmed writes remain in the reported progress

## Ownership and verification

- Named/comma `webhooks` and standalone dynamic `all` require a present list.
  Omission stays unmanaged; ordinary empty lists retain all hooks. Explicit empty
  retention selects all acquired IDs, with no candidate-set growth or tombstones.
- Global complete acquisition happens before every unrelated or unarchive write.
  Every retained and candidate hook needs coherent complete identity/nonsecret
  state and valid timestamps. Unknown parent/configuration blocks destruction.
- Known delivery/transport fields are not configuration; Secret strings are
  discarded. Missing/masked Secret never proves absence or equality. A changed
  timestamp blocks; an equal timestamp is not proof against a Secret-only change.
- Config changes, creations and required ID recording finish before deletion.
  Expected state overlays only each verified own effect, never an unrelated
  refreshed inventory. New verified IDs remain retained; pending-delete objects
  remain visible to creation collision checks.
- Before each DELETE, fresh complete inventory and exact candidate/target reads
  must match the fixed expected state. External additions, removals, replacements,
  nonsecret drift or timestamp drift fail before attempting that delete.
- Exact empty 204 and independent complete absence/retained-state verification
  are both required. No isolated 404 success, response-URL following, retry or
  rollback. Partial prior progress and uncertain attempts remain distinguishable.
- Detailed deletion plans expose ID, URL, MIME, SSL verification, Active/events and the whole-object
  Secret consequence. Status-only hides them. Private snapshots, raw SSL evidence and timestamps,
  raw bodies and delivery evidence never enter reports or API debug logs.
- Deletion does not revoke external credentials or undo/cancel dispatched
  requests. There is no atomic read/delete lock. Fixtures establish the local
  boundary, not server timestamp semantics or live delivery cancellation.

`internal/settings/webhook_prune_test.go`, `internal/github/webhook_prune_test.go`
and `internal/app/webhook_prune_test.go` own selection/evidence, fixed REST
acquisition/deletion and global sequencing/progress respectively. Built-CLI
coverage is in `tests/acceptance/webhook_prune_test.go`; the supplied-product
Action scenario exercises the same closed public path and unchanged checkout.
