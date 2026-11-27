# Feature: Manage public repository search during Copilot code review

The **General** section of **Settings → Copilot → Code review** on a personally
owned public repository shows **Search public repositories** between the MCP
tools and custom instructions controls. See the
[configuration reference](../../references/settings/copilot.md#search-public-repositories)
and [official GET/PATCH contract](https://docs.github.com/en/rest/copilot/copilot-code-review#update-copilot-code-review-settings-for-a-repository).

## Scenario: Export readable preferences in General screen order

- Given all four General preferences are readable, including disabled search
- When the CLI exports code review settings
- Then the General mapping lists review effort, MCP tools, public search, and custom instructions in that order
- And search is explicitly false

## Scenario: Search availability is unknown

- Given the repository GET provides no search boolean
- When export runs
- Then it reports the unavailable field and preserves any prior search declaration
- And a plan or apply selecting search fails before its PATCH

## Scenario: Change only public repository search

- Given a selected search preference differs from its observed value
- When apply runs
- Then it sends only `cross_repo_context_enabled` and independently reads the retained setting and its companions
- And repeated apply with the same stored preference performs no PATCH

Archived matching reads each selected toggle twice. Denied, missing, ambiguous,
or unequal selected evidence prevents a successful read-only apply.
