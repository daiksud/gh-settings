---
type: Reference
title: "About settings"
description: "About settings declaration, ownership and API contracts."
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
---

# Repository About settings

Use `repository.about` for the description, website and topics displayed on the repository home page. Each field is optional.

| Field | Value |
| --- | --- |
| `description` | String. An empty string clears the description. |
| `website` | String. An empty string clears the website. |
| `topics` | The complete set of topic names. An empty list clears the set. |

Export represents a missing description or homepage from GitHub as an empty string. An explicit YAML `null` is invalid for these fields.[^github-update-repository][^github-repository-openapi]

## Topics

`repository.about.topics` manages the complete topic set. It is a direct array, not a mapping. Omission leaves it unmanaged; `[]` explicitly clears all topics; `{}` and null are invalid. Names are unique lowercase letters/numbers/hyphens, 1–50 characters each, at most 20 entries. Names are public metadata even on private repositories. Export, comparison and PUT use sorted set order without rewriting existing input YAML. Apply reads complete, validated current membership and skips PUT when the set matches, regardless of order. Changed membership uses one full-set PUT followed by an independent GET; the PUT response alone is not confirmation. A failed prerequisite read makes no Topics write. A rejected update or failed/mismatching readback stops later resources with uncertain effects and no automatic retry or rollback. Metadata read and Administration write must already exist. See [apply and failure recovery](../commands.md#apply).

[^github-update-repository]: Repository update request fields; original inspection 2026-10-11. This does not expand the supported UI inventory.

[^github-repository-openapi]: Fixed GitHub OpenAPI evidence for nullable observed description/homepage; original inspection 2026-10-12.
