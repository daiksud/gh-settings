# YAML schema design for v1

This ledger records the YAML that an owner of a personal public repository could
use if every eligible setting had a complete GitHub settings API. Use the
[public settings reference](../../references/settings/README.md) to write an
operational declaration. Use the [coverage ledger](../settings-coverage.md) to
check target applicability and the dated UI and API evidence.

## Reading the ledger

- [Current schema](current.yml) records the implemented document shape in
  export order, scalar types and closed choices. A sequence in the shape
  describes the element. The field reference for each family gives conditional
  validation, empty values and object ownership. The legacy v0 entries retain
  their existing contracts under the [adoption policy](../../references/settings/scope.md).
- [Editor order](export-order.yml) lists verified differences in the order of
  existing fields. Each entry is marked implemented when the encoder matches.
- The [General and About](future/general.yml), [automation](future/automation.yml),
  [security](future/security.yml) and [integrations](future/integrations.yml)
  tables reserve additional UI controls and choices individually.
  Each record identifies an absolute YAML path, type, implementation status,
  UI label and evidence. An entry with `extension: enum` adds choices to its field.
  `before` and `after` name siblings in the same mapping. Sequence items in paths
  use `[]`. Group rows inherit their UI context and API limits to their children
  unless a child records distinct evidence. The order of input keys is free.
- A `reserved` field or enum choice is unavailable for operational input.
  `reshape` records a replacement for a currently supported path.
  An `unconfirmed` design is provisional. List records specify an item type, or `unconfirmed` when the UI
  does not establish a stable element identity. Publishing the design does not make
  these keys usable. A release can accept a new field once it implements the
  applicable observation, update and independent verification behavior.
- The `gaps-*.yml` tables record evidence that would change the design. An
  `additive` gap can be filled without altering an existing declaration.
  A `blocking` gap keeps the overall v1 schema open until its compatibility
  question is resolved. An `excluded` entry identifies a product or target
  boundary. Record the date and repository or guide when adding UI evidence.
  A conditional control may be visible only on a differently configured repository.

The November 27, 2026 review used the English desktop UI on
[gh-settings](https://github.com/daiksud/gh-settings) and
[test-gh-settings-public](https://github.com/daiksud/test-gh-settings-public).
The [general](gaps-general.yml), [automation](gaps-automation.yml),
[security](gaps-security.yml) and [integration](gaps-integrations.yml) gap
records describe the conditional forms and operations still to verify.
Run the ledger checks from the repository root.

```sh
go test ./internal/workspace -run 'Test(CurrentSchema|FutureSchema|Schema|RemovedRulesetDelete)'
```

## Ownership and compatibility

Omitting an ordinary scalar leaves that control unmanaged. Explicit false,
zero, empty strings and empty lists retain the field-specific meanings in the
current reference. Keep a list's resource identity, update policy, child
ownership and prune behavior stable across a schema change. An independently
configurable UI checkbox gets its own boolean. Place conditional inputs under
their actual group; separate page headings can also be siblings.

Whole-object writers need special attention. A declared Ruleset can own its
existing rules and bypass actors. Before adding a rule to the codec, exercise an
old declaration against an acquired object carrying that new rule. Preserve the
new rule if omitted or stop before the write if exact preservation cannot be
verified. Likewise, adding a collaborator identity type cannot silently broaden
the set of people or invitations that an existing prune can remove.

Use the current English UI labels and order when designing the initial shape.
A later UI relabel or move can update the evidence without renaming an already
published v1 key. A new optional key must leave existing input, output semantics
and ownership intact. Unavailable observation remains unknown; export never
invents a desired false value or an empty collection for a future field.

## v0 migration and v1 freeze

The reviewed schema changes land in v0.x. Move an explicitly configured Wiki
checkbox from `general.features.wikis` to `general.features.wikis.enabled`.
Move `advanced_security.private_vulnerability_reporting` to
`advanced_security.private_vulnerability_reporting.enabled`. Both examples
are relative to `repository.settings`. Carry the original boolean, including
false, and keep an omitted setting omitted. The Wiki edit restriction and the
private reporting detail controls have independent ownership.

Keep the root marker `version: daiksud/gh-settings@v0` throughout v0.x.
Switch it to `daiksud/gh-settings@v1` at CLI v1 publication. A declaration
using only v1-eligible settings in the final v0 layout should need only the
marker change. Declarations using classic branch protection or Autolinks follow
their separate v0 contracts and need a separate v1 transition decision.

Review each pending reshape and blocking evidence item before declaring the
design frozen. For each implementation, update the ledger, public examples,
behavior specifications, CLI and Action reports together. Test the previous
shape's rejection before settings API access, unordered input, canonical
export, explicit and omitted values, a complete apply cycle and repeat plan.
After a compatible v0 package is published, update both operational Action
pins, the workflow lockfile and the repository declaration in the same PR.
Verify its read-only plan before applying and compare live state afterwards.
