# Repository immutable release policy

The [configuration reference](../settings.md#immutable-release-policy) owns the
public contract. This specification covers #348's repository policy only; the
release publisher and individual releases/tags/assets are outside ownership.

## Declaration and observation

- `settings.security.immutable_releases` is an optional boolean. Omission is
  unmanaged, false is explicit, and null/type/key/duplicate errors fail locally.
- Opted-in `export --security immutable-releases` acquires only this additional
  family. Ordinary export retains its existing scope and permissions; failed
  selected reads preserve the destination file.
- GET `/repos/{owner}/{repo}/immutable-releases`: 200 requires native boolean
  `enabled` and `enforced_by_owner`; malformed/null/missing values fail. An
  enforced-but-disabled pair is inconsistent and fails.
- The October 28 user decision maps every GET 404 to enabled=false, with unknown
  owner enforcement. No additional endpoint, inferred grant or fallback identity
  is used to distinguish disabled state from missing authorization. This scoped
  exception can misclassify insufficient access as disabled. It is an accepted
  product tradeoff, not a live observation that this endpoint returns such a
  permission-related 404. 401/403/redirects/transport/other failures remain errors.

## Plan, apply and reporting

- Plan acquires only selected families, compares native booleans, and performs no
  mutation. False intent against observed owner enforcement fails.
- Apply follows private vulnerability reporting and precedes Pages. Read-only
  preparation does not count as a confirmed earlier write. A fresh matching
  policy skips the toggle. Fresh owner enforcement blocks disabling before DELETE.
- Changed enable/disable sends exactly one body-free PUT/DELETE to the policy
  endpoint, accepts only 204, then reads fresh policy. No owner-enforcement field
  or individual release/tag/asset endpoint is written. Repeat matching apply is
  a no-op for this family.
- Readback uses the same 404 decision. An unchanged/ignored requested value,
  failed readback or refused write fails with uncertain-write diagnostics and
  retains any confirmed earlier operations. No retries or rollback occur.
- CLI and credential-free Action Summary render
  `settings.security.immutable_releases: false → true`. The existing closed
  security report registry accepts only distinct native boolean changes, not
  owner metadata, unavailable values, guessed defaults or arbitrary fields.

## Test mapping and evidence limits

- [Model](../../internal/settings/immutable_releases_test.go): acquired versus
  unknown metadata, omission, comparisons and owner enforcement.
- [YAML](../../internal/workspace/immutable_releases_test.go) and
  [CLI options](../../internal/cli/immutable_releases_test.go): false round trip,
  invalid types/duplicates, independent selections and duplicate-option failure.
- [API](../../internal/github/immutable_releases_test.go): complete/incomplete
  200, explicit 404 interpretation (including a modeled authorization message),
  distinct 403/409/redirect/transport failures, fresh enforcement, exact body-free
  writes, matching state, refused/ignored writes and readback certainty.
- [Application](../../internal/app/immutable_releases_test.go): independent plan,
  repeated matching apply, failed reads and retained earlier write progress.
- [Action](../../internal/actions/immutable_releases_test.go): native Summary and
  rejection of invalid values through the shared closed report contract.
- [Built process](../../tests/acceptance/immutable_releases_test.go): ordinary and
  selected export, 404→false→zero plan, changed plan/report/Summary, enable/disable
  and repeat apply, file preservation, owner enforcement, redirected reads and a
  409 after an earlier confirmed repository write.

All mutations use local fixtures and dummy credentials. No live immutable-release
policy, release, tag, asset, App grant, workflow, version or release publication
was changed or tested. See the official [GET](https://docs.github.com/en/rest/repos/repos#check-if-immutable-releases-are-enabled-for-a-repository),
[enable](https://docs.github.com/en/rest/repos/repos#enable-immutable-releases) and
[disable](https://docs.github.com/en/rest/repos/repos#disable-immutable-releases)
contracts for endpoint permissions and status codes.
