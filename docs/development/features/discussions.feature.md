# Feature: Manage the Discussions checkbox

The optional boolean `repository.settings.general.features.discussions` controls repository Discussions.

## Rule: Manage only explicitly declared values

- An omitted Discussions checkbox remains unmanaged and adds no internal `has_discussions` PATCH member.
- Explicit `true` and `false` are native booleans and count as one managed field.
- Null, quoted booleans, numbers, mappings, sequences, and legacy boolean spellings are rejected before origin/authentication/API work.
- Comparison uses the stable REST-field order after `has_wiki`, before existing merge-related values, topics, and labels.

### Scenario: Reject a string in a boolean field

- Given the declaration contains `discussions: "false"`
- When the maintainer validates the declaration
- Then validation fails locally before contacting GitHub

## Rule: Observe a complete boolean

- Repository REST GET must contain a non-null boolean `has_discussions`; missing or malformed values fail instead of becoming false.
- Export includes the observed value after complete base-group acquisition and projection. An unavailable base group follows the [readable-settings contract](readable-settings-export.feature.md); other groups still export.
- Plan remains read-only; declarations containing only this field do not trigger merge-related GraphQL reads.

### Scenario: Keep prior intent when GitHub omits Discussions

- Given a valid file declares Discussions enabled and the base repository read omits `has_discussions`
- When the maintainer refreshes the file with export
- Then the unavailable base group retains its prior intent and other readable groups can still update

## Rule: Verify declared writes without enabling implicitly

- Apply follows [repository reconciliation](../../references/commands.md#apply): read fresh state, skip exact matches, PATCH only changed declared controls and independently verify original intent.
- For `has_discussions` sent in PATCH, the response must return the requested boolean; an absent, null, wrong-type, or unequal value fails the operation.
- Permission errors remain errors. Verification failure happens after PATCH, can leave earlier repository writes applied, and stops later resources without rollback.
- Omitting the field retains existing repository PATCH behavior without requiring a Discussions response.
- The field does not manage discussion categories, individual discussions, access policy, or any prerequisite setting.

### Scenario: Disable Discussions and verify the result

- Given Discussions is enabled and the declaration sets `discussions: false`
- When the maintainer runs `gh settings apply`
- Then gh-settings requests `has_discussions: false` and verifies the disabled state
- And a repeat apply against that state sends no Discussions update

## API and evidence boundaries

The [official GitHub CLI input](https://github.com/cli/cli/blob/6fcf2ea52718ad1ef8b12130f07430269e871352/pkg/cmd/repo/edit/edit.go#L76-L94) includes `has_discussions`, and its [update path](https://github.com/cli/cli/blob/6fcf2ea52718ad1ef8b12130f07430269e871352/pkg/cmd/repo/edit/edit.go#L301-L318) sends repository PATCH. GitHub's [update permission](https://docs.github.com/en/rest/repos/repos#update-a-repository) is Administration write; no additional persistent token grant is required by this implementation. The field is absent from the current public REST update schema, so an HTTP success without the expected returned boolean is insufficient.

A same-value `false` PATCH can establish response/read agreement, but does not establish a live true-to-false round trip or prove that an already-false request changed state. Live verification must not implicitly enable Discussions. True/false transitions, ignored writes, incomplete responses, and permission failures are checked with local API fixtures and dummy credentials.

## Test mapping

- [Schema](../../../internal/workspace/ui_general_test.go): omission, explicit booleans, round trip, and invalid types.
- [Comparison](../../../internal/settings/comparison_test.go): false, omission, no difference, and stable order.
- [REST adapter](../../../internal/github/discussions_test.go): complete reads, declared-only PATCH, response mismatch/missing data, and forbidden writes.
- [Use case](../../../internal/app/discussions_test.go) and [export](../../../internal/app/export_test.go): REST-only plan and observed export values.
- [Summary](../../../internal/actions/discussions_test.go): native boolean rendering and rejection of invalid value types.
- [Process plan](../../../tests/acceptance/plan_test.go) and [apply](../../../tests/acceptance/apply_test.go): actual executable behavior, false writes, invalid-input ordering, ignored writes, and permission failures.
