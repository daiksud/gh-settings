---
type: Feature
title: Validate repository settings
description: Specify local validation of the five supported YAML settings and its Go test mapping.
---

## Feature: Validate repository settings locally

`validate` reads `.github/settings.yml` from the current Git checkout's root, including when invoked from a subdirectory. The document requires root integer `schemaVersion: 1`; `settings.repository` is optional. It does not require a GitHub origin or authentication, make API requests, change the file, or write repository settings. Success has exit status zero and no output; failure has a nonzero status, a diagnostic on standard error, and no standard output.

### Rule: Accept only supported fields with the update API's value types

#### Scenario: Validate all supported settings

- Given `.github/settings.yml` declares `schemaVersion: 1` and the five supported fields under `settings.repository` as strings and booleans
- When the maintainer runs `gh settings validate`
- Then validation succeeds without output
- And no GitHub API request is made

#### Scenario: Preserve partial declarations and zero values

- Given `.github/settings.yml` declares only `settings.repository.description: ''` and `settings.repository.has_issues: false`
- When the maintainer runs `gh settings validate`
- Then validation succeeds without adding omitted fields or changing either declared value

#### Scenario: Resolve an alias or merge key

- Given `.github/settings.yml` uses an alias or merge key within `settings.repository` that resolves to supported keys and values
- When the maintainer runs `gh settings validate`
- Then validation succeeds using the resolved values

### Rule: Reject unsupported keys, values, and document shapes

#### Scenario: An unknown key is declared

- Given `.github/settings.yml` declares `settings.repository.visibility: private`
- When the maintainer runs `gh settings validate`
- Then validation fails with an unknown-setting diagnostic

#### Scenario: A key is declared twice

- Given `.github/settings.yml` declares `settings.repository.has_issues` twice
- When the maintainer runs `gh settings validate`
- Then validation fails with a duplicate-setting diagnostic

#### Scenario: A value has the wrong type

- Given `.github/settings.yml` declares `settings.repository.has_issues: 'false'`
- When the maintainer runs `gh settings validate`
- Then validation fails because the value is a string rather than a boolean

#### Scenario: Explicit null is not a typed value

- Given `.github/settings.yml` declares `settings.repository.description: null` or `settings.repository.has_issues: null`
- When the maintainer runs `gh settings validate`
- Then validation fails for the field's required string or boolean type
- And null is not treated as omission or an empty string

#### Scenario: Quoted null is a string

- Given `.github/settings.yml` declares `settings.repository.description: 'null'`
- When the maintainer runs `gh settings validate`
- Then validation succeeds because the value is a string

#### Scenario: YAML is malformed or has more than one document

- Given `.github/settings.yml` contains malformed YAML or multiple documents
- When the maintainer runs `gh settings validate`
- Then validation fails with a document diagnostic

#### Scenario: Old flat configuration is rejected

- Given `.github/settings.yml` declares `schemaVersion: 1` and `has_issues: false` at the root
- When the maintainer runs `gh settings validate`
- Then validation fails with an unknown-root-key diagnostic
- And the maintainer must move the field under `settings.repository`

#### Scenario: The schema version is missing or unsupported

- Given `.github/settings.yml` omits `schemaVersion` or declares `schemaVersion: 2`
- When the maintainer runs `gh settings validate`
- Then validation fails instead of guessing a file format

#### Scenario: Invalid YAML key and scalar tags are rejected

- Given `.github/settings.yml` contains a null key, a duplicate alias key, or a binary-tagged description
- When the maintainer runs `gh settings validate`
- Then validation fails without a GitHub API request

#### Scenario: The configuration file is missing

- Given the current Git checkout has no `.github/settings.yml`
- When the maintainer runs `gh settings validate`
- Then validation fails with a file-read diagnostic

### Test mapping

The scenarios map to ordinary Go tests and the [process acceptance harness](../../tests/acceptance/main.go). The process cases launch the already-built executable and verify that validation makes no request to the local API fixture.

| Scenario | Go test | Process acceptance case |
| --- | --- | --- |
| Validate all supported settings | `TestValidateSettings` | `validate-all-fields` |
| Preserve partial declarations and zero values | `TestValidateSettings`, `TestValidateCommandReadsOnlySettingsFile` | `validate-partial-zero-values` |
| Resolve an alias or merge key | `TestValidateSettings` | `validate-alias`, `validate-merge` |
| An unknown key is declared | `TestValidateSettings` | `validate-unknown-key` |
| A key is declared twice | `TestValidateSettings` | `validate-duplicate-key` |
| A value has the wrong type | `TestValidateSettings` | `validate-wrong-type` |
| Explicit null is not a typed value | `TestValidateSettings` | `validate-null-string`, `validate-null-boolean` |
| Quoted null is a string | `TestValidateSettings` | `validate-quoted-null-string` |
| YAML is malformed or has more than one document | `TestValidateSettings` | `validate-malformed`, `validate-multiple-documents` |
| Old flat configuration is rejected | `TestValidateSettings` | `validate-old-flat-format` |
| The schema version is missing or unsupported | `TestValidateSettings` | `validate-missing-schema-version`, `validate-unsupported-schema-version` |
| Invalid YAML key and scalar tags are rejected | `TestValidateSettings` | `validate-null-key`, `validate-alias-duplicate-key`, `validate-binary-string` |
| The configuration file is missing | `TestValidateCommandMissingFile` | `validate-missing-file` |

The Go table also checks a versioned document with no settings, empty `settings` and `repository` mappings, wrong mapping shapes, nested values, non-string keys, an unsupported bare `repository` wrapper, invalid version types, and a quoted `"null"` string. Alias and merge values are checked after YAML v4 resolves them; duplicate explicit keys remain errors. Explicit null rejection follows [ADR 0019](../adr/0019-reject-explicit-null.md) for these five update-request fields. It does not establish how GitHub handles live null updates.
