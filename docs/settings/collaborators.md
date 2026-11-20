---
type: Reference
title: "Personal repository collaborators"
description: "Personal repository collaborators declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Personal collaborators

`repository.settings.collaborators` manages people on Settings > Collaborators
for personally owned public repositories. Each entry has a required `username`
and optional positive immutable GitHub **user** `id`:

```yaml
repository:
  settings:
    collaborators:
      - username: existing-person
        id: 123456
      - username: new-person
```

An ID identifies the person, not an invitation, membership or proof of accepted
access. Export includes supported active people and pending invitees with their
user IDs, excluding the owner by owner ID. Personal collaborators have GitHub's
fixed write permission; no role or permission field is accepted. Email-only
invitations have an unconfirmed complete API roundtrip and remain unsupported.

Omission is unmanaged. Without prune, `[]` and unlisted people cause no access
mutation. Explicit [`--prune collaborators`](#collaborator-pruning), including
standalone `all`, adds bounded invitation cancellation and active-grant removal
for a present collection.
An ID-bound person must still be active or pending; disappearance fails without
reinviting. A username-only entry is initial-add intent. Existing active access
or a matching pending invitation makes no PUT; apply records the verified user ID
locally. Otherwise it sends one bodyless invitation PUT and independently reads
back the complete active and pending inventories. A 201 must match the exact
invitation or the same user's now-active access; a 204 must verify active access.
A successful invitation is not acceptance. CLI plan and detailed Action Summary
show pending acceptance even when no new invitation is planned.

A username-only binding is a local change, distinct from a remote notification.
After confirmation, normal CLI apply records the user ID using the source
snapshot and safe-file checks shared with created-resource IDs. It never records
an unverified ID. Earlier confirmed bindings remain if a later operation fails;
local binding alone is not reported as a completed remote write. `--no-writeback`
or `GITHUB_ACTIONS=true` suppresses file changes and reports the verified ID for a
reviewed declaration update. Persist that ID before relying on repeated scheduled
runs. No Git commit or push is made. Output or file-write failure can occur after
remote effects; diagnostics preserve verified user IDs and reported invitation IDs
for deliberate recovery, without claiming acceptance or inviting again.

Saved user IDs prevent ordinary reinvitation after expiry, decline, cancellation
or revocation. Deliberately removing an ID restores initial-add intent and can
notify that person again. There is no hidden journal: if an ambiguous request was
sent, no verified binding was saved and the invitation later disappears, another
username-only invocation cannot prove the earlier request happened. Resolve access
before retrying. No automatic retry, polling, acceptance or rollback occurs.
Cancellation and removal require explicit prune intent.

Both inventories must be fully paginated and typed. Missing/null invitees,
unsupported actors/roles, malformed state, explicit `expired: true`, duplicate
identities and active/pending overlap make the family unavailable. Missing expiry
metadata does not promise an expiry date. Export omits an initially unavailable
family or retains its complete valid prior declaration; it never combines fresh
active rows with retained pending rows. Declared plan/apply observations fail
strictly. Denial or 404 is not an empty access list.

Case-only usernames match; export uses acquired spelling. ID-bound entries resolve
the user by ID and reject genuine renames with the current username for deliberate
correction. Reused usernames never silently select a different ID. Apply resolves
all declared people before the first collaborator operation, then rechecks the
repository/owner IDs, personal-public status, complete inventories and current
username identity before each request. A prepared pending invitation that vanished
is not resent. Without prune, an explicit own unarchive occurs before these reads; archived
repositories cannot receive a new invitation. These reads are not atomic, and the
username-addressed PUT has no atomic expected-user-ID condition, leaving a narrow
concurrent rename/reassignment race. Later failures stop subsequent operations
without undoing earlier changes.

The [Collaborators REST API](https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2026-03-10),
[Invitations REST API](https://docs.github.com/en/rest/collaborators/invitations?apiVersion=2026-03-10)
and [user identity API](https://docs.github.com/en/rest/users/users?apiVersion=2026-03-10)
define these operations. Complete observation requires collaborator-list access
and Administration read for invitations; mutation requires Administration write.
Installation-token support is documented, but local fixtures do not establish a
real App's grants or a recipient's notification/acceptance. See the
[behavior specification](../features/repository-access.feature.md).

### Collaborator pruning

`plan --prune collaborators` and `apply --prune collaborators` use the existing
`repository.settings.collaborators` list. Omission stays unmanaged even under
`all`; `[]` selects every supported non-owner active grant and pending invitation.
A nonempty list retains the union of ID-bound people and resolved username-only
people, including initial-add intent. The owner is always protected. No deletion
marker, role expansion, email invitation, recipient acceptance or installation
removal is introduced.

Before **any** remote write, including unarchive and unrelated settings, apply
acquires complete supported active/pending inventories and freezes selected user
IDs, active removal candidates and invitation-ID cancellation candidates. Failed,
partial, duplicate, unsupported or ambiguous evidence blocks the whole preflight.
The same credential must positively identify the personal repository owner, or
prove installation target access through a complete authenticated
[installation repository inventory](https://docs.github.com/en/rest/apps/installations?apiVersion=2026-03-10#list-repositories-accessible-to-the-app-installation)
and complete admin-only invitation visibility. A failed `/user` alone is not
installation evidence. Ambiguous positive identities and a non-owner user fail.
No App/installation ID is invented, no second credential is requested, and token
prefixes or `GITHUB_ACTOR` do not determine the management path. Administration
write remains a mutation prerequisite; successful reads do not prove that grant.

Inviter identities are private dependency evidence, never YAML or report values.
An active removal is refused if a retained invitation has unknown inviter identity
or was sent by that candidate. Only initially selected invitations are cancelled;
a late invitation cannot be cancelled to unblock removal. Ordinary selected
reconciliation and verified local bindings run first, followed by fixed invitation
cancellations, then fixed active removals, before Environment work. A selected
pending person's acceptance may advance to active; a cancellation candidate's
acceptance, disappearance or replacement fails instead of becoming a grant removal.
Frozen selected IDs prevent later username reassignment from changing retention.

Every DELETE freshly checks the exact repository/owner, management path and complete
remaining inventories, accounting only for confirmed operations. Only a verified
own unarchive may change expected archive state. After cancellations, active
removal requires no remaining invitation dependency and a last immutable-ID and
username-to-ID check. A bodyless repository invitation-ID or collaborator-username
DELETE must return empty 204, followed by independent complete readback. It must
confirm absence and unchanged remaining access; 204 alone is not verified success.
CLI/detailed Summary distinguish cancellation from removal. Apply diagnostics
separate attempted, acknowledged and verified effects, retain earlier bindings and
verified deletions, and stop later work without retry, rollback or failed-readback
adoption. Local binding alone is not a remote partial write.

GitHub documents additional collaborator-removal effects on issue assignments,
stars, package access, invitations and forks. These may settle asynchronously;
verified list absence does not certify their completion. Removing a grant does
not make public content inaccessible. Inventories are not atomic, the DELETE is
username-addressed without an expected-ID condition, and invitation creation can
race the final dependency check. Removing and re-adding an identical active grant
between reads is indistinguishable because there is no membership incarnation ID.
Review these effects before apply; there is no transaction or hidden journal.
