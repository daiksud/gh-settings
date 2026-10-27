# Feature: Named Environment policies

Status: implements named policies (#352), custom deployment branch/tag patterns (#353), and typed administrator-bypass management. Historical [ADR 0029](../adr/0029-environment-ownership-and-protection.md) records an earlier proposal; the current contract is [Environments](../settings.md#environments).

## Ownership and API-controlled omissions

Only listed names are managed. Omitted and empty collections select no objects and make no Environment requests. Names select ordinary REST create-or-update, not numeric identity. Changing a name selects another object; the old unlisted object remains. Deletion, rename, secrets, variables, deployment execution and numeric-ID writeback are outside this feature.

Each listed policy owns supported settings as a whole, delegating omitted built-in request fields to GitHub rather than supplying client defaults or copying old values. Plan displays the observed policy and partial request with an API-controlled outcome warning. A partial request remains an update intent; full explicit matching requests can be no-ops. Readback verifies explicit requested values, not presumed omission outcomes. A name-only declaration sends `{}`, not a read-only existence assertion.

Names use documented client Unicode simple folding, preserve display spelling, reject controls/surrounding whitespace and encode slash/percent as one path segment. Reviewers are at most six unique typed positive numeric references, sorted independent of response order. GitHub validates principal access and account eligibility; this client never grants access or resolves a reused human name to a different principal. True self-review prevention cannot accompany an explicitly empty reviewer set. Only branch policy accepts explicit null; a non-null policy requires both opposite booleans.

## Typed deployment patterns

`deployment_branch_patterns` is a separate authoritative collection of case-sensitive `(type, name)` identities. Branch and tag may share a name; duplicate pairs are invalid. API IDs stay internal. List omission sends no pattern CRUD, because the Environment PUT has no pattern-list field; the client does not synthesize an empty collection or predict mode-transition effects on inactive patterns. Explicit `[]` removes every observed pattern. An explicit list requires custom mode, either explicitly selected in the request or completely observed on an existing Environment.

Custom-mode reads require all pages, stable total count, unique positive IDs, typed nonempty names, and unique pairs. Missing `type` is an error even though some official examples omit it. A read-only request to this repository's existing Environment returned `type: branch`; this does not prove all servers, grants, modes or writes behave identically.

## Complete reads and unsupported settings

Inventory requires explicit total count, complete pages and unambiguous names/IDs. Selected existing names receive individual detail and enabled-custom-protection reads. Unlisted details are not acquired by plan/apply; export explicitly selects every name. Failed/hidden reads never become absent objects. Detail 404 after inventory is a read failure, not create permission.

An explicitly present complete `protection_rules: []` represents no enabled built-in rules. Missing/null containers are errors. Present timer/reviewer rules require their complete typed values, including false/zero/empty. Missing policy is not null. An explicit root `can_admins_bypass` boolean is required for a complete observation; neither an absent nor a null/malformed value implies false or true. Unknown rule kinds/fields, enabled custom deployment rules and other unsupported root settings fail rather than being silently erased. Read-only principal profile metadata is reduced to its typed numeric reference; Environment URLs/timestamps/internal IDs do not enter writable policy.

An explicit YAML administrator-bypass boolean is sent natively as `can_admins_bypass`, including false. Omission sends no field and supplies no true/default value. Export and plan carry observed false/true values; apply verifies a requested value in fresh typed readback. Request spelling is supported by the pinned [go-github implementation](https://github.com/google/go-github/blob/6eccc42c1ea8cd175b190f8e785d1a55656e3c49/github/repos_environments.go), independently of that library's default-inserting marshaler. The client uses its own omission-preserving payload and does not copy arbitrary response fields into a mutation. The REST documentation's incomplete parameter table is not treated as proof of server behavior.

An explicitly authorized 2026-10-27 live characterization used one uniquely named temporary Environment in a public, personally owned GitHub repository, with API version `2026-03-10`. Explicit false and true each returned HTTP 200 and matched fresh GET. With timer zero, self-review false, reviewers empty and branch policy null held constant in the requests, subsequent flag omission retained each preceding false/true value. Other observable fields stayed equal. Creation without the flag returned true in this one test; the client still does not encode that as a default. The newly created ID was checked before HTTP 204 deletion, absence was verified, and pre-existing Environment observations/inventory matched before and after. No deployment, workflow, secrets or permissions were created, and no effective administrator-bypass attempt was tested. These results support the tested writable field; they do not generalize omission or entitlement guarantees to all requests/servers.

## Apply and reporting

Local validation precedes remote work. Apply acquires complete policies for selected names after earlier resources, then writes in stable name order. Each adapter call rechecks repository identity/archive state and selected protections, skips a fully explicit fresh match, and sends specified built-in fields without name/ID/pattern-list metadata. After PUT, explicit patterns are freshly read, missing pairs created, then obsolete pairs removed after rechecking their API ID and typed name. Complete fresh readback must agree with the requested fields/list. A 303 create response is an error, never permission to follow a redirect or adopt an unknown object.

Ordinary upsert is intentional: concurrent name creation/update is not rejected by an invented create-only rule, and a different numeric Environment incarnation is not a declaration identity mismatch. There is no lock, transactional snapshot, atomic policy replacement, rollback or hidden retry. Earlier successful writes remain applied when a later read/write fails. Prerequisite failures are classified before-write; PUT, malformed response, mismatched/denied readback failures may have applied.

Plan and Action reports show complete selected before policies and requested after policies, identifying API-controlled omissions instead of fabricated resulting values. A nonempty collection counts once. The same-version report rejects incomplete observations, null/unknown request fields, invalid deletes, wrong identity, duplicate/unordered changes and inconsistent outcomes/counts. Explicit null policy remains distinct from omission. Summary escapes content; status-only mode validates the same report without exposing names or policy.

Explicit `export --environments` saves supported observed policies and complete typed patterns in custom mode only after successful reads. Normal export retains its scope. Export has no Environment/pattern IDs and an unchanged complete export should produce zero differences. Inactive-mode patterns are not inferred. Unsupported/unreadable policy fails before file replacement. Apply never changes caller YAML for Environments, including local/default, opt-out and Actions execution; there is no automatic commit/push.

## Behavior/test mapping

| Behavior | Executable evidence |
| --- | --- |
| Name identity, API-controlled omission, unlisted preservation, no deletion, reviewer ordering | `internal/settings/environments_test.go` |
| Typed pattern identity, authoritative empty list, omission, duplicate rejection | `internal/settings/environment_patterns_test.go` |
| Explicit bypass false/true, omitted request uncertainty, missing-observation rejection and repeat no-op | `internal/settings/environment_bypass_test.go` |
| Strict YAML, null/type/unknown/ID rejection, encode/decode round trip | `internal/workspace/environments_test.go` |
| Complete configured/disabled reads, selected-only details, unavailable/custom protections, exact escaped path | `internal/github/environments_test.go` |
| Explicit upsert payload, matching no-op, server rejection, ignored values, uncertain readback, accepted changed incarnation | `TestEnvironmentUpsertWritesExplicitPolicyAndVerifiesWithoutNumericBinding` |
| Typed paginated reads, custom-mode entry, omitted/empty lists, 303/403, partial writes and identity races | `internal/github/environment_patterns_test.go` |
| Native bypass request booleans, no implicit true, 422/ignored values and failed typed reads | `internal/github/environment_bypass_test.go` |
| Plan/apply ordering, no ID recorder, earlier-success partial failure, empty/read-failure no write | `internal/app/environments_test.go` |
| Report round trip, default visibility, HTML escaping and closed report boundary | `internal/actions/environments_test.go` |
| Actual binary create/update/repeat plan/apply, API-controlled omissions, unchanged caller YAML including Actions mode, unsupported protection and readback failure | `TestEnvironmentBinaryExplicitPolicyAndOmissionsWithoutWriteback` |
| Actual binary typed pattern reconciliation, same-name branch/tag, repeat no-op and unchanged YAML | `TestEnvironmentPatternBinaryTypedReconciliationAndRepeat` |

## Primary API evidence and limits

- [REST Environments](https://docs.github.com/en/rest/deployments/environments): paginated inventory, individual detail and name-addressed create-or-update; timer 0–43200, at most six typed reviewers and nullable branch policy. Reads require Actions read, mutations Administration write.
- [Enabled custom deployment protections](https://docs.github.com/en/rest/deployments/protection-rules#get-all-deployment-protection-rules-for-an-environment): separate complete custom-protection check for selected existing objects.
- [Deployment branch policies](https://docs.github.com/en/rest/deployments/branch-policies): separate paginated typed-name collection, custom-mode requirement and CRUD permission/status contracts. Pattern list omission is not a supported Environment PUT field.
- [Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments): case-insensitive names up to 255 characters and lifecycle consequences.
- [Deployment availability/effectiveness](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments): account/visibility restrictions and protected-branches behavior. No subscription, visibility, credential or permission expansion is attempted.

Fixtures prove client serialization, parsing and failure/order behavior, not live omitted-field preservation/reset, entitlements, exact token grants, inactive-pattern retention or effective/intermediate deployment protection. The bounded administrator-bypass experiment above is separate evidence, not a claim that fixtures performed live mutations. Source implementation does not change production Environment policies, establish operational App coverage, or publish a release.
