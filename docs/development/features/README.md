# Behavior specifications

Use these scenarios and test mappings when changing how the CLI or public Action behaves. The [settings reference](../../references/settings/README.md), [command reference](../../references/commands.md) and [Action reference](../../references/action.md) describe the public contracts. Update the affected reference, scenarios and tests together.

Write each specification in [Markdown with Gherkin](https://github.com/cucumber/gherkin/blob/main/MARKDOWN_WITH_GHERKIN.md). Start the file with a `# Feature:` heading. Group examples under `## Rule:` and give each one a `### Scenario:` heading. Use list items for the starting state (`Given`), action (`When`) and observable result (`Then`). Link each behavior to its implementing tests. The Go suites exercise the CLI and Action against fixtures. The [testing guide](../testing.md) explains which checks to run and what hosted runs verify.

## CLI and public Action

| Specification | Behavior |
| --- | --- |
| [Export](export.feature.md) | Read supported state into a configuration file. |
| [Automatic readable settings](readable-settings-export.feature.md) | Default family acquisition, complete inventories, info and prior-intent retention. |
| [Readable Security settings](readable-security-export.feature.md) | Acquire independent Security families and retain prior intent when a family is unreadable. |
| [Validate](validate.feature.md) | Validate declarations locally. |
| [Plan](plan.feature.md) | Compare declared values without writes. |
| [Apply](apply.feature.md) | Apply declarations and report partial failure. |
| [Collection pruning](prune.feature.md) | Explicit command-selected deletions, presence, safety and staged scope. |
| [GitHub Actions](actions.feature.md) | Operate the public composite Action. |

## Repository and pull request settings

| Specification | Behavior |
| --- | --- |
| [Default branch](default-branch.feature.md) | Manage the repository default branch. |
| [Repository lifecycle](repository-lifecycle.feature.md) | Archive and unarchive repositories. |
| [Template repository](template-repository.feature.md) | Manage template availability. |
| [Discussions](discussions.feature.md) | Manage Discussions availability. |
| [Sponsorships](sponsorships.feature.md) | Manage Sponsor button visibility. |
| [Web signoff](web-commit-signoff.feature.md) | Manage web commit sign-off requirement. |
| [Immutable releases](immutable-releases.feature.md) | Manage repository release immutability policy and its explicit 404 tradeoff. |
| [Labels](labels.feature.md) | Manage selected label objects by ID. |
| [Merge methods](merge-methods.feature.md) | Manage allowed merge methods. |
| [Merge commit defaults](merge-commit-defaults.feature.md) | Manage normal merge message defaults. |
| [Squash commit defaults](squash-commit-defaults.feature.md) | Manage squash commit title and body defaults. |
| [Automatic branch deletion](delete-branch-on-merge.feature.md) | Manage head-branch deletion after merge. |
| [Auto merge / branch update](pull-request-options.feature.md) | Manage optional PR merge behavior. |

## Access, moderation and branch rules

| Specification | Behavior |
| --- | --- |
| [Repository access](repository-access.feature.md) | Manage personal collaborator identities and pending invitations without resending matching requests. |
| [Pull request access](pull-request-access.feature.md) | Manage pull-request availability and creation policy. |
| [Persistent pull request limits](pull-request-limits.feature.md) | Manage the open pull request cap and the declared exemption list. |
| [Issue creation](issue-creation-policy.feature.md) | Manage issue creation policy. |
| [Rulesets](rulesets.feature.md) | Manage selected repository rulesets. |
| [Classic branch protection](classic-protection.feature.md) | Maintain existing v0 classic rules by exact pattern and independently owned control. |

## Actions, workflows and variables

| Specification | Behavior |
| --- | --- |
| [Actions execution](actions-execution-policy.feature.md) | Manage repository Actions policy. |
| [Actions event policies](actions-event-policies.feature.md) | Manage listed repository-owned event policies and workflow targets without implicit deletion. |
| [Actions cache limits](actions-general-controls.feature.md) | Manage repository cache retention and the size eviction limit independently. |
| [Workflow permissions](workflow-permissions.feature.md) | Manage default Actions token and approval policy. |
| [Fork workflows](fork-workflows.feature.md) | Set approval requirements for workflows from public fork contributors. |
| [Retention](retention.feature.md) | Manage Actions retention policy. |
| [Actions OIDC subject template](oidc-subject.feature.md) | Manage the complete active template while preserving known immutable mode. |
| [Existing runner custom labels](runners.feature.md) | Reconcile custom labels for selected existing runner IDs while preserving readonly labels. |
| [Repository variables](repository-variables.feature.md) | Manage selected non-sensitive Actions and Agents repository variables. |

## Integrations, deployment and security

| Specification | Behavior |
| --- | --- |
| [Webhook configuration and creation](webhooks.feature.md) | Reconcile selected config, guard parent state and create hooks with verified ID recording. |
| [Webhook pruning](webhook-prune.feature.md) | Explicitly delete unlisted whole hooks with private evidence and confirmed absence. |
| [Autolink references](autolinks.feature.md) | Maintain the existing v0 Autolink contract for creation, ID binding and explicit pruning. |
| [Deploy keys](deploy-keys.feature.md) | Manage public-key creation, immutable ID binding and explicit guarded deletion. |
| [Copilot custom instructions](copilot-custom-instructions.feature.md) | Manage the stored repository Use custom instructions toggle. |
| [Copilot review effort](copilot-review-effort.feature.md) | Manage the observable repository review effort choice without enabling reviews. |
| [Copilot public repository search](copilot-public-search.feature.md) | Select whether reviews can use public repository search for context. |
| [Copilot MCP tools](copilot-mcp.feature.md) | Manage the stored MCP tools permission toggle without configuring servers or executing reviews. |
| [Environments](environments.feature.md) | Manage supported named environment policies and explicitly selected non-sensitive variables. |
| [Pages](pages.feature.md) | Manage Pages creation, configuration and explicit deletion. |
| [Security](security.feature.md) | Manage adopted security controls. |
