---
type: Feature
title: Manage selected label objects by ID
description: Create ID-less labels and reconcile whole supported ID-selected objects without deleting unlisted labels.
---

## Feature: Manage repository labels

Current source adopts ID-based ownership; this is unreleased. Published v0.19.0 and earlier keep the legacy authoritative collection contract. The [settings reference](../settings.md#labels) owns the schema and migration instructions.

### Rule: Select objects, never delete unlisted labels

- Omitted and empty collections select nothing and make no label calls.
- A positive ID selects only an existing label in the target repository. Missing/wrong IDs fail before label mutation.
- No ID means create-only. A colliding name is an error showing the existing ID, not auto-adoption.
- Duplicate IDs and names under Unicode simple case folding fail. Case-only spelling is retained; a different noncolliding name with the same ID renames without deletion.
- Name and color are required; description omission is the fixed empty-string default on both create and update. Null is invalid.
- Complete selected object values are managed. Unlisted objects, assignments, archive state and server metadata are not removed or replayed.

### Rule: Observe complete identity and report honest failures

- Follow every labels page and require positive unique IDs and complete supported values. Nullable response descriptions normalize to empty strings.
- Export saves IDs and complete supported fields only after all selected reads/encoding succeed; failure preserves the output file.
- Plan shows creates, ID-bound updates and default-derived changes. It never writes configuration.
- Apply reads fresh labels after ordinary resources, preflights all label changes, and performs only creates/updates in stable identity order.
- Resolve update paths from the observed ID/current name, check ID again before PATCH, and verify response identity and supported settings. This is not a lock against concurrent remote replacement.
- Response mismatch, denial, redirect, unknown effect or later failure stops subsequent writes. Earlier writes remain applied, with no rollback/retry.

### Rule: Persist confirmed create IDs without committing configuration

- Each verified creation reports its ID and is recorded before the next write.
- Normal CLI apply adds only new IDs to the source snapshot; preserve existing IDs, unrelated values, file mode and comments as practicable.
- Refuse changed snapshots, unsafe files/directories, and aliased/merged collection entries instead of overwriting unrelated work. Formatting may normalize.
- Writeback failure after creation is a partial failure with the recoverable remote ID. Earlier saved IDs survive later failures.
- `--no-writeback` or `GITHUB_ACTIONS=true` disables recording; display created IDs and leave YAML untouched. A repeat ID-less create then fails with ID guidance rather than adopting the object.
- No Git commit/push, saved-plan mutation input, secret read or credential expansion is involved.

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
| Pagination/completeness, exact payloads, escaped paths | [GitHub adapter tests](../../internal/github/labels_test.go), [individual request tests](../../internal/github/label_operations_test.go) |
| Fresh read, conditional resources, sequencing and partial failures | [Application tests](../../internal/app/labels_test.go), [operation sequence tests](../../internal/app/label_sequence_test.go) |
| Typed report, attribute presence, escaped Summary, malformed-report preservation | [Actions tests](../../internal/actions/labels_test.go) |
| Built CLI export/plan/apply, failed-page preservation, redirects, report and credential-free Summary | [Process acceptance](../../tests/acceptance/labels_test.go) |

Tests use isolated files/Git and a local API with dummy credentials. They do not mutate live labels or establish real credential permissions.
