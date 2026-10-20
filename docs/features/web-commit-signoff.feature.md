# Web commit sign-off requirement

## Status

Implemented in current source, unreleased. Published v0.15.1 and older CLI/Action packages reject `settings.repository.web_commit_signoff_required`. The field is an optional boolean under `schemaVersion: 1`, not a new resource group. The [settings reference](../settings.md#web-commit-sign-off-unreleased) owns adoption and usage guidance.

## Behavior

- Omission leaves the field unmanaged; explicit true and false each count as one managed field. Null, quoted booleans and other types fail local validation before target, authentication or API access.
- The complete repository REST observation requires a non-null boolean, including when another REST field triggered the read. Missing or malformed data fails the read and discards any partial comparison.
- Export includes the observed boolean only after all required reads succeed. A failed read preserves the destination. Exported false remains an explicit declaration.
- Plan is read-only, compares declared values only, and places this field after `has_discussions` and before merge-related fields. This field alone needs no GraphQL request.
- Apply sends only declared fields through one repository PATCH, with no preliminary GET. An explicit false is sent as native JSON false; omission does not add the key or alter other fields. Existing API errors stop later resources and do not imply rollback.
- CLI output and Actions report/Summary use `settings.repository.web_commit_signoff_required` with native boolean values. The source report maximum is twenty: eighteen repository fields, topics and labels.
- The setting applies to web-based commit sign-off. It does not require cryptographic signatures, install a DCO check, rewrite commits or change protections.

## API and evidence boundaries

GitHub documents the field in [repository PATCH](https://docs.github.com/en/rest/repos/repos#update-a-repository), requiring Administration write. Reads use the existing repository GET and the existing authentication boundary. No extra permission or mutation probe is introduced.

Local HTTP fixtures and dummy credentials verify the requested transitions and failure behavior. They do not prove a live GitHub write, a particular token's permissions, or published-package support. No real repository setting is changed to test this feature.

## Test mapping

- [Codec](../../internal/workspace/web_commit_signoff_test.go): true/false, omission, invalid types and encode/decode round trip.
- [Model](../../internal/settings/web_commit_signoff_test.go) and [field definitions](../../internal/settings/repository_fields_test.go): managed values, comparisons, field identity and types.
- [REST](../../internal/github/web_commit_signoff_test.go): complete boolean observations and exact declared-only PATCH for omission, false, true, mixed declarations and forbidden writes.
- [Plan](../../internal/app/web_commit_signoff_test.go) and [export](../../internal/app/export_test.go): requested REST field and complete state-to-declaration export.
- [Summary](../../internal/actions/web_commit_signoff_test.go): native boolean rendering and rejection of invalid types.
- [Process plan](../../tests/acceptance/plan_test.go), [apply](../../tests/acceptance/apply_test.go) and [export](../../tests/acceptance/export_validate_test.go): real executable output/requests, missing reads, validation before remote work, permission failures and saved-file round trip.
