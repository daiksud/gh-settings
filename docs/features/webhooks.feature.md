# Webhook configuration and creation

The [configuration contract](../settings.md#webhook-configuration-and-creation)
owns declarations and evidence limits. Explicit [whole-object pruning](webhook-prune.feature.md) is a separate ownership selection.

- Bound positive IDs select optional URL/Content type and optional Active/events
  matching guards. All guards pass before this family's first write. A mismatch
  or missing selected ID fails without parent PATCH or recreation.
- IDless creation requires all four explicit public controls. Nulls, invalid URL
  bytes, duplicate/nonstring/whitespace event members and guessed defaults are
  rejected locally. Event sets include explicit empty and opaque `*` values.
- Omission and an empty family make no plan/apply requests. Bound-only work reads
  selected IDs. Any creation requires complete original inventory and exact URL
  collision checks, including inactive/unlisted hooks and hooks moving away.
  URLs never adopt identity. No deletion/prune ownership is implied.
- Export acquires complete coherent parent/config evidence for all hooks and writes
  IDs plus four public controls. Unknown/inaccessible state omits or retains the
  whole family, including valid prior IDless intent. Empty inventory exports `[]`.
- Apply prepares all selections before writes. Fresh matching config skips PATCH.
  Needed writes require the bound unarchived personal-public target; own unarchive
  precedes configuration and archive remains last. Mixed writes guard complete
  inventory and advance expected state only for their own verified effects.
- Dedicated config PATCH sends changed selected members only. Secret is discarded
  and never replayed/compared. Omitted nonsecret controls, SSL and parent state
  remain preserved. Independent parent/config/target GETs follow complete HTTP 200.
- Creation POST sends explicit URL, native Content type, Active/events, `name: web`
  and enabled SSL verification, without Secret. The detailed plan states GitHub's
  automatic creation ping and no supplied Secret; no extra lifecycle or destination
  request occurs. HTTP 201 must identify an unseen positive ID and complete state.
- Independent fixed-ID parent/config and full-inventory GETs verify new state,
  retained hooks and target identity before completion. A valid unambiguous 201 ID
  with bad remaining acknowledgment/readback is recovery-only. Error bodies,
  Location headers and ambiguous/old IDs cannot supply creation identity.
- Verified creation permits guarded ID-only source recording by exact URL. Failed
  confirmation leaves YAML unchanged. Concurrent edits/aliases/merge keys or
  symlink conflicts stop local recording while preserving verified remote progress.
  Actions and `--no-writeback` preserve files; manual binding precedes repeat apply.
  An IDless repeat colliding with a live URL stops without another POST.
- Prewrite failure and uncertain effects remain distinct. Earlier verified writes
  remain applied after later failure; no retry, rollback or false success occurs.
- Closed create/update reports contain only selected public projections. Status-only
  hides IDs/values; API/debug/error bodies and private companions never appear.
  Local fixtures make no live delivery, receiver authentication or secret-equality
  claim and do not establish atomicity between requests.

## Verification mapping

Literal declaration and ID-recording tests live in `internal/workspace`;
`internal/settings/webhook_creation_test.go` covers identity, guards, collisions and
inventory advancement. `internal/github/webhook_creation_test.go` and
`webhook_mixed_test.go` cover transport, recovery and independent verification.
`internal/app/webhook_creation_test.go` covers sequencing and actual progress;
`internal/actions/webhooks_test.go` covers closed report/Summary boundaries.
`tests/acceptance/webhook_creation_test.go` runs the built CLI;
`tests/github-actions/composite_webhook_creation_acceptance_test.go` runs the actual
public Action using the supplied product. Existing webhook tests retain selected
config behavior and complete export/retention coverage.
