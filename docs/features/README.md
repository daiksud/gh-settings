# Behavior specifications

These specifications describe detailed behavior and map scenarios to existing Go tests. They are contributor reference material, written in Markdown; no BDD runner executes these files.

For usage, start with the [command reference](../commands.md), [configuration reference](../settings.md), or [GitHub Actions guide](../github-actions.md). For choosing checks and understanding what they prove, use the [testing guide](../contributing/testing.md).

| Specification | Responsibility |
| --- | --- |
| [Export](export.feature.md) | Read the checkout's supported state and save configuration. |
| [Validate](validate.feature.md) | Accept valid declarations and reject invalid configuration locally. |
| [Plan](plan.feature.md) | Compare declared resources and report differences without mutation. |
| [Apply](apply.feature.md) | Write declared values and report partial or failed updates. |
| [GitHub Actions](actions.feature.md) | Public Action behavior, consumer scenarios, and internal adapter test mappings. |
| [Release](release.feature.md) | Manual publication, tested binary identity, and recovery. |

When behavior changes, update its user reference and affected scenarios together. Keep test mappings accurate, put detailed fixture implementation and evidence limits in the testing guide, and use [ADRs](../adr/README.md) for decision rationale.
