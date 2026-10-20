---
type: Feature
title: Repository security design
description: Planned declarations, complete observations, dependencies, and acceptance gates for security controls.
---

## Status and scope

**Secret scanning subset implemented in current source, unreleased; other families remain planned/gated.** Published v0.16.0 and older packages reject security keys/options. [ADR 0026](../adr/0026-explicit-security-observations.md) records the decisions for [#343](https://github.com/daiksud/gh-settings/issues/343). This specification is the implementation acceptance contract for #344–#347, not a claim that unimplemented families are supported. Implementation PRs must update the public configuration, command, and Action references before claiming support.

Manage repository feature enablement only. Exclude alert contents or dismissal, dependency/version-update files, scanning workflows/default setup/query suites, bypass actors/patterns, advisories/reports/notifications, contract purchases or product migration, and organization/enterprise security configuration attachment, detachment, or editing. Applying an enabled setting can cause GitHub to perform its documented background work; the CLI does not start a separate scan or manage its results.

## Planned declaration and ownership

The schema version remains 1. Under optional `settings.security`, use these exact optional boolean keys:

| Key | Implementation | Decision |
| --- | --- | --- |
| `dependabot_alerts` | #344 | Adopt subject to the disabled-read gate below. |
| `dependabot_security_updates` | #344 | Same gate; paused state is separate operational metadata. |
| `secret_scanning` | #345 | Adopt the repository feature, not partner notifications. |
| `secret_scanning_push_protection` | #345 | Adopt repository push protection, not user-level push protection. |
| `code_security` | #346 | Reserved candidate; not adopted for implementation yet. |
| `advanced_security` | #346 | Distinct reserved legacy-product candidate; not an alias. |
| `private_vulnerability_reporting` | #347 | Adopt for public repositories only. |

Omitted group/key means unmanaged. `security: {}` manages nothing and performs no security reads. Explicit false means disable; true means enable. Null at the group or any key, strings such as `enabled`, sequences, duplicate/unknown keys, and wrong types are invalid before remote work. There are no security collections: empty mappings do not mean disable all or clear anything. Only `secret_scanning` and `secret_scanning_push_protection` are currently accepted. Other planned/reserved keys remain unknown to the executable until their separate implementation/adoption.

Plan reads only declared families and the prerequisite/companion observations needed to establish a safe transition. Unrelated hidden security fields do not prevent private vulnerability reporting or an ordinary repository-only plan. API response-only metadata, including pause, entitlement, and enforcement evidence, cannot be declared as desired state.

## API, permissions, and availability matrix

Paths below are relative to `/repos/{owner}/{repo}`. Permission names are the documented fine-grained repository permissions, not proof of a token's actual grants. Classic OAuth/PAT scopes, user roles, App installation scope, SSO, and policy restrictions must also permit the specific operation. Do not infer an entire credential's authority from repository `permissions.admin`, another endpoint's success, or a public metadata read.

| Control | Observation | Mutation | Documented read / write boundary | Availability and upper policy |
| --- | --- | --- | --- | --- |
| Dependabot alerts | GET `/vulnerability-alerts`: 204 enabled; documented disabled response 404 requires the gate below | PUT / DELETE same path: 204 | Administration read / write; authenticated admin access | Public/private; basic Dependabot is available across plans. Endpoint also enables/disables dependency graph; upper security configuration can constrain changes. |
| Dependabot security updates | GET `/automated-security-fixes`: 200 with required boolean `enabled` and `paused`; ambiguous 404 is unknown | PUT / DELETE same path: 204 | Administration read / write; authenticated admin access | Public/private; requires dependency graph and alerts. Enabled can be paused; does not mean a PR can be produced for every ecosystem/dependency. |
| Secret scanning | GET repository, required `security_and_analysis.secret_scanning.status` enum | PATCH repository with only declared nested status | Metadata read for repository endpoint plus admin/owner/security-manager visibility; Administration write and applicable role for security PATCH | Public feature availability differs from private/internal Secret Protection entitlement. Partner scanning is separate. Upper configuration can enforce either state. |
| Repository push protection | GET repository, required `security_and_analysis.secret_scanning_push_protection.status` enum | Same narrow nested PATCH | Same visibility / update requirements | Secret scanning prerequisite; public availability does not establish private/internal entitlement. User push protection is not this setting. |
| Code Security candidate | GET repository, `security_and_analysis.code_security.status` if completely visible | Candidate nested `code_security.status` PATCH | Same repository visibility / Administration write boundary, subject to adoption evidence | Standalone product; private/internal use needs applicable existing entitlement. Public scanning availability alone does not establish a writable product toggle. |
| Legacy Advanced Security candidate | GET repository, `security_and_analysis.advanced_security.status` if completely visible | Candidate nested `advanced_security.status` PATCH | Same repository visibility / Administration write boundary, subject to adoption evidence | Legacy/bundled product. API explicitly disallows this parameter for standalone Code Scanning or Secret Protection products. Never substitute it for `code_security`. |
| Private vulnerability reporting | GET `/private-vulnerability-reporting`: 200 with required boolean `enabled` | PUT / DELETE same path: 204 | Metadata read / Administration write; admin access for update | Public repositories; private/internal target is inapplicable even when desired false. No paid-product prerequisite. Do not alter advisory/report data. |

The permission matrix records the [repository REST reference](https://docs.github.com/en/rest/repos/repos) and the [official OpenAPI description](https://github.com/github/rest-api-description/blob/755ee33b4a81d51f6f07a902cce26cbec3b822f2/descriptions/api.github.com/api.github.com.json) reviewed on 2026-10-20. The latter separately specifies both product PATCH fields, visibility requirements, and Dependabot response shapes. Repository endpoint permission is necessary but does not guarantee visibility of each nested feature. A missing/null `security_and_analysis`, missing/null requested child/status, unknown status string, or wrong type is incomplete observation, never disabled. Unknown unrequested sibling fields are not replayed into requests.

For plan/entitlement boundaries see [GitHub security features](https://docs.github.com/en/code-security/getting-started/github-security-features) and [Advanced Security products](https://docs.github.com/en/get-started/learning-about-github/about-github-advanced-security). Basic dependency security is distinct from premium product capabilities. Do not purchase, trial, enroll, or switch products to make a failed read/write succeed. Existing organization entitlement is a prerequisite, not something gh-settings provisions.

### Disabled-state gate for Dependabot

The documented 404 disabled response overlaps GitHub's concealment of inaccessible resources. A successful repository GET or an unrelated Actions/runner endpoint cannot prove permission for this security endpoint. A user role alone cannot prove the token's fine-grained grants. Therefore initial implementation must treat an unqualified 404 as unknown and fail before comparison, mutation, or file replacement.

Before claiming full #344 disabled-state support, establish and document an endpoint- and credential-specific authorization discriminator, with denied and genuinely disabled fixtures. If none is available, defer #344 for a separate contract decision rather than silently weakening the observation guarantee. A narrower readable-state-only feature must be explicitly adopted as such; it cannot claim disabled export or verified disable success. This gate does not require new live permissions, toggling a production repository, or broader organization reads.

### Product candidate gate

For #346, retain separate reserved boolean names. A future adoption must identify the existing product model from authoritative evidence, define the exact supported visibility/entitlement cases, and establish companion effects for both enable and disable. Do not infer a model solely from a missing field or attempt the other product after a failed request. Reject both candidate keys in one declaration; neither requests conversion or purchase. Unknown applicability blocks that family, not independent security features.

A product-level toggle can affect subordinate security features. Before adoption, specify which companions must be observed and explicitly declared for each destructive transition, and test preservation or acknowledged intrinsic effects. The current design deliberately does not adopt an unverified product mutation. #346 remains pending this evidence/adoption decision; #345 and #347 do not depend on it.

## Explicit export selection

Choose proposed repeatable `--security FAMILY` with exact implemented family values `dependabot`, `secret-scanning`, and `private-vulnerability-reporting`. Only `secret-scanning` is currently implemented. Each future family selects its adopted keys in its table row(s), not every security setting. Reserve no accepted `all` value or product option. Reject unknown, missing, or duplicate family values before reads or destination changes. Compose distinct families with existing `--rulesets`, `--actions`, and `--workflow-permissions` choices without making security selection implicit.

Ordinary export preserves all existing requests and credential requirements. Selected security reads follow the existing resources in table-family order: Dependabot, Secret scanning, private vulnerability reporting. Complete all selected reads and serialization before opening/replacing the destination. Selection is an explicit request: unavailable/inapplicable/unknown data fails, rather than being omitted with success. Export false values when authoritatively observed, omit unselected families, and exclude operational metadata. An unchanged export must yield zero differences for exported booleans; zero does not prove writable access, active scans, or an unlocked policy.

## State and failure table

| Observation or event | Required result |
| --- | --- |
| Complete enabled/disabled boolean or supported status enum | Compare native booleans; equal yields no difference. |
| Updates `enabled: true, paused: true` | Preserve enabled=true and pause metadata; emit a diagnostic in plan/export/apply. Do not report running/healthy, synthesize false, or disable/re-enable to resume. |
| Any other complete `enabled`/`paused` boolean pair | Compare enabled independently and retain pause metadata; warn whenever paused. Do not invent a correlation between the two booleans that the API schema does not establish. |
| Missing/null/wrong-type enabled or pause metadata | Reject incomplete state; do not discard pause to manufacture a usable observation. |
| Missing/null/invalid requested response field | Fail that read; no successful empty/false comparison or export. |
| 401/403, unqualified 404, redirect, transport failure, invalid JSON | Fail read as unknown/denied as evidenced; never infer disabled or follow a changed target. |
| Confirmed unsupported visibility/entitlement | Report inapplicable/unavailable as evidenced; never false. Do not guess licensing from arbitrary 403/404 text. |
| Authoritative upper-policy conflict known before write | Fail before mutation of that security family; no detach or policy bypass. |
| Fresh prerequisite read/preflight fails | Record a confirmed pre-write read failure, preserving prior completed operations. |
| Write returns 2xx but readback differs, is incomplete, or fails | Fail verification; mutation effects are uncertain. Do not count success, retry, or roll back automatically. |
| Earlier resource/family succeeded, later family fails | Retain and report confirmed progress and any uncertain mutation; overall failure. |

Upper [security configuration enforcement](https://docs.github.com/en/code-security/reference/security-at-scale/configuration-enforcement) may make a REST change appear successful while leaving values unchanged. Complete fresh readback is mandatory. The initial feature does not require organization-wide configuration reads or claim to diagnose all enforcement before writing. If available evidence identifies a conflict, surface it; otherwise an unmet readback is a verification failure with no invented cause. Repository declarations never own inherited configuration membership. Changing an unenforced repository value may affect its configuration status; no attachment/detachment request is sent to repair that status.

## Dependency and mutation table

These rules constrain transition preflight; they do not introduce a saved-plan transaction. Plan is read-only. Apply rereads prerequisites with its own credential immediately before each family's mutation; an earlier plan does not authorize using stale observations. Validate the entire local declaration before any remote operation. Within each family, complete necessary state and transition checks before its first write.

| Desired transition | Required prerequisites and order |
| --- | --- |
| Security updates true with alerts false | Reject contradiction locally. |
| Security updates true; alerts omitted | Require a fresh known-enabled alerts observation; otherwise fail before writes. Never enable omitted alerts. |
| Alerts true and security updates true | Enable/verify alerts before enabling/verifying updates. The alerts endpoint's documented dependency-graph effect is intrinsic and must be disclosed. |
| Alerts false | Observe updates; if currently enabled, require explicit updates=false and disable/verify updates first. Unknown updates blocks the transition. An omitted known-disabled child needs no write. The documented graph-disable effect is intrinsic. |
| Push protection true with secret scanning false | Reject contradiction locally. |
| Push protection true; scanning omitted | Require fresh known-enabled scanning; no implicit parent enable or product enrollment. |
| Scanning true and push protection true | Enable/verify scanning before enabling/verifying push protection, with narrow sequential nested PATCHes. |
| Scanning false | Observe push protection; if currently enabled, require explicit secret_scanning_push_protection=false and disable/verify it first. Unknown child blocks the transition. |
| Child false alone | Update only the declared child when a change is needed; never send omitted parent fields. |
| Private vulnerability reporting true/false | Verify public target and complete current boolean, then PUT/DELETE only if different and GET-verify; independent of paid products and other families. |

The dependency graph/alerts relationship is documented in the REST endpoint descriptions; [Dependabot security updates](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependabot-security-updates) requires both. [Push protection](https://docs.github.com/en/code-security/concepts/secret-security/push-protection) is a secret scanning feature. The conservative explicit-child-disable requirement prevents using a parent declaration to silently change an unmanaged child. Unavoidable GitHub-internal side effects cannot be represented as additional successful CLI writes; disclose them and verify relevant companions where the supported observation contract permits.

For Secret scanning, send only supported declared nested status members, never the complete observed security block. Do not combine parent/child transitions into a PATCH whose order GitHub does not guarantee. Readback checks the desired member and relevant preserved companions; unknown unrelated fields remain outside the owned payload. For each family, preflight must cover every companion it intends to preserve. On an unknown write outcome, stop; rerunning later starts from fresh observations and does not automatically compensate.

Security families append after existing apply resources in the same fixed order as export. Each adapter owns observation/payload/readback; settings owns declarations, transition rules, and native comparisons; app owns operation sequencing and partial progress. Use closed field identities/native booleans and bounded report validation for CLI/Action summaries. Do not invent a generic privileged backend framework. Existing direct apply and no automatic rollback contracts remain unchanged.

## Implementation acceptance and fixture plan

The matrix remains the acceptance plan for all families; the implemented Secret scanning mappings follow it. Unimplemented rows are planned checks, **not claims of completed tests or live verification**. Each implementation supplies its owning Go test names and public references when merged. Use local fake clients, bounded HTTP fixtures, and built-CLI process tests with dummy credentials. Do not enable scans, change production settings, purchase contracts, or expand permissions for these tests.

| Issue / layer | Required evidence before completion |
| --- | --- |
| All: codec/model | Omitted/group-empty no-op; explicit true/false preserved; null, wrong types, duplicate/unknown keys rejected before network; zero/unacquired observation rejected; copied values cannot corrupt invariants. |
| All: API/app | Correct target and method, declared-only payload, complete fresh prerequisite and readback checks, redirect rejection, error classification before/after writes, deterministic order, earlier successes retained, no automatic retry/rollback. |
| #344 | Resolve disabled-read gate first; authorized disabled vs permission-concealed 404, 204 alerts, complete enabled/paused combinations, contradictory/omitted dependencies, enable/disable ordering, graph effects, no version-update/individual-alert operations. |
| #345 | Public and entitled/nonentitled private/internal fixtures; hidden/null/unknown status; parent/child combinations and fresh drift; no omitted fields in nested PATCH; enforced 2xx/no-change must fail verification; no bypass/pattern/configuration writes. |
| #346 | Separate adoption evidence first; distinct model/entitlement fixtures, both-key rejection, no alias/fallback/conversion, complete companion effects and lossless supported-state round-trip. No implementation completion while product behavior remains unverified. |
| #347 | Public true/false/equal/omitted, private/internal inapplicability, Metadata-only read shape, missing/null/wrong enabled value, 422/permission errors, PUT/DELETE+readback, no advisory/report/notification operations. |
| Export/process | Ordinary requests unchanged; explicit family composition and invalid options; complete export→unchanged plan zero; requested unavailable state leaves destination untouched; false retained; pause diagnostic; no stdout changes. |
| CLI/Action | Native boolean differences, strict report reconstruction and bounds, read/write failure and partial-progress output, escaping of untrusted diagnostics; published pinned CLI must reject unsupported additions rather than pretend source support is released. |

The [private vulnerability reporting guide](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository) establishes public-repository scope. The source/runtime privilege boundary and fixture limitations belong in the public references and testing guide at implementation time. API schema inspection and local fixtures do not prove all token types, billing availability, upper enforcement, or real scan outcomes.

## Implemented Secret scanning test mappings

The [public configuration contract](../settings.md#secret-scanning-and-push-protection-unreleased) owns current-source usage. The source implementation accepts only the two Secret scanning booleans and `--security secret-scanning`. Other families and product candidates remain unsupported.

- Declaration/observation/transition/forged-change boundaries: [TestSecretScanningObservationAndTransitions](../../internal/settings/security_test.go) and [TestSecurityComparisonsRejectForgedValues](../../internal/settings/security_test.go).
- Strict YAML, explicit false and omission round-trip: [TestSecurityYAMLContract](../../internal/workspace/security_test.go).
- Complete API reads, unknown/denied/redirect responses, fresh prerequisites, narrow payload and readback/companion failures: [TestSecretScanningCompleteReads](../../internal/github/security_test.go) and [TestSecretScanningNarrowPayloadReadbackAndFreshPreflight](../../internal/github/security_test.go).
- Dependency order, read certainty, matching state and confirmed partial progress: [TestSecretScanningApplyOrderAndPartialProgress](../../internal/app/security_test.go) and [TestSecretScanningReadFailureAndOmission](../../internal/app/security_test.go).
- Export selection/composition and forged native Summary values: [TestSecurityExportSelectionIsExplicitAndComposable](../../internal/cli/security_test.go) and [TestSecuritySummaryHasNativeClosedValues](../../internal/actions/security_test.go).
- Built-CLI ordinary/explicit export, zero plan, native Action report/Summary, enable/disable/idempotence, unavailable reads preserving files, redirect rejection and enforced/partial writes: [TestSecretScanningExportPlanApplyAndSummary](../../tests/acceptance/security_test.go).

Fixtures use dummy credentials and local modeled responses. A read-only observation of this repository confirmed both enabled statuses and additional unowned siblings; no live PATCH, private entitlement, enforcement or token-grant behavior was exercised or established. No scan, contract, organization setting, workflow permission, grant or release changed.
