---
type: Feature
title: Manage selected label objects by ID
description: Create ID-less labels and reconcile whole supported ID-selected objects with separately selected collection pruning.
---

## Feature: Manage repository labels

Current source uses `repository.labels` with ID-selected ownership. The [settings reference](../settings.md#labels) owns the current UI schema and published-package boundary; there is no legacy migration.

### Rule: Select objects and prune only when explicitly requested

- Omitted collections always select nothing. Without prune, empty collections also make no label calls; with explicit prune, [] selects all candidates.
- A positive ID selects only an existing label in the target repository. Missing/wrong IDs fail before label mutation.
- No ID means create-only. A colliding name is an error showing the existing ID, not auto-adoption.
- Duplicate IDs and names under Unicode simple case folding fail. Case-only spelling is retained; a different noncolliding name with the same ID renames without deletion.
- Name and quoted UI color (`#` plus six hex digits) are required; description omission is the fixed empty-string default on both create and update. Null is invalid.
- Complete selected object values are managed. Unlisted objects remain unless `--prune labels` selects the explicitly present list. See the [prune contract](prune.feature.md) for omission/empty semantics, preflight and assignment loss. Archive state and server metadata are not replayed.

### Rule: Observe complete identity and report honest failures

- Follow every labels page and require positive unique IDs and complete supported values. Nullable response descriptions normalize to empty strings.
- Export adopts IDs and complete supported fields only after all label pages succeed; unavailable acquisition emits info and retains prior Labels while independent reads continue.
- Plan shows creates, ID-bound updates and default-derived changes. It never writes configuration.
- Apply reads fresh labels after ordinary resources, preflights all label changes, and performs creates/updates in stable identity order, then fixed prune deletions when requested.
- Matching selected objects emit no mutation. Resolve update paths from the observed ID/current name, check ID again before PATCH, and verify response identity and supported settings.
- After a valid create/update response, independently GET the resulting name and verify the same ID and complete supported settings. Renames read the new name. Missing/incomplete responses, changed identity or mismatching settings fail. These checks are not a lock against concurrent remote replacement.
- Response or readback mismatch, denial, redirect, unknown effect or later failure stops subsequent writes, including prune deletions. Postwrite read failure has uncertain effects and is not a prewrite failure. Earlier confirmed writes remain applied, with no rollback/retry.

### Rule: Persist confirmed create IDs without committing configuration

- Each creation verified by acknowledgment and independent readback reports its ID and is recorded before the next write.
- A positive ID from a create response is retained in recovery diagnostics even if validation/readback fails. That failed creation is not written into YAML; earlier confirmed saved IDs remain intact.
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
| Strict schema, aliases/merges, null rejection, stable serialization | [Workspace codec tests](../../internal/workspace/ui_labels_test.go) |
| Pagination/completeness, exact payloads, escaped paths | [GitHub adapter tests](../../internal/github/labels_test.go), [individual request tests](../../internal/github/label_operations_test.go) |
| Fresh read, conditional resources, sequencing and partial failures | [Application tests](../../internal/app/labels_test.go), [operation sequence tests](../../internal/app/label_sequence_test.go) |
| Typed report, attribute presence, escaped Summary, malformed-report preservation | [Actions tests](../../internal/actions/labels_test.go) |
| Built CLI export/plan/apply, failed-page preservation, redirects, report and credential-free Summary | [Process acceptance](../../tests/acceptance/labels_test.go) |
| Independent create/update readback, recovery without ID writeback, rename before prune and matching repetition | [Readback process tests](../../tests/acceptance/label_readback_test.go), [ID writeback tests](../../tests/acceptance/label_ids_test.go) |

Tests use isolated files/Git and a local API with dummy credentials. They do not mutate live labels or establish real credential permissions.
