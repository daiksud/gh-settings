# Feature: Require sign-off on web commits

`repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits` selects one boolean in the General settings. The setting applies to commits made through the web UI. Cryptographic signatures, DCO checks and existing commits have separate controls. See the [settings reference](../../references/settings/general.md#web-commit-sign-off).

## Rule: Manage only explicitly declared web commit sign-off

### Scenario: Disable an enabled control

- Given web commit sign-off is enabled and the declaration selects `false`
- When the owner runs plan
- Then the CLI and detailed Action Summary show the declared field changing from true to false
- When the owner applies the declaration
- Then GitHub receives the changed boolean and a fresh repository read must confirm it
- And applying again after confirmation sends no update

### Scenario: Leave the control unmanaged

- Given a declaration manages another repository field and omits `repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits`
- When apply changes that field
- Then the request omits `web_commit_signoff_required` and leaves web commit sign-off alone

### Scenario: Reject a quoted boolean

- Given the declaration sets `repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits` to the string `"false"`
- When the owner validates it
- Then validation fails locally before origin resolution or an API call

## Rule: Export only a complete observation

### Scenario: GitHub omits the required boolean

- Given an existing declaration and a repository GET with a missing or invalid `web_commit_signoff_required` value
- When the owner exports settings
- Then export reports the unavailable observation and can retain compatible previous intent for that group
- And independently readable settings still export

A complete observation exports a native boolean, including false. For mixed General plans, the sign-off value follows Discussions and precedes merge settings. Declaring this field alone requires only the REST observation. The [CLI apply contract](../../references/commands.md#apply) covers failed writes, readback errors and earlier confirmed changes.

## API and evidence boundaries

GitHub documents the field in [repository PATCH](https://docs.github.com/en/rest/repos/repos#update-a-repository), requiring Administration write. Reads use the existing repository GET and the existing authentication boundary. No extra permission or mutation probe is introduced.

Local HTTP fixtures and dummy credentials verify the requested transitions and failure behavior. They do not prove a live GitHub write, a particular token's permissions, or published-package support. No real repository setting is changed to test this feature.

## Test mapping

- [Codec](../../../internal/workspace/ui_general_test.go): true/false, omission, invalid types and encode/decode round trip.
- [Model](../../../internal/settings/comparison_test.go) and [field definitions](../../../internal/settings/repository_fields_test.go): managed values, comparisons, field identity and types.
- [REST](../../../internal/github/web_commit_signoff_test.go): complete boolean observations and exact declared-only PATCH for omission, false, true, mixed declarations and forbidden writes.
- [Plan](../../../internal/app/web_commit_signoff_test.go) and [export](../../../internal/app/export_test.go): requested REST field and complete state-to-declaration export.
- [Summary](../../../internal/actions/web_commit_signoff_test.go): native boolean rendering and rejection of invalid types.
- [Process plan](../../../tests/acceptance/plan_test.go), [apply](../../../tests/acceptance/apply_test.go) and [export](../../../tests/acceptance/export_validate_test.go): real executable output/requests, missing reads, validation before remote work, permission failures and saved-file round trip.
