# Discussions setting

## Status

CLI/Action v0.15.0 and current consumer pins support `has_discussions`. Published v0.14.0 and older packages reject the key. This is an optional field in `settings.repository` under `schemaVersion: 1`, not a new resource family.

## Behavior

### Rule: Manage only explicitly declared values

- Omitted `has_discussions` remains unmanaged and does not appear in PATCH payloads.
- Explicit `true` and `false` are native booleans and count as one managed field.
- Null, quoted booleans, numbers, mappings, sequences, and legacy boolean spellings are rejected before origin/authentication/API work.
- Comparison uses the stable REST-field order after `has_wiki`, before existing merge-related values, topics, and labels.

### Rule: Observe a complete boolean

- Repository REST GET must contain a non-null boolean `has_discussions`; missing or malformed values fail instead of becoming false.
- Export includes the observed value only after all required reads succeed, preserving the existing file on failure.
- Plan remains read-only; declarations containing only this field do not trigger merge-related GraphQL reads.

### Rule: Verify declared writes without enabling implicitly

- Apply sends only declared repository fields in one PATCH and makes no prerequisite read.
- For declared `has_discussions`, PATCH must return the requested boolean; an absent, null, wrong-type, or unequal value fails the operation.
- Permission errors remain errors. Verification failure happens after PATCH, can leave earlier repository writes applied, and stops later resources without rollback.
- Omitting the field retains existing repository PATCH behavior without requiring a Discussions response.
- The field does not manage discussion categories, individual discussions, access policy, or any prerequisite setting.

## API and evidence boundaries

The [official GitHub CLI input](https://github.com/cli/cli/blob/6fcf2ea52718ad1ef8b12130f07430269e871352/pkg/cmd/repo/edit/edit.go#L76-L94) includes `has_discussions`, and its [update path](https://github.com/cli/cli/blob/6fcf2ea52718ad1ef8b12130f07430269e871352/pkg/cmd/repo/edit/edit.go#L301-L318) sends repository PATCH. GitHub's [update permission](https://docs.github.com/en/rest/repos/repos#update-a-repository) is Administration write; no additional persistent token grant is required by this implementation. The field is absent from the current public REST update schema, so an HTTP success without the expected returned boolean is insufficient.

A same-value `false` PATCH can establish response/read agreement, but does not establish a live true-to-false round trip or prove that an already-false request changed state. Live verification must not implicitly enable Discussions. True/false transitions, ignored writes, incomplete responses, and permission failures are checked with local API fixtures and dummy credentials.

## Test mapping

- [Schema](../../internal/workspace/discussions_test.go): omission, explicit booleans, round trip, and invalid types.
- [Comparison](../../internal/settings/discussions_test.go): false, omission, no difference, and stable order.
- [REST adapter](../../internal/github/discussions_test.go): complete reads, declared-only PATCH, response mismatch/missing data, and forbidden writes.
- [Use case](../../internal/app/discussions_test.go) and [export](../../internal/app/export_test.go): REST-only plan and observed export values.
- [Summary](../../internal/actions/discussions_test.go): native boolean rendering and rejection of invalid value types.
- [Process plan](../../tests/acceptance/plan_test.go) and [apply](../../tests/acceptance/apply_test.go): actual executable behavior, false writes, invalid-input ordering, ignored writes, and permission failures.
