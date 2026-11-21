---
type: Reference
title: "Self-hosted runner labels and removal"
---

# Existing runner custom labels

Use `repository.settings.actions.runners` to manage custom labels on existing
repository self-hosted runners in a personally owned public repository:

```yaml
runners:
  - id: 23
    custom_labels: [gpu, accelerated]
  - id: 24
    custom_labels: []
```

Each entry requires a unique positive numeric runner `id` and an explicit
`custom_labels` list of at most 100 strings. The ID selects an existing runner;
there is no name-based adoption, creation, rename or ID writeback. Omitting the
family leaves it unmanaged. Without prune, `runners: []` selects no runners and
makes no family requests during plan/apply. An entry with `custom_labels: []`
clears that runner's custom labels. Unlisted runners remain untouched unless
`--prune runners` or standalone `--prune all` is explicitly selected. With either
selector, a present list retains the listed IDs and removes unlisted remote
registrations; an explicit empty list selects all acquired registrations. This
changes the effect of `all` for existing present runner declarations. Review the
plan and your list before applying it.

Labels form a case-insensitive set. The client uses Unicode simple folding,
without trimming or Unicode normalization; order and case-only changes are no-ops.
Submitted spelling remains in YAML and the request. Duplicate case identities,
blank or control-containing names, nulls, unknown YAML fields and malformed IDs
fail locally. The nonblank/control rule is a product guard, not an additional
GitHub schema claim. A desired custom label that collides with an acquired
read-only label is rejected before this family's first write.

Export acquires the complete paginated inventory plus each runner's detail and
labels before replacing this family. Known empty inventory exports `runners: []`.
Any incomplete, denied or inconsistent runner/label observation omits the family
on first export or retains its valid prior declaration with bounded info; other
families continue. Missing/null/unknown label type never implies custom or
read-only, and default labels are never inferred from OS. Label IDs are optional
observation evidence, not YAML bindings. Present IDs must be positive; conflicting
known IDs fail. Both reads contribute the strongest available ID evidence.

Without runner pruning, plan reads only selected runner IDs and compares their complete custom sets.
Apply prepares every selected runner before the first label update, then freshly
reads each runner and repository before writing. A match makes no PUT. For an already archived target declared `archived: true`,
non-prune matching requires the independent identity-bound checks described in
the [archive lifecycle](../general.md#repository-archive-lifecycle). A needed write on an archived repository fails; an explicitly requested unarchive still
precedes configuration. Replacement sends only the complete declared custom list.
HTTP 200 must contain a complete matching label acknowledgment, followed by
independent repository, same-ID runner and label GETs. Verification preserves
fresh runner name/OS and exact read-only label names/types, including every
previously known read-only ID. Custom-label IDs are not owned. Missing prior IDs
remain unknown, and later missing known read-only IDs fail verification.

For ordinary label updates, operational status, busy, version and other liveness metadata are not drift
conditions. Failed pre-write reads are distinguished from uncertain effects after
a PUT attempt. Earlier verified writes remain applied; later work stops, without
retry or rollback. Detailed plans and Action Summaries show selected IDs and
custom labels; status-only hides them. For ordinary label updates, name/OS/read-only
evidence and raw API errors do not enter reports. Input files are not modified.

## Runner registration pruning

Explicit pruning owns only remote deregistration, corresponding to GitHub's
[Force remove control](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/remove-runners).
It does not stop services, remove host software or credentials, cancel jobs, or
restore a registration. Removing a runner can affect routing and queued or running
work. GitHub provides no atomic busy or version condition for DELETE: a job may
start between the final check and removal.

Before any write, pruning acquires the complete personal-public target and stable
paginated inventory, details and labels, fixes the unlisted IDs, prepares selected
label updates, and requires each candidate to have a valid online/offline state and
`busy: false`. Unknown or incomplete data fails closed. Both online and offline
idle runners may be selected. The complete inventory is checked again before the
family's first write and each fixed-ID DELETE. Additions, disappearance, changed
candidate state or metadata, and changed retained identity or configuration fail
rather than expanding or replacing the candidate set. Retained operational status,
busy and version changes alone do not block an unrelated deletion. Selected label
updates complete first; only independently verified own changes are incorporated
into later checks. A confirmed explicit unarchive updates the expected archive state.

Deletion requires an empty HTTP 204 and a fresh complete inventory confirming the
fixed ID's absence and preservation of retained objects. A 404 or acknowledgment
alone does not prove removal. A naturally disappearing runner requires replanning.
An attempted deletion or failed verification can have uncertain effects; earlier
confirmed changes remain, later work stops, and no retry or rollback is automatic.
Detailed deletion plans show ID, name, OS and custom labels as the before value;
status-only hides these values. Operational state, optional raw label IDs and other
private evidence remain outside reports. Fixtures do not establish live job behavior.

The [official labels guide](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/apply-labels)
confirms the user-repository UI and case-insensitivity. The
[repository runner REST API](https://docs.github.com/en/rest/actions/self-hosted-runners)
provides observation and custom-label replacement. Reads require repository
Administration read and writes require Administration write. Local fixtures do
not prove a populated UI, exact server Unicode comparison, live writes, runner
availability, routing, physical OS or job completion. This support does not
register runners, generate tokens, install software, manage groups or modify
workflows and hosts.

[All Actions settings](README.md) · [GitHub App permissions](../permissions.md#actions)
