# Documentation

## For users

If this is your first checkout, [install the CLI](../README.md#install) and [manage a repository description](tutorials.md). For an existing checkout or workflow, choose the task below.

| Task | Read |
| --- | --- |
| Understand how the checkout, declaration, and remote state relate | [How gh-settings manages settings](concepts.md) |
| Look up a command, flag, exit status, or failure | [CLI reference](references/commands.md) |
| Find a YAML field by its GitHub Settings page | [Settings reference](references/settings/README.md) |
| Update one label while preserving other labels | [Manage labels by ID](guides/manage-labels.md) |
| Check what happens to omitted or unlisted resources | [Value semantics](references/settings/configuration.md#configuration-semantics) · [Collection pruning](references/settings/pruning.md#collection-pruning) |
| Find the GitHub App permissions a command needs | [Permission matrix](references/settings/permissions.md) |
| Set up pull request plans and protected applies on main | [Workflow setup](guides/github-actions.md) · [Protect the apply path](guides/protect-apply.md) |
| Look up Action inputs, outputs, or which CLI a pin runs | [Action reference](references/action.md) · [Package selection](guides/github-actions.md#published-package-and-consumer-pins) |
| Check the supported repository types and v1 scope | [Support policy](references/settings/scope.md#feature-adoption-scope) |

## For contributors

Start with [CONTRIBUTING](../CONTRIBUTING.md) to propose a change. For implementation and maintenance work, use the following guides.

| Task | Read |
| --- | --- |
| Set up the tools, build the CLI, and run checks | [Local development](development/setup.md) · [Testing and CI](development/testing.md) |
| Find the code that owns a change | [Architecture](development/architecture.md) · [Model glossary](development/glossary.md) |
| Check the constraints for a design or documentation change | [Development policies](development/policies.md) · [Architecture decision records](development/adr/README.md) |
| Find behavior scenarios and the tests that cover them | [Feature specifications](development/features/README.md) |
| Assess a setting's UI availability, API support, and implementation | [Settings coverage and evidence](development/settings-coverage.md) |
| Change the internal Action reports or job summary | [Actions internals](development/actions-internals.md) |
| Review a PR or troubleshoot a required check | [Code-review policy](../.agents/skills/code-review/SKILL.md) · [PR validation](development/pr-validation.md) |
| Apply this repository's own settings | [Repository settings changes](development/repository-automation.md#repository-settings-changes) |
| Publish or recover a release | [Release procedure](development/releases.md) |

The UI mappings record the evidence for individual settings pages. See [Repository and Labels](development/ui-repository-mapping.md), [Rulesets](development/ui-ruleset-mapping.md), [Actions](development/ui-actions-mapping.md), [Security](development/ui-security-mapping.md), [Pages](development/ui-pages-mapping.md), or [Environments](development/ui-environments-mapping.md).
