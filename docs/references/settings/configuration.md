---
type: Reference
title: YAML configuration
description: Structure, types, field ownership and exported values in .github/settings.yml.
sources:
  - id: ui-inventory
    resource: ../../development/ui-repository-mapping.md
---

# YAML configuration

Put one YAML mapping document in `.github/settings.yml` at the root of the target checkout. The required version marker is the literal string `daiksud/gh-settings@v0`. It identifies the schema independently of the installed CLI version.

## Supported fields

Use `repository.about` for the repository home page, `repository.labels` for labels, and `repository.settings` for controls on Settings pages. These three keys are siblings.

```yaml
version: daiksud/gh-settings@v0
repository:
  about:
    description: My project
    website: https://example.com
    topics: [github-cli, settings-as-code]
  labels:
    - name: bug
      color: '#d73a4a'
      description: Something is not working
  settings:
    general:
      features: {issues: true}
```

Within `settings`, follow the paths listed in the [settings index](README.md). Keys follow English UI names in snake_case. For example, Moderation contains `interaction_limits.pull_request_limits`. Some controls sit directly on a page, while others live under a section, such as `workflow_permissions.default_permissions` on the Actions General page. The [UI mapping](../../development/ui-repository-mapping.md) records the naming evidence. Use only the documented names and types for your selected [CLI or Action package](../../guides/github-actions.md#published-package-and-consumer-pins).

## Configuration semantics

| Declaration | Meaning |
| --- | --- |
| Omitted field | Leave it unmanaged. A version-only document or `repository: {}` has no managed settings. CLI plan and apply finish locally for this case. |
| `false`, `0`, or `''` | An explicit value. The field must accept its type and range. |
| Empty mapping | An ordinary section selects no scalar fields. In a collection, an entry can select a whole object or rule with its own required fields and defaults. |
| Empty list | The effect depends on the collection. Empty topics clears all topics. Empty labels preserves existing labels in ordinary apply and selects them for deletion with `--prune labels`. [Collection pruning](pruning.md#collection-pruning) |
| `null` | Valid only for [the Pages custom domain](pages.md#pages-configuration), where it requests removal. |

Labels and Rulesets manage listed whole objects under their respective field rules. Existing Environments preserve omitted built-in settings and a missing Environment can be created with a name-only entry. Each [resource reference](README.md) defines how omission works inside its entries.

Validation rejects unknown, duplicate or non-string keys, unsupported tags, multiple documents, wrong scalar types, and out-of-range or unrecognized choices. Aliases and merges are checked after resolution. Use a YAML string for a string field and a true boolean for a checkbox. Validation finishes before settings API work.

## Exported values

Export writes the settings it could fully observe. An unavailable group is omitted on first export and can retain compatible prior declarations on re-export. Treat a retained value as unverified intent. Check the diagnostics and the resulting file before applying it. [Export behavior](../commands.md#export)

Some GitHub APIs return values outside the choices shown in the UI. Where a resource defines a fallback, export writes its documented UI value and reports the normalization. Plan can still show a required change even when the displayed values match, because apply must verify the requested value against the raw state. Missing data and incomplete inventories never supply default values.

For field-specific rules, see [merge menus](general.md#commit-message-observation-normalization), [Actions](actions/general.md#execution-policy-observation-normalization), [Rulesets](rulesets.md), [Environments](environments.md#environment-deployment-mode-observation-normalization), [Pages](pages.md#pages-configuration), and [Advanced Security](security.md). The CLI and detailed Action Summary use the same UI paths and choices.
