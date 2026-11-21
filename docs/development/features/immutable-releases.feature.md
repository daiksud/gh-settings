# Feature: Manage the repository release immutability policy

The [configuration reference](../../references/settings/general.md#immutable-release-policy) owns the
public contract. This specification covers #348's repository policy only; the
release publisher and individual releases/tags/assets are outside ownership.

## Rule: Respect owner-enforced release immutability

### Scenario: Owner enforcement blocks a disable request

- Given the repository reports release immutability enabled and enforced by its owner
- And the declaration sets `enable_release_immutability: false`
- When the maintainer applies the declaration
- Then apply fails before sending a request to disable the policy
- And existing releases, tags and assets remain unchanged

## Declaration and observation

- `repository.settings.general.releases.enable_release_immutability` is an optional boolean. Omission is
  unmanaged, false is explicit, and null/type/key/duplicate errors fail locally.
- Ordinary export automatically attempts this independent
  family. An unavailable family retains compatible prior intent and emits bounded info while other families continue.
- GET `/repos/{owner}/{repo}/immutable-releases`: 200 requires native boolean
  `enabled` and `enforced_by_owner`; malformed/null/missing values fail. An
  enforced-but-disabled pair is inconsistent and fails.
- Under the required-permissions precondition, target GET 404 means
  enabled=false, with unknown owner enforcement. No additional authorization
  discriminator, inferred grant or fallback identity is used. This endpoint's
  disabled-state interpretation does not establish actual credential grants or
  supply owner-enforcement metadata, and does not define other endpoints' 404
  semantics. Explicit 401/403 denials, redirects, transport and other failures
  remain errors.

## Plan, apply and reporting

- Plan acquires only selected families, compares native booleans, and performs no
  mutation. False intent against observed owner enforcement fails.
- Read-only
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
  `repository.settings.general.releases.enable_release_immutability: false → true`.
  The private report retains its native field identity. The closed
  security report registry accepts only distinct native boolean changes, not
  owner metadata, unavailable values, guessed defaults or arbitrary fields.

## Test mapping and evidence limits

- [Model](../../../internal/settings/immutable_releases_test.go): acquired versus
  unknown metadata, omission, comparisons and owner enforcement.
- [Security/General Releases UI codec](../../../internal/workspace/ui_security_test.go) and
  [CLI options](../../../internal/cli/security_test.go): false round trip,
  invalid types/duplicates, independent selections and duplicate-option failure.
- [Page/section schema](../../../internal/workspace/ui_page_sections_test.go): the UI-label
  release key roundtrips, and the superseded release key is rejected without an alias.
- [API](../../../internal/github/immutable_releases_test.go): complete/incomplete
  200 and endpoint-specific 404 interpretation under the permission precondition,
  distinct 403/409/redirect/transport failures, fresh enforcement, exact body-free
  writes, matching state, refused/ignored writes and readback certainty.
- [Application](../../../internal/app/immutable_releases_test.go): independent plan,
  repeated matching apply, failed reads and retained earlier write progress.
- [Action](../../../internal/actions/immutable_releases_test.go): native Summary and
  rejection of invalid values through the shared closed report contract.
- [Built process](../../../tests/acceptance/immutable_releases_test.go): automatic
  export, 404→false→zero plan, changed plan/report/Summary, enable/disable
  and repeat apply, file preservation, owner enforcement, redirected reads and a
  409 after an earlier confirmed repository write.

All mutations use local fixtures and dummy credentials. No live immutable-release
policy, release, tag, asset, App grant, workflow, version or release publication
was changed or tested. See the official [GET](https://docs.github.com/en/rest/repos/repos#check-if-immutable-releases-are-enabled-for-a-repository),
[enable](https://docs.github.com/en/rest/repos/repos#enable-immutable-releases) and
[disable](https://docs.github.com/en/rest/repos/repos#disable-immutable-releases)
contracts for endpoint permissions and status codes.

## Canonical-key and package boundary

- The UI-derived key is `enable_release_immutability`; `immutable_release` is
  rejected even when the canonical key is also present. Validate, plan, apply
  and export reject that existing invalid input before API work or file writes.
- Current-source export and reports use only the canonical name. Native
  comparison fields, REST paths/payloads, booleans and permission requirements
  remain unchanged.
- A workflow needs a CLI package that accepts the declared key. The [package selection guide](../../guides/github-actions.md#published-package-and-consumer-pins) explains which CLI a pinned Action invokes.
