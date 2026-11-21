---
type: Reference
title: "Autolinks: existing v0 contract"
description: "Autolinks: existing v0 contract declaration, ownership and API contracts."
---

# Autolink references

Autolinks are available in v0 when the repository meets GitHub’s plan and access requirements. [Support scope](scope.md#feature-adoption-scope)

`repository.settings.autolinks` manages the documented **Autolink references**
controls: Reference prefix, Target URL and reference identifier format.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    autolinks:
      - id: 123
        reference_prefix: TICKET-
        target_url: https://tickets.example.com/<num>?view=full
        reference_format: numeric
```

Every entry requires all three controls. `reference_format` is `numeric` or
`alphanumeric`; omission is invalid. Prefix and URL spelling are retained exactly.
Prefixes must be nonblank, contain no control characters and must not overlap
another prefix. No case folding is inferred. URLs must be absolute HTTP(S), have
no userinfo and contain literal `<num>`. These are product input constraints;
they do not describe every URL GitHub might accept. The placeholder is substituted
only for local URL parsing; gh-settings never fetches the destination. Use
non-secret URLs. Nulls, aliases for field names, unknown fields, `delete: true`,
nonpositive/duplicate IDs and missing controls are rejected before API work.

## Binding and replacement

An optional positive `id` binds an existing immutable reference. A matching bound
entry is unchanged and sends no mutation. A missing ID or differing configuration
fails; the API has no documented in-place update. An IDless entry is create-only
and cannot adopt a live overlapping prefix, including an object selected for prune.
Replace a reference explicitly: remove its declaration and prune it, then declare
the replacement without an ID in a later apply. This leaves a temporary
configuration gap; there is no automatic replacement, retry or rollback.

## Pruning

Omission is unmanaged. `autolinks: []` preserves existing references without prune.
For an already archived repository declared with `archived: true`, non-prune
apply can succeed without writes when every selected positive ID and all three
controls match complete observations. It checks the personal-public repository
and owner identity and selected state twice, and verifies repository identity and
archive state around the checks. Missing IDs, creation requests, differences,
unavailable or ambiguous observations, identity changes and prune are rejected.
Unselected references remain unmanaged. These independent reads are not an atomic
snapshot and cannot prevent later races. Archived GitHub readability has not been
verified live; local fixtures exercise this behavior.
`--prune autolinks`, its comma-separated combinations, and standalone `--prune all`
make a present list authoritative for retention. All deletion candidates and the
complete private inventory are frozen before any remote write, including unrelated
settings and unarchive. Candidates never expand. Before each deletion, complete
collection and individual-object reads must match the expected identity,
configuration and optional timestamp presence. Expected state changes only for
this invocation's verified creations/deletions and its verified unarchive.
Unknown additions or changed, missing or replaced retained/candidate objects stop
the operation. Timestamp equality alone is never identity evidence.

## Creation and recovery

Creation sends all three native fields and requires exact HTTP 201, a positive
previously unseen identity, a complete matching acknowledgment, independent GET by
ID and a complete inventory check. Only verified creations receive guarded local
ID recording. All declared creations and required recording finish before deleting
the fixed candidates. Deletion requires an empty HTTP 204 and a successful complete
inventory proving absence and retained-state preservation. A failed/denied GET or
isolated 404 does not prove absence. Races after these checks remain possible.

Only a positive unseen ID from the intended successful creation response is kept
as recovery information; it is not proof of a completed creation. Readback or
recording failure stops later work. Earlier verified changes remain applied and
uncertain effects remain uncertain. CLI `--no-writeback` and GitHub Actions leave
the file unchanged. Record confirmed IDs through the existing
[creation-ID procedure](../../development/repository-automation.md#repository-settings-changes)
before repeating an IDless declaration; repeated IDless creation will collide.

## Export and permissions

Automatic export independently reads the complete collection and exact
personal-public repository/owner identity. Confirmed empty state exports `[]`.
Unavailable, partial, malformed or unsupported state is omitted initially or
retains valid previous intent with bounded information. No required acquired
value is defaulted. Optional `updated_at` is private evidence, not YAML or report
configuration. Detailed CLI plans and Action Summary can contain prefix, URL and
format; status-only plans hide them. HTTP debug payloads and upstream error bodies
are suppressed. A mixed create/delete plan counts this family once.

The [UI guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/configuring-autolinks-to-reference-external-resources)
documents the controls and paid-plan prerequisites. Free is not listed, so
Autolinks are outside supported v1 scope. The current repository's missing sidebar
entry is bounded UI evidence, not proof of global unavailability or API absence. The
[Autolinks REST API](https://docs.github.com/en/rest/repos/autolinks)
supports App installation tokens with Administration read for acquisition and
write for creation/deletion. This implementation uses list-all, individual GET,
POST and DELETE; it grants no permission and performs no external-service work.
No fresh authenticated form, live write or rendered-link behavior was verified.
See the [behavior specification](../../development/features/autolinks.feature.md).

[Settings index](README.md) · [GitHub App permissions](permissions.md)
