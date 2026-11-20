---
type: Reference
title: "Repository webhooks"
description: "Repository webhooks declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Webhook configuration and creation

Use `repository.settings.webhooks` for repository webhooks in a personally owned
public repository. An existing positive `id` selects optional **Payload URL**,
**Content type** and **SSL verification** updates. Its optional **Active** and **events** fields are
**equality guards**, not parent-update requests:

```yaml
webhooks:
  - id: 12345678
    payload_url: https://example.com/github
    content_type: application/json
    ssl_verification: true
    active: true
    events: [pull_request, push]
  - id: 87654321
    content_type: application/x-www-form-urlencoded
```

Every bound entry needs a unique positive integer ID and at least one selected
control. Omitted controls and unlisted hooks remain unmanaged. Declaring only
Active/events is a useful guard: matching values make no change; a mismatch fails
before this family's first write. A missing selected ID fails, without recreation.
Existing **Active/events updates are outside v1** under the current design,
which does not own Secret values. The [parent PATCH endpoint][webhook-parent-update]
exists and accepts Active/events, but warns that an existing Secret must be
supplied again or replaced to avoid removing it. A general safe preservation
contract without Secret ownership is unconfirmed. This is a technical-contract
limitation, not a claim that the API is absent or that creation is unsupported.
The dedicated config PATCH remains supported for URL, Content type and SSL
verification; it cannot update Active/events.

To change Active/events on an existing hook:

1. In the target repository's GitHub **Settings → Webhooks**, edit the existing
   hook's Active/event selection and save it through GitHub's UI.
2. From that repository's checkout, run `gh settings export`, then inspect the
   `.github/settings.yml` diff. Export reads GitHub settings and changes the local
   file; it does not update the hook. It also refreshes other supported readable
   settings, so review the whole diff before retaining it.
3. Confirm the hook's existing ID and newly observed Active/events were exported,
   then run `gh settings plan`. These fields now guard the observed values;
   editing them in YAML cannot request an update.

