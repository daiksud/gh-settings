---
type: Feature
title: Validate repository settings
description: Specify local validation of supported YAML settings and its Go test mapping.
---

## Feature: Validate repository settings locally

`validate` reads `.github/settings.yml` from the current Git checkout's root, including when invoked from a subdirectory. The document requires the string marker `version: daiksud/gh-settings@v0` and groups supported intent beneath `repository.about`, `repository.labels`, and `repository.settings`. It does not require a GitHub origin or authentication, make API requests, change the file, or write repository settings. Success has exit status zero and no output; failure has a nonzero status, a diagnostic on standard error, and no standard output.

### Rule: Use the UI-oriented versioned repository document

#### Scenario: Preserve About intent in the new envelope

- Given the document uses `version: daiksud/gh-settings@v0`
- And `repository.about` declares an empty description, a website and an empty topics list
- When the document is read and exported
- Then those explicit values remain present under the same UI field names
- And omitted About fields remain unmanaged

#### Scenario: Reject obsolete envelopes instead of migrating them

- Given a document uses `schemaVersion`, root `settings`, or the obsolete About field `homepage`
- When the maintainer validates the document
- Then validation fails without network access or changing the document

These scenarios map to `TestUIConfigAboutRoundTrip`, `TestUIConfigAboutOmission`, and `TestUIConfigRejectsObsoleteEnvelopeAndInvalidAbout` in the [UI codec tests](../../internal/workspace/ui_config_test.go).

#### Scenario: Preserve General controls and selected menu wording

- Given `repository.settings.general` contains supported nested controls with explicit false values and snake_case menu choices
- When the document is read and exported
- Then the selected values retain their UI names and types
- And omitted controls do not become desired defaults
- And API-shaped keys, unknown controls, duplicate keys and incorrectly typed values are rejected

#### Scenario: Reject transport choices and invalid dependent controls

- Given a General commit-message menu contains the API token `PR_TITLE` instead of a supported UI choice
- Or Issues is explicitly disabled together with an explicit creation-permission choice
- When the maintainer validates the document
- Then validation fails before API access
- And no partial declaration is used

Test mapping: the [General UI codec tests](../../internal/workspace/ui_general_test.go).

The [UI declaration composition tests](../../internal/github/ui_declaration_test.go) check independent expected adapter values and local domain validation.

#### Scenario: Preserve the Labels collection and selected identity

- Given `repository.labels` contains selected numeric IDs and create-only entries without IDs
- When the document is read and exported
- Then names, quoted `#rrggbb` colors and explicit empty descriptions remain present
- And omitted IDs and descriptions remain omitted
- And an omitted collection remains distinct from an explicit empty list
- And unknown keys, nulls, non-string label values and incorrectly typed or overflowing IDs are rejected

Test mapping: the [Labels UI codec tests](../../internal/workspace/ui_labels_test.go).

Colors require the UI `#rrggbb` form; selected IDs and names retain their existing whole-object ownership rules. Invalid colors or duplicate selected IDs fail locally without using a partial declaration. The [UI declaration composition tests](../../internal/github/ui_declaration_test.go) cover this adapter boundary independently of YAML syntax.

#### Scenario: Preserve Ruleset controls and selected identity

- Given `repository.settings.rulesets` declares selected IDs or create-only entries using `ruleset_name`
- And branch or tag targets use UI include controls and short name patterns
- When the document is read and exported
- Then enforcement, target choices, simple rule checkboxes and explicit false values retain their UI names and types
- And omitted and empty collections remain distinct
- And obsolete API-shaped keys and incorrectly typed controls are rejected

Test mapping: the [Ruleset UI codec tests](../../internal/workspace/ui_rulesets_test.go).

Ruleset PR controls preserve explicit zero approvals, false checkboxes, and selected merge methods. Typed actor collections retain selected IDs, and team-review file patterns retain ordering, repetition and negation. Status checks distinguish Any source from a selected App. Code scanning and code quality use UI choice names. Wrong types, null collections and transport-shaped nested keys fail locally. The [Ruleset request tests](../../internal/github/ui_config_rulesets_test.go) independently check new YAML against exact create/update HTTP payloads and selected-ID URLs.

#### Scenario: Preserve Actions General controls

- Given `repository.settings.actions.general` declares UI permission choices and nested fork workflow controls
- When the document is read and exported
- Then explicit false values, ordered action patterns and omitted controls remain distinct
- And UI control names use snake_case, including `github` and `sha`
- And API-shaped aliases, null controls and incorrectly typed values are rejected

Test mapping: the [Actions UI codec tests](../../internal/workspace/ui_actions_test.go).

#### Scenario: Keep Security controls at their UI locations

