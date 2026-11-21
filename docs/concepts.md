# How gh-settings manages a repository

The CLI reads the declaration in `.github/settings.yml` at the checkout root.
The checkout's directly configured GitHub.com `origin` identifies the repository
to read or update. The [command reference](references/commands.md#repository-and-authentication)
describes the accepted origins and credentials.

The public Action runs a CLI release selected by the workflow's Action pin. To
share a declaration between your terminal and a workflow, use settings supported
by both CLI versions. See [published packages and pins](guides/github-actions.md#published-package-and-consumer-pins).

## What the declaration owns

Each field you declare selects something to manage. For example, a declaration
containing only `repository.about.description` can update the repository's
description while leaving its website and topics alone. Setting the description
to an empty string explicitly clears it. When you omit the field, gh-settings leaves the current description alone.

Collections have their own ownership rules. Declaring a topic list selects the
entire set of topics. Declaring a label by ID selects that label, so other labels
survive an ordinary apply. These rules also determine what an empty list means.

| Declaration | Effect of ordinary apply |
| --- | --- |
| Omit `repository.about.topics` | Leave the repository's topics alone. |
| Set `repository.about.topics: []` | Clear all topics. |
| Set `repository.labels: []` | Leave all labels alone. |

Explicit pruning adds deletions for a named collection. With `--prune labels`,
an empty label list selects every label for deletion, including its assignments
to issues and pull requests. An omitted label list still selects none. Check the
[ownership rules for your setting](references/settings/README.md) and the
[prune scopes](references/settings/pruning.md#collection-pruning) when working with lists.

## From observation to change

**Export** saves the settings gh-settings can read into the checkout. If a
setting is unavailable, export reports it and may retain a previous declaration.
Review the file and the diagnostics before deciding which values to manage.

**Validate** checks the declaration locally. **Plan** reads the managed settings
and shows how they compare with the declaration. A successful plan may show
changes. The preview describes the repository at the time of those reads.

**Apply** reads current state again and acts on the declaration. If an update
fails partway through, earlier writes can remain in GitHub. Check the reported
resource IDs and the repository's current state before retrying. The
[command reference](references/commands.md#recovering-from-a-failed-apply)
explains partial failures and ID recovery.
