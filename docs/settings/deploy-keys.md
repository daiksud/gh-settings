---
type: Reference
title: "Repository deploy keys"
description: "Repository deploy keys declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Deploy keys

Current source manages the **Title**, public **Key** and **Allow write access**
controls for a personally owned public repository. [GitHub documents these UI
controls](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/managing-deploy-keys)
and [list, get, create and delete REST operations](https://docs.github.com/en/rest/deploy-keys/deploy-keys).
Administration read is needed for observation and write for creation/deletion.
The client never generates keys, reads private keys, uses an SSH agent or tests SSH
access. API acceptance and a stored configuration do not prove actual key usability.

```yaml
repository:
  settings:
    deploy_keys:
      - id: 456
        title: build-server
        key: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB
        allow_write_access: false
```

The example public material is a fixture, not a usable deployment credential.
Every entry requires native string `title` and `key` and boolean
`allow_write_access`. Empty titles are accepted; titles may repeat. Reject title
control characters, nulls, duplicate positive IDs and duplicate parsed key material.
`key` must contain one valid authorized-public-key line, optionally ending in one
LF/CRLF and a trailing comment. Options, private formats, internal newlines,
multiple keys and mismatched algorithm tokens fail locally without echoing input.
A maintained SSH parser determines syntax; it does not promise GitHub algorithm
support. Parsing is bounded to 64 KiB. Export and creation use canonical public
material without comments, and `allow_write_access` is the inverse of REST
`read_only`, including explicit false.

Omission is unmanaged. `deploy_keys: []` preserves existing keys during ordinary
apply. Export requires complete target identity and paginated, typed inventory;
unavailable reads emit bounded info and retain valid prior declarations, or omit
this family on first export. Missing fields never become empty inventories or
false permissions. Export includes IDs. The optional server `enabled` field is
private evidence: missing means unknown, null is invalid, and neither state is
converted into declared access.

An ID-bound exact match is a no-op. A missing ID or changed title, material or
access flag fails: there is no update endpoint, implicit replacement or fallback
creation. IDless entries create only when their canonical key does not exist;
existing material requires explicit ID binding. Titles and comments cannot cause
adoption. Before creating, apply checks the complete target/inventory again. A
201 acknowledgment, independent GET by the new ID and complete inventory readback
must confirm the configuration and preservation of previous keys. Only then may
local CLI writeback add the ID to the unique canonical-key entry, preserving its
original key scalar/comment. Existing IDs, ambiguous/aliased entries and concurrent
file changes block writeback. Action and `--no-writeback` leave YAML unchanged;
record the reported fingerprint and ID against the matching key before repeating.

A positive new ID from a successful creation response may be reported separately
for recovery when confirmation fails. It is not written into YAML, does not prove
successful creation verification and never comes from an error response or reused
ID. Later work stops; earlier verified writes remain. Reports show title,
SHA256 fingerprint and grant, with numeric ID when selected; raw key material and
private snapshots stay out of plan reports. Status-only also hides this identity.
Raw transport logging is suppressed even with `GH_DEBUG=api`.

### Deploy key pruning

`--prune deploy-keys` and standalone `all` add deletion only for a present list.
Selected IDs are retained. Complete target and all key snapshots are frozen before
any write, including unrelated changes and unarchive. Before each DELETE, complete
inventory and candidate GET must still match. Expected state includes only this
run's verified creations/deletions and known own-unarchive transition; new external
keys never become inferred candidates. ID, parsed material, title, grant, creation
time, canonical API URL and exact unknown/false/true `enabled` state are guarded.
Dynamic `verified`, usage time and creator attribution are typed but do not trigger
identity drift. The server URL is validated against the fixed GitHub.com target
route and is never followed. Unknown fields or incomplete pages block acquisition.

Deletion requires a 204 empty acknowledgment and fresh complete inventory proving
absence and preserved remaining keys. GET404 alone proves nothing. A failed
readback can leave a deleted key and uncertain remote effects; no rollback or retry
is attempted. Deletion revokes that repository credential. Replace immutable
configuration by explicitly pruning the old ID, then creating the replacement in
a later apply. A distinct new key can be created before pruning an old key, with
temporary overlapping access. This is not automatic rotation. Removing a creator
or uninstalling an App is not deploy-key cleanup.

See the [behavior and verification specification](../features/deploy-keys.feature.md).