- Given Secret Protection, Push protection and Private vulnerability reporting are declared under `repository.settings.security_and_quality.advanced_security`
- And release immutability is declared under `repository.settings.general.releases`
- When the document is read and exported
- Then each explicit boolean stays at its UI location and omitted controls remain unmanaged
- And API aliases, the internal `general_releases` container and owner enforcement metadata are rejected

Test mapping: the [Security UI codec tests](../../internal/workspace/ui_security_test.go).

#### Scenario: Preserve Pages controls and domain removal

- Given `repository.settings.pages` declares Build and deployment choices, a custom domain and Enforce HTTPS
- When the document is read and exported
- Then source choices, branch, folder and explicit false values retain their UI names and types
- And an omitted custom domain remains unmanaged while `custom_domain: null` selects removal
- And `custom_domain` is a scalar, without an internal `value` member
- And API aliases, owner metadata and nulls in other controls are rejected

Test mapping: the [Pages UI codec tests](../../internal/workspace/ui_pages_test.go).

#### Scenario: Preserve Environment names and separately owned patterns

- Given `repository.settings.environments` contains name-addressed environments with deployment protection controls
- And deployment branches and tags use UI choices and typed branch/tag patterns
- When the document is read and exported
- Then zero timers, false flags, selected person/team IDs and explicit empty reviewer lists remain present
- And omitted and empty root collections remain distinct
- And numeric environment IDs, API-shaped aliases, null collections and incorrectly typed values are rejected

Test mapping: the [Environment UI codec tests](../../internal/workspace/ui_environments_test.go).

### Rule: Accept only supported fields with the update API's value types

#### Scenario: Validate all supported settings

- Given `.github/settings.yml` declares `schemaVersion: 1` and supported fields under `settings.repository` with their required strings, booleans and validated enums
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

- Given `.github/settings.yml` contains a null key, a duplicate alias key, an alias of a merge key, a binary-tagged description, or a legacy boolean spelling
- When the maintainer runs `gh settings validate`
- Then validation fails without a GitHub API request

#### Scenario: A mapping or merge source has a contradictory YAML tag

- Given a root, `settings`, `repository`, or merge-source mapping is explicitly tagged as a string
- When the maintainer runs `gh settings validate`
- Then validation fails because the value is not a YAML mapping

#### Scenario: The configuration file is missing

- Given the current Git checkout has no `.github/settings.yml`
- When the maintainer runs `gh settings validate`
- Then validation fails with a file-read diagnostic

### Test mapping

The scenarios map to ordinary Go tests and [process acceptance cases](../../tests/acceptance/scenarios_validate.go). The process cases launch the already-built executable and verify that validation makes no request to the local API fixture. Detailed YAML parser cases remain in `TestParseConfigValidation` in the [workspace codec tests](../../internal/workspace/config_test.go). File-reading outcomes have [workspace file tests](../../internal/workspace/config_file_test.go). The [application tests](../../internal/app/validate_test.go) check injected declarations and loading errors without Git or files. The [checkout tests](../../internal/workspace/checkout_test.go) cover explicit directories, nested/trailing-space roots, unchanged bytes, and Git lookup errors. The process cases retain the CLI, file, Git, output, and request-observation checks.

| Scenario | Go test | Process acceptance case |
| --- | --- | --- |
| Validate all supported settings | `TestParseConfigValues/all_fields` | `validate-all-fields` |
| Preserve partial declarations and zero values | `TestParseConfigValues/partial_zero_values`, `TestCheckoutDeclaration` | `validate-partial-zero-values` |
| Resolve an alias or merge key | `TestParseConfigValidation/alias_value`, `TestParseConfigValidation/merge_key` | `validate-alias`, `validate-merge` |
| An unknown key is declared | `TestParseConfigValidation/unknown_field` | `validate-unknown-key` |
| A key is declared twice | `TestParseConfigValidation/duplicate_field` | `validate-duplicate-key` |
| A value has the wrong type | `TestParseConfigValidation/string_instead_of_boolean` | `validate-wrong-type` |
| Explicit null is not a typed value | `TestParseConfigValidation/null_instead_of_string`, `TestParseConfigValidation/null_instead_of_boolean` | `validate-null-boolean` |
| Quoted null is a string | `TestParseConfigValidation/quoted_null_is_a_string` | — |
| YAML is malformed or has more than one document | `TestParseConfigValidation/malformed_YAML`, `TestParseConfigValidation/two_documents` | `validate-malformed` |
| Old flat configuration is rejected | `TestParseConfigValidation/old_flat_format` | — |
| The schema version is missing or unsupported | `TestParseConfigValidation/missing_schema_version`, `TestParseConfigValidation/unsupported_schema_version` | `validate-missing-schema-version`, `validate-unsupported-schema-version` |
| Invalid YAML key and scalar tags are rejected | `TestParseConfigValidation/null_key`, `TestParseConfigValidation/duplicate_alias_key`, `TestParseConfigValidation/alias_merge_key`, `TestParseConfigValidation/binary_instead_of_string`, `TestParseConfigValidation/legacy_boolean_yes` | `validate-legacy-tagged-boolean` |
| A mapping or merge source has a contradictory YAML tag | `TestParseConfigValidation/tagged_settings_mapping_as_string`, `TestParseConfigValidation/mistagged_merge_mapping` | `validate-mistagged-mapping` |
| The configuration file is missing | `TestReadConfigErrors/missing_file` | `validate-missing-file` |

