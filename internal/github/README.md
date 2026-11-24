# GitHub API adapters

This package reads repository settings through REST and GraphQL, translates them into settings observations, and writes declared changes. Files are grouped by resource. For example, [label reads](labels.go) acquire the inventory and [label operations](label_operations.go) create, update or delete labels and verify the outcome.

When changing how a configuration field maps to native settings, start with [declaration conversion](ui_declaration.go). For the reverse direction, start with [export conversion](ui_export.go). The [architecture guide](../../docs/development/architecture.md#following-a-setting-through-the-code) traces these conversions alongside acquisition and readback.
