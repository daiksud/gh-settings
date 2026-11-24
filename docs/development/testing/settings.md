# Setting verification by family

Find the fixtures that check the changed setting from decoded YAML through API requests and user-visible output. For complete setup and commands, see [Local process acceptance](../testing.md#local-process-acceptance) and [supplied-product Action tests](ci.md#composed-callers-and-examples).

[Public configuration and export](#public-configuration-and-export) · [Repository fields and labels](#repository-fields-and-labels) · [Rulesets](#rulesets) · [Actions and Copilot](#actions-and-copilot) · [Deployment](#deployment) · [Security](#security) · [Access and integrations](#access-and-integrations) · [Existing v0 contracts outside v1](#existing-v0-contracts-outside-v1)

## Public configuration and export

### Declaration and export composition

| Owner | Contract exercised |
| --- | --- |
| [Current codec](../../../internal/workspace/ui_config_test.go), [strict YAML safety](../../../internal/workspace/ui_yaml_safety_test.go) and family `ui_*_test.go` files | Exact marker, obsolete/API aliases rejected, independent literal inventories, null/false/zero/empty/omitted values, aliases/merges and zero partial decoding |
| [Desired composition](../../../internal/github/ui_declaration_test.go) | Desired fields translated into independently expected native values, semantic failure returns zero whole declaration |
| [Export composition](../../../internal/github/ui_export_test.go) | Actual acquired data only, pure policy projections, hidden/raw distinctions, no metadata replay, stable Labels and no partial UI output |
| [Checkout adapters](../../../internal/command/checkout_config_test.go) | Deferred bound saver before reads, UI/native conversion, current-schema snapshot ID recording, no-op documents and failure-before-recorder behavior |
| [Ruleset HTTP fixture](../../../internal/github/ui_config_rulesets_test.go) | Literal UI YAML through decode/conversion/reconcile to exact independently authored POST/PUT and ID-bound paths |
| [Process suite](../../../tests/acceptance) | A supplied binary executes user-facing operations against local API fixtures with independently expected YAML, responses and writes |
| [Candidate Action fixture](../../../tests/github-actions/candidate-plan.sh) and [composite tests](../../../tests/github-actions/composite_test.go) | Current literal UI consumer configs with unchanged installer/token/identity/escaping boundaries; actual hosted `uses` expansion remains separate evidence |

### Metadata, reports, and Summary

[Metadata acceptance](../../../tests/acceptance/metadata_test.go) checks the supplied executable's actual version/digest and a separate committed consumer identity without settings, credentials, or API requests. Detailed origin syntax and isolation matrices remain in workspace tests. Mock metadata and owned test-executable adapters prove wiring or failure handling, rather than the installed product's identity.

The Linux-only [`TestMetadataUsesRunningImage`](../../../internal/actions/metadata_linux_test.go) launches an owned copy of the test executable and replaces only that child's pathname before requesting metadata. The digest must identify the original running image; the parent test runner is not replaced. Native checks on other platforms cover an unchanged executable path, and cross-compilation alone does not run this Linux regression.

[Plan-report acceptance](../../../tests/acceptance/plan_report_test.go) inspects native typed JSON and categorical failures through the real interface. Label operations occupy their own typed report entries and count as one managed collection, independently of the number of operations. [Rulesets acceptance](../../../tests/acceptance/rulesets_test.go) verifies the built CLI, report and credential-free Summary expose complete supported ID-selected objects and default-derived changes while leaving unlisted objects untouched. [Rulesets ID acceptance](../../../tests/acceptance/ruleset_ids_test.go) covers independent create confirmation, writeback/repeat apply, opt-out/Actions suppression and reported-ID recovery without YAML adoption after failed readback. [Basic readback acceptance](../../../tests/acceptance/ruleset_readback_test.go) verifies branch/tag updates, empty-rule removal, fresh confirmation before prune, failure stopping and repeat no-op. [Adapter regressions](../../../internal/github/ruleset_readback_test.go) cover complete raw policy, identity/ownership, malformed/denied/redirected readback and transport failures; [application cases](../../../internal/app/ruleset_ids_test.go) retain uncertain effects and prevent later writes or ID recording. [Summary acceptance](../../../tests/acceptance/summary_test.go) checks escaped values, separate identity links, append behavior, and zero API requests. Detailed report/state/storage cases, including label reports and the maximum managed count, remain in Actions owner tests. Report/Summary output preserves a failed plan's nonzero status and never becomes apply input.

### Export diagnostic delivery

[Application tests](../../../internal/app/export_warning_failure_test.go) verify
that warning failures in Repository, direct families and Security prevent saving,
retain their error identity and cannot become ordinary unavailable-family info.
[Export warning adapters](../../../internal/command/export_test.go) cover every typed warning writer;
Team-reviewer normalization remains behind the existing personal-repository
policy rejection, with no new eligibility implied by its callback.

The [built-CLI regression](../../../tests/acceptance/export_warning_output_test.go)
passes a real read-only regular file as stderr. A complete read-only API fixture
has no incidental info diagnostics: canonical acquisition succeeds even with that
unwritable fd, while a required normalization warning makes export fail before
initial creation or existing-file replacement. Writable controls verify the exact
warning and effective YAML. This avoids false positives from denied-family info,
closed-pipe signals or parent-side output copying. It establishes local failure
propagation, not runner log delivery or live GitHub behavior.

Action export directly invokes this same CLI. Its existing [composition failure
cases](../../../tests/github-actions/composite_export_test.go) establish nonzero-status
propagation and absent success outputs. The real-fd test above establishes the
CLI warning failure; it is not a hosted runner log-delivery test.

## Repository fields and labels

### About, General and Labels

These fixtures compare the [About, General and Labels mappings](../ui-repository-mapping.md) with independently specified UI values and API payloads.

| Test owner | Independent expected behavior |
| --- | --- |
| [About](../../../internal/github/ui_about_test.go) | Omission vs empty strings/list; Website to `homepage`; raw null-string/topic reads; exact empty PATCH/PUT payloads; owned values and invalid topics |
| [General scalars](../../../internal/github/ui_general_test.go) | Singleton mappings for each true/false control and branch; reverse projection; invalid branch; exact mixed REST payload; complete raw GraphQL observation |
| [Commit menus](../../../internal/github/ui_commit_message_test.go) | Seven literal UI/API pairs and exact payloads; absent vs incomplete pairs; API-only choices and invalid/disabled declarations |
| [Creation policies](../../../internal/github/ui_creation_policy_test.go) | Both policies and choices; unknown tokens; exact GraphQL input with native access/sponsorship booleans and verified target identity |
| [Labels](../../../internal/github/ui_labels_test.go) | UI color prefix; complete raw list conversion; omission/empty; missing observation; selected-ID rename leaving unlisted objects alone; exact update/create bodies and ID validation |

[Conditional desired-policy fixtures](../../../internal/github/ui_creation_policy_visibility_test.go) reject explicitly disabled feature/policy combinations while retaining omitted/true gates. [Conditional observation fixtures](../../../internal/github/ui_general_conditional_observation_test.go) exercise independent mixed gates for both feature policies and merge menus, six owned raw values, absence and strict validation before hiding. Exact composed GraphQL tests retain visible policy intent and an independent false Sponsorships control.

The new fixtures use literal expectations rather than production mapping definitions. Existing repository, merge-method, access-policy and label owner tests continue to cover complete/malformed/denied reads, pagination, identity checks, redirects and partial failures. Existing archive tests retain lifecycle guards and sequencing. Codec and process tests exercise the same mappings through the CLI.

[Label readback acceptance](../../../tests/acceptance/label_readback_test.go) checks
successful acknowledgments followed by stale or denied reads, recovery IDs without
premature YAML recording, earlier confirmed IDs, rename paths, stopping before
prune deletions and matching repetition. [ID writeback acceptance](../../../tests/acceptance/label_ids_test.go)
also covers opt-out, Actions suppression, concurrent edits and later failures.
These fixtures use independent remote state and dummy credentials; they do not
establish live GitHub behavior.

Current-source [archive lifecycle model](../../../internal/settings/archive_test.go), [API](../../../internal/github/archive_test.go), [application](../../../internal/app/archive_test.go), and [built-process acceptance](../../../tests/acceptance/archive_test.go) tests verify typed state, target identity, narrow updates, ordered transitions, partial failures, and automatic export. They use local fixtures only; live lifecycle permissions and recovery are not established.

### Topics

The [application tests](../../../internal/app/topics_test.go) and [built CLI cases](../../../tests/acceptance/topics_test.go) check complete-set ownership, a matching no-op, one replacement of a changed set and an independent readback. An empty list clears Topics. Omission leaves Topics unmanaged.

### Merge response acceptance

[Merge transport tests](../../../internal/github/merge_methods_test.go), [PR option tests](../../../internal/github/pr_options_test.go), and [squash](../../../internal/github/squash_commit_test.go)/[normal merge enum tests](../../../internal/github/merge_commit_test.go) own response completeness: each required field must be present with the correct type and a supported enum value. Any GraphQL errors discard even complete data. [Observation tests](../../../internal/github/observation_test.go) also reject missing fields that were not declared; expected responses use literal fixtures independent of production field definitions.

[Merge acceptance](../../../tests/acceptance/merge_methods_test.go) retains representative permission, incomplete-response, and partial-data/error failures, including rejection of an earlier REST comparison with empty stdout and nonzero status. For export, unavailable merge settings emit bounded info and retain compatible prior intent while readable siblings are saved. The [About/General export regressions](../../../tests/acceptance/repository_export_isolation_test.go) exercise partial acquisition and local validation. Detailed null/type/enum matrices remain at the adapter boundary; successful plan cases and complete declared-only PATCH cases remain at the process boundary.

### Labels resource acceptance

[Labels acceptance](../../../tests/acceptance/labels_test.go) runs the current-source binary through empty selections, create-only collisions, ID-selected updates and renames, description defaults, unmanaged objects, invalid input, complete pagination and failed reads. Typed reports and a separate credential-free Summary expose complete supported selected objects, escaped names and ID identity, without implicit deletions. [ID writeback acceptance](../../../tests/acceptance/label_ids_test.go) checks repeat apply after recording IDs, explicit opt-out, GitHub Actions suppression, concurrent local edits and partial remote creation failure with recoverable IDs.

[Application labels tests](../../../internal/app/labels_test.go) separately own repository → topics → labels-read → labels-write sequencing, fresh comparison, already matching collections with no mutation, earlier-write preservation, and rejection of an earlier partial plan when a later labels read/comparison fails. [Adapter tests](../../../internal/github/labels_test.go) own exact request payloads and escaped name segments; full validation/identity matrices stay in the model and codec tests.

These checks use local API responses and dummy credentials. They verify current-source behavior without establishing live label permissions or support in the published Action/CLI pair.

## Rulesets

### Control mappings and acquisition

The [Ruleset mapping](../ui-ruleset-mapping.md) records the supported UI controls and complete-acquisition boundaries. The public YAML codec uses that mapping.

| Fixture owner | Independent contract checked |
| --- | --- |
| [Root](../../../internal/github/ui_ruleset_root_test.go) | Required enforcement/one target, sentinels, short/full-ref patterns, empty criteria, blank rejection, ownership and out-of-UI values |
| [Bypass](../../../internal/github/ui_ruleset_bypass_test.go) | Three verified role IDs, dynamic actor IDs, three UI modes, marker validation and target restrictions |
| [Simple rules](../../../internal/github/ui_ruleset_rules_test.go) | Four singleton false/true checkbox mappings and complete observed absence |
| [PR settings](../../../internal/github/ui_ruleset_pr_test.go) | Singleton flags, finite count choices, merge methods and complete read projection |
| [Checks](../../../internal/github/ui_ruleset_checks_test.go) | Singleton flags, empty collections, Any source vs App identity, raw null source and owned values |
| [Analysis](../../../internal/github/ui_ruleset_analysis_test.go) | All 20 scanning threshold combinations, four quality choices and required desired values |
| [Collection and transport](../../../internal/github/ui_ruleset_collection_test.go) | Omitted and empty collections, raw eight-rule reads through export, exact no-ID POST/selected-ID PUT and independent readback |

[Unsupported Ruleset acceptance](../../../tests/acceptance/ruleset_unsupported_controls_test.go) rejects removed declarations before remote work, including false/empty values, and verifies that unsupported observed enforcement, rule types and review-dismissal parameters block export, update and explicit deletion without writes or configuration replacement. These local cases preserve supported PR behavior and do not change live protection.

Expected payloads and choices are literal fixtures, independent of production maps. Existing Ruleset owner tests retain denied/malformed/incomplete reads, pagination, inherited filtering and unsupported Push rejection, identity/target failures and partial-write coverage. Local conversion/transport tests do not establish live policy changes, effective enforcement, new YAML acceptance or normalization warnings.

### Ruleset bypass-mode normalization

[Mapping](../../../internal/settings/ruleset_bypass_mapping_test.go), [owned state](../../../internal/settings/ruleset_bypass_observation_test.go), [closed changes](../../../internal/settings/ruleset_bypass_change_test.go) and [reconciliation](../../../internal/settings/ruleset_bypass_reconcile_test.go) cover exact actor identity, nil IDs, multiple raw records, fallback equality, removal and independent scalar evidence. The [pure projector](../../../internal/github/normalization_ruleset_bypass_test.go) and [complete reader](../../../internal/github/ruleset_bypass_read_test.go) reject incomplete/malformed/ambiguous acquisition without partial output. Original acknowledgment and fresh GET use [strict confirmation](../../../internal/github/ruleset_bypass_verification_test.go); [pre-write](../../../internal/github/ruleset_bypass_warning_test.go) and [fresh](../../../internal/github/ruleset_bypass_readback_warning_test.go) diagnostics preserve context without supplying successful verification values.

[Export](../../../internal/app/normalization_ruleset_bypass_export_test.go), [presentation](../../../internal/presentation/ruleset_bypass_test.go) and [private reports](../../../internal/actions/normalization_ruleset_bypass_test.go) verify ordering, owned callbacks, ordinary silence, malformed/forged metadata, removal and privacy. The [built-executable sequence](../../../tests/acceptance/normalization_ruleset_bypass_test.go) verifies current export, plan, reports, Summary, literal updates, ignored raw drift, canonical repetition, actual removal, unlisted-ID privacy and file preservation. These local fixtures use dummy credentials and establish no live grants, protection changes or release behavior.

### Ruleset Allowed merge methods normalization

[Mapping](../../../internal/settings/ruleset_merge_methods_mapping_test.go), [owned state](../../../internal/settings/ruleset_merge_methods_observation_test.go), [closed changes](../../../internal/settings/ruleset_merge_methods_change_test.go) and [reconciliation](../../../internal/settings/ruleset_merge_methods_reconcile_test.go) cover typed raw lists, deep ownership, invalid evidence beside valid companions, fallback equality, declaration defaults and PR-rule removal. The [pure projector](../../../internal/github/normalization_ruleset_merge_methods_test.go) rejects null elements and duplicates before whole-list projection; the [complete reader](../../../internal/github/ruleset_merge_methods_read_test.go) requires complete policy and pagination with no partial observation. [Strict confirmation](../../../internal/github/ruleset_merge_methods_verification_test.go) and [fresh diagnostics](../../../internal/github/ruleset_merge_methods_readback_warning_test.go) keep raw verification independent from warnings.

[Export](../../../internal/app/normalization_ruleset_merge_methods_export_test.go), [presentation](../../../internal/presentation/ruleset_merge_methods_test.go) and [private reports](../../../internal/actions/normalization_ruleset_merge_methods_test.go) verify scoped owned callbacks, raw-order/named-choice display, typed closed lists, removal and privacy. The [built-executable sequence](../../../tests/acceptance/normalization_ruleset_merge_methods_test.go) checks export, plan, report/Summary, literal updates, rejected raw drift, canonical repetition, actual removal, unlisted privacy and destination-preserving errors against the candidate. All use isolated local APIs and dummy credentials; they establish no live settings change or published artifact behavior.

### Ruleset scanning Alerts normalization

[Domain tests](../../../internal/settings/ruleset_scanning_alerts_observation_test.go) cover shared UI mapping, exact tool identity, owned raw/effective state, invalid evidence, closed changes, selected reconciliation and removal/replacement. [Reader tests](../../../internal/github/ruleset_scanning_alerts_read_test.go) cover the pure projector, complete acquisition, all-or-zero failure and independent companion evidence. [Strict confirmation tests](../../../internal/github/ruleset_scanning_alerts_verification_test.go) verify original acknowledgment and fresh GET, scoped warnings and honest uncertain effects.

[Report/presentation tests](../../../internal/actions/normalization_ruleset_scanning_alerts_test.go) exercise empty raw strings, closed metadata, exact ordering, escaping, ownership and status-only privacy; [export tests](../../../internal/app/normalization_ruleset_scanning_alerts_export_test.go) cover the independently optional sink and acquisition/save boundaries. The [built-executable regression](../../../tests/acceptance/normalization_ruleset_scanning_alerts_test.go) checks current export/plan/report/Summary/apply, literal payloads, raw drift, canonical repetition and destination preservation. No test changes live settings or establishes a published artifact's behavior.

### Ruleset scanning Security alerts normalization

[Domain tests](../../../internal/settings/ruleset_scanning_security_alerts_observation_test.go) cover the five-choice mapping, named fallback, exact identities, owned raw values, independent companion records, closed changes and selected intent/removal. [Reader tests](../../../internal/github/ruleset_scanning_security_alerts_read_test.go) cover complete typed acquisition, simultaneous unknown thresholds and all-or-zero failure. [Strict confirmation tests](../../../internal/github/ruleset_scanning_security_alerts_verification_test.go) retain independent original acknowledgment/fresh GET and contextual warnings.

[Report/presentation tests](../../../internal/actions/normalization_ruleset_scanning_security_alerts_test.go) validate separate records together, explicit empty strings, ownership, escaping and privacy. [Export tests](../../../internal/app/normalization_ruleset_scanning_security_alerts_export_test.go) cover independent sinks and save boundaries. The [built-executable regression](../../../tests/acceptance/normalization_ruleset_scanning_security_alerts_test.go) verifies current commands, exact mutations, strict failure, no-op repetition and both thresholds on one tool. Local fixture evidence does not establish live protection, hosted CI or a released artifact's behavior.

## Actions and Copilot

### Actions UI mapping

[Actions mapping fixtures](../../../internal/github/ui_actions_composition_test.go) connect literal UI declarations to existing exact request/readback behavior. The [mapping reference](../ui-actions-mapping.md) links the owning enum, singleton, acquisition, retention and fork tests. Codec and process tests compare declared values with independently expected native payloads.

### OIDC subject template

[Owner tests](../../../internal/github/oidc_subject_test.go) cover complete active
observations, exact ordered keys, missing immutable no-op versus write blocking,
true/false companion replay, default transitions, bounded acknowledgments and
independent identity/state readback. [Application tests](../../../internal/app/oidc_subject_test.go)
cover independent export retention, managed counts and verified prior progress.
The [built CLI lifecycle](../../../tests/acceptance/oidc_subject_test.go) verifies
export, plan, repeated apply, failure and raw-debug privacy against an isolated API;
the
[supplied-product composite](../../../tests/github-actions/oidc_subject_acceptance_test.go)
checks actual installer/runtime/report/output behavior and is required by the
candidate gate. These fixtures issue no real tokens or cloud requests and do not
prove live permissions or server acceptance of an empty custom key list.

The
`oidc_immutable_test.go` owner tests distinguish absent from false, selected units
from fresh companions, leaf retention from acquired data and diagnostic failures
from save success. Adapter fixtures cover exact custom-key replay, known no-ops,
400/422 without retry, prefix changes and requested/companion readback drift.
The supplied CLI and additional scenarios within `TestBuiltCompositeOIDCSubject`
exercise immutable export, plan, apply/repeat, unknown evidence, privacy, unchanged
YAML and success-only output. These do not establish reversible legacy opt-in,
effective token format or live cloud authentication.

### Runner labels and pruning

`internal/settings/runners_test.go` owns explicit-set validation, Unicode
simple-fold identity and immutable observations. Workspace literal round-trip
and invalid-shape tests cover the YAML boundary; these include ID checks and explicit empty custom label sets. GitHub owner tests cover selected-only reads, complete
pagination, optional-ID evidence, classifications/counts, conditional no-ops,
exact replacement/clear, acknowledgment and independent readback failures.
App tests verify family-local all-selection preparation, actual-write progress,
unarchive ordering, independent export retention and diagnostic save failures.
Action report tests enforce the closed private schema, exact large numeric IDs and family counts.
`TestRunnerLabelsLifecycle` runs the built CLI, while `TestBuiltCompositeRunnerLabels`
runs the supplied product through the actual installer/composite for export,
plan, status-only privacy, change/repeat/clear and failed verification without
output publication or input edits. These local fixtures do not certify live
labels, server Unicode semantics, host state or job execution.

Runner pruning additionally uses `TestRunnerPruneCompleteAndFailClosed`,
`TestRunnerPrunePaginationAndCompleteInventory`, and
`TestRunnerPruneDeleteRequiresFreshEvidenceAndIndependentAbsence` for private
acquisition and fixed-ID removal. App tests cover global preflight, own label
and unarchive overlays, and partial progress. `TestRunnerPruneBuiltCLI` and
`TestBuiltCompositeRunnerPrune` exercise named and all selectors with the built
product and local fixtures; they do not establish live runner or job behavior.

### Actions event policies

[Actions event policy scenarios](../features/actions-event-policies.feature.md)
map strict codec, domain reconciliation, complete API acquisition/mutation,
application sequencing, typed reports and snapshot-ID recording to their owners.
[Process acceptance](../../../tests/acceptance/actions_event_policies_test.go)
checks the built CLI's automatic collection export, create/update/repeat apply,
partial ID recovery and destination preservation. The [actor-policy process
fixture](../../../tests/acceptance/actions_actor_policies_test.go) also checks actor-only
creation, combined policies, omitted-rule/target preservation, checked-empty versus
explicit removal, complete private reports, status-only privacy, unsupported actor
isolation and recoverable creation failure. These local fixtures use dummy
credentials and cover active/disabled metadata transitions; actual enforcement
and workflow execution were not tested.

### Cache retention and size

The [built-product fixtures](../../../tests/acceptance/actions_general_controls_test.go) cover independent acquisition, retained prior declarations on unavailable reads, validation, numeric planning and credential-free Summary output. They verify one-member PUTs, a fresh GET, unchanged limits on repeat apply and failure on refused writes or mismatched readback. The [cache specification](../features/actions-general-controls.feature.md) defines the public controls.

## Deployment

### Pages UI mapping

[Pages mapping fixtures](../../../internal/github/ui_pages_composition_test.go) connect literal UI declarations, acquired raw state and exact existing-site updates/readback. The [mapping reference](../ui-pages-mapping.md) identifies source/folder, nullable-domain, ownership and metadata checks. Current public/process and lifecycle-limit coverage remains with the existing Pages tests.

### Environment UI mapping

[Environment mapping fixtures](../../../internal/github/ui_environments_composition_test.go) connect literal UI declarations to complete raw reads, exact name-addressed PUT and separately owned typed pattern CRUD. The [mapping reference](../ui-environments-mapping.md) links builtin, principal, finite-choice, acquisition/metadata and partial-ownership preservation tests. Current UI schema/process tests exercise the mapping with fresh existing-policy companions, matching no-ops, API-controlled creation omissions and original partial private report semantics.

### Environment protection Apps

[Owner fixtures](../../../internal/github/environment_custom_rules_test.go) check
complete enabled and paginated available inventories, strict identities/statuses,
add-before-remove capacity, omitted companions, independent readback, recovery
and the retained parent-prune gate. [Built CLI cases](../../../tests/acceptance/environment_apps_test.go)
exercise validation/export, unavailable whole-family retention, debug privacy,
replacement/no-op lifecycle, omitted bindings and failed-readback recovery.
[Supplied-product Action checks](../../../tests/github-actions/environment_apps_acceptance_test.go)
run the actual installer and plan/apply steps, including status-only privacy,
unchanged configuration and success-only outputs. The candidate selector includes
these cases alongside every earlier product scenario. Fixtures do not establish
live activation, App grants or combined built-in/custom capacity counting.

### Environment deployment pattern ref types

[Reader tests](../../../internal/github/environment_ref_type_read_test.go) own complete pagination, raw identity retention and genuine projected collisions. [Update tests](../../../internal/github/environment_ref_type_update_test.go) and [strict creation tests](../../../internal/github/environment_ref_type_update_failure_test.go) exercise declared intent, canonical companion IDs, omitted/empty lists, raw deletion confirmation and independent final readback. Detailed malformed/identity matrices stay at their decoder, reconciliation and deletion owners.

[Executable export](../../../tests/acceptance/normalization_environment_ref_type_test.go) verifies exact diagnostics, literal UI policy, latent privacy and destination preservation for collisions or incomplete acquisition. [Executable command composition](../../../tests/acceptance/normalization_environment_ref_type_commands_test.go) verifies fallback-equal write intent, private report evidence, detailed/status-only Summary, literal ordered mutations, canonical repetition and honest final-readback failure. All use isolated local APIs and dummy credentials; none establishes live mutation or a published artifact's behavior.

## Security

### Security UI mapping

[Security mapping fixtures](../../../internal/github/ui_security_composition_test.go) connect literal UI booleans with existing acquired raw status, narrow PATCH and readback behavior. The [mapping reference](../ui-security-mapping.md) identifies reporting and immutability endpoint fixtures, presence, dependency, applicability and owner-known metadata checks. The [security scenarios](../features/security.feature.md) link these API checks to CLI and Action reporting.

[CodeQL threat-model acceptance](../../../tests/acceptance/codeql_threat_model_test.go) covers both UI/API choices, threat-only payloads, plan/report and private Summary, repeated no-op, unavailable-detail retention, disabled export cleanup and failure stopping before later setup work. [Configuration adapter tests](../../../internal/github/codeql_configuration_test.go) own strict raw decoding and independent readback, including unknown values, omitted companion drift and transport failures. The [configuration process case](../../../tests/acceptance/codeql_configuration_test.go) also verifies omitted threat-model preservation while other controls change. These fixtures confirm stored-configuration behavior, not completed analysis or live entitlement.

### Repository AI Scan

The [security specification](../features/security.feature.md#ai-scan-for-repositories)
maps strict domain/API cases and the built CLI and actual supplied-product Action
scenarios. They cover automatic export retention, independent PR status, native
booleans, narrow PATCH, no-op repeat, target/companion readback failure, unchanged
checkout and private/success-only publication. No live scan or write is performed.

## Access and integrations

### Personal collaborators and invitations

[Model tests](../../../internal/settings/collaborators_test.go) cover user-ID selection,
case matching, duplicate identities and pending versus active observations.
[Adapter tests](../../../internal/github/collaborators_test.go) cover full pagination,
strict targets/roles, no-op races, bodyless requests, independent 201/204 readback
and uncertain effects. [Application tests](../../../internal/app/collaborators_test.go)
separate local binding from remote progress and retain whole export families.
[Workspace tests](../../../internal/workspace/ui_collaborators_test.go) exercise
safe snapshot binding and codec shape. [Action reports](../../../internal/actions/collaborators_test.go)
keep matching pending state visible and protect status-only publication.
[Built CLI scenarios](../../../tests/acceptance/collaborators_test.go) verify lifecycle,
validation before requests, file persistence, export retention and recovery.
[Supplied-product composite checks](../../../tests/github-actions/composite_collaborators_acceptance_test.go)
exercise the actual installer and Action plan/apply steps against a local service.
These are fixture observations, not live invitations, access grants, App
permission verification or recipient acceptance.

The supplied-product collaborator-prune composite uses an installation-shaped
fixture with one explicit token for `/user`, complete installation target access,
admin invitation inventory and both DELETE endpoints. It exercises detailed and
status-only plans, comma scopes and `all`, no-writeback, repeated apply, bounded
readback failure and success-only outputs. It does not establish live installation
grants, recipient notification or GitHub's asynchronous removal side effects.
Local CLI lifecycle fixtures explicitly set `GITHUB_ACTIONS=false`; run the full
CLI suite with host `GITHUB_ACTIONS=true` to verify that isolation.

### Deploy keys

The [Deploy keys specification](../features/deploy-keys.feature.md) maps parser,
complete transport, application sequencing, snapshot binding and report tests.
The supplied-product
`CompositeDeployKeys` scenario uses the actual installer/CLI/Action against a local
API with explicit token and `GH_DEBUG=api`: detailed/status-only plan, create without
writeback, collision, bound no-op, ordinary empty preservation and unconfirmed
prune failure. No real key is installed, generated, deleted or used for SSH.

### Existing webhook configuration

Model and
workspace tests cover partial ownership, exact URL bytes, strict shapes, set
preservation and closed report projections. REST tests cover fixed-route complete
pagination, selected-only reads, full identity/parent/config coherence, partial
PATCH, complete ACK, independent selected/omitted-control verification and bounded
private diagnostics. App checks cover no-ops, whole-family preparation, actual
partial progress and export retention/output failures.
`TestWebhooksExecutableLifecycle` and `TestWebhooksExecutableExportRetention`
exercise the built CLI, including unarchive ordering and unchanged input files.
`TestBuiltCompositeWebhooks` uses the supplied product and actual public Action for
export, detailed/status-only planning, change/repeat and failed confirmation.
Secret/body/member-name canaries remain private under `GH_DEBUG=api`. These local
fixtures do not certify live secret equality, receiver behavior or delivery.

### Webhook creation

Model tests cover explicit full creation, guards, exact URL collisions, opaque and
empty event sets, closed create/update report unions and expected inventory
advancement without aliasing. Adapter tests assert the exact POST, successful-201
identity recovery independent of acknowledgment validity, fresh detail/config/full
inventory confirmation, mixed-update drift, strict target/SSL/identity checks and
bounded diagnostics. App tests distinguish attempted, verified and locally recorded
progress across mixed and multiple writes; source tests preserve comments and
reject stale/aliased recording.

`TestWebhookCreationExecutableLifecycle` runs real CLI creation, guarded ID
writeback, repeat no-op, no-writeback/hosted manual binding, unarchive ordering,
whole-family guards, collisions and HTTP/ACK/readback/local/late failures.
`TestBuiltCompositeWebhookCreation` exercises the actual public Action with the
supplied binary, detailed/status-only privacy, exact outputs, unchanged source,
recovery-only IDs and no duplicate POST on IDless repeats. Existing complete export
checks now require Active/events and retain unavailable prior IDless intent.
`GH_DEBUG=api` and private secret/body/delivery canaries exercise the logging boundary.
These are local fixtures, not live delivery or secret-preservation certification.

### Webhook SSL verification and pruning

Webhook fixtures cover native `insecure_ssl` zero/one mapping, omission and
explicit boolean selection through export, creation, selected update and prune.
Detailed CLI and supplied-product Action reports show disabled-verification risk;
status-only hides values. Mixed operations retain fixed candidates and verified
ID recovery. Fixtures do not prove live TLS validation, delivery or Secret equality.

The built CLI and supplied-product Action mixed-prune fixtures also reject a
failed DELETE after an update and confirmed creation. Both retain the earlier
changes and created-ID recovery diagnostics, report uncertain deletion effects,
and leave the undeleted candidate in the local fixture. The CLI records the
created ID locally when enabled; the Action leaves the settings file and checkout
HEAD unchanged and publishes no successful outputs on failure. Secret and delivery
canaries remain private in logs, Summary and Action outputs. This is local failure
evidence, not a live webhook operation.

## Existing v0 contracts outside v1

The CLI and public Action retain the v0 Autolink and classic branch protection contracts. These tests verify existing behavior. The [support policy](../../references/settings/scope.md#feature-adoption-scope) defines the v1 exclusions.

### Autolink references

The [behavior specification](../features/autolinks.feature.md) maps model, adapter, app, codec and report ownership. Supplied CLI checks exercise export/validate/plan/create/ID-recording/prune/repeat, immutable failures, retained unavailable export, own unarchive and host Action-environment isolation. `TestBuiltCompositeAutolinks` uses the actual supplied binary and public composite for export, private/detailed plans, matching/no-op, create/prune, empty/all, unchanged YAML and failed-readback output suppression. All use local transports; no live reference or external destination is changed or fetched.

### Classic branch protection pruning

Classic whole-rule pruning has model/adapter/application safety checks in the
`classic_prune_test.go` owner files and actual CLI coverage in
[`tests/acceptance/classic_prune_test.go`](../../../tests/acceptance/classic_prune_test.go).
The [supplied-product Action case](../../../tests/github-actions/classic_prune_acceptance_test.go)
checks private snapshots, both reporting modes, standalone `all`, repeat no-op
and unchanged YAML using the candidate artifact. It does not delete live rules.

### Branch locking

The supplied CLI lifecycle covers acquired false export, lock changes and repeated
apply. The supplied-product ClassicLock composite exercises lock/unlock, matching
no-op, detailed/private plans, exact changed-only GraphQL input, independent
fork-sync drift failure and success-only outputs. Model and adapter fixtures also
check omission, complete acquisition and preservation of fork-sync true/false.
These fixtures do not establish live branch enforcement or API side effects.

### Administrator enforcement

Model and adapter tests cover both acquired boolean values, clone isolation,
omitted-state preservation, direct true/false mutation inputs, creation ACK and
independent readback failures. The supplied CLI lifecycle exercises explicit false;
`TestBuiltCompositeClassicAdmin` runs the actual public Action for both directions,
matching repeat, private plans and collateral fork-sync drift. The Action keeps
checkout YAML unchanged and publishes outputs only after confirmed success.
Fixture results do not prove live administrator access or branch enforcement.

### Required status checks

Owner tests cover the strict active union, exact names and
explicit sources, alias coherence, raw inactive preservation, set-order no-ops,
activation with false/empty children, minimal updates and complete acknowledgment.
Adapter tests also cover deduplicated App lookups, prewrite and postwrite failures,
raw companion drift and independently verified policy. Combined deployment cases
check simultaneous edits and deactivation, both independent reference checks,
repeated no-ops and collateral drift in either omitted group. The supplied CLI lifecycle
checks export/validate/plan/apply/repeat and creation recovery without ID writeback
after failed App confirmation. `TestBuiltCompositeClassicStatusChecks` checks the
actual public Action with an explicit token, detailed/status-only plans, matching
and repeated no-ops, changed apply, unchanged YAML and failure output suppression.
These local fixtures do not establish live App grants, storage or enforcement.

### Required deployments

Owning model and
adapter tests distinguish active empty/null/missing states, exact-name sets and
raw inactive preservation. They verify nullable numeric Environment IDs, complete
pagination, changed-only inputs, no-op without Environment access, reference drift,
partial creation recovery and other companion preservation. App cases prove that
Environment-prune conflicts fail before unrelated writes or unarchive.

The supplied CLI lifecycle checks export, plan, update and repeat no-op.
`TestBuiltCompositeClassicDeployments` runs the real public Action through enabled,
disabled, empty and matching configurations, private plans and post-write reference
replacement. `TestClassicDeploymentReferenceDiagnostics` covers GraphQL, HTTP,
transport and malformed-member acquisition failures before/after a mutation,
including response-controlled member names. The real CLI lifecycle and supplied
Action include pre/post-write diagnostic regressions; the Action also uses a
malformed HTTP response to exercise the actual transport failure.
These checks retain mutation counts, certainty markers, unchanged checkout and
success-only outputs. These isolated fixtures do not establish live deployment
execution or effective branch protection.

### Required pull request reviews

Model and adapter fixtures
check complete active unions, counts zero/one/six and unavailable values, clone
isolation, activation from null/populated latent values, explicit all-five inputs,
changed-only active edits and false-only deactivation. Omitted raw state remains
exact; selecting disabled mode permits hidden-child reset. Plan and fresh checks
reject dependent bypass/dismissal protection on deactivation, while disabled no-op
and count-zero preserve those unowned identities. Independent readback, incomplete
acknowledgment and partial creation recovery retain existing failure boundaries.
Combined review/deployment cases verify simultaneous edits and deactivation,
matching repeats without Environment reads, and rejection of collateral drift in
either omitted group after a write. The combined status/review/deployment adapter
cases additionally verify simultaneous creation and update, all-group disable,
matching no-ops without reference lookups, omitted-group drift, and a fresh PR
dependency blocking the whole combined update.

The supplied CLI exercises export, validation, plan, count-zero/update/disable and
matching repeats. `TestBuiltCompositeClassicReviews` uses the actual public Action
for activation, count-zero, deactivation, no-op, guarded failure and readback drift,
checking private plans, unchanged checkout and success-only outputs. These isolated
fixtures do not establish live approval behavior or effective merge enforcement.

### Fork syncing

The [Classic specification](../features/classic-protection.feature.md#rule-manage-fork-syncing-only-on-a-locked-fork)
links the literal baseline and model/API/app cases. They distinguish unavailable
fork metadata from false, bind every private target comparison, preserve inactive
raw values and verify prewrite versus uncertain postwrite failures. The CLI fork
cases export and validate both values after application; the actual supplied
`TestBuiltCompositeClassicForkSync` verifies matching, enable/disable, private plans,
readback failure and source immutability. The official UI documentation is evidence
for the control; fixtures do not establish live syncing or branch authorization.
Combined fork/force/PR-review/status-check/deployment cases verify exact create and
update payloads, matching repeats, both raw omission directions, selected matching
sync acknowledgment drift, fork identity drift and prewrite actor lookup failure.

### Force-push actors

The [Classic specification](../features/classic-protection.feature.md#rule-preserve-actors-outside-the-selected-force-push-mode)
links literal codec, model, API, app and product tests. Fixtures distinguish native
false with allowances from disabled, preserve hidden actors under Everyone and
retain known Team for read-only Before/prune. The actual supplied
`TestBuiltCompositeClassicForcePush` covers matching, App selection, explicit
clearing, private plans, required lookup on unrelated changes and uncertain actor
readback. Static identities and local transport do not prove live App access,
force-push success or enforcement. Combined force-push/PR-review/status-check/deployment
cases verify creation, updates, disabling and repeat no-ops, omitted-group drift
and fresh PR transition guards.
