---
type: Feature
title: Repository security design
description: Planned declarations, complete observations, dependencies, and acceptance gates for security controls.
---

## Status and scope

[ADR 0026](../adr/0026-explicit-security-observations.md) records the decisions for [#343](https://github.com/daiksud/gh-settings/issues/343). This specification is the implementation acceptance contract for #344–#347, not a claim that unimplemented families are supported. Implementation PRs must update the public configuration, command, and Action references before claiming support.

Manage repository feature enablement only. Exclude alert contents or dismissal, dependency/version-update files, scanning workflows/default setup/query suites, bypass actors/patterns, advisories/reports/notifications, contract purchases or product migration, and organization/enterprise security configuration attachment, detachment, or editing. Applying an enabled setting can cause GitHub to perform its documented background work; the CLI does not start a separate scan or manage its results.

## Planned declaration and ownership

Current source uses the exact marker `version: daiksud/gh-settings@v0`. Optional booleans under `repository.settings.advanced_security` are:

| Public UI key | Implementation | Scope |
| --- | --- | --- |
| `dependabot.alerts`, `dependabot.security_updates` | #344 | Coupled optional booleans with explicit 404 reading and accepted-disable policies |
| `secret_protection.enabled` | #345 | Repository Secret Protection feature, not partner notifications |
| `secret_protection.push_protection` | #345 | Repository push protection, not user-level protection |
| `private_vulnerability_reporting` | #347 | Public repositories only |

Release immutability belongs separately at `repository.settings.general.releases.immutable_release`; its [contract](immutable-releases.feature.md) owns the explicit GET-404 tradeoff and owner enforcement.

Omitted groups/keys are unmanaged; empty mappings manage nothing. False disables, true enables, and null/wrong types/duplicate/unknown keys fail before remote work. Navigation-category wrappers, scalar Secret Protection, flat push protection and API-shaped aliases are not accepted. The two fields within Secret Protection remain independently optional; declaring the child never populates `enabled`. Empty mappings do not disable all. Product candidates below remain future API concepts and adoption gates, not accepted YAML or CLI families. Dependabot uses the two independent controls above; paused state is not a declaration.

Plan reads only declared families and the prerequisite/companion observations needed to establish a safe transition. Unrelated hidden security fields do not prevent private vulnerability reporting or an ordinary repository-only plan. API response-only metadata, including pause, entitlement, and enforcement evidence, cannot be declared as desired state.

## API, permissions, and availability matrix

Paths below are relative to `/repos/{owner}/{repo}`. Permission names are the documented fine-grained repository permissions, not proof of a token's actual grants. Classic OAuth/PAT scopes, user roles, App installation scope, SSO, and policy restrictions must also permit the specific operation. Do not infer an entire credential's authority from repository `permissions.admin`, another endpoint's success, or a public metadata read.

| Control | Observation | Mutation | Documented read / write boundary | Availability and upper policy |
| --- | --- | --- | --- | --- |
| Dependabot alerts | GET `/vulnerability-alerts`: 204 enabled; 404 interpreted as disabled by explicit decision, retaining provenance | PUT / DELETE same path: exactly 204 | Administration read / write; authenticated admin access | Public/private; basic Dependabot is available across plans. Endpoint also enables/disables dependency graph; upper security configuration can constrain changes. |
| Dependabot security updates | GET `/automated-security-fixes`: 200 with required boolean `enabled` and `paused`; 404 interpreted as disabled with pause unknown | PUT / DELETE same path: exactly 204 | Administration read / write; authenticated admin access | Public/private; requires dependency graph and alerts. Enabled can be paused; does not mean a PR can be produced for every ecosystem/dependency. |
| Secret scanning | GET repository, required `security_and_analysis.secret_scanning.status` enum | PATCH repository with only declared nested status | Metadata read for repository endpoint plus admin/owner/security-manager visibility; Administration write and applicable role for security PATCH | Public feature availability differs from private/internal Secret Protection entitlement. Partner scanning is separate. Upper configuration can enforce either state. |
| Repository push protection | GET repository, required `security_and_analysis.secret_scanning_push_protection.status` enum | Same narrow nested PATCH | Same visibility / update requirements | Secret scanning prerequisite; public availability does not establish private/internal entitlement. User push protection is not this setting. |
| Code Security candidate | GET repository, `security_and_analysis.code_security.status` if completely visible | Candidate nested `code_security.status` PATCH | Same repository visibility / Administration write boundary, subject to adoption evidence | Standalone product; private/internal use needs applicable existing entitlement. Public scanning availability alone does not establish a writable product toggle. |
| Legacy Advanced Security candidate | GET repository, `security_and_analysis.advanced_security.status` if completely visible | Candidate nested `advanced_security.status` PATCH | Same repository visibility / Administration write boundary, subject to adoption evidence | Legacy/bundled product. API explicitly disallows this parameter for standalone Code Scanning or Secret Protection products. Never substitute it for `code_security`. |
| Private vulnerability reporting | GET `/private-vulnerability-reporting`: 200 with required boolean `enabled` | PUT / DELETE same path: 204 | Metadata read / Administration write; admin access for update | Public repositories; private/internal target is inapplicable even when desired false. No paid-product prerequisite. Do not alter advisory/report data. |

The permission matrix records the [repository REST reference](https://docs.github.com/en/rest/repos/repos) and the [official OpenAPI description](https://github.com/github/rest-api-description/blob/755ee33b4a81d51f6f07a902cce26cbec3b822f2/descriptions/api.github.com/api.github.com.json) reviewed on 2026-10-20. The latter separately specifies both product PATCH fields, visibility requirements, and Dependabot response shapes. Repository endpoint permission is necessary but does not guarantee visibility of each nested feature. A missing/null `security_and_analysis`, missing/null requested child/status or wrong type is incomplete observation, never disabled. Complete Secret Protection/Push protection strings outside UI choices follow the typed normalization rules below; unknown unrequested sibling fields are not replayed into requests.

For plan/entitlement boundaries see [GitHub security features](https://docs.github.com/en/code-security/getting-started/github-security-features) and [Advanced Security products](https://docs.github.com/en/get-started/learning-about-github/about-github-advanced-security). Basic dependency security is distinct from premium product capabilities. Do not purchase, trial, enroll, or switch products to make a failed read/write succeed. Existing organization entitlement is a prerequisite, not something gh-settings provisions.

### Dependabot state-reading decision

The [scoped decision in ADR 0026](../adr/0026-explicit-security-observations.md#dependabot-state-reads) accepts GET 404 as disabled for both Dependabot endpoints, intentionally giving up the distinction from insufficient permission. No authorization discriminator, successful unrelated endpoint, token inspection or GraphQL replacement is required. This choice can misclassify inaccessible settings as disabled, producing false differences/equality or false exported values in public plan/export. Retain 404 provenance rather than representing it as direct boolean evidence. Explicit 401/403 and all other unsuccessful statuses remain errors.

The [Dependabot adapter](../../internal/github/dependabot.go) uses the existing security REST version/timeout and redirect rejection. Each family observation completely acquires alerts then updates with one GET per endpoint and returns no partial family on failure. Only declared family plan/apply or explicit export requests these reads. No credential-discovery requests are made.

#### Response evidence and pause metadata

The [official OpenAPI description at `f9ed742`](https://github.com/github/rest-api-description/blob/f9ed74230d54847b5ca00a1343e8e349e858b8fb/descriptions/api.github.com/api.github.com.json), reviewed on 2026-11-01, specifies alerts GET 204/404, updates GET 200/404, and body-free 204 acknowledgments for both PUT/DELETE pairs. The updates schema requires both `enabled` and `paused` booleans without constraining their relationship. Accept all four complete pairs. A 200 containing false is direct boolean evidence if returned; the schema does not promise disabled repositories return 200 instead of 404. Missing/null/wrong-type members, malformed JSON and unexpected successful statuses acquire no observation.

The [typed observations](../../internal/settings/dependabot.go) distinguish unacquired state, direct endpoint evidence, and disabled-by-404 state. Updates 404 acquires enabled=false but no pause metadata. The `paused` value returned alongside `pauseKnown=false` is unavailable, not an observed false; presentation preserves this distinction. Complete 200 metadata remains independent of enabled, including enabled=false/paused=true.

[GitHub's REST troubleshooting guidance](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api#404-not-found-for-an-existing-resource) explains authorization-concealing 404 responses. The [App permissions reference](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps) describes `X-Accepted-GitHub-Permissions` as endpoint requirements, not the caller's actual grants. Neither resolves the ambiguity accepted by the product decision.

| Evidence | State-reading interpretation | Post-update verification |
| --- | --- | --- |
| Alerts REST 204 | Enabled, direct evidence | Can verify enabled=true, never disabled. |
| Updates REST 200 with both booleans | Direct enabled state plus known pause metadata | Verify declared enabled independently of pause; mismatch fails. |
| Either REST GET 404 | Disabled by accepted interpretation; permission may be insufficient | Cannot verify either desired value, including false. |
| Updates DELETE 204 followed by GET 404 | Acknowledgment plus interpreted disabled read; pause unknown | Accepted disable with explicit uncertainty after the separately adopted decision; may proceed to an explicitly declared parent disable. |

[Dependabot's inactivity guidance](https://docs.github.com/en/code-security/reference/supply-chain-security/troubleshoot-dependabot/dependabot-updates-stopped) distinguishes automatic pause from disablement and lists enabling security updates among activity that can reactivate updates. It does not specify an HTTP PUT pause guarantee. Therefore boolean intent true means enabled, not resume: fresh enabled=true/paused=true must skip PUT and retain a diagnostic. After an enable, acquire both booleans again without promising paused=false or running pull requests. Pause changes alone neither request an extra write nor fail verification of matching enabled intent.

#### Mutation and accepted-disable contract

The [separate accepted-disable decision](../adr/0026-explicit-security-observations.md#dependabot-accepted-disable) permits DELETE 204 plus a fresh target GET 404. The result is accepted with a warning/notice and `OperationAcceptedUnverified`, never independent verification. Permission loss or enforcement may be hidden by the interpreted false. Observation-only `VerifyEnabled` still rejects 404; the adapter requires the actual 204 acknowledgment before accepting this narrow outcome. PUT followed by 404, any non-204 write, target mismatch, companion mismatch or incomplete/failed readback remains a failure with possible effects.

Apply observes the complete family before selecting changes. Every changed member rereads both controls immediately before its body-free PUT/DELETE, rechecks prerequisites, then reads both controls again. Matching fresh intent skips the write, including enabled=true/paused=true. A changed alerts value must preserve the interpreted updates value; a changed updates value must preserve interpreted alerts. These checks inherit the explicit 404 reading risk and do not prove authorization or actual preservation behind a concealed response. Pause is acquired when available and diagnosed, not replayed or compared as desired state.

Enable alerts before updates; disable updates before alerts. An accepted child disable can precede the explicitly declared parent disable under the accepted uncertainty policy. Never enable an omitted parent, disable an active omitted child, or send undeclared siblings. The alert endpoint's intrinsic dependency-graph enable/disable effects are disclosed. Unexpected companion changes fail instead of inventing GitHub's side effects. Local contradictions fail before remote work; missing prerequisites and failed complete family reads fail before this family's first write. Later failure preserves verified and accepted-unverified prior progress with its warning/notice; no automatic retry or rollback occurs.

#### Implemented Dependabot test mapping

- [Observation tests](../../internal/settings/dependabot_test.go) cover unacquired values, explicit disabled-by-404 provenance, unavailable pause on 404, all four enabled/paused pairs and rejection of 404 for update verification.
- [Endpoint fixtures](../../internal/github/dependabot_test.go) cover exact selected GET targets, 204 alerts, 200 updates, disabled-looking and permission-looking 404 bodies with the same deliberate interpretation, malformed/incomplete observations, 401/403/other statuses, rejected redirects, body/transport failure and no acquired state on error. They also exercise verification against actual decoded endpoint observations.
- [Toggle fixtures](../../internal/github/dependabot_update_test.go) check exact body-free methods, fresh prerequisites, paused no-write equality, positive readback, DELETE 204 plus 404 acceptance (including a permission-loss fixture retaining enabled state), ignored writes, companion drift, rejected redirects and pre-/post-write certainty.
- [Application fixtures](../../internal/app/dependabot_test.go) check dependency order, matching/omitted intent, local contradiction, read failure and accepted-unverified progress on later failure. [UI codec](../../internal/workspace/ui_dependabot_test.go) and [closed Summary](../../internal/actions/dependabot_test.go) fixtures retain optional booleans and native field bounds.
- [Built-CLI cycle](../../tests/acceptance/dependabot_test.go) checks ordinary/explicit export, false round-trip to zero plan, native report and credential-free Summary, ordered enable/disable, paused no-write behavior, omitted-parent preservation, accepted-child/later-parent failure notices and destination preservation on incomplete/denied reads.

All fixtures use local APIs and dummy credentials. They establish modeled behavior, not live permission grants, update effects, enforcement or scan activity.

### Product candidate gate

For #346, retain separate product concepts and inventory UI paths before adoption. A future adoption must identify the existing product model from authoritative evidence, define the exact supported visibility/entitlement cases, and establish companion effects for both enable and disable. Do not infer a model solely from a missing field or attempt the other product after a failed request. Reject both candidate keys in one declaration; neither requests conversion or purchase. Unknown applicability blocks that family, not independent security features.

A product-level toggle can affect subordinate security features. Before adoption, specify which companions must be observed and explicitly declared for each destructive transition, and test preservation or acknowledged intrinsic effects. The current design deliberately does not adopt an unverified product mutation. #346 remains pending this evidence/adoption decision; #345 and #347 do not depend on it.

## Explicit export selection

Repeatable `--security FAMILY` accepts exactly the implemented `secret-scanning`, `dependabot`, `private-vulnerability-reporting` and `immutable-releases` selections. Product options and `all` are not accepted. Distinct families compose with other export options; unknown/missing/duplicate values fail before reads or file changes.

Ordinary export adds no security requests. Secret protection/Push protection reads precede Dependabot, private reporting, then immutable policy. Complete selected acquisition and UI conversion before destination replacement. Inapplicable, denied or incomplete observations fail instead of being omitted with success. Complete out-of-UI status strings retain provenance and normalize to the named false fallback; inconsistent projected declarations fail without saving. Export observed false, exclude owner/entitlement metadata and never replay internal API names as public intent. An exact supported unchanged export has zero differences without proving writable access, active scanning or an unlocked policy. Normalization is active for these two status strings; other families follow their documented policies.

## State and failure table

| Observation or event | Required result |
| --- | --- |
| Complete enabled/disabled boolean or supported status enum | Compare native booleans; exact equality yields no difference. |
| Complete out-of-UI Secret Protection/Push protection status string | Retain raw provenance and named false fallback; declared equality remains write-required, never independent proof of raw equality. |
| Updates `enabled: true, paused: true` | Preserve enabled=true and pause metadata; emit a diagnostic in plan/export/apply. Do not report running/healthy, synthesize false, or disable/re-enable to resume. |
| Any other complete `enabled`/`paused` boolean pair | Compare enabled independently and retain pause metadata; warn whenever paused. Do not invent a correlation between the two booleans that the API schema does not establish. |
| Updates 200 with missing/null/wrong-type enabled or pause metadata | Reject incomplete state; do not discard pause to manufacture a usable observation. The scoped 404 interpretation separately retains unavailable pause. |
| Missing/null/wrong-type requested response field | Fail that read; no successful empty/false comparison or export. |
| Dependabot GET 404 | Interpret enabled=false by explicit product decision; retain provenance and unavailable pause metadata; never use it alone to verify a mutation; only DELETE 204 plus target 404 qualifies for the explicit accepted-disable outcome. |
| 401/403, unqualified 404 outside the scoped Dependabot/immutable policies, redirect, transport failure, invalid JSON on successful body-bearing reads | Fail read as unknown/denied as evidenced; never infer disabled or follow a changed target. |
| Confirmed unsupported visibility/entitlement | Report inapplicable/unavailable as evidenced; never false. Do not guess licensing from arbitrary 403/404 text. |
| Authoritative upper-policy conflict known before write | Fail before mutation of that security family; no detach or policy bypass. |
| Fresh prerequisite read/preflight fails | Record a confirmed pre-write read failure, preserving prior completed operations. |
| Write returns 2xx but readback differs, is incomplete, or fails, outside the explicit Dependabot DELETE 204/GET 404 acceptance | Fail verification; mutation effects are uncertain. Do not count success, retry, or roll back automatically. |
| Earlier resource/family succeeded, later family fails | Retain and report confirmed progress and any uncertain mutation; overall failure. |

Upper [security configuration enforcement](https://docs.github.com/en/code-security/reference/security-at-scale/configuration-enforcement) may make a REST change appear successful while leaving values unchanged. Complete fresh readback is mandatory. The initial feature does not require organization-wide configuration reads or claim to diagnose all enforcement before writing. If available evidence identifies a conflict, surface it; otherwise an unmet readback is a verification failure with no invented cause. Repository declarations never own inherited configuration membership. Changing an unenforced repository value may affect its configuration status; no attachment/detachment request is sent to repair that status.

## Dependency and mutation table

These rules constrain transition preflight; they do not introduce a saved-plan transaction. Dependabot transitions use the explicit reading/accepted-disable policies above; interpreted disabled-by-404 state is not proof of actual companion preservation. Plan is read-only. Apply rereads prerequisites with its own credential immediately before each family's mutation; an earlier plan does not authorize using stale observations. Validate the entire local declaration before any remote operation. Within each family, complete necessary state and transition checks before its first write.

| Desired transition | Required prerequisites and order |
| --- | --- |
| Security updates true with alerts false | Reject contradiction locally. |
| Security updates true; alerts omitted | Require a fresh known-enabled alerts observation; otherwise fail before writes. Never enable omitted alerts. |
| Alerts true and security updates true | Enable/verify alerts before enabling/verifying updates. The alerts endpoint's documented dependency-graph effect is intrinsic and must be disclosed. |
| Alerts false | Observe updates; if currently enabled, require explicit updates=false and disable/verify or explicitly accept the scoped 204/404 outcome first. A failed updates read blocks the transition; a disabled-by-404 read follows the explicit risk policy. An omitted known-disabled child needs no write. The documented graph-disable effect is intrinsic. |
| Push protection true with secret scanning false | Reject contradiction locally. |
| Push protection true; scanning omitted | Require fresh known-enabled scanning; no implicit parent enable or product enrollment. |
| Scanning true and push protection true | Enable/verify scanning before enabling/verifying push protection, with narrow sequential nested PATCHes. |
| Scanning false | Observe push protection; if currently enabled, require explicit `push_protection: false` and disable/verify it first. Unknown child blocks the transition when the companion is omitted; explicit push protection=false must be written and verified first. |
| Child false alone | Update only the declared child when a change is needed; never send omitted parent fields. |
| Private vulnerability reporting true/false | Verify public target and complete current boolean, then PUT/DELETE only if different and GET-verify; independent of paid products and other families. |

The dependency graph/alerts relationship is documented in the REST endpoint descriptions; [Dependabot security updates](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependabot-security-updates) requires both. [Push protection](https://docs.github.com/en/code-security/concepts/secret-security/push-protection) is a secret scanning feature. The conservative explicit-child-disable requirement prevents using a parent declaration to silently change an unmanaged child. Unavoidable GitHub-internal side effects cannot be represented as additional successful CLI writes; disclose them and verify relevant companions where the supported observation contract permits.

For Secret scanning, send only supported declared nested status members, never the complete observed security block. Do not combine parent/child transitions into a PATCH whose order GitHub does not guarantee. Readback checks the desired member and relevant preserved companions; unknown unrelated fields remain outside the owned payload. For each family, preflight must cover every companion it intends to preserve. On an unknown write outcome, stop; rerunning later starts from fresh observations and does not automatically compensate.

Security families append after existing apply resources in the same fixed order as export. Each adapter owns observation/payload/readback; settings owns declarations, transition rules, and native comparisons; app owns operation sequencing and partial progress. Use closed field identities/native booleans and bounded report validation for CLI/Action summaries. Do not invent a generic privileged backend framework. Existing direct apply and no automatic rollback contracts remain unchanged.

## Implementation acceptance and fixture plan

The matrix remains the acceptance plan for all families; the implemented family mappings follow it. Unimplemented rows are planned checks, **not claims of completed tests or live verification**. Each implementation supplies its owning Go test names and public references when merged. Use local fake clients, bounded HTTP fixtures, and built-CLI process tests with dummy credentials. Do not enable scans, change production settings, purchase contracts, or expand permissions for these tests.

| Issue / layer | Required evidence before completion |
| --- | --- |
| All: codec/model | Omitted/group-empty no-op; explicit true/false preserved; null, wrong types, duplicate/unknown keys rejected before network; zero/unacquired observation rejected; copied values cannot corrupt invariants. |
| All: API/app | Correct target and method, declared-only payload, complete fresh prerequisite and readback checks, redirect rejection, error classification before/after writes, deterministic order, earlier successes retained, no automatic retry/rollback. |
| #344 | Intentional disabled/permission-concealed 404 equivalence, unavailable pause, complete 200 pairs, independent verification versus accepted DELETE 204/404, fresh prerequisites, omitted/contradictory dependencies, ordered body-free writes, target/companion failure, partial accepted progress and graph effects; no version-update/individual-alert operations. |
| #345 | Public and entitled/nonentitled private/internal fixtures; hidden/null/unknown status; parent/child combinations and fresh drift; no omitted fields in nested PATCH; enforced 2xx/no-change must fail verification; no bypass/pattern/configuration writes. |
| #346 | Separate adoption evidence first; distinct model/entitlement fixtures, both-key rejection, no alias/fallback/conversion, complete companion effects and lossless supported-state round-trip. No implementation completion while product behavior remains unverified. |
| #347 | Public true/false/equal/omitted, private/internal inapplicability, Metadata-only read shape, missing/null/wrong enabled value, 422/permission errors, PUT/DELETE+readback, no advisory/report/notification operations. |
| Export/process | Ordinary requests unchanged; explicit family composition and invalid options; complete exact supported export→unchanged plan zero; normalized export retains raw-versus-fallback decisions; requested unavailable state leaves destination untouched; false retained; pause diagnostic; no stdout changes. |
| CLI/Action | Native boolean differences, strict report reconstruction and bounds, read/write failure and partial-progress output, escaping of untrusted diagnostics; published pinned CLI must reject unsupported additions rather than pretend source support is released. |

The [private vulnerability reporting guide](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository) establishes public-repository scope. The source/runtime privilege boundary and fixture limitations belong in the public references and testing guide at implementation time. API schema inspection and local fixtures do not prove all token types, billing availability, upper enforcement, or real scan outcomes.

## Implemented Secret scanning test mappings

The [public configuration contract](../settings.md#secret-scanning-and-push-protection) owns current-source usage. The Secret scanning subset uses its two booleans and `--security secret-scanning`, independently of private vulnerability reporting below. Other families and product candidates remain unsupported.

- Declaration/observation/transition/forged-change boundaries: [TestSecretScanningObservationAndTransitions](../../internal/settings/security_test.go) and [TestSecurityComparisonsRejectForgedValues](../../internal/settings/security_test.go).
- Strict YAML, explicit false and omission round-trip: [Security UI codec](../../internal/workspace/ui_security_test.go) and [page/section schema](../../internal/workspace/ui_page_sections_test.go), including nested controls and rejection of superseded shapes.
- Complete API reads, malformed/denied/redirect responses, fresh prerequisites, narrow payload and readback/companion failures: [TestSecretScanningCompleteReads](../../internal/github/security_test.go) and [TestSecretScanningNarrowPayloadReadbackAndFreshPreflight](../../internal/github/security_test.go).
- Dependency order, read certainty, matching state and confirmed partial progress: [TestSecretScanningApplyOrderAndPartialProgress](../../internal/app/security_test.go) and [TestSecretScanningReadFailureAndOmission](../../internal/app/security_test.go).
- Export selection/composition and forged native Summary values: [TestSecurityExportSelectionIsExplicitAndComposable](../../internal/cli/security_test.go) and [TestSecuritySummaryHasNativeClosedValues](../../internal/actions/security_test.go).
- Built-CLI ordinary/explicit export, zero plan, native Action report/Summary, enable/disable/idempotence, unavailable reads preserving files, redirect rejection and enforced/partial writes: [TestSecretScanningExportPlanApplyAndSummary](../../tests/acceptance/security_test.go).

Fixtures use dummy credentials and local modeled responses. A read-only observation of this repository confirmed both enabled statuses and additional unowned siblings; no live PATCH, private entitlement, enforcement or token-grant behavior was exercised or established. No scan, contract, organization setting, workflow permission, grant or release changed.

## Implemented private vulnerability reporting test mappings

The [public configuration contract](../settings.md#private-vulnerability-reporting) owns usage. Current source accepts `repository.settings.advanced_security.private_vulnerability_reporting` and `--security private-vulnerability-reporting`; distinct implemented family selections compose. Product candidates remain gated/unsupported.

- Acquired observations, omitted/false/true comparison and independent field domains: [TestPrivateReportingObservationAndIndependentDeclaration](../../internal/settings/private_reporting_test.go).
- Strict boolean YAML, omission/round-trip, invalid types, duplicate/unknown keys and mixed independent declarations: [Security UI codec](../../internal/workspace/ui_security_test.go).
- Public/private/internal applicability, missing/null/wrong-case/type responses and API errors: [TestPrivateReportingReadApplicabilityAndCompleteBoolean](../../internal/github/private_reporting_test.go).
- Fresh preflight, body-free PUT/DELETE, matching state, upper-policy no-change and readback certainty: [TestPrivateReportingToggleAndCertainty](../../internal/github/private_reporting_test.go). [TestSecurityClientsRejectCrossFamilyChangesBeforeRequests](../../internal/github/private_reporting_test.go) guards both concrete clients.
- Independent reads, matching apply, confirmed progress and pre-/post-write failure: [TestPrivateReportingIndependentPlanAndMatchingApply](../../internal/app/private_reporting_test.go) and [TestPrivateReportingFailureRetainsOnlyConfirmedWrites](../../internal/app/private_reporting_test.go).
- Distinct family composition and native report validation: [TestPrivateReportingExportSelectionAndFamilyComposition](../../internal/cli/private_reporting_test.go) and [TestPrivateReportingNativeSummaryAndIndependentSecurityFields](../../internal/actions/private_reporting_test.go).
- Built-CLI export→zero-plan→edit→apply, native credential-free Summary, false/idempotence, nonpublic applicability, file preservation, GET/PUT/DELETE redirects and partial writes after Secret scanning: [TestPrivateReportingExportPlanApplyAndSummary](../../tests/acceptance/private_reporting_test.go).

Fixtures use dummy credentials and local modeled responses. A read-only GET on this public repository returned enabled=true; no live PUT/DELETE, report/advisory/notification operation, permission expansion, upper-policy change or release was performed. These checks do not establish live update compatibility or other credential grants.

The private same-version security report retains native API member identifiers/booleans and existing bounds. Exact-observation CLI/Summary use UI paths; native paths are not public YAML aliases. Normalized status decisions instead retain typed raw provenance and use their UI paths with boolean write-required decisions.

## Feature: Retain provenance for out-of-UI security statuses

### Rule: Complete typed statuses remain distinct from their effective UI booleans

#### Scenario: A complete successful status is outside the adopted choices

- Given both required Secret Protection and Push protection status strings are acquired
- When one status is outside `enabled` and `disabled`, including empty or whitespace strings
- Then its effective UI value is the named boolean fallback `false`
- And one owned record retains the exact raw string, UI field and typed boolean fallback
- And the supported companion remains unchanged
- But missing/null members, wrong types and failed reads remain acquisition errors
- And an inconsistent projected declaration remains a consistency error

#### Scenario: Managed intent equals or differs from the normalized fallback

- Given a complete normalized observation and an explicitly declared boolean
- When plan compares the declaration with its effective value
- Then it retains one write-required decision with raw provenance even for false-to-false
- And omitted controls remain unmanaged without replaying their defaults
- And malformed managed provenance discards the entire comparison

### Rule: Raw prerequisites and independent readback govern apply

#### Scenario: A companion has an unknown raw status

- Given a complete acquired companion outside the supported status choices
- When its raw state is required to prove a prerequisite
- Then its effective false fallback cannot establish that prerequisite
- And explicit intent must resolve the dependency through safe ordered narrow writes
- But safely unmodified companion raw evidence remains exact through fresh readback
- And unrelated unmanaged raw values are not printed

#### Scenario: Fresh raw confirmation follows a normalized write decision

- Given a declared normalized false-to-false or false-to-true decision
- When apply rereads the complete family before and after the narrow PATCH
- Then fresh canonical target equality can skip a write
- And otherwise the declared member is written and exact raw target/companion values are verified
- And retained unknown target or changed companion status fails with possible write effects
- And earlier completed writes remain applied without rollback

Test mapping: [closed observations](../../internal/settings/security_status_observation_test.go), [typed changes](../../internal/settings/security_status_change_test.go), [comparison](../../internal/settings/security_status_comparison_test.go), [raw dependencies](../../internal/settings/security_status_transitions_test.go), [typed decoder](../../internal/settings/security_status_json_test.go), [private reports and presentation](../../internal/actions/normalization_security_status_test.go), [scoped export](../../internal/app/normalization_security_export_test.go), [raw pair verification](../../internal/github/security_status_verification_test.go), [HTTP normalization/apply](../../internal/github/security_status_apply_test.go), and [executable commands](../../tests/acceptance/normalization_security_status_test.go).