The Go table also checks a versioned document with no settings, empty `settings` and `repository` mappings, wrong mapping shapes, nested values, non-string keys, an unsupported bare `repository` wrapper, invalid version types, and a quoted `"null"` string. Alias and merge values are checked after YAML v4 resolves them; duplicate explicit keys remain errors. Explicit null rejection follows [ADR 0019](../adr/0019-reject-explicit-null.md) for the original five update-request fields; the same non-null rule applies to later fields, including Discussions. It does not establish how GitHub handles live null updates.

### Rule: Distinguish omitted topics from explicit empty names

- Given topics is omitted or an empty mapping.
- Then topics remains unmanaged.
- Given names is an explicit empty string array.
- Then validation accepts an authoritative empty set.
- Given names is null, not a string array, duplicated or outside the documented naming limits.
- Then validation fails before API access.

Test mapping: [TestTopicsConfigPresenceAndRoundTrip and TestTopicsConfigRejectsInvalidLists](../../internal/workspace/topics_test.go), [TestTopicNames](../../internal/settings/topics_test.go).

### Rule: Validate label declarations

Current unreleased source validates listed whole label objects with positive optional IDs, required names/colors and optional descriptions. Empty selections manage nothing; duplicate IDs/names, null and unknown fields fail before API access. The [Labels specification](labels.feature.md#rule-select-objects-never-delete-unlisted-labels) owns the contract.

### Rule: Validate the optional Discussions field

Omission remains unmanaged; false remains declared. Null, quoted booleans, numbers, collections and legacy boolean spellings fail before origin/authentication/API access. The [Discussions specification](discussions.feature.md) owns its behavior and test boundaries.

Test mapping: [TestDiscussionsDeclaration](../../internal/workspace/discussions_test.go) and [TestApplyAcceptance](../../tests/acceptance/apply_test.go).

### Rule: Actions execution policy validation

Current source accepts only the strict optional `actions` mapping. The [Actions policy specification](actions-execution-policy.feature.md#rule-validate-only-explicit-native-values) owns omission/false/empty/null semantics and zero-network failure cases.

Validation accepts twenty-four repository fields, ID-selected whole-object Rulesets, fourteen Actions policy fields, three adopted security booleans, five Pages fields. Omission remains unmanaged; false and supported empty collections remain explicit; Pages cname null is the endpoint-specific clear operation; other nulls follow their resource contracts. Unknown fields/types/enums fail locally before origin/authentication/API access. Current-state prerequisites, entitlement/applicability and upper-policy checks belong to read/compare/apply, not local validation. Pages lifecycle/visibility, organization-only settings and other gated candidates are not accepted.

| Subset | Existing owning tests and specification |
| --- | --- |
| Branch/template/sign-off | [Default branch](default-branch.feature.md), [template](template-repository.feature.md), [sign-off](web-commit-signoff.feature.md) |
| PR and Issue policies | [PR codec](../../internal/workspace/pr_access_test.go), [Issue codec](../../internal/workspace/issue_creation_policy_test.go) |
| Rulesets and nested identities | [Ruleset codec](../../internal/workspace/rulesets_test.go), [Ruleset contract](rulesets.feature.md) |
| Workflow permissions and retention | [Workflow codec](../../internal/workspace/workflow_permissions_test.go), [retention codec](../../internal/workspace/retention_test.go) |
| Secret scanning and public reporting | [Security codec](../../internal/workspace/security_test.go), [security contract](security.feature.md) |
| Fork workflow policies | [Fork codec](../../internal/workspace/fork_workflows_test.go), [fork contract](fork-workflows.feature.md) |
| Archive and Sponsor | [Archive codec](../../internal/workspace/archive_test.go), [Sponsor codec](../../internal/workspace/sponsorships_test.go) |
| Existing-site Pages and removed settings | [Pages codec](../../internal/workspace/pages_test.go), [removed settings](../../internal/workspace/removed_settings_test.go) |

Dedicated process cases exercise the same supplied CLI through these interfaces; detailed type matrices remain with the codec/model owners. Validation success does not establish remote policy, grants, applicability or published availability.
