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

Published v0.16.0 adds the [web commit sign-off requirement](web-commit-signoff.feature.md) through the existing repository REST read/write path.

Published v0.16.0 also adds the [template repository setting](template-repository.feature.md) through the existing repository REST read/write path.

Published v0.16.0 adds the [default branch setting](default-branch.feature.md), including literal-name validation and PATCH response verification.

These three features are published in v0.16.0; older consumer pins must be updated before adopting them.

The [Rulesets design](rulesets.feature.md) records the accepted declaration/reconciliation contract and its implementation published in v0.17.0, including PR/check rules; published v0.16.0 and older CLI/Action packages reject Rulesets.

Published v0.17.0 adds [pull request access controls](pull-request-access.feature.md) with independent availability/policy declarations, complete GraphQL reads and declared-only mutations. Published v0.16.0 and older packages reject these keys.

Published v0.17.0 adds [Issue creation policy](issue-creation-policy.feature.md), an independent enum with selective GraphQL reads and a shared declared-only repository mutation. It does not change `has_issues` implicitly; published v0.16.0 and older packages reject the key.

Published v0.17.0 adds [Actions execution policy](actions-execution-policy.feature.md), with explicit export opt-in, complete conditional reads, inactive-value comparisons and fresh nontransactional apply. Published v0.16.0 and older packages reject it.

Published v0.17.0 adds [default workflow permissions](workflow-permissions.feature.md), independently of execution policy, with declared-only updates and explicit export selection. Published v0.16.0 and older packages reject it.

The [Security design](security.feature.md) defines proposed feature-family selection, complete observations, dependency ordering, and adoption gates for #344–#347. Secret scanning and private vulnerability reporting for public repositories are implemented in v0.17.0; other families remain planned/gated. Published v0.16.0 and older packages reject all proposed security keys/options.

Published v0.17.0 adds [repository retention policy](retention.feature.md), with complete configured/maximum observations, explicit export opt-in, native integer differences and fresh upper-limit verification. Published v0.16.0 and older packages reject the key and option.

Published v0.17.0 adds [fork workflow policies](fork-workflows.feature.md): public contributor approval and private fork execution/token/secrets/approval choices, with explicit export selections, declared-only updates and fresh visibility/readback boundaries. Published v0.16.0 and older packages reject the keys and options.

The v0.17.0 feature set and the bounded additions below are published in [v0.18.0](https://github.com/daiksud/gh-settings/releases/tag/v0.18.0) from immutable source `6d3146b30860692f84dc54e32d746e6fbbb55c66`. Native acceptance, immutable assets and published Action smoke checks passed. Unimplemented/gated candidates remain outside this package. See the [package boundary](../settings.md#v0180-package).

Published v0.18.0 adds [Sponsor button visibility](sponsorships.feature.md), with an independent complete boolean read and declared-only shared repository mutation. It does not manage funding destinations or payments; published v0.17.0 and older packages reject the key.

Published v0.18.0 adds [repository archive/unarchive](repository-lifecycle.feature.md), with explicit sequencing, identity/readback verification, and opt-in `export --archive`. Published v0.17.0 and older packages reject this field and option. Visibility and rename remain deferred; no live repository has been archived.

Published v0.18.0 adds the [existing-site Pages contract](pages.feature.md) from #350, including cname-only null clearing and explicit export. Creation/deletion and visibility changes remain deferred until their observation and exposure gates are resolved; #350 retains that lifecycle work. Published v0.17.0 and older packages reject all Pages keys/options.

The [Environment design](environments.feature.md) for #351 separates selected names from authoritative membership, protection reductions from omission, and typed reviewer identity from display names. All Environment syntax remains unimplemented. #352/#353 require the documented observation, preservation and transition evidence before adoption; lifecycle and undocumented administrator-bypass mutations remain deferred.

Published v0.18.0 adds the bounded [custom property values](custom-properties.feature.md) contract, with explicit export selection, schema-bound validation and null unset. Published v0.17.0 and older packages reject it; default-backed and other unadopted constraints remain gated in #354.

The [repository access design](repository-access.feature.md) reserves collaborator/team ownership, pending invitation outcomes and revocation gates for #356/#357. It is unimplemented; current source and published packages reject access declarations and export options.

Published v0.18.0 adds the bounded [private repository forking policy](private-forking.feature.md), with complete organization-policy observation and explicit export selection. Published v0.17.0 and older packages reject it; public/internal/personal/fork/archive targets and lifecycle composition remain unadopted.
