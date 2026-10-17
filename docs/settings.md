# Repository settings

## Scope and status

Configuration lives in `.github/settings.yml` at the Git checkout root and manages only that checkout's GitHub.com repository. Supported resources are five repository fields and the repository's topics. See the [command reference](commands.md) for reading, validating, planning and applying this configuration.

## Supported fields

The keys inside `settings.repository` follow GitHub's repository update request body schema.[^github-update-repository]

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The booleans accept `true` or `false`; the string fields accept strings, including empty strings. Explicit null and other value types are invalid. All other repository fields and read-only response metadata are excluded, even when GitHub exposes them as writable parameters.

## Topics

`settings.topics.names` manages the complete repository topic set. Topics have been supported since [v0.8.0](https://github.com/daiksud/gh-settings/releases/tag/v0.8.0), using the same `schemaVersion: 1` format.

```yaml
schemaVersion: 1
settings:
  topics:
    names:
      - github-cli
      - settings-as-code
```

`names` follows the [replace-topics request body](https://docs.github.com/en/rest/repos/repos#replace-all-repository-topics): an array of unique lowercase strings using letters, numbers and hyphens, each 1–50 characters long, with at most 20 entries. Uppercase names, null, scalars, non-string elements and duplicates fail local validation. GitHub's [topic naming rules](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/classifying-your-repository-with-topics) apply to public and private repositories; topic names are public metadata even on private repositories.

| Declaration | Meaning |
| --- | --- |
| Omitted `topics`, or `topics: {}` | Topics unmanaged; plan and apply skip that resource. |
| `names: [github-cli]` | Manage the complete topic set, replacing any other remote topics. |
| `names: []` | Explicitly clear all topics. |
| `names: null` | Invalid; never interpreted as omission or clearing. |

Names are compared as a set. Export, plan and update requests use sorted lists without rewriting an existing input configuration. An empty observed set is exported as `names: []`, which becomes an explicit managed declaration.

Topic reads require Metadata read; public resources may be read without authentication. Writes require Administration write. The [command reference](commands.md#apply-behavior) explains request ordering and partial failures when repository settings and topics are declared together.

## Configuration semantics

The document requires integer `schemaVersion: 1`, a file-format version independent of the CLI release. The `settings` mapping is optional and can contain optional `repository` and `topics` mappings. Unknown root keys, unsupported versions and unsupported groups fail validation; there is no automatic migration or compatibility parsing.

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. No defaults are injected. For example:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
  topics:
    names:
      - github-cli
      - settings-as-code
```

This declaration manages `has_issues` and the complete topic set. Description, homepage, projects and wiki remain unmanaged. The version and outer YAML mappings are not sent as API fields; the repository update body contains only `{"has_issues": false}` and the topics request contains the declared `names` array.

The supported repository fields distinguish omission from explicit values:

| YAML under `settings.repository` | Meaning |
| --- | --- |
| Field omitted | Unmanaged; apply leaves it out of the request. |
| `has_issues: false` | A declared boolean value. |
| `description: ""` | A declared empty string; single-quoted empty strings are equivalent. |
| `description: null` or `has_issues: null` | Invalid for these string and boolean fields. A bare `description:` is also YAML null and invalid. |

The same type rules apply to `homepage`, `has_projects` and `has_wiki`. Quoted `'null'` is a string value, not YAML null. Null is invalid for the version, mappings and topic list too.

A version-only document manages nothing:

```yaml
schemaVersion: 1
```

Empty `settings`, `repository` or `topics` mappings are also valid. When no fields are managed across either resource, plan and apply succeed locally without an origin, authentication or API requests. An explicit `names: []` still manages topics and therefore performs a comparison or update.

Validation rejects unknown or duplicate keys, non-string keys, wrong types, malformed YAML and multiple documents. YAML aliases and merge keys are resolved before supported values are checked. Partial declarations are accepted; unsupported root-level field names are rejected. See [validate behavior](commands.md#validate-behavior) for output and failure handling.

## Exported values

Export declares every supported observed repository field and the complete topic set. Review the saved file and remove fields you want to leave unmanaged before applying it.

GitHub can return null for an unset description or homepage. Export normalizes those response values to empty strings and writes quoted YAML strings (`""`); it preserves existing empty strings and false booleans. Missing response fields and null booleans are errors. Topic names must be present as a valid array; missing, null, invalid or unreadable names fail export instead of becoming an empty set.

This response normalization does not make explicit YAML null valid input. The published read schema permits null for description/homepage, while the update schema defines optional strings without `nullable`.[^github-repository-openapi] The [export command](commands.md#export-behavior) documents file replacement and write-failure behavior.

## Unsupported settings

Labels, rulesets, Pages and Actions settings are not supported. Organization settings and multi-repository management are outside the tool's scope. Contributor [architecture and future direction](contributing/architecture.md) explains design choices without extending the configuration contract.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.
