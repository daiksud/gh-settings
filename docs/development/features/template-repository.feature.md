# Feature: Enable or disable a template repository

`repository.settings.general.template_repository` selects one boolean in the General settings. Creating repositories from a template and managing Git LFS are separate operations. See the [settings reference](../../references/settings/general.md#template-repository).

## Rule: Manage only explicitly declared template status

### Scenario: Disable an enabled control

- Given template status is enabled and the declaration selects `false`
- When the owner runs plan
- Then the CLI and detailed Action Summary show the declared field changing from true to false
- When the owner applies the declaration
- Then GitHub receives the changed boolean and a fresh repository read must confirm it
- And applying again after confirmation sends no update

### Scenario: Leave the control unmanaged

- Given a declaration manages another repository field and omits `repository.settings.general.template_repository`
- When apply changes that field
- Then the request omits `is_template` and leaves template status alone

### Scenario: Reject a quoted boolean

- Given the declaration sets `repository.settings.general.template_repository` to the string `"false"`
- When the owner validates it
- Then validation fails locally before origin resolution or an API call

## Rule: Export only a complete observation

### Scenario: GitHub omits the required boolean

- Given an existing declaration and a repository GET with a missing or invalid `is_template` value
- When the owner exports settings
- Then export reports the unavailable observation and can retain compatible previous intent for that group
- And independently readable settings still export

A complete observation exports a native boolean, including false. For mixed General plans, the template value follows web commit sign-off and precedes merge settings. Declaring this field alone requires only the REST observation. The [CLI apply contract](../../references/commands.md#apply) covers failed writes, readback errors and earlier confirmed changes.

## API and evidence boundaries

The [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents `is_template` and Administration write. GitHub's [template prerequisites](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-template-repository) still apply. No new permission or live mutation is introduced to verify this feature.

Local fakes and HTTP fixtures establish typed reads, exact requests and error handling. They do not establish live write permission, supported template conversion for every repository type, or availability in a published package.

## Test mapping

- [Codec](../../../internal/workspace/ui_general_test.go) and [model](../../../internal/settings/comparison_test.go): omission, explicit booleans, invalid types, round trips, managed count and comparison.
- [REST](../../../internal/github/template_test.go): required boolean observations, exact omitted/true/false/mixed PATCH payloads and forbidden writes.
- [Plan](../../../internal/app/template_test.go) and [export](../../../internal/app/export_test.go): REST-only requests and complete observed declarations.
- [Summary](../../../internal/actions/template_test.go): native boolean rendering and type rejection.
- [Process plan](../../../tests/acceptance/plan_test.go), [apply](../../../tests/acceptance/apply_test.go) and [export](../../../tests/acceptance/export_validate_test.go): actual executable output/requests, incomplete reads, validation ordering, permission failure and export round trips.
