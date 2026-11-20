---
type: Reference
title: "About settings"
description: "About settings declaration, ownership and API contracts."
---

Description and Website use the [basic field contract](configuration.md#supported-fields).
The Topics contract below owns complete-set membership.

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Topics

`repository.about.topics` manages the complete topic set. It is a direct array, not a mapping. Omission leaves it unmanaged; `[]` explicitly clears all topics; `{}` and null are invalid. Names are unique lowercase letters/numbers/hyphens, 1–50 characters each, at most 20 entries. Names are public metadata even on private repositories. Export, comparison and PUT use sorted set order without rewriting existing input YAML. Apply reads complete, validated current membership and skips PUT when the set matches, regardless of order. Changed membership uses one full-set PUT followed by an independent GET; the PUT response alone is not confirmation. A failed prerequisite read makes no Topics write. A rejected update or failed/mismatching readback stops later resources with uncertain effects and no automatic retry or rollback. Metadata read and Administration write must already exist. See [Topics and ordering](../commands.md#apply-behavior).
