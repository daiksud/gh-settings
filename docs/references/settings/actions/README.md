# GitHub Actions settings

Choose the part of the repository's Actions configuration you want to manage. The table lists paths relative to `repository.settings.actions` in the order `gh settings export` writes them.

| GitHub setting | YAML | What you can declare |
| --- | --- | --- |
| [General](general.md) | `general` | Allowed actions and reusable workflows, token defaults, public-fork approval, workflow retention and cache limits. |
| [Runners](runners.md) | `runners` | Custom labels on existing runners and explicit removal of unlisted registrations. |
| [Policies](policies.md) | `policies` | Workflow targeting, event and actor restrictions for repository-owned policies. |
| [OIDC](oidc.md) | `oidc.subject_claim` | The active subject template and the immutable-subject setting. |
| [Repository variables](variables.md) | `variables` | Named plaintext Actions variables, listed separately from Agents and Environment variables. |

To choose credentials for these resources, see the [permission matrix](../permissions.md#actions). To run the gh-settings Action in a workflow, see the [workflow guide](../../../guides/github-actions.md).
