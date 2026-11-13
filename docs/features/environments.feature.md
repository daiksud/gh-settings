---
type: Feature
title: Named Environment policies and variables
description: Manage name-selected UI policies, typed patterns and explicit non-sensitive variables with separate ownership.
---

Status: implements named policies (#352), custom deployment branch/tag patterns (#353), typed administrator-bypass management, and selected non-sensitive variables. Historical [ADR 0029](../adr/0029-environment-ownership-and-protection.md) records an earlier proposal; the current contract is [Environments](../settings.md#environments).

Ordinary `export` now follows the [automatic readable-settings contract](readable-settings-export.feature.md): observe all supported families without selectors, retain unavailable declarations with info, and keep plan/apply strict.

## Ownership and preservation

Public entries live under `repository.settings.environments`. Only listed names are managed. The policy rules here apply to policy-selected entries; [variable-only entries](#selected-environment-variables) have independent ownership. Omitted and empty collections select no objects and make no Environment requests. Names select ordinary REST create-or-update, not numeric identity. Changing a name selects another object; the old unlisted object remains. Deletion, rename, secrets, deployment execution and numeric-ID writeback are outside this feature.

Each listed existing policy owns its declared built-in controls. Matching partial and name-only requests skip mutation. A needed policy PUT overlays original partial intent on the writer’s latest complete built-in snapshot, preserving timer, reviewers, self-review, administrator bypass and canonical deployment mode. Plan and Summary keep the original partial request; companions never become desired ownership. Creation still sends partial intent, with omitted outcomes controlled by GitHub; a missing name-only selection sends `{}`. Complete independent readback verifies requested controls and preservation companions.

Names use documented client Unicode simple folding, preserve display spelling, reject controls/surrounding whitespace and encode slash/percent as one path segment. Reviewers are at most six unique typed positive numeric references, sorted independent of response order. GitHub validates principal access and account eligibility; this client never grants access or resolves a reused human name to a different principal. True self-review prevention cannot accompany an explicitly empty reviewer set. Conflicts with freshly preserved omitted reviewers or self-review also fail before writing rather than changing unowned controls. Public `deployment_branches_and_tags` selects `no_restriction`, `protected_branches_only` or `selected_branches_and_tags`; API null/boolean-pair aliases are not public YAML. All public nulls are invalid. `deployment_protection_rules` contains optional `wait_timer` (0–43200), `required_reviewers` entries selecting one positive `person` or `team`, `prevent_self_review` and `allow_administrators_to_bypass_configured_protection_rules`.

## Typed deployment patterns

`deployment_branch_or_tag_rules` is a separate authoritative collection of case-sensitive `(ref_type, name_pattern)` identities. `ref_type` is `branch` or `tag`; raw API type/name/ID fields remain internal. Branch and tag may share a name; duplicate pairs are invalid. API IDs stay internal. List omission sends no pattern CRUD, because the Environment PUT has no pattern-list field; the client does not synthesize an empty collection. When selected mode stays selected, independent readback must retain exact raw pattern IDs/types/names, including unknown types and projection collisions. Plan and apply reject disabling raw custom mode with omitted nonempty patterns, including unknown ref types and projection collisions. First apply an explicit empty list while Selected, then change mode; a complete known-empty collection may transition. Apply reacquires state independently of an earlier plan. Explicit `[]` removes every observed pattern. An explicit list requires selected-branches-and-tags mode (internal custom mode), either explicitly selected or completely observed on an existing Environment.

Custom-mode reads require all pages, stable total count, unique positive IDs, typed nonempty names, and unique pairs. Missing `type` is an error even though some official examples omit it. A read-only request to this repository's existing Environment returned `type: branch`; this does not prove all servers, grants, modes or writes behave identically.

## Complete reads and unsupported settings

Inventory requires explicit total count, complete pages and unambiguous names/IDs. Selected existing names receive individual detail and enabled-custom-protection reads. Unlisted details are not acquired by plan/apply; export explicitly selects every name. Failed/hidden reads never become absent objects. Detail 404 after inventory is a read failure, not create permission.

An explicitly present complete `protection_rules: []` represents no enabled built-in rules. Missing/null containers are errors. Present timer/reviewer rules require their complete typed values, including false/zero/empty. Missing policy is not null. An explicit root `can_admins_bypass` boolean is required for a complete observation; neither an absent nor a null/malformed value implies false or true. Unknown rule kinds/fields, enabled custom deployment rules and other unsupported root settings fail rather than being silently erased. Read-only principal profile metadata is reduced to its typed numeric reference; Environment URLs/timestamps/internal IDs do not enter writable policy.

The explicit UI administrator-bypass checkbox translates to native API `can_admins_bypass`, including false. Creation omission sends no field and supplies no true/default value. Existing updates preserve a freshly observed omitted value as a send-time companion. Export and plan carry observed false/true values; apply verifies declared values and sent companions in fresh typed readback. Request spelling is supported by the pinned [go-github implementation](https://github.com/google/go-github/blob/6eccc42c1ea8cd175b190f8e785d1a55656e3c49/github/repos_environments.go), independently of that library's default-inserting marshaler. The client constructs its own typed payload and does not copy arbitrary response fields into a mutation. The REST documentation's incomplete parameter table is not treated as proof of server behavior.

An explicitly authorized 2026-10-27 live characterization used one uniquely named temporary Environment in a public, personally owned GitHub repository, with API version `2026-03-10`. Explicit false and true each returned HTTP 200 and matched fresh GET. With timer zero, self-review false, reviewers empty and branch policy null held constant in the requests, subsequent flag omission retained each preceding false/true value. Other observable fields stayed equal. Creation without the flag returned true in this one test; the client still does not encode that as a default. The newly created ID was checked before HTTP 204 deletion, absence was verified, and pre-existing Environment observations/inventory matched before and after. No deployment, workflow, secrets or permissions were created, and no effective administrator-bypass attempt was tested. These results support the tested writable field; they do not generalize omission or entitlement guarantees to all requests/servers.

## Apply and reporting

Local validation precedes remote work. Apply acquires complete policies for selected names after earlier resources, then writes in stable name order. Each adapter call rechecks repository identity/archive state and selected protections and skips any fresh match of owned values. A needed existing-policy PUT sends the fresh supported built-in snapshot with original requested controls overlaid, without name/ID/pattern-list metadata. Pattern-only changes skip PUT. After a needed PUT, independent policy acquisition must verify all sent controls before pattern CRUD; otherwise the existing complete selected-mode observation supplies the prerequisite. Missing pairs are created, then obsolete pairs removed after rechecking their API ID and typed name. Complete final acquisition must agree with original fields/list and the preserved built-in snapshot. Omitted active patterns must retain their exact raw identities when selected mode stays selected. A 303 create response is an error, never permission to follow a redirect or adopt an unknown object.

Ordinary upsert is intentional: concurrent name creation/update is not rejected by an invented create-only rule, and a different numeric Environment incarnation is not a declaration identity mismatch. There is no lock, transactional snapshot, atomic policy replacement, rollback or hidden retry. Earlier successful writes remain applied when a later read/write fails; matching skips and reads are not earlier writes. An independently verified policy update before a later pattern failure is recorded as partial progress. Prerequisite failures before any mutation are classified before-write; PUT/pattern mutations, malformed responses and mismatched/denied readbacks can have uncertain effects.

Plan and detailed Summary show the requested fields of each selected effective before policy and the actual requested after policy, identifying preservation on existing updates and API-controlled omission on creation instead of fabricated resulting values. The private report retains the complete acquired before policy for validation. A nonempty collection counts once. The same-version report rejects incomplete observations, null/unknown request fields, invalid deletes, wrong identity, duplicate/unordered changes and inconsistent outcomes/counts. The private report keeps internal nullable API policy distinct from omitted intent; public YAML expresses no restriction with its explicit UI choice, not null. UI object rendering uses current names and choices. Summary escapes content; status-only mode validates the same report without exposing names or policy.

Ordinary `export` adopts complete supported policy and active typed patterns, without object IDs, and independently inventories variables. Unavailable policy emits info and retains the prior collection; no missing value becomes a safe-looking default.

## Behavior/test mapping

| Behavior | Executable evidence |
| --- | --- |
| Name identity, partial ownership, unlisted preservation, no deletion, reviewer ordering | `internal/settings/environments_test.go` |
| Fresh companion overlay, partial no-op, dependency rejection and unchanged original intent | `internal/settings/environment_preservation_test.go` |
| Partial payload, latest companions, raw-mode refusal, raw-pattern retention, drift/read failures and pattern-only preflight | `internal/github/environment_preservation_test.go` |
| Typed pattern identity, authoritative empty list, omission, duplicate rejection | `internal/settings/environment_patterns_test.go` |
| Explicit bypass false/true, omitted matching no-op, missing-observation rejection | `internal/settings/environment_bypass_test.go` |
| Strict YAML, null/type/unknown/ID rejection, encode/decode round trip | [UI workspace tests](../../internal/workspace/ui_environments_test.go) |
| Complete configured/disabled reads, selected-only details, unavailable/custom protections, exact escaped path | `internal/github/environments_test.go` |
| Explicit upsert payload, matching no-op, server rejection, ignored values, uncertain readback, accepted changed incarnation | `TestEnvironmentUpsertWritesExplicitPolicyAndVerifiesWithoutNumericBinding` |
| Typed paginated reads, custom-mode entry, omitted/empty lists, 303/403, partial writes and identity races | `internal/github/environment_patterns_test.go` |
| Native bypass request booleans, no implicit true, 422/ignored values and failed typed reads | `internal/github/environment_bypass_test.go` |
| Plan/apply ordering, no ID recorder, earlier-success partial failure, empty/read-failure no write | `internal/app/environments_test.go` |
| Report round trip, default visibility, HTML escaping and closed report boundary | `internal/actions/environments_test.go` |
| Actual binary create/update/repeat plan/apply, omitted name-only no-op and partial preservation, unchanged caller YAML including Actions mode, unsupported protection and readback failure | `TestEnvironmentBinaryExplicitPolicyAndOmissionsWithoutWriteback` |
| Actual binary typed pattern reconciliation, same-name branch/tag, repeat no-op and unchanged YAML | `TestEnvironmentPatternBinaryTypedReconciliationAndRepeat` |

## Primary API evidence and limits

- [REST Environments](https://docs.github.com/en/rest/deployments/environments): paginated inventory, individual detail and name-addressed create-or-update; timer 0–43200, at most six typed reviewers and nullable branch policy. Reads require Actions read, mutations Administration write.
- [Enabled custom deployment protections](https://docs.github.com/en/rest/deployments/protection-rules#get-all-deployment-protection-rules-for-an-environment): separate complete custom-protection check for selected existing objects.
- [Deployment branch policies](https://docs.github.com/en/rest/deployments/branch-policies): separate paginated typed-name collection, custom-mode requirement and CRUD permission/status contracts. Pattern list omission is not a supported Environment PUT field.
- [Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments): case-insensitive names up to 255 characters and lifecycle consequences.
- [Deployment availability/effectiveness](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments): account/visibility restrictions and protected-branches behavior. No subscription, visibility, credential or permission expansion is attempted.

Fixtures prove client serialization, parsing and failure/order behavior, not live omitted-field preservation/reset, entitlements, exact token grants, effective/intermediate deployment protection. The bounded administrator-bypass experiment above is separate evidence, not a claim that fixtures performed live mutations. Source implementation does not change production Environment policies, establish operational App coverage, or publish a release.

## Rule: Keep raw deployment mode distinct from effective No restriction

- Given a complete typed equal branch-policy pair, project No restriction with independent pair provenance. Valid null is exact; missing/null/type/unknown fields and incomplete acquisition fail without partial observation.
- If raw custom is true, acquire every pattern page and custom-protection check before attaching provenance. Keep latent patterns in separate state; effective fallback neither activates them nor proves Selected applicability.
- Compare only selected names. Explicit mode fallback equality requires one update and scoped UI warning/decision. Every before-object is labeled effective policy; partial existing-update omissions remain unowned preservation companions. Private closed pair metadata, HTML safety, status-only privacy and collection count one remain.
- Existing-policy PUT includes fresh supported built-in companions overlaid with declared controls. Omitted canonical mode is preserved and independently verified without unowned diagnostics; unreplayable equal raw pairs block a needed policy write, while matching selections skip. An explicitly requested nullable mode must match the raw acknowledgment and independent complete fresh read; canonical acknowledgment followed by retained equal booleans fails with possible write effects.
- Pattern intent under an unknown mode requires explicit Selected. Confirm actual canonical Selected in the intermediate fresh read before ordered create-before-delete CRUD. Neither fallback nor raw custom true is proof. Failed pattern operations retain earlier mode/pattern effects without rollback; earlier resource writes remain reported.
- Pure readers never warn or write; export warns after complete acquisition and retains prior family intent with info on failure. Actual null exports no invented pattern list.

Test mapping: [mapping](../../internal/settings/environment_deployment_mapping_test.go), [owned state/latents](../../internal/settings/environment_deployment_observation_test.go), [closed changes](../../internal/settings/environment_deployment_change_test.go), [scoped reconciliation](../../internal/settings/environment_deployment_reconcile_test.go), [typed JSON](../../internal/settings/environment_deployment_json_test.go), [preliminary details](../../internal/github/normalization_environment_deployment_test.go), [reader pagination](../../internal/github/environment_deployment_read_test.go), [raw requested verification](../../internal/github/environment_deployment_verification_test.go), [preliminary acknowledgment](../../internal/github/environment_deployment_ack_test.go), [scoped diagnostics](../../internal/github/environment_deployment_warning_test.go), [real Update](../../internal/github/environment_deployment_update_test.go), [export](../../internal/app/normalization_environment_deployment_export_test.go), [private reports](../../internal/actions/normalization_environment_deployment_test.go), [presentation](../../internal/presentation/environment_deployment_test.go), [built executable](../../tests/acceptance/normalization_environment_deployment_test.go).

## Rule: Keep raw ref-type identity distinct from effective branch

Status: implemented in current source. Release publication is a separate operation.

- Given complete acquisition of active selected deployment patterns with an unsupported string ref type, export projects that type to `branch` and warns with the Environment name, pattern ID, exact pattern name, original type and fallback. Canonical branch/tag companions remain unchanged and silent.
- Warn in normalized Environment-name order, then ascending pattern-ID order, before materializing or saving the export. Latent patterns under an effective mode other than Selected are neither exported nor disclosed by ref-type warnings. A missing warning sink does not change the effective policy.
- Distinct raw identities that project to the same typed name must not be collapsed. Emit scoped warnings and info, retain prior family intent and continue independent reads. Read failures and incomplete acquisition produce neither fallback export nor ref-type warnings.
- During apply, ref-type warnings require an explicit managed pattern list, the selected Environment name and complete fresh acquisition in canonical Selected mode. Explicit empty lists still own the observed patterns. Omitted lists, unlisted names and latent patterns remain silent; validate the entire raw collection before warning, including when its effective projection collides.
- Given complete unsupported string types, retain original types, case-sensitive names and positive IDs through all pages. Missing/null/wrong-type fields, invalid identities, duplicate raw pairs and incomplete counts fail before publishing any observation or diagnostic.
- An explicit list, including empty, owns raw reconciliation and remains actionable at fallback equality. Plan/private reports/detailed Summary retain the full acquired list and canonical companions; status-only output hides those details. A projected collision is unavailable effective list data, not an empty collection. Collection count stays one.
- Apply creates canonical pairs before deleting obsolete raw IDs; existing canonical companions keep their IDs. Each removal confirms exact raw ID/type/name. Unsupported/malformed or mismatching creation acknowledgments stop before removal, and final complete readback never accepts fallback equality as canonical confirmation.
- Later failures preserve earlier effects, return nonzero status without success output, retain fresh scoped diagnostics and leave caller YAML unchanged. No corrective retry or rollback occurs. Confirmed canonical repeat is silent and makes no write.

Test mapping: [mapping and observation](../../internal/settings/environment_ref_type_observation_test.go), [reader completeness and collisions](../../internal/github/environment_ref_type_read_test.go), [raw deletion identity](../../internal/github/environment_ref_type_delete_test.go), [Update intent and fresh confirmation](../../internal/github/environment_ref_type_update_test.go), [strict creation confirmation](../../internal/github/environment_ref_type_update_failure_test.go), [private reports](../../internal/actions/normalization_environment_ref_type_test.go), [executable export](../../tests/acceptance/normalization_environment_ref_type_test.go), [executable plan/report/apply](../../tests/acceptance/normalization_environment_ref_type_commands_test.go), [owned UI projection](../../internal/github/ui_environment_ref_type_observation_test.go), [export warnings, effective policy and failure preservation](../../internal/app/normalization_environment_ref_type_export_test.go), [scoped fresh-state warnings](../../internal/github/environment_ref_type_warning_test.go).

## Selected Environment variables

- Each entry may select `variables`, an array of required nonempty `name`/string `value` pairs. Variable names canonicalize to uppercase and reject duplicates within the folded Environment identity. The same name in different Environments and repository families remains independent.
- Omitted variable lists always stay unmanaged. Without variable prune, [] selects no values and unlisted values remain unchanged. Explicit `--prune environment-variables` adds unlisted deletions for each listed Environment with a present variable list; [] selects all its acquired variables. There is no YAML delete or rename member. Values are explicitly non-sensitive plaintext, not Secrets.
- An entry with `name` and `variables` only selects no Environment policy. It sends no policy PUT or pattern CRUD, even for an empty list. Missing Environments fail instead of being created. A name-only entry skips an existing policy or creates a missing one; explicitly combined policy/variable intent applies the policy first.
- Plan reads selected individual names. If the same plan explicitly creates a missing Environment policy, it shows the selected variables as creates without reading nonexistent variable endpoints. Apply checks the Environment identity, reads fresh variable state, conditionally POSTs or PATCHes and verifies an independent GET. It skips matching values. Failed Environment reads never become missing variables, and variable 404 is absence only after the Environment check succeeds.
- Environment names are escaped as one URL segment. No unselected Environment, repository variable family or Secret is read or changed.
- Ordinary export inventories Environment names completely, then variables per parent. One unavailable parent inventory retains prior variables with info while other parents continue. No secret API is used.
- Stable Environment/name-order execution retains prior completed writes and later unexecuted operations after failure. Failed mutations/readback may have applied; there is no rollback. Detailed output contains values, escaped in Summary; status-only output exposes no variable details.

### Live adoption evidence

On 2026-11-11, a disposable Environment in the Free, personally owned public `daiksud/test-gh-settings-public` repository exposed configurable Environment variables. UI create/update/delete and API create/update/readback/delete succeeded; the UI and API rejected an empty value. The pinned API version also accepted a slash-containing Environment name as one escaped path segment and value-only PATCH. Temporary variables and Environments were removed and the final Environment collection was empty. This verifies bounded UI/API availability and persistence, not workflow consumption, all credential permissions or effective protection. Automated tests use local fixtures and dummy credentials, not live mutations.

### Variable test mapping

| Boundary | Executable evidence |
| --- | --- |
| Scoped identity, selection and empty ownership | `internal/settings/environment_variables_test.go` |
| Strict YAML, variable-only versus combined policy intent and round trip | `internal/workspace/ui_environment_variables_test.go` |
| Escaped paths, existing-Environment guard, selected variable reads/writes and readback failures | `internal/github/environment_variables_test.go` |
| Selected plan/export, no policy upsert and honest partial progress | `internal/app/environment_variables_test.go` |
| Explicit export pairs and typed CLI presentation | `internal/cli/environment_variables_test.go` |
| Closed scoped reports, escaped detailed Summary and status-only privacy | `internal/actions/environment_variables_test.go` |
| Actual executable selection, exact requests, preservation and failure behavior | `tests/acceptance/environment_variables_test.go` |

Environment-variable prune additionally binds complete inventories and deletion
checks to the parent ID, rejecting detected replacement before the family write
and before/after DELETE. Only explicitly declared Environment policy creation can
precede variables for an absent parent. Unexpected candidates after that creation
are rejected instead of deleting them. This adopts no Environment-object deletion
or secret ownership. See [variable pruning](../settings.md#variable-pruning) and
`internal/github/variable_prune_test.go`, `internal/app/variable_prune_test.go`,
`tests/acceptance/prune_variables_test.go` for bounded fixture evidence.

## Mode changes and omitted pattern preservation

A bounded 2026-11-13 API experiment used a new temporary Environment in the dedicated personally owned public test repository, with no workflows, deployments, secrets, variables or custom protections. Disabling Selected mode through either No restriction or Protected branches made pattern listing unavailable; returning to Selected produced a complete empty inventory instead of the two original branch/tag patterns. The temporary Environment was removed only after its identity and empty child inventories were checked, and a complete fresh inventory confirmed the original empty baseline. This establishes observed pattern loss for those transitions, not universal credential behavior or a supported inactive-read contract.

The [mode guard](../../internal/settings/environment_mode_preservation_test.go) rejects omission-based loss before planning an update or preparing its payload. It uses raw acquisition, including normalized modes and colliding ref-type projections; missing acquisition is not empty. [Fresh adapter coverage](../../internal/github/environment_mode_preservation_test.go) proves that a pattern appearing after an earlier empty plan blocks apply. [Built-product coverage](../../tests/acceptance/environment_mode_preservation_test.go) checks CLI/Action plan rejection, no Environment or later write, unchanged YAML, explicit clear-then-switch and repeated no-ops. Existing normalization tests use known-empty collections when exercising mode acknowledgments and readback errors. Reads and writes remain non-atomic; there is no conditional mode-update API or rollback promise.
