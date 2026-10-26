# Feature: Named Environment policies

Status: implemented in current **unreleased** source for the bounded protection subset of #352. Published v0.19.0 and earlier reject all Environment keys/options. Historical [ADR 0029](../adr/0029-environment-ownership-and-protection.md) retains the earlier existing-only/field-preservation proposal; the accepted current contract is [Environments](../settings.md#environments).

## Ownership and defaults

Only listed names are managed. Omitted and empty collections select no objects and make no Environment requests. Names select ordinary REST create-or-update, not numeric identity. Changing a name selects another object; the old unlisted object remains. Deletion, rename, secrets, variables, deployment execution and numeric-ID writeback are outside this feature.

Each listed policy owns timer, self-review, reviewer membership and supported branch policy as a whole. Omitted fields receive fixed defaults on both create and update: zero minutes, false self-review prevention, empty reviewer list and null/all-refs branch policy. Review plan before applying a name-only declaration: it can remove supported protections.

Names use documented client Unicode simple folding, preserve display spelling, reject controls/surrounding whitespace and encode slash/percent as one path segment. Reviewers are at most six unique typed positive numeric references, sorted independent of response order. GitHub validates principal access and account eligibility; this client never grants access or resolves a reused human name to a different principal. True self-review prevention requires a nonempty reviewer set. Only branch policy accepts explicit null; a non-null policy requires both opposite booleans and supports protected-branches mode only.

## Complete reads and unsupported settings

Inventory requires explicit total count, complete pages and unambiguous names/IDs. Selected existing names receive individual detail and enabled-custom-protection reads. Unlisted details are not acquired by plan/apply; export explicitly selects every name. Failed/hidden reads never become absent objects. Detail 404 after inventory is a read failure, not create permission.

An explicitly present complete `protection_rules: []` represents no enabled built-in rules. Missing/null containers are errors. Present timer/reviewer rules require their complete typed values, including false/zero/empty. Missing policy is not null. Unknown rule kinds/fields, custom branch mode, enabled custom deployment rules and observed unsupported root settings (including administrator bypass) fail rather than being silently erased. Read-only principal profile metadata is reduced to its typed numeric reference; Environment URLs/timestamps/internal IDs do not enter writable policy.

The supported public request body has no administrator-bypass setter. This feature does not implement or guess one. It also does not preserve unknown settings by blindly copying response JSON into a mutation.

## Apply and reporting

Local validation precedes remote work. Apply acquires complete policies for all selected names after earlier resources, then writes in stable name order. Each adapter call rechecks repository identity/archive state and selected protections, skips a fresh match, and sends every supported writable field without name/ID metadata. The endpoint path identifies the name. A complete response and fresh detail/custom-protection readback must agree with the desired supported policy.

Ordinary upsert is intentional: concurrent name creation/update is not rejected by an invented create-only rule, and a different numeric Environment incarnation is not a declaration identity mismatch. There is no lock, transactional snapshot, atomic policy replacement, rollback or hidden retry. Earlier successful writes remain applied when a later read/write fails. Prerequisite failures are classified before-write; PUT, malformed response, mismatched/denied readback failures may have applied.

Plan and Action reports show complete selected before/after policies, including default-derived protection reductions. A nonempty collection counts once. The same-version report rejects missing/null/unknown fields, no-op updates, deletes, wrong identity, duplicate/unordered changes and inconsistent outcomes/counts. Null policy is explicit in reports, unlike omission-defaulted declarations. Summary escapes content; status-only mode validates the same report without exposing names or policy.

Explicit `export --environments` saves all supported observed policies only after complete successful reads. Normal export retains its scope. Export has no Environment IDs and an unchanged export should produce zero differences. Unsupported/unreadable policy fails before file replacement. Apply never changes caller YAML for Environments, including local/default, opt-out and Actions execution; there is no automatic commit/push.

## Behavior/test mapping

| Behavior | Executable evidence |
| --- | --- |
| Name identity, whole-object defaults, unlisted preservation, no deletion, reviewer ordering | `internal/settings/environments_test.go` |
| Strict YAML, null/type/unknown/ID rejection, encode/decode round trip | `internal/workspace/environments_test.go` |
| Complete configured/disabled reads, selected-only details, unavailable/custom protections, exact escaped path | `internal/github/environments_test.go` |
| Complete upsert payload, matching no-op, server rejection, ignored values, uncertain readback, accepted changed incarnation | `TestEnvironmentUpsertWritesWholePolicyAndVerifiesWithoutNumericBinding` |
| Plan/apply ordering, no ID recorder, earlier-success partial failure, empty/read-failure no write | `internal/app/environments_test.go` |
| Report round trip, default visibility, HTML escaping and closed report boundary | `internal/actions/environments_test.go` |
| Actual binary create/update/repeat plan/apply, unchanged caller YAML including Actions mode, unsupported protection and readback failure | `TestEnvironmentBinaryNameUpsertDefaultsAndRepeatWithoutWriteback` |

## Primary API evidence and limits

- [REST Environments](https://docs.github.com/en/rest/deployments/environments): paginated inventory, individual detail and name-addressed create-or-update; timer 0–43200, at most six typed reviewers and nullable branch policy. Reads require Actions read, mutations Administration write.
- [Enabled custom deployment protections](https://docs.github.com/en/rest/deployments/protection-rules#get-all-deployment-protection-rules-for-an-environment): separate complete custom-protection check for selected existing objects.
- [Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments): case-insensitive names up to 255 characters and lifecycle consequences.
- [Deployment availability/effectiveness](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments): account/visibility restrictions and protected-branches behavior. No subscription, visibility, credential or permission expansion is attempted.

Fixtures prove client serialization, parsing and failure/order behavior, not live server preservation, entitlements, exact token grants or effective deployment protection. No live Environment mutation or privileged test is part of this verification. Undocumented administrator-bypass behavior and custom branch/tag transitions (#353) remain outside adoption. Source implementation does not activate operational #380–#382 or publish a release.
