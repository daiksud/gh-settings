# Public references

These files define the interfaces callers can rely on. Update the reference that owns the interface when its behavior changes.

| Interface | Where to edit |
| --- | --- |
| CLI | [Commands](commands.md) covers arguments, output, exit status and recovery after a partial apply. |
| Public GitHub Action | [Action reference](action.md) covers inputs, outputs, plan reporting and failures. |
| Settings declaration | The [settings index](settings/README.md) points to YAML fields by GitHub Settings page, collection semantics and required permissions. |

The corresponding [feature specifications](../development/features/README.md) identify the scenarios and tests that verify those contracts.
