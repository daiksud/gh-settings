---
type: Feature
title: Manage repository labels
description: Reconcile declared label membership and selected attributes.
---

## Feature: Manage repository labels

Labels support is published in v0.9.0 and included in the current pinned workflows, without changing `schemaVersion: 1`. Older v0.8.0 packages do not include it. [Settings](../settings.md#labels) defines the configuration contract. This feature manages repository label definitions, not individual issue/PR assignments.

### Rule: Declare authoritative membership explicitly

```yaml
schemaVersion: 1
settings:
  labels:
    - name: bug
      color: d73a4a
      description: Something is not working
    - name: help wanted
```

- Omitted `labels` leaves the collection unmanaged; plan/apply make no label request.
- A provided list manages the entire membership: undeclared remote labels are deleted. `labels: []` deletes every remote label.
- Deletion removes the label from issues and pull requests. Recreating the same name does not restore those associations. Review plan before applying.
- `name` is required and identifies a label. Names must be strings that are neither empty nor whitespace-only, without control characters. Spaces, punctuation and emoji are allowed.
- Identity uses Unicode simple case folding, without trimming or renaming. Duplicate/ambiguous identities in declarations or observations fail; case-only spelling changes retain the remote spelling.
- `color` and `description` are optional strings. Omitted attributes preserve existing values; when creating a label, omission delegates those values to GitHub. No defaults are synthesized.
- `color` is six hexadecimal characters without `#`, normalized to lowercase. `description` permits up to 100 Unicode characters; `description: ""` explicitly clears it.
- Explicit null, wrong shapes/types, duplicate keys and unsupported attributes fail validation before API access. Aliases and merges use the existing YAML rules.
- No rename inference or `new_name` is supported. Changing a name to a different identity creates the new label and deletes the old label, losing old associations. Archive settings and read-only metadata are not managed.

### Rule: Complete reads before export or reconciliation

- Export reads repository settings, topics, then every labels page. It saves all observed label names/colors/descriptions, sorted by identity, including `labels: []` when none exist.
- Labels reads use `GET /repos/{owner}/{repo}/labels` with `per_page=100` and successive `page` values until a short page. Exactly 100 results require another request.
- Missing required response values, malformed state, ambiguous names or a failed later page fail the whole operation. Nullable response descriptions become empty strings.
- Export finishes all reads and serialization before opening the destination; a read failure preserves the previous file. Exporting makes the observed membership authoritative in the saved declaration.
- Plan reads only declared resources and shows ordered create/update/delete operations. A Labels collection counts as one managed item, including an explicitly empty list.

### Rule: Reconcile fresh state during apply

- Repository PATCH and topics PUT retain their direct-write behavior, in that order. If Labels is declared, apply then reads all label pages using the apply credential and compares against the declaration.
- Execute creates/updates in identity order, followed by deletes in identity order. Stop on the first failure, leaving earlier successful resource/label writes applied. There is no transaction, rollback or automatic retry.
- Create uses POST with `name` and only declared attributes. Update uses PATCH with only changed declared attributes. Delete uses DELETE for each undeclared name. Names are escaped as individual URL path segments; rename/archive fields are never sent.
- Apply rejects redirects for both its label reads and writes, preserves the configuration file, and never consumes a saved plan. Concurrent remote edits can change the operations after review; this is not drift protection.
- CLI plan and the same-version Actions report/Summary show explicit operation names. Deletes state that issue/PR associations are removed. Names and attribute values remain escaped in terminal/HTML output.

### API and permissions

[GitHub's Labels REST API](https://docs.github.com/en/rest/issues/labels) supports these repository endpoints. Fine-grained tokens and GitHub App tokens need Issues **or** Pull requests read for listing, and write for mutations; public lists can be read unauthenticated. Administration write alone does not grant label writes. No paid-plan prerequisite is documented for these endpoints.

Credentials and caller permissions remain the owner's responsibility. This source change does not expand the existing App token, workflow permissions, or the repository's own managed settings. [Deleting labels](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/managing-labels#deleting-a-label) explains the assignment loss.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Identity, validation, omission, attribute presence, stable reconciliation | [Settings model tests](../../internal/settings/labels_test.go) |
| Validated declaration ownership, independent retrieved values and operations | [Declaration collection tests](../../internal/settings/label_declarations_test.go) |
| Completed observation ownership, invalid state, missing versus empty read | [Observation collection tests](../../internal/settings/label_observation_test.go) |
| Closed operations, canonical attributes, unique sequence and independent visitor values | [Operation model tests](../../internal/settings/label_operations_test.go) |
| Strict schema, aliases/merges, null rejection, stable serialization | [Workspace codec tests](../../internal/workspace/labels_test.go) |
| Pagination/completeness, exact payloads, escaped paths, failure stop | [GitHub adapter tests](../../internal/github/labels_test.go) |
| Fresh read, conditional resources, sequencing and partial failures | [Application tests](../../internal/app/labels_test.go) |
| Typed report, attribute presence, escaped Summary, malformed-report preservation | [Actions tests](../../internal/actions/labels_test.go) |
| Built CLI export/plan/apply, failed-page preservation, redirects, report and credential-free Summary | [Process acceptance](../../tests/acceptance/labels_test.go) |

Tests use isolated files/Git and a local API with dummy credentials. They do not mutate live labels or establish real credential permissions.
