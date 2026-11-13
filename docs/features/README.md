# Behavior specifications

These files describe observable behavior and its Go test mappings. They are
contributor references; no BDD runner executes them. The [configuration
reference](../settings.md), [CLI reference](../commands.md), and [Action
guide](../github-actions.md) own user-facing behavior.

| Specification | Responsibility |
| --- | --- |
| [Export](export.feature.md) | Read supported state into a configuration file. |
| [Automatic readable settings](readable-settings-export.feature.md) | Default family acquisition, complete inventories, info and prior-intent retention. |
| [Validate](validate.feature.md) | Validate declarations locally. |
| [Plan](plan.feature.md) | Compare declared values without writes. |
| [Apply](apply.feature.md) | Apply declarations and report partial failure. |
| [Persistent pull request limits](pull-request-limits.feature.md) | Manage the persistent open PR cap without temporary restrictions or exemptions. |
| [Actions OIDC subject template](oidc-subject.feature.md) | Manage the complete active template while preserving known immutable mode. |
| [Copilot MCP tools](copilot-mcp.feature.md) | Manage the stored MCP tools permission toggle without configuring servers or executing reviews. |
| [Copilot review effort](copilot-review-effort.feature.md) | Manage the observable repository review effort choice without enabling reviews. |
| [Copilot custom instructions](copilot-custom-instructions.feature.md) | Manage the stored repository Use custom instructions toggle. |
| [Repository variables](repository-variables.feature.md) | Manage selected non-sensitive Actions repository variables. |
| [Collection pruning](prune.feature.md) | Explicit command-selected deletions, presence, safety and staged scope. |
| [Labels](labels.feature.md) | Manage selected label objects by ID. |
| [Rulesets](rulesets.feature.md) | Manage selected repository rulesets. |
| [Environments](environments.feature.md) | Manage supported named environment policies and explicitly selected non-sensitive variables. |
| [GitHub Actions](actions.feature.md) | Operate the public composite Action. |
| [Actions event policies](actions-event-policies.feature.md) | Manage listed repository-owned event policies and workflow targets without implicit deletion. |
| [Actions execution](actions-execution-policy.feature.md) | Manage repository Actions policy. |
| [Workflow permissions](workflow-permissions.feature.md) | Manage default Actions token and approval policy. |
| [Fork workflows](fork-workflows.feature.md) | Manage fork pull-request workflow controls. |
| [Security](security.feature.md) | Manage adopted security controls. |
| [Immutable releases](immutable-releases.feature.md) | Manage repository release immutability policy and its explicit 404 tradeoff. |
| [Retention](retention.feature.md) | Manage Actions retention policy. |
| [Cache and sharing access](actions-general-controls.feature.md) | Manage independent repository cache limits and private Actions/reusable workflow sharing. |
| [Pages](pages.feature.md) | Manage Pages creation, configuration and explicit deletion. |
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
| [Deploy keys](deploy-keys.feature.md) | Manage public-key creation, immutable ID binding and explicit guarded deletion. |
| [Repository access](repository-access.feature.md) | Manage personal collaborator identities and pending invitations without resending matching requests. |

Update affected scenarios, tests, and the authoritative user reference together
when current behavior changes. Use [testing](../contributing/testing.md) for
validation boundaries and [ADRs](../adr/README.md) for decision rationale.

- [Classic branch protection](classic-protection.feature.md): exact patterns, independent controls including branch locking, partial preservation and verified identity.
- [Autolink references](autolinks.feature.md): immutable ID binding, verified creation, explicit pruning and private plans.
