---
type: Reference
title: "Label settings"
description: "Label settings declaration, ownership and API contracts."
sources:
  - id: github-labels
    resource: https://docs.github.com/en/rest/issues/labels
---

# Labels

`repository.labels` selects whole label objects by positive REST `id`; no ID means create-only. Without prune, omission or `[]` selects nothing and unlisted objects remain untouched. Explicit `--prune labels` changes present-list ownership: [] selects all current labels and nonempty lists retain declared IDs. Omission always stays unmanaged. Deletion loses Issue/PR assignments; see [collection pruning](pruning.md#collection-pruning).

Export writes the label ID first, followed by the editable fields in GitHub's form order. Input YAML accepts these keys in any order.

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive integer; selects an existing label in the target repository; missing ID errors, never create fallback |
| `name` | Required nonempty/non-whitespace string without control characters; spaces, punctuation and emoji allowed |
| `description` | Optional string, at most 100 Unicode characters; omission defaults to `''` on create/update, clearing an existing description |
| `color` | Required `#` plus six hex digits; quote the string; canonical lowercase, no guessed default |

Duplicate IDs and Unicode-simple-folded names fail without trimming. ID-less collisions show the existing ID for explicit adoption. A valid ID can rename to a noncolliding name while preserving assignments; case-only differences keep remote spelling. Renames cannot collide even with another planned rename; apply is not a rename transaction. Export requires complete IDs/settings and pagination, includes only writable supported values and sorts labels consistently. Incomplete reads fail rather than becoming empty state.

Apply compares a fresh complete collection and skips matching selected objects.
Updates check the selected ID at its observed name before writing. After each
create/update response passes validation, apply independently reads the resulting
name and requires the same positive ID, name, color and description. Null API
descriptions normalize to empty strings. A failed or mismatching readback stops
later operations, including prune deletions, with uncertain write effects; a
matching acknowledgment alone does not establish success. These checks do not
lock the label against concurrent changes.

## Apply ID writeback

After verifying a newly created label, CLI apply inserts its numeric ID before the existing fields in the original declaration, preserving their order. A later failure can leave that binding in place. Read [how apply records and recovers IDs](../commands.md#recording-ids) for file checks, aliases, `--no-writeback` and GitHub Actions behavior. Listing and changing labels require the appropriate [Issues or Pull requests grants](permissions.md#general-about-and-labels).[^github-labels]

A positive Label ID from the creation response remains in recovery diagnostics
even if later validation or readback fails. It is not automatically written to
YAML on that failure. Retain the reported ID and inspect the remote object before
retrying; earlier confirmed IDs already recorded in YAML remain intact.

[^github-labels]: Labels listing/mutation and existing permission boundary.

[Settings index](README.md) · [GitHub App permissions](permissions.md)
