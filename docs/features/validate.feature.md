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

### Rule: Validate typed UI intent without remote work

#### Scenario: Preserve partial declarations and zero values

- Given the exact current version marker
- And `repository.about.description: ''` and `repository.settings.general.features.issues: false`
- When the maintainer runs `gh settings validate`
- Then validation succeeds silently without adding omitted values or changing the file
- And no origin, authentication or API request is needed

#### Scenario: Resolve safe aliases, merges and string styles

- Given aliases or merge keys resolve to supported UI fields and typed values
- When the document is validated and encoded
- Then resolved values preserve explicit false, empty strings and ordered collections
- And quoted `null` and quoted boolean-looking About text remain strings

#### Scenario: Reject unsafe document shapes without partial intent

- Given an unknown, duplicate or non-string key, unsupported tag, contradictory mapping tag, wrong type, malformed YAML or multiple documents
- When the maintainer validates the document
- Then validation fails with the specific local cause and no API request or file change
- And no partly decoded declaration is used

#### Scenario: Distinguish null from omission

- Given About description or a General boolean is explicitly null
- When validation runs
- Then it fails rather than treating null as omission, false or an empty string
- But Pages `custom_domain: null` remains the explicit scalar removal operation

#### Scenario: Require the exact version string

- Given the version is missing, a different string or a non-string value
- When validation runs
- Then it fails instead of guessing a file format
- And the old envelope and root settings alias are not migrated

#### Scenario: Empty documents are local no-ops

- Given a version-only document, `repository: {}` or `repository: {settings: {}}`
- When validate, plan or apply runs
- Then no origin, authentication or API access is needed
- And plan/apply report no managed settings

#### Scenario: A missing file is a file-read failure

- Given the checkout has no `.github/settings.yml`
- When validation runs
- Then it fails before any remote work

### Rule: Preserve each resource's ownership and collection semantics

- About Topics omission is unmanaged; `topics: []` is an authoritative empty set and `topics: {}` is invalid.
- Labels and Rulesets select only listed objects; empty root lists select nothing.
- Ruleset omitted rule groups follow whole-object defaults, not remote retention.
- Environment omitted built-ins remain API-controlled without invented defaults or numeric-ID writeback.
- Actions UI radios select explicit execution state/policy; omitted radios never enable implicitly.
- Unknown controls such as organization custom properties and General forking policy fail at their current paths.
- Current-state prerequisites, entitlement, caps and upper-policy checks belong to observation/apply, not local validation.

### Test mapping

The supplied-binary [validation scenarios](../../tests/acceptance/scenarios_validate.go) cover current literals, exact local causes, aliases/merges, types, marker/obsolete envelopes, malformed/tagged mappings, missing files and three no-op forms. They launch the already-built executable, preserve source bytes and observe zero fixture API requests.

| Boundary | Current owning tests |
| --- | --- |
| About, marker, obsolete aliases and organization-only controls | [UI config](../../internal/workspace/ui_config_test.go) |
| Safe aliases/merges/styles and unsafe zero-result decoding | [YAML safety](../../internal/workspace/ui_yaml_safety_test.go) |
| General choices and types | [General UI codec](../../internal/workspace/ui_general_test.go), [General API conversion](../../internal/github/ui_general_test.go) |
| Label identity/color/omission | [Labels UI codec](../../internal/workspace/ui_labels_test.go), [composition](../../internal/github/ui_declaration_test.go) |
| Ruleset nested controls/identities | [Ruleset UI codec](../../internal/workspace/ui_rulesets_test.go), [independent HTTP requests](../../internal/github/ui_config_rulesets_test.go) |
| Actions, Security, Pages, Environment types | [Actions](../../internal/workspace/ui_actions_test.go), [Security](../../internal/workspace/ui_security_test.go), [Pages](../../internal/workspace/ui_pages_test.go), [Environments](../../internal/workspace/ui_environments_test.go) |
| No partial semantic intent and snapshot composition | [Root composition](../../checkout_config_test.go) |
| File, checkout and injected application failures | [Files](../../internal/workspace/config_file_test.go), [Checkout](../../internal/workspace/checkout_test.go), [application](../../internal/app/validate_test.go) |
| Resource semantics beyond YAML types | [Topics model](../../internal/settings/topics_test.go) and each linked family specification |

`TestUIConfigYAMLUnsafeShapesReturnNoPartialValues` owns the parser safety matrix. `TestCheckoutUICompositionNoOpDocuments` and `TestCheckoutUIInvalidDomainReturnsNoPartialIntent` cover root wiring. There is one current public codec; no retained legacy parser is an alternate contract. Validation success does not establish live grants, effective protection, applicability or published availability.
