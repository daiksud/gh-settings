# Behavior specifications

These files describe observable behavior and its Go test mappings. They are
contributor references; no BDD runner executes them. The [configuration
reference](../settings.md), [CLI reference](../commands.md), and [Action
guide](../github-actions.md) own user-facing behavior.

| Specification | Responsibility |
| --- | --- |
| [Export](export.feature.md) | Read supported state into a configuration file. |
| [Validate](validate.feature.md) | Validate declarations locally. |
| [Plan](plan.feature.md) | Compare declared values without writes. |
| [Apply](apply.feature.md) | Apply declarations and report partial failure. |
| [Labels](labels.feature.md) | Manage selected label objects by ID. |
| [Rulesets](rulesets.feature.md) | Manage selected repository rulesets. |
| [Environments](environments.feature.md) | Manage supported named environment policies. |
| [GitHub Actions](actions.feature.md) | Operate the public composite Action. |
| [Actions execution](actions-execution-policy.feature.md) | Manage repository Actions policy. |
| [Workflow permissions](workflow-permissions.feature.md) | Manage default Actions token and approval policy. |
| [Fork workflows](fork-workflows.feature.md) | Manage fork pull-request workflow controls. |
| [Security](security.feature.md) | Manage adopted security controls. |
| [Immutable releases](immutable-releases.feature.md) | Manage repository release immutability policy and its explicit 404 tradeoff. |
| [Retention](retention.feature.md) | Manage Actions retention policy. |
| [Pages](pages.feature.md) | Manage supported existing Pages configuration. |
| [Release](release.feature.md) | Publish and verify immutable artifacts. |
| [Default branch](default-branch.feature.md) | Manage the repository default branch. |
| [Repository lifecycle](repository-lifecycle.feature.md) | Archive and unarchive repositories. |
| [Discussions](discussions.feature.md) | Manage Discussions availability. |
| [Template repository](template-repository.feature.md) | Manage template availability. |
| [Web signoff](web-commit-signoff.feature.md) | Manage web commit sign-off requirement. |
| [Merge methods](merge-methods.feature.md) | Manage allowed merge methods. |
| [Automatic branch deletion](delete-branch-on-merge.feature.md) | Manage head-branch deletion after merge. |
| [Squash commit defaults](squash-commit-defaults.feature.md) | Manage squash commit title and body defaults. |
| [Merge commit defaults](merge-commit-defaults.feature.md) | Manage normal merge message defaults. |
| [Auto merge / branch update](pull-request-options.feature.md) | Manage optional PR merge behavior. |
| [Pull request access](pull-request-access.feature.md) | Manage pull-request availability and creation policy. |
| [Issue creation](issue-creation-policy.feature.md) | Manage issue creation policy. |
| [Sponsorships](sponsorships.feature.md) | Manage Sponsor button visibility. |
| [Repository access](repository-access.feature.md) | Describe separately proposed access management boundaries. |

Update affected scenarios, tests, and the authoritative user reference together
when current behavior changes. Use [testing](../contributing/testing.md) for
validation boundaries and [ADRs](../adr/README.md) for decision rationale.
