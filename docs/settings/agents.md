---
type: Reference
title: "Agents variables"
description: "Agents variables declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Agents repository variables

`repository.settings.agents.variables` corresponds to **Secrets and variables → Agents → Variables → Repository variables**. This current-source feature manages only explicitly named, non-sensitive repository Agents variables:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    agents:
      variables:
        - name: PROJECT_CONTEXT
          value: public-documentation
```

The [name/value, omission and upsert rules above](actions.md#actions-repository-variables) also apply to Agents variables: names are canonical uppercase and values are required nonempty strings. Without prune, omission and `[]` select nothing and unlisted variables remain unchanged. `--prune agents-variables` adds unlisted deletions only for a present list; [] selects all, omission stays unmanaged. There is no YAML deletion/rename member. Without prune, plan reads only selected names; apply reads fresh state, conditionally creates or updates, then verifies the exact value. Failed reads do not become empty state, and failed mutations or verification can have unknown effects without rollback.

Ordinary `export` lists all supported Agents variables through its separate paginated inventory API. An unavailable/incomplete/unsupported inventory emits info and retains prior intent; it never supplies an empty fallback. The same canonical name can exist with different values in Actions and Agents. Values remain **plaintext** in configuration, exports and detailed output; secrets are outside scope.

The [Agents Variables REST API](https://docs.github.com/en/rest/agents/variables) uses `/repos/{owner}/{repo}/agents/variables` and requires the repository **Agent variables** permission at read level for acquisition and write level for mutation with fine-grained/App tokens. This is separate from the **Variables** permission for Actions variables. No permission or token request is added automatically. The Free/personal/public UI and API persistence were verified separately; the probe does not prove actual agent consumption or access for an untested credential. See the [behavior specification](../features/repository-variables.feature.md) for evidence and its limits.