If webhook acquisition is unavailable, re-export retains valid prior declarations
as unverified intent. Retained old values do not confirm the UI change; resolve
acquisition and inspect a successful export before relying on the guards. This
workflow does not provision, export or verify Secrets, authenticate a receiver or
verify delivery. The bounded observations in [#776][webhook-observations] do not
establish general Secret preservation or supersede the official warning.

Omit `id` to create a hook, and explicitly supply URL, Content type, Active and
events. SSL verification is optional; false and an empty event set are meaningful values:

```yaml
webhooks:
  - payload_url: https://example.com/new-hook
    content_type: application/json
    ssl_verification: true
    active: false
    events: []
```

Creation sends `name: web`, the exact URL, native Content type, Active and events,
with the selected SSL verification and **no Secret**. Omitted `ssl_verification`
keeps verification enabled (`insecure_ssl: "0"`); `true` enables verification
and `false` disables it (`insecure_ssl: "1"`). Disabling certificate verification
can allow man-in-the-middle attacks. Detailed plans identify that consequence. GitHub
[documents an automatic ping when a webhook is created][webhook-ui]; the CLI does
not promise that inactive creation suppresses it. Detailed plans show these
consequences. Receiver authentication, secret provisioning and delivery success
are not established by successful creation. No additional ping, test, redelivery
or destination request is made.

`ssl_verification` accepts only a non-null boolean. For existing hooks omission
preserves the fresh observed value. Content type accepts exactly the two MIME strings above, mapped to GitHub's
`json` and `form`. Payload URL must be absolute HTTP(S) with a host and without
userinfo, fragments, whitespace, control characters or malformed escapes. These
are product guards, not extra GitHub schema claims. Valid ports, paths and queries
are accepted byte-for-byte, without trimming, normalization or DNS lookup.
Events are an exact, order-insensitive set of nonempty strings without whitespace
or control characters; duplicates fail. `[]` is not omission, and no default
`push`, case folding, fixed event allowlist or `*` expansion is inferred. An
acquired `*` remains literal. GitHub may still reject a declared event combination.
Nulls, wrong types, duplicate members/IDs, explicit `id: 0`, incomplete creation,
and unsupported keys such as `url`, `secret`, `config` or `insecure_ssl` fail locally.

URLs are nonsecret configuration visible in YAML and detailed plans. GitHub
[advises against embedding credentials in webhook URLs][webhook-practices].
Path/query tokens cannot be reliably detected or redacted while preserving an
exact, reviewable destination. Status-only plans hide IDs, URLs, SSL verification and events; they
are not credential storage. Recovery diagnostics identify the declared URL and ID.

Omitting the family or using `webhooks: []` without pruning makes no family plan/apply requests. Explicit [webhook pruning](#explicit-webhook-pruning) owns whole-object deletion for a present list; `--prune all` includes this scope and may remove unlisted hooks and their opaque Secrets.
Bound-only plan/apply reads selected IDs. Any IDless entry requires a complete
paginated inventory, including unlisted and inactive hooks, before any family
write. Its exact URL must be absent from that original inventory and must not
collide with another IDless or declared bound URL. A hook moving away from that
URL does not make it eligible in the same apply. Existing bound hooks may share
URLs. URL is a collision guard and local writeback selector, never remote identity
or permission to adopt an existing hook. Recover and declare an existing ID instead.

Export completes pagination and obtains every parent/config record, then writes
positive IDs and all five public fields, including observed SSL verification, with sorted event sets. Complete empty
inventory exports `webhooks: []`. Inaccessible, incomplete, conflicting or
unsupported evidence omits this entire family on first export or retains its valid
prior declaration, including an IDless creation request, on re-export. Independent
families continue; no subset replaces the collection.

Apply prepares every selected hook and parent guard before the family's first
write. Fresh matching configuration skips PATCH, including on an archived target.
Needed writes require a freshly unarchived repository; explicit unarchive precedes
webhooks, and archive remains last. Mixed creation/configuration work rechecks the
complete inventory before each mutation and advances its expected state only for
its own independently verified effects. Outside additions, removals, replacement
or nonsecret state changes stop later work rather than being adopted as a new base.

Reads require complete personal-public repository/owner identity, archive state,
repository hook identity and creation timestamp. Parent and dedicated config
records must agree on URL, Content type and SSL state. Active and all events must
be known. Unknown config members, missing/null controls, malformed values and
identity drift fail rather than imply defaults. SSL accepts native string/numeric
zero and one and maps them to the positive `ssl_verification` boolean. Future valid nonempty event names remain an exact set.

A needed config PATCH sends only changed selected URL/Content type/SSL members to
`/repos/{owner}/{repo}/hooks/{id}/config`, without Secret, parent controls,
omitted members or an empty payload. Complete HTTP 200 acknowledgment is followed
by independent parent/config/target GETs verifying selected values, omitted
nonsecret configuration, creation identity, Active and events. A value already
matching before PATCH must still match afterward.

Creation uses POST `/repos/{owner}/{repo}/hooks`. HTTP 201 must contain a new
positive ID and complete matching nonsecret state. Independent fixed-ID parent and
config GETs and a complete inventory read must confirm that new hook exactly once,
its identity and SSL verification, retained hooks and the bound target. A valid
unambiguous new ID in a successful 201 body remains recovery evidence if the rest
of acknowledgment or readback fails. Error bodies, Location headers, malformed or
repeated identity members and already-known IDs never establish a new identity.
Acknowledgment alone does not authorize local ID recording.

After verification, ordinary CLI apply adds only the new ID to the matching
IDless URL entry using guarded source-file checks, preserving controls/comments.
Concurrent edits, aliases, merge keys, symlink or identity conflicts block recording
and report the verified remote progress for manual recovery. `--no-writeback` and
GitHub Actions leave YAML unchanged and report the ID for a reviewed follow-up.
Record and verify that ID before retrying: repeating an IDless request after
creation fails the URL collision guard, without another POST. A bound matching
repeat performs no mutation. A failed readback never writes an ID to YAML.

Secret strings are discarded during decoding; absent, empty or masked strings
cannot prove secret presence or equality. Secret is neither compared nor replayed.
Existing updates rely on the dedicated endpoint's documented partial-update
contract. Raw response/error/debug bodies, unknown member names and private
snapshots do not enter logs or reports. Closed reports contain selected public
controls only. Prewrite failure differs from unconfirmed effects after a mutation
attempt; earlier verified writes remain applied. There is no retry, rollback or
success output after failure, and each hook completes before the next.

The [creation/editing guides][webhook-ui], [repository webhook REST][webhook-rest]
and [partial configuration announcement][webhook-partial] establish these paths.
Reads require Webhooks read; creation/updates require Webhooks write. Local fixtures
do not prove live delivery, authentication, reachability, cryptographic secret
equality or absence of races between requests. Existing parent Active/events mutation is outside v1; receiver configuration and Secret values are outside policy.

## Explicit webhook pruning

`--prune webhooks` (including comma selection or standalone `all`) adds deletion
for a present `repository.settings.webhooks` list. Omission stays unmanaged.
Ordinary `webhooks: []` retains every hook; the same empty list with explicit
prune selects every acquired hook. Bound IDs retain those objects. Verified new
IDs join retention and never become deletion candidates. There are no tombstones
or automatic adoption by destination URL.

Explicit pruning owns the **whole unlisted webhook**, including its associated
opaque Secret. Deletion is irreversible: it removes that hook and its Secret,
but does not revoke credentials embedded in a destination URL, undo prior
requests or prove cancellation of an already dispatched delivery. Detailed plans
show the candidate ID, Payload URL, Content type, SSL verification, Active and exact event set and
identify this whole-object effect. Status-only hides identities and values.

Before any remote write, including unarchive or unrelated settings, apply obtains
a complete paginated inventory and individual parent/config evidence for every
hook. It binds the repository and owner IDs, public/User applicability and
archive state, hook IDs, fixed API routes, creation identity, nonsecret config,
Active/events and validated exact `updated_at` values. Missing, malformed or
unknown configuration and unrecognized parent members stop all writes. Known
transport/delivery metadata is ignored as configuration; it is never followed,
retained in the snapshot or displayed. Secret strings are discarded rather than
compared, retained or replayed.

The candidate IDs are frozen by that acquisition. Before each DELETE, complete
inventory and candidate/target reads must match the expected state. Added or
removed objects, changed identity, nonsecret state or timestamps stop deletion.
Only independently verified changes made by this apply update expected state:
config writes refresh the written object's evidence, creations join retention,
confirmed deletions remove their IDs, and confirmed unarchive changes archive
state. Refreshing an inventory never permits unrelated drift. Selected updates,
creations and required local ID recording complete before deletion; pending-delete
hooks still count for creation collision checks. A capacity failure does not
cause delete-first recovery.

DELETE uses the fixed repository/ID route and requires an empty HTTP 204. A fresh
complete inventory must then prove absence and unchanged retained evidence.
An isolated 404, denied read or acknowledgment alone is not success. Prewrite
failures remain distinct from unconfirmed effects after an attempt; earlier
verified updates, creations or deletions remain applied. There is no automatic
retry or rollback. CLI and the public Action use these same checks, and Action
files remain unchanged.

`updated_at` is only an additional comparison token. Any change blocks
conservatively; there is no exception for presumed ping/delivery activity.
GitHub documents no version precondition on DELETE and no guarantee that every
Secret rotation changes this timestamp. Equal timestamps or missing/masked Secret
representations do not prove Secret equality, absence or protection from a
same-timestamp change. There is no atomic lock between the last read and DELETE.
This contract deliberately permits whole-object deletion with those limits; it
is not secret-value management or secret-aware concurrency control.

The [repository deletion guide](https://docs.github.com/en/webhooks/using-webhooks/disabling-webhooks)
and [repository webhook DELETE](https://docs.github.com/en/rest/repos/webhooks#delete-a-repository-webhook)
establish the UI/API path. Deterministic tests prove local comparison, ordering,
privacy and failure boundaries, not a server guarantee about Secret changes or
live delivery cancellation. See the [prune behavior specification](../features/webhook-prune.feature.md).

[webhook-practices]: https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks

[webhook-ui]: https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks

[webhook-parent-update]: https://docs.github.com/en/rest/repos/webhooks#update-a-repository-webhook

[webhook-observations]: https://github.com/daiksud/gh-settings/issues/776

[webhook-rest]: https://docs.github.com/en/rest/repos/webhooks

[webhook-partial]: https://github.blog/changelog/2020-10-26-webhooks-configuration-enhancements/
