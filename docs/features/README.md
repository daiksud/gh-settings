# Behavior specifications

These specifications describe detailed behavior and map scenarios to existing Go tests. They are contributor reference material, written in Markdown; no BDD runner executes these files.

For usage, start with the [command reference](../commands.md), [configuration reference](../settings.md), or [GitHub Actions guide](../github-actions.md). For choosing checks and understanding what they prove, use the [testing guide](../contributing/testing.md).

| Specification | Responsibility |
| --- | --- |
| [Export](export.feature.md) | Read the checkout's supported state and save configuration. |
| [Validate](validate.feature.md) | Accept valid declarations and reject invalid configuration locally. |
| [Plan](plan.feature.md) | Compare declared resources and report differences without mutation. |
| [Apply](apply.feature.md) | Write declared values and report partial or failed updates. |
| [Labels](labels.feature.md) | Authoritative membership, retained attributes, fresh-state reconciliation, and label-specific test mappings. |
| [GitHub Actions](actions.feature.md) | Public Action behavior, consumer scenarios, and internal adapter test mappings. |
| [Release](release.feature.md) | Manual publication, tested binary identity, and recovery. |

When behavior changes, update its user reference and affected scenarios together. Keep test mappings accurate, put detailed fixture implementation and evidence limits in the testing guide, and use [ADRs](../adr/README.md) for decision rationale.

The v0.10.0 [repository merge-method extension](merge-methods.feature.md) adds three booleans to the existing repository group, with a narrow GraphQL read and the existing REST apply path.

The v0.11.0 [automatic branch-deletion setting](delete-branch-on-merge.feature.md) adds one optional repository boolean using the same fixed GraphQL read and direct REST PATCH.

The v0.12.0 [squash commit defaults](squash-commit-defaults.feature.md) add two optional repository enums with explicit message/title dependency and the same read/write paths.

The v0.13.0 [auto-merge and branch-update options](pull-request-options.feature.md) add two independent repository booleans through the existing fixed GraphQL read and direct REST PATCH.

The v0.14.0 [normal merge commit defaults](merge-commit-defaults.feature.md) add two repository enums with an independent message/title dependency, without enabling normal merges.

v0.15.0 adds the optional [`has_discussions` setting](discussions.feature.md) through the repository REST read/write path with a verified PATCH response.

Current source adds the unreleased [web commit sign-off requirement](web-commit-signoff.feature.md) through the existing repository REST read/write path.

Current source also adds the unreleased [template repository setting](template-repository.feature.md) through the existing repository REST read/write path.

Current source adds the unreleased [default branch setting](default-branch.feature.md), including literal-name validation and PATCH response verification.

These three current-source features are planned for v0.16.0 and remain unreleased until the separate release workflow publishes and verifies the matching package.

The [Rulesets design](rulesets.feature.md) records the accepted declaration/reconciliation contract and its current-source, unreleased implementation, including PR/check rules; published v0.16.0 and older CLI/Action packages reject Rulesets.

Current source adds unreleased [pull request access controls](pull-request-access.feature.md) with independent availability/policy declarations, complete GraphQL reads and declared-only mutations. Published v0.16.0 and older packages reject these keys.

Current source adds unreleased [Issue creation policy](issue-creation-policy.feature.md), an independent enum with selective GraphQL reads and a shared declared-only repository mutation. It does not change `has_issues` implicitly; published v0.16.0 and older packages reject the key.

Current source adds unreleased [Actions execution policy](actions-execution-policy.feature.md), with explicit export opt-in, complete conditional reads, inactive-value comparisons and fresh nontransactional apply. Published v0.16.0 and older packages reject it.

Current source adds unreleased [default workflow permissions](workflow-permissions.feature.md), independently of execution policy, with declared-only updates and explicit export selection. Published v0.16.0 and older packages reject it.
