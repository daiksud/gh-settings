# GitHub configuration and automation

This directory configures how contributors work on gh-settings and how this repository
tests, releases and applies its own settings.

| When changing | Start here |
| --- | --- |
| The settings gh-settings manages on its own repository | [Settings declaration](settings.yml) and [settings change procedure](../docs/development/repository-automation.md#repository-settings-changes) |
| When a job runs and what permissions it receives | [Workflows](workflows/README.md) |
| Steps shared by CI and release jobs | [Local Actions](actions/README.md) and their [helper scripts](scripts/README.md) |
| What contributors submit and who reviews changes | [Issue forms](ISSUE_TEMPLATE/), [PR template](pull_request_template.md) and [code owners](CODEOWNERS) |
| How files map to PR labels or how dependencies receive updates | [Labeler rules](labeler.yml) and [Dependabot configuration](dependabot.yml) |

The Action that other repositories call is defined in the root [action.yml](../action.yml).
