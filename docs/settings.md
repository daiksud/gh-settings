---
type: Reference
title: Repository UI YAML
description: Define the current-source UI-oriented declaration, ownership, validation and API boundaries.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: github-labels
    resource: https://docs.github.com/en/rest/issues/labels
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
  - id: github-actions-policies
    resource: https://docs.github.com/en/rest/actions/policies
  - id: ui-inventory
    resource: contributing/ui-repository-mapping.md
---

## Scope and status

Current source reads and exports UI-oriented YAML in `.github/settings.yml` at the Git checkout root. That checkout's directly configured GitHub.com `origin` selects the repository. Public configuration follows pages and sections, using English Web UI terms and meaningful control keys in snake_case; REST and GraphQL names are translated internally.[^ui-inventory]

```yaml
version: daiksud/gh-settings@v0
repository:
  about:
    description: My project
    website: https://example.com
    topics: [github-cli, settings-as-code]
  labels:
    - name: bug
      color: '#d73a4a'
      description: Something is not working
  settings:
    general:
      features: {issues: true}
```

All existing and future gh-settings specifications assume the caller already has the required permissions. Endpoint-specific documented 404 semantics apply under that precondition; explicit denials, malformed/incomplete responses, timeouts, policy enforcement and readback mismatches remain failures. This product precondition does not establish a live credential's grants or authorize access changes.

The exact string `version: daiksud/gh-settings@v0` is required. Superseded envelopes, navigation-category wrappers, scalar permission sections and API-shaped aliases are rejected. There is no compatibility decoder, migration or legacy-format fallback.

This reference describes current source, not the contract of every published CLI or pinned Action. Read the [published package boundary](github-actions.md#published-package-and-consumer-pins) before adoption. Current-source output uses UI names and field-specific normalization warnings. Out-of-UI Actions execution/workflow permission strings, public-fork approval policies, Security status strings, creation policies, complete merge/squash message pairs, complete Pages modes and legacy folders, Ruleset enforcement/bypass-mode/scanning Alerts/scanning Security alerts strings, Required approvals integers, Allowed merge methods lists and Environment deployment-mode pairs/ref-type strings use the explicit observation policies below; unsupported identities and malformed or incomplete observations still fail. Observed fallbacks never populate omitted desired fields. CLI and detailed Action Summary use UI paths and choices for exact and normalized decisions. Same-version private reports retain native identities and are not apply input.

## Feature adoption scope

The [v1.0.0 goal (#668)](https://github.com/daiksud/gh-settings/issues/668)
targets **personally owned public repositories**. A setting is eligible when it
is configurable in that target's GitHub Web UI. Account plan, Copilot subscription,
feature execution entitlement and Preview status are separate facts; none is a
blanket exclusion. A writable API-only field does not create a target UI control.

Eligibility, technical feasibility and implementation are separate dimensions.
Complete YAML/export/plan/apply support requires the applicable official API
observation and update paths. A read endpoint alone does not prove update support,
and a write endpoint alone does not prove export support. An unconfirmed API
contract remains unconfirmed, rather than being labeled unavailable. The
[coverage ledger](contributing/settings-coverage.md) records these distinctions
and the limits of fixture and live evidence.

Personal-repository collaborators and invitations are supported through the
[identity and invitation contract](#personal-collaborators). Existing Environment
protection App selection follows the [custom protection contract](#custom-protection-apps).
Remaining classic protection controls and additional configurable custom rules
remain adopted future work when supported by the target UI and applicable APIs. Organization-owned repositories and organization
settings, repository rename/visibility/transfer/deletion, authoring GitHub feature
configuration files and secret values are excluded. This file-authoring boundary
does not exclude gh-settings' own declaration file. Existing App selection does
not authorize installing an App or increasing its grants.

This supersedes the Free-only policy recorded on 2026-11-08 and blanket Preview
and custom-rule exclusions. Historical investigations remain evidence with their
original dates, not current scope rules. The contracts below describe **current
source**: newly eligible work does not make an unimplemented key valid or change
current export, reconciliation, preservation or pruning behavior. The v1 goal is
not a claim that current source or a pinned published package is complete.

## Collection pruning

Normal apply creates/updates the declared independent objects and preserves
unlisted objects. `plan --prune labels,rulesets` previews additional deletions;
`apply --prune labels,rulesets` executes them alongside all other declared work.
Use one comma-separated option, not repeated flags. Empty elements, duplicate or
unknown names fail locally. Current source supports `labels`, `rulesets`, `actions-variables`,
`agents-variables`, `environment-variables`, `actions-policies`, `environments`, `collaborators`,
`branch-protection-rules`, `autolinks` and `deploy-keys`. Standalone `all` selects all currently supported scopes; combining it with another
scope is invalid. Each collection still requires explicit presence in YAML.

Omitted lists remain unmanaged. Explicit `[]` selects all current objects in that
scope; a nonempty list prunes objects whose identities are not declared (IDs for
labels/Rulesets/Actions Policies/classic rules/Autolinks/deploy keys, canonical names for variables, folded names for Environments, resolved immutable user IDs for collaborators). ID-less label/Ruleset/Actions Policy/classic rule/Autolink entries
remain create-only and never adopt a name collision. Acquire complete inventories
and validate every prune candidate before any remote write, including unrelated
settings writes. Unsupported candidates fail the whole preflight. Create/update
before deletion; keep the prepared deletion set fixed, check drift before DELETE
and verify absence afterward. Completed earlier writes survive a later failure;
readback failure can leave uncertain effects. There is no lock or rollback.

Deleting labels removes their Issue/PR assignments; recreating them does not
restore assignments. Automatically created Dependabot labels are deletion
candidates too if undeclared. Plan displays this consequence. Topics, PR exemption
lists and deployment branch/tag pattern replacement are unchanged. The
[behavior specification](features/prune.feature.md) records stages and tests.

### Environment pruning

`--prune environments` owns unlisted parent lifecycles only when
`repository.settings.environments` is present. Omission remains unmanaged even
with `all`; `[]` selects every acquired Environment. A nonempty list retains all
listed names, including variable-only entries and `variables: []`. Policy upserts
and all selected Environment variable work finish before parent deletions. New
parents never join the fixed deletion set. No ID is written to YAML and no
`delete` member is accepted.

Before any writes, acquire every candidate's positive parent ID, complete
supported policy, active raw pattern identities, variable inventory and secret
name/creation/update metadata. The dedicated custom-deployment-protection endpoint
must independently confirm a complete empty list. Nonempty, denied, malformed or
normalized unsupported state blocks the whole operation. Retained names do not
need deletion snapshots. Recheck the fixed candidate set before Environment work
and each candidate immediately before DELETE. Account only for this command's
confirmed unarchive; other target/archive or observable child drift stops deletion.
After DELETE 204, a complete fresh inventory must omit both the original ID and
folded name. Same-name recreation fails confirmation. Reads are not an atomic
conditional-delete lock; a race remains between the final check and deletion.

Parent deletion removes contained configuration, including variables and patterns.
[GitHub documents](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments#deleting-an-environment)
secret/protection removal and failure of jobs waiting on protection rules. A later
workflow reference can recreate a name without restoring the deleted resource's
protection. Secret values are never read, exported or backed up; YAML cannot
restore them. Secret metadata is only private drift evidence, not complete
rotation detection, and is excluded from plans/reports. Prune snapshot transport
ignores inherited `GH_DEBUG` so child evidence is not copied to debug logs.

[Environment list/detail](https://docs.github.com/en/rest/deployments/environments)
and [custom-protection reads](https://docs.github.com/en/rest/deployments/protection-rules#get-all-deployment-protection-rules-for-an-environment)
require Actions read. Child variable and [secret metadata reads](https://docs.github.com/en/rest/actions/secrets#list-environment-secrets)
require Environments read; deletion requires Administration write. These are
preconditions, not automatic permission grants. Inactive patterns are inaccessible
under the documented custom-mode prerequisite: explicit parent deletion is allowed
with that observation limit, without claiming an acquired empty latent list.
Detailed plans show complete supported before-policy and absence, with deletion
consequences; status-only output hides names and policy. Failed prewrite reads
make no deletion; failed DELETE/absence verification can have uncertain effects.
Earlier verified writes survive later failure. No retry or rollback occurs.

## Supported fields

`repository.about`, `repository.labels` and `repository.settings` are siblings. Settings contains `general`, `rulesets`, `branch_protection_rules`, `actions.general`, `actions.policies`, `actions.variables`, `agents.variables`, `copilot.code_review`, `interaction_limits.pull_request_limits`, `advanced_security`, `pages` and `environments`. Unknown keys, including otherwise writable API fields, are rejected.

The basic structure is `repository.settings.<page>[.<subpage>][.<section>].<control>`. Navigation-only sidebar categories are omitted. Do not invent a section solely to equalize depth: page-level controls remain direct, and compound values or resource entries retain their own nesting. The Actions sections use `actions_permissions.policy` and `workflow_permissions.default_permissions`; Secret Protection uses `secret_protection.enabled` with `push_protection` beside it. There is no generic `selection` or `value` wrapper for controls (variable entries have their own required `value`), no second Actions enable flag, and no universal `enabled` wrapper around simple booleans. These are public declaration names, not claims about HTML form attributes.

Under `repository.about`, `description` and `website` are strings, including `''`; `topics` is a direct string array. These translate to repository REST fields and the Topics endpoint. Nullable observed repository strings become empty strings through the existing reader; YAML null does not request that behavior.[^github-update-repository][^github-repository-openapi]

The following paths are relative to `repository.settings.general`:

| Path | Type |
| --- | --- |
| `template_repository` | boolean |
| `default_branch` | literal branch-name string |
| `features.wikis`, `features.issues`, `features.sponsorships`, `features.discussions`, `features.projects`, `features.pull_requests` | boolean |
| `features.issue_permissions.creation_allowed_by`, `features.pull_request_permissions.creation_allowed_by` | `all_users` or `collaborators_only` |
| `pull_requests.allow_merge_commits`, `pull_requests.allow_squash_merging`, `pull_requests.allow_rebase_merging` | boolean |
| `pull_requests.always_suggest_updating_pull_request_branches`, `pull_requests.allow_auto_merge`, `pull_requests.automatically_delete_head_branches` | boolean |
| `pull_requests.merge_commit.default_commit_message`, `pull_requests.squash_merge.default_commit_message` | UI menu choice below |
| `commits.require_contributors_to_sign_off_on_web_based_commits` | boolean |
| `danger_zone.archive_this_repository` | boolean; automatic export |
| `releases.immutable_release` | boolean; automatic export |

Omitting a scalar leaves it unmanaged. Explicit false and empty strings remain intent. REST-backed About/General controls use [fresh reconciliation](commands.md#apply-behavior): skip exact matches and independently verify declared state after changed writes. An explicitly disabled Issues/Pull requests feature cannot accompany its creation-permission choice. Likewise, an explicitly disabled merge method cannot accompany its commit-message menu. Omitting the enabling checkbox never implicitly enables it. Export hides choices behind explicitly disabled controls after validating the acquired raw values.

## Repository archive lifecycle

Use `repository.settings.general.danger_zone.archive_this_repository`. Omission makes no archive-specific request; false explicitly unarchives and true explicitly archives. Ordinary export includes the observed boolean after complete reads; unavailable state retains prior intent with info.

```yaml
version: daiksud/gh-settings@v0
repository:
  about: {description: Active maintenance}
  settings:
    general:
      danger_zone: {archive_this_repository: false}
```

Plan validates positive repository ID, canonical owner/name and archive state, then completely observes other managed resources without writing. Apply validates current identity before writes. Explicit unarchive runs first; final archive runs only after every other update succeeds and a fresh identity/state check. Archive PATCH contains only the internal archive member, and response/readback must confirm ID, name and state. Matching archive-only intent makes no lifecycle write.

On an already archived target, archive=true combined with other managed settings fails apply before all writes, even if plan showed a match or difference. Explicitly unarchive before making other changes. Failed unarchive stops later work. A confirmed unarchive survives a later failure; final archive failure can leave earlier updates applied. A write/readback failure can leave lifecycle state uncertain. There is no rollback, automatic retry, temporary unarchive, transaction or identity lock. Keep an external recovery route if the archived repository's own Actions cannot run.

Existing Administration read/write grants and actor authorization are prerequisites. Local fixtures do not establish live permissions or lifecycle behavior. See the [lifecycle specification](features/repository-lifecycle.feature.md) and [ADR 0027](adr/0027-repository-lifecycle.md).

## Dependabot alerts and security updates

Use independently optional booleans under `repository.settings.advanced_security.dependabot`:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    advanced_security:
      dependabot:
        alerts: true
        security_updates: true
```

Omission leaves each control unmanaged; false explicitly disables, null is invalid and an empty mapping manages nothing. `paused` is observed metadata, not a YAML control. This current-source capability is not added to the pinned published package by documentation or configuration changes.

Ordinary `export` attempts both controls as one complete family; unavailable acquisition emits info and preserves prior declarations when present. Administration read and applicable actor access are prerequisites for complete reads; updates require Administration write and actor authorization. GET 404 means disabled under the required-permissions precondition. This does not certify actual grants or live behavior. Updates 200 requires both enabled and paused booleans; updates 404 supplies no pause metadata. Paused updates remain enabled when enabled=true; warnings do not request a resume or promise active pull requests.

Plan is read-only. Apply completely reads the family, then refreshes prerequisites before each narrow body-free PUT/DELETE and reads back both controls. Enable alerts before security updates; disable security updates before alerts. Explicit alerts=false with updates=true fails locally. Updates=true with alerts omitted requires acquired enabled alerts; no parent is enabled implicitly. Disabling alerts while updates are active requires explicit updates=false. Omitted companions must retain their freshly acquired enabled values. The alerts endpoint intrinsically enables/disables Dependency Graph too.

A fresh matching value skips its write, including enabled=true/paused=true. An enable requires enabled readback. A disable returning DELETE 204 followed by matching complete family readback, including target GET 404, completes under required permissions. Pause remains unknown on 404; the operation reports enabled intent only. There is no permission-risk warning or accepted-unverified notice. GET alone does not establish a write; non-204 writes, explicit 401/403, timeouts, malformed/incomplete reads or target/companion mismatch fail. Later failures retain earlier completed progress without retry or rollback. No Dependabot configuration files, individual alerts, grouping, auto-triage, credentials or organization settings are managed. See the [decision and fixture contract](features/security.feature.md#dependabot-state-reading-decision).

## Secret scanning and push protection

Use the independently optional booleans `enabled` and `push_protection` under `repository.settings.advanced_security.secret_protection`.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    advanced_security:
      secret_protection:
        enabled: true
        push_protection: true
```

Omission is unmanaged; false disables. Explicit `secret_protection.enabled=false` with `push_protection=true` fails locally. No parent is enabled implicitly. Push protection true with `enabled` omitted requires fresh observed secret protection=true. Disabling secret protection while push protection is active requires declaring push protection false too. Unknown raw companions cannot prove prerequisites; explicit intent must resolve an unsafe transition. Safely unmodified companions retain exact raw evidence for readback. Enable parent before child; disable child before parent.

Ordinary `export` attempts both complete statuses; unavailable acquisition emits info and is omitted on first export or retained from a valid existing declaration. Plan is read-only. Apply preflights, makes narrow per-member repository updates in dependency order, rereads prerequisites and verifies requested values plus preserved companions. Exact canonical matching members are skipped. Upper policy can ignore an apparently successful update; readback mismatch fails. Later failure retains earlier confirmed writes and uncertain effects; no rollback or automatic retry occurs.

A complete successful read requires both status strings. Canonical `enabled` / `disabled` map to true / false; correctly typed unknown, empty or whitespace statuses use the named per-field boolean fallback `false` with exact original provenance. Missing/null members, invalid types and failed reads remain acquisition errors. An inconsistent projected false/true pair remains acquired but cannot be exported as a consistent declaration; selected export warns and preserves the destination.

Plan retains one write-required decision per declared normalized control, including false-to-false. Omitted controls remain unmanaged. Apply verifies fresh raw statuses for the requested member and preserved companion; unchanged unknown status cannot confirm a write, and a changed unknown companion fails verification. Each successful managed initial/preflight/readback observation can warn; unowned companion values are not disclosed. Detailed CLI/Summary use UI paths and native booleans, while status-only Summary suppresses values. Export saves effective booleans after complete conversion and never writes GitHub settings.

Existing entitlement, token grants and appropriate admin/organization-owner/security-manager visibility are required. API visibility alone does not prove writable access or licensing. No product is purchased, credential expanded, alert/pattern/bypass managed or scanning workflow created. Unadopted security product controls remain unsupported. See the [security specification](features/security.feature.md).

## Immutable release policy

Use `repository.settings.general.releases.immutable_release`, a boolean. This General-page control is independent from Advanced Security. Ordinary `export` automatically attempts this policy, with info and omission/prior-declaration retention when unavailable. Owner enforcement is read-only metadata, never YAML intent.

A complete successful 200 observation requires both enabled and owner-enforced booleans. Under required permissions, HTTP 404 means disabled without an authorization-discrimination request, but supplies no owner-enforcement metadata. This does not certify actual grants or change other endpoints' 404 semantics. Explicit 401/403, timeouts and malformed/incomplete successful reads remain errors.

An owner-enforced policy cannot be planned/applied as false. Apply runs after private vulnerability reporting and before Pages, reads fresh state, skips a match, sends body-free PUT/DELETE, requires 204 and verifies fresh readback. A post-DELETE 404 confirms disabled intent under the same precondition; unknown owner enforcement remains unknown. Failed writes/readback can already have changed the policy; no retry or rollback occurs. Individual releases, tags and assets are never mutated by this feature, and disabling the policy does not make existing immutable releases editable. Administration read/write and applicable role must already exist. See [the full contract and evidence limits](features/immutable-releases.feature.md).

## CodeQL default setup

Use the optional boolean `repository.settings.advanced_security.code_security.codeql_analysis.default_setup` for a personally owned public repository. Omission leaves default setup unmanaged; false explicitly disables it. Null and other types fail. Ordinary `export` automatically attempts this enablement, independently of other Security reads. Unavailable acquisition emits info and retains prior intent when present.

The public [default setup API](https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2026-03-10#update-a-code-scanning-default-setup-configuration) requires Administration read/write. Reads require public visibility, a User owner and a complete `configured` or `not-configured` state; missing/null/unknown states, 403 and 404 fail without fallback. Plan is read-only. Apply rereads before writing, skips a fresh matching state, and sends only `state`. Language selection, runner and query suite are separate [configuration controls](#codeql-default-configuration); they are not part of this enablement control. No product, Actions prerequisite or advanced workflow configuration is automatically enabled or edited. Ineligible repositories fail through the public API.

Changing this setting can start CodeQL validation/analysis and consume Actions resources. Disabling stops default-setup scanning and may affect features that depend on its results, including Autofix and required scanning checks. Existing analyses are not deleted. HTTP 202 accepts asynchronous work; fresh readback confirms configuration state only, never completion or success of a scan. A mismatching readback may indicate pending validation and fails with uncertain write effects. No automatic retry or rollback occurs; inspect state before retrying. Local fixtures do not establish live entitlement, analysis results or consumption.

## AI Scan for pull requests

Use the optional boolean `repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_pull_requests` for a personally owned public repository. Omission leaves it unmanaged; false disables it. Null/other types fail. Ordinary `export` automatically attempts this toggle, with info and omission/prior-declaration retention when unavailable. The separate [repository-wide toggle](#ai-scan-for-repositories) is supported; AI-application scanning remains unimplemented pending its API assessment.

The [public repository AI Scan API](https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2026-03-10#get-ai-scan-enablement-for-a-repository) requires Code scanning alerts read and Administration write. Acquisition requires public visibility, a User owner and exact `pr_scan` enabled/disabled. Missing/null/unknown values, 403/404 and incomplete applicability remain errors. Plan only reads. Apply rereads, skips a matching value, sends only `pr_scan` and confirms it with fresh readback. Other AI scans, CodeQL and product/prerequisite controls are not sent or automatically enabled.

Enabling permits GitHub to process pull requests with AI; applicable billing and AI credits can apply. Disabling affects future AI findings. Neither configuration readback nor a successful PATCH establishes a completed scan, coverage, entitlement, grants or actual consumption. Write/readback errors retain unknown effects; there is no automatic retry or rollback. No alert fix, Copilot assignment, scan-execution or credits-purchase operation is performed.

## AI Scan for repositories

`repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_repositories`
is an optional boolean for the **AI Scan for repositories** toggle on a personally
owned public repository. Omission leaves it unmanaged; false disables it. Null and
other types are invalid. It is independent of `ai_scan_for_pull_requests`:

```yaml
repository:
  settings:
    advanced_security:
      code_security:
        ai_scan:
          ai_scan_for_repositories: true
```

Ordinary export attempts this setting automatically. The native `repository_scan` status must be enabled or disabled; missing/null,
unsupported or denied observations are unavailable, never false. The optional API
member's absence remains unknown. Re-export preserves valid prior repository-scan
intent while independent PR-scan export can succeed. Declared plan/apply are
strict. Repository and owner IDs, name, personal ownership and public visibility
must be acquired; 403/404 is not disabled state.

Apply rereads the declared status and skips matches. A needed change requires
both native statuses. A missing or unsupported PR companion blocks mutation,
while independently readable repository state can
still be exported or planned. A needed PATCH sends only `repository_scan`,
requires a matching acknowledgment and independently verifies the requested
setting, original repository/owner identity and unchanged acquired PR companion.
Failure stops later work and preserves confirmed earlier progress; attempted
writes or failed readback may have changed configuration. There is no retry,
rollback or concurrency lock. CLI and detailed Action reports show the boolean;
status-only publication hides the setting and value.

This controls eligibility to run separately requested scans; apply does not
start a scan, enable CodeQL, edit workflows or change budgets. Scans may consume
AI credits under GitHub's applicable billing and entitlements. The
[AI Scan guide](https://docs.github.com/en/code-security/concepts/code-scanning/ai-scan-for-repositories)
explains execution and availability; the
[REST contract](https://docs.github.com/en/rest/code-scanning/code-scanning#get-ai-scan-enablement-for-a-repository)
documents Code scanning alerts read and Administration write. A read-only target
GET confirmed both statuses; modeled write tests do not establish real App grants,
scan execution or billing. Check support in the selected published CLI/Action;
this source addition does not upgrade its pin or publish a release.

## CodeQL default configuration

The optional controls below belong under `repository.settings.advanced_security.code_security.codeql_analysis`, alongside the independent `default_setup` boolean:

```yaml
languages: [github_actions, go, python]
query_suite: extended
threat_model: remote_sources
runner:
  type: labeled_runner
  label: codeql-pool
```

| Control | Accepted values and ownership |
| --- | --- |
| `languages` | A nonempty unique set: `github_actions`, `c_cpp`, `csharp`, `go`, `java_kotlin`, `javascript_typescript`, `python`, `ruby`, `swift`. Order is immaterial; export/report order is canonical. Unsupported or undetected selections may be refused by GitHub. |
| `query_suite` | `default` or `extended` |
| `threat_model` | `remote_sources` or `remote_and_local_sources`; maps exactly to API `remote` or `remote_and_local`. Raw API spellings are not YAML choices. |
| `runner` | Owns the type/label pair. `type: standard_github_runner` forbids `label` and explicitly clears an old label through API null. `type: labeled_runner` requires a nonblank `label`; it can select an existing labeled GitHub or self-hosted runner. No runner/label/capacity is created. |

GitHub documents local sources as [public preview](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/manage-your-configuration/edit-default-setup#including-local-sources-of-tainted-data-in-default-setup), currently supported for Java/Kotlin and C#. A repository's supported choices and source languages remain GitHub-controlled; gh-settings does not enable languages or infer UI availability from the selected analysis-language list. A complete read requires exactly the API `remote` or `remote_and_local` value. Missing, null, mistyped or unknown values fail acquisition without a fallback to remote-only scanning.

Omission leaves each control unmanaged. Null, invalid choices, duplicate/empty language lists and incomplete runner pairs fail before remote work. Declare a runner object to change the pair; omitting it preserves both members. This configuration requires an **already configured** default setup in a personally owned public repository. It never enables CodeQL or its prerequisites. If currently disabled, enable default setup in a separate operation and confirm it before configuring these controls. Configuration plus explicit `default_setup: false` is invalid; declaring true does not waive the already-configured requirement.

Ordinary `export` attempts all four controls as a complete configuration family. State and details share one response. Confirmed `not-configured` exports `default_setup: false` and removes existing languages/query suite/runner/threat model from the output with info. Unknown state (including denied, failed or malformed reads) emits info and omits the family on first export or retains its existing declaration. Configured state exports true; complete details replace prior values, while unavailable details emit info and retain existing details only. Reads require Administration read and complete exact state/languages/query suite/runner type and nullable label, plus the exact threat model. Unknown, missing, mistyped, denied or inaccessible observations are errors, not inferred defaults. Standard-runner null/empty labels represent the inactive label; a nonempty label with standard runner is rejected.

Plan only reads and reports declared controls. Apply fresh-reads and sends a single PATCH containing only declared configuration members, never `state`. It verifies the resulting configuration **and every omitted configuration companion** against the fresh pre-write state; concurrent changes can make verification fail. Changing only the threat model preserves languages, query suite and runner; omitting threat model preserves its acquired choice. An unchanged declaration writes nothing. Configuration changes may launch asynchronous validation/analysis and consume Actions resources. HTTP 202 and matching readback confirm configuration only; pending/mismatching state, companion drift or denied verification retain unknown effects and fail without retry or rollback. No analysis completion, live grant, runner availability or billing outcome is established by fixtures. See the [public API](https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2026-03-10#update-a-code-scanning-default-setup-configuration).

## Private vulnerability reporting

Use boolean `repository.settings.advanced_security.private_vulnerability_reporting`. Omission is unmanaged; false explicitly disables. It is independent from Secret protection. Ordinary `export` automatically attempts this family, with info and omission/prior-declaration retention when unavailable.

This control applies to public repositories. Fresh visibility and complete status are required; private/internal applicability failures, malformed data, permission errors and redirects do not become disabled state. Apply reads fresh prerequisites, skips matching state, performs the narrow enable/disable operation and verifies readback. Earlier writes remain on later failure. No rollback, permission expansion or automatic retry occurs. See [private reporting](features/security.feature.md).

## Security UI settings not managed by gh-settings

The investigation recorded on **2026-11-08** did not establish a formal public API path for complete settings reads and updates for the controls below. They remain unimplemented; that historical result does **not** prove that no API exists today. Reassess target Web UI availability and the applicable official API contract under the [personal-public adoption policy](#feature-adoption-scope), recording fresh evidence in the [coverage ledger](contributing/settings-coverage.md). UI availability can depend on prerequisites.

| UI control | Unconfirmed settings API boundary |
| --- | --- |
| Independent Dependency graph enablement | Public dependency comparison, SBOM and snapshot APIs operate on dependency data; organization/enterprise configuration APIs do not establish a personal-repository setter. |
| Automatic dependency submission | Submitting a dependency snapshot does not configure GitHub's automatic submission setting. |
| Dependabot malware alerts | Existing vulnerability-alert enablement does not manage this separate toggle. |
| Dependabot grouped security updates | Existing security-update enablement does not manage grouping. |
| Dependabot GitHub preset ON/OFF | No complete public preset identity/read/update contract was confirmed for the development-dependency or package-malware dismissal presets. API access to auto-dismissed alerts does not configure the presets. |
| Copilot Autofix / Agentic Autofix settings | Alert-specific fix generation, commits and assignment execute remediation rather than configure these repository controls. |
| Global code-scanning check-run failure thresholds | [Ruleset scanning thresholds](#code-scanning-parameters) are supported separately; they do not configure these global check-run settings. |
| Private vulnerability reporting details: Require a CWE assignment, Daily advisory limit, Trusted reporters | The public enablement endpoint exposes only enabled state, not complete detail settings. |

The existing [Dependabot alerts and security updates](#dependabot-alerts-and-security-updates) and [private reporting enablement](#private-vulnerability-reporting) remain supported. The alerts API intrinsically enables/disables Dependency graph too; this side effect is not independent graph management. Individual alert operations, advisory operations, Rulesets and organization/enterprise configuration are not substitutes for the missing settings contracts.

Configuration-file creation/editing remains excluded. Custom Dependabot auto-triage rules are eligible for assessment under the target UI/API policy but are not implemented. GitHub preset ON/OFF above is distinct from custom rules. [Code Security / Advanced Security product switches (#346)](https://github.com/daiksud/gh-settings/issues/346) are not planned because the target Web UI has no corresponding product-wide switch; they are not API-unconfirmed individual controls. The implemented [CodeQL default setup](#codeql-default-setup), [CodeQL configuration](#codeql-default-configuration) and [AI Scan for pull requests](#ai-scan-for-pull-requests) remain separate supported controls.

Sources reviewed on 2026-11-08: [public REST definition](https://github.com/github/rest-api-description/blob/2f8f80e47ea4d3292cf4e37ba9d9b25475e06140/descriptions/api.github.com/api.github.com.json), [public GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql), [GitHub preset documentation](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependabot-auto-triage-rules), and the [historical personal-public inventory](https://github.com/daiksud/gh-settings/issues/323#issuecomment-6719839333) with its [API recheck evidence](https://github.com/daiksud/gh-settings/issues/323#issuecomment-6728386507), both historical evidence rather than current eligibility rules. The public [Dependabot runner settings API](https://docs.github.com/en/rest/dependabot/runner-settings) was also assessed: labeled runners apply only to private/internal repositories with Actions enabled; public repositories report `standard`, so this does not add a configurable personal-public control.

## Default workflow permissions

These fields belong to `repository.settings.actions.general.workflow_permissions`:

| Field | Choice / type |
| --- | --- |
| `default_permissions` | `read_and_write_permissions` or `read_repository_contents_and_packages_permissions` |
| `allow_github_actions_to_create_and_approve_pull_requests` | boolean |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        workflow_permissions:
          default_permissions: read_repository_contents_and_packages_permissions
          allow_github_actions_to_create_and_approve_pull_requests: false
```

Each is independent and optional. Ordinary `export` includes both after complete acquisition, independently of execution policy. Omission is unmanaged, false remains explicit, and unknown choices/null/wrong types fail locally. Apply compares a complete fresh observation, skips matching intent and sends only changed declared workflow-policy members. Independent readback verifies the original declaration, including initially matching values, and preserves omitted raw companions. A matching repeat apply sends no update. Direct apply remains nontransactional: organization refusal can leave an earlier execution-policy update applied, and a failed PUT/readback can have uncertain effects. This declaration does not create tokens or waive a PR's approval requirements. See [workflow permissions](features/workflow-permissions.feature.md).

A complete successful workflow-policy GET with a correctly typed permission string outside the UI choices uses `read_repository_contents_and_packages_permissions` and emits a warning naming the YAML field, raw token and fallback. Missing/null members, wrong types and failed reads remain acquisition errors. Export saves the UI fallback without changing GitHub. Plan retains an actionable write-required comparison even if the desired choice equals that fallback; it does not claim independently observed zero drift. Apply sends only declared members and confirms the requested permission against a fresh raw readback. An unchanged out-of-UI token fails verification with possible write effects retained. An omitted permission remains unmanaged and is verified as an unchanged raw companion during approval-only writes, without replaying or printing the fallback.

## Fork PR workflow policies

Public-fork approval is `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`:

- `require_approval_for_first_time_contributors_who_are_new_to_github`
- `require_approval_for_first_time_contributors`
- `require_approval_for_all_external_contributors`

A complete successful public-fork policy GET with an out-of-UI string projects to `require_approval_for_first_time_contributors` (native `first_time_contributors`). Original unknown, empty or whitespace strings remain exact provenance. Missing/null members, wrong types, unsupported visibility and failed reads remain errors. Export warns and saves only the UI choice; plan retains one write-required decision even when desired intent equals the fallback. Omitted intent remains unmanaged.

Apply checks exact raw values before skipping a write and in fresh readback after PUT. A retained unknown value fails verification with possible write effects; a canonical exact match remains a no-op. Each successful managed apply read can warn: the initial application read, updater preflight and a normalized readback. Pure plan readers emit no stderr; detailed plan/Summary carry the decision warning, while status-only Summary suppresses values.

Ordinary `export` observes this public-repository control. Visibility, complete observations, narrow payloads, readback, redirect rejection and partial-failure reporting remain required. Upper policy may refuse or ignore changes; no retry, rollback or credential expansion occurs. See [fork workflows](features/fork-workflows.feature.md).

## Check, run, status, artifact and log retention

Use integer `repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention` (1–400 days). It translates to the existing retention endpoint. Ordinary `export` attempts retention automatically. The server's maximum is read-only metadata and is never replayed as intent. Existing repository visibility, range and freshly observed cap checks apply; a plan/readback cannot invent missing cap information or prove licensing. Apply verifies the requested value and preserves failure/uncertainty reporting. See [retention](features/retention.feature.md).

## Actions cache policy

The optional `cache` section under `repository.settings.actions.general` manages two independent integer controls:

| Field below `cache` | Value | Export option |
| --- | --- | --- |
| `cache_retention` | 1–365 days; public repositories are limited to 90 | `--cache-retention` |
| `cache_size_eviction_limit` | 1–10000 GB | `--cache-storage` |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        cache:
          cache_retention: 7
          cache_size_eviction_limit: 10
```

These controls are independent of check/workflow/artifact/log retention and Actions execution policy. Omitted controls are neither read nor changed by plan/apply; selecting one export option does not acquire the other. Empty `cache: {}` declares nothing. Nulls, strings, fractions and out-of-range integers are invalid.

A cache size eviction limit above 10 GB can incur additional charges. It is an eviction threshold, **not a spending budget**. Retention does not guarantee that caches survive until that age: eviction or manual deletion can remove them sooner. gh-settings manages repository limits only, not cache entries, usage, billing budgets, or organization/enterprise policy.

Each selected control requires a complete current read before its PUT and an independent readback afterward. Matching values skip PUT. GitHub can reject a value because of a tighter parent policy; gh-settings does not clamp it or change that policy. Missing, denied or malformed reads fail rather than supply defaults. A successful HTTP response with different readback fails verification and may have changed remote state. Later failure leaves earlier successful writes applied, without rollback or retry. See [behavior and API boundaries](features/actions-general-controls.feature.md).

## Actions OIDC subject template

Use `repository.settings.actions.oidc.subject_claim` for the **Subject claim**
page of a personally owned public repository:

```yaml
repository:
  settings:
    actions:
      oidc:
        subject_claim:
          use_default_template: false
          subject_claim_template: [repo, context, job_workflow_ref]
          use_immutable_subject_claim: true
```

The group has two independently selectable units: the complete active template
and the optional `use_immutable_subject_claim` boolean. Omission or an empty mapping
leaves it unmanaged. An immutable-only declaration is valid. If either template
field is selected, `use_default_template: false` requires an explicit ordered array;
`true` forbids that array and selects GitHub's default template. An array without
an explicit false mode is invalid. Keys are unique nonempty ASCII alphanumeric
or underscore strings; order and case are preserved. There is no closed claim
name list. Null, unsupported keys and coercions are rejected. An explicit
`[]` is accepted locally because neither the official API schema nor the inspected
form establishes a minimum; GitHub acceptance and exact readback are still required.

Selecting default deactivates the custom template. It does not promise to retain
hidden inactive keys that GitHub ignores in default mode. Changing a subject can
stop cloud authentication: coordinate the corresponding cloud trust configuration
before applying it. This tool does not issue tokens, run workflows, or change
cloud trust, credentials, grants or billing.

`use_immutable_subject_claim` requests the API configuration corresponding to the
UI checkbox, including explicit true or false. It does not certify effective token
format, universal UI editability or a reversible transition. GitHub documents
[mandatory immutable subjects](https://docs.github.com/en/actions/reference/security/oidc#immutable-subject-claims)
for some repositories independently of manual opt-in. gh-settings does not infer
eligibility from creation dates, rename/transfer history or the returned prefix.
The editable legacy control was observed unsaved on 2026-11-14; persisted legacy
opt-out has not been live-verified. GitHub decides whether a particular update is
permitted. A false configuration readback is not proof of a name-only cloud token.

Ordinary export acquires this family independently. Known default exports true
without keys; known custom exports false with its ordered keys. Known immutable
configuration exports its boolean, including false. A missing optional immutable
field remains unknown: export refreshes the readable active template and retains
only valid previous immutable intent, or omits that leaf on first export, with
bounded info. Retained intent is not acquired evidence. Whole-family transport,
identity or malformed-data failure retains compatible prior whole intent. Required
diagnostic delivery must succeed before saving.

Plan compares one managed group. Apply reacquires complete personal-public target
identity and active configuration, then skips known matching selected values.
Selecting immutable requires its known boolean even for comparison; unknown is
neither false nor a proven difference. A matching template-only declaration can
still skip with unknown immutable. A needed PUT requires both complete active
template and known immutable evidence. Immutable-only writes replay the fresh
active mode and exact ordered custom keys; template-only writes replay immutable.
Combined intent supplies both. Omission never silently selects a different mode.
Missing or malformed required state fails before writing. A successful PUT must
return HTTP 201 with a bounded empty acknowledgment, followed by independent GET
confirmation of the original repository/owner identity, archive state, requested
values and replayed active companions. The read-only prefix may legitimately
change and is not an equality guard. A failed write or readback reports uncertain effects;
earlier verified progress remains and later work stops. No rollback or automatic
retry occurs, and fresh reads are not a concurrency lock.

The official [repository OIDC API](https://docs.github.com/en/rest/actions/oidc)
uses Actions read for observation and Actions write for updates. Its read-only
subject prefix is neither YAML intent nor identity evidence; it and raw API bodies
are excluded from diagnostics even with `GH_DEBUG=api`. Current-source support
requires adoption through a package that includes this capability.

## Copilot code review custom instructions

`repository.settings.copilot.code_review.use_custom_instructions` corresponds to **Copilot → Code review → Use custom instructions**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        use_custom_instructions: false
```

This is one optional boolean. Omission, `copilot: {}` and `code_review: {}` are unmanaged; explicit false is managed. Null and non-booleans are invalid. The [review effort level](#copilot-code-review-effort-level) is independently selectable. The [MCP toggle](#copilot-code-review-mcp-tools), approvals, licensing, access and automatic-review configuration remain separate. No instruction file is read, created or edited.

The toggle changes whether future eligible Copilot reviews use repository instruction files. It does **not** enable Copilot code review, grant access, request a review or launch a workflow. Organization instructions apply independently. In the verified Free/personal/public UI, the No access banner expressly allowed storing settings for people with review access, and the toggle was enabled. Storing configuration is distinct from review entitlement and instruction consumption.

Plan requires a complete acquired boolean and never writes. Apply reads fresh state, skips matches, PATCHes only `custom_instructions_enabled`, then requires matching independent GET readback. HTTP 404 is an error, not false; denied, incomplete, null or mistyped reads fail. Mutation/readback failures may have changed stored configuration; earlier successful writes remain and later operations stop. No rollback, automatic retry, concurrency lock or saved-plan application is provided.

Ordinary `export` independently includes the stored custom-instructions boolean, including false. Unavailable observations retain prior declarations with info. Detailed CLI/Action reports show the UI path and native booleans; status-only output omits them.

The official [Copilot code review API schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json) documents repository administrators, classic `repo` scope and GitHub App support for GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings`. Exact fine-grained/App-token grants were not independently established by the owner-authenticated probe; callers must authorize this endpoint with their chosen credential. No grant is added automatically. The [behavior specification](features/copilot-custom-instructions.feature.md) records the verified storage/restoration and its limits.

## Copilot code review MCP tools

`repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests` corresponds to **Copilot → Code review → Allow Copilot to use MCP tools when reviewing pull requests**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false
```

This is one optional boolean. Omission, `copilot: {}` and `code_review: {}` leave it unmanaged; explicit false is managed. Null and non-booleans are invalid. Custom instructions and review effort remain independently selectable.

The toggle stores whether future eligible Copilot reviews may use MCP tools. It does not configure MCP servers, create or edit MCP configuration or secrets, grant permissions, enable or request reviews, launch workflows, manage billing or establish entitlement. The personally owned public repository under GitHub Free exposed a configurable toggle even with a No access banner. Configurable storage does not establish actual MCP use or review access.

Plan requires a complete typed settings read and never writes. Apply reads fresh settings, skips a matching boolean and PATCHes only `mcp_enabled` when needed. An independent GET must confirm the requested boolean and preservation of all acquired sibling settings, including custom instructions, configured/effective review effort, available effort choices and severity filtering. Missing, null or non-boolean MCP values, incomplete or malformed sibling settings, denied reads and HTTP 404 are errors rather than false. A configured-null review effort remains a valid sibling value and is preserved as observed. A preservation mismatch fails verification without attributing a concurrent change to this write. Preflight is not a concurrency lock. Mutation/readback failures may leave unknown effects; earlier successful writes remain and later operations stop. There is no automatic retry, rollback or saved-plan application.

Ordinary `export` independently includes the stored MCP boolean, including false. Unavailable observations retain prior declarations with info. Detailed CLI/Action reports use the fixed UI path and native booleans; status-only output hides the control and values.

The official [Copilot code review API schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json) defines GET/PATCH `/repos/{owner}/{repo}/copilot/code-review/settings` and `mcp_enabled`. Repository administrators and classic `repo` scope are documented, and GitHub App tokens are supported; exact fine-grained/App-token grants were not independently established. The credential must already authorize this endpoint; gh-settings does not expand grants. See GitHub's [code-review configuration guide](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review) and the [behavior specification](features/copilot-mcp.feature.md).

## Copilot code review effort level

`repository.settings.copilot.code_review.review_effort_level` corresponds to **Copilot → Code review → Review effort level**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    copilot:
      code_review:
        review_effort_level: lite
```

The optional string accepts `default`, `lite`, `balanced` and `max`. Omission leaves effort unmanaged, independently of `use_custom_instructions`. Null, empty strings, API spelling `inherit` and unknown choices are invalid declarations. Max is available only when the repository reports it as an available effort. This control does not enable reviews, request a review, grant access, change licensing or edit instruction files. Stored configuration does not establish review entitlement or actual review execution.

GitHub reports configured effort, effective effort and available efforts separately. A configured `lite`, `balanced` or `max` remains an explicit override, even when its value equals the inherited effective effort. `default` compares against configured null, not against effective `balanced`. GitHub can also return configured null for a stored override that is no longer available. Therefore export and comparison use `default` only as the **observable** default state: it does not prove that an underlying override was erased. A matching configured-null state produces no difference and apply skips it; applying this declaration is not a forced reset of a hidden override.

Plan is read-only. A selected read requires complete, correctly typed configured/effective effort and availability. Unknown configured or effective choices fail rather than being converted to a supported choice. An explicit `lite`, `balanced` or `max` must be available in the acquired state; it is not silently replaced by another choice.

Apply reads fresh state and availability, skips an observable match, and PATCHes only `review_effort` when a change is needed. `default` sends `inherit`; the other choices send their literal API strings. An independent GET must confirm the requested observable state. Sibling controls, including custom instructions, MCP and severity filtering, are not included in this effort PATCH. Their complete typed values must remain unchanged in readback; a mismatch fails verification without claiming whether the cause was this write or a concurrent change. Mutation or readback failures can leave unknown effects; earlier successful writes remain applied and later work stops. There is no rollback, automatic retry, concurrency lock or saved-plan application.

Ordinary `export` includes the observable effort choice. Unavailable observations retain prior declarations with info. Detailed CLI/Action reports show the fixed UI path and string choices; status-only output hides them.

The [repository settings API](https://docs.github.com/en/rest/copilot/copilot-code-review#get-copilot-code-review-settings-for-a-repository) documents GitHub App and fine-grained token support with Administration read for acquisition and Administration write for updates. These documented grants do not certify a particular installation or review entitlement. The other [credential boundaries](#copilot-code-review-custom-instructions) apply. GitHub documents the [configuration control](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review), [effort choices and Preview status](https://docs.github.com/en/copilot/concepts/agent-capabilities/agent-powered-workflows-and-operations/code-review) and [GET/PATCH schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json). The [behavior specification](features/copilot-review-effort.feature.md) records test ownership and the limits of the live storage evidence.

## Actions repository variables

`repository.settings.actions.variables` corresponds to **Secrets and variables → Actions → Variables → Repository variables**. Navigation-only sidebar groups are omitted from YAML. This current-source feature manages only explicitly named, non-sensitive repository variables:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      variables:
        - name: BUILD_CHANNEL
          value: stable
        - name: OPTIONAL_LABEL
          value: nightly
```

Both `name` and nonempty string `value` are required. Names contain only ASCII letters, digits and underscores, cannot start with a digit or `GITHUB_` (case insensitive), and are unique ignoring case. Internal identity and export names are uppercase. Missing/null/empty values are invalid; quote values that YAML would interpret as numbers or booleans. Without prune, omission and `[]` select nothing and unlisted variables remain unchanged. `--prune actions-variables` makes a present list authoritative for retention; [] deletes all acquired variables, while omission always remains unmanaged. There is no YAML delete or rename member.

Plan reads each selected name individually and shows creates or value updates. Apply reads fresh state, creates absent names with POST, updates changed values with value-only PATCH, skips equal values, and requires exact individual GET readback after a mutation. Only GET 404 means absent under the caller-permissions precondition. Other failed/incomplete reads fail. Mutations and verification failures can have remote effects; earlier writes remain applied, without rollback, retry or concurrency protection.

Ordinary `export` lists all supported Actions repository variables through complete paginated inventory reads. Unavailable/incomplete/unsupported inventories retain prior valid declarations with info. Values are **plaintext** in YAML, exports, detailed plans and Summary. Secrets and organization variables remain outside scope. Agents and Environment inventories are separate independent families; output does not classify or mask variable values.

GitHub's [Variables REST API](https://docs.github.com/en/rest/actions/variables) requires repository Variables read for reads and write for mutations with fine-grained/App tokens. An empty-value create was rejected with HTTP 422 (`Variable value cannot be empty.`); validation rejects empty desired values before requests. No whitespace-trimming rule is inferred. GitHub enforces its [variable limits](https://docs.github.com/en/actions/reference/workflows-and-actions/variables#limits-for-configuration-variables); server rejections are errors. No permission is granted automatically. Free/personal/public UI eligibility and API persistence were observed separately; this is not a runtime entitlement gate. See the [behavior specification](features/repository-variables.feature.md).

### Variable pruning

`--prune actions-variables`, `--prune agents-variables` and
`--prune environment-variables` select independent collections. Their
comma-separated combination uses the common [prune contract](#collection-pruning).
Pruning requires complete name/value inventories, including every page (API
maximum 30 entries per page) and a stable, exact `total_count`. Duplicate names,
missing/null values, count drift and incomplete/denied pages fail before writes.
Prune acquires all current values in a selected collection, not just declared
names; values remain non-sensitive plaintext and can appear in detailed output.
Named export continues to read only its explicit selections.

All candidates are prepared before any remote write. Each family fresh-reads its
inventory, rejects changed candidates, upserts declared values before deleting
its fixed candidates, rechecks the exact acquired name/value before each DELETE,
and requires HTTP 204 plus fresh complete inventory absence. A concurrent change
can still race these checks; there is no lock, retry or rollback. Earlier writes
remain applied after later failure and uncertain DELETE/readback effects are
reported honestly. Plan and detailed Summary display deletion to `absent`, not
an empty value. Status-only output still hides names/values.

Environment-variable pruning touches only listed Environments with an explicit
`variables` array. `variables: []` prunes that Environment; an omitted array does
not. Existing-parent inventory and deletion are bound to its numeric identity,
so detected same-name parent replacement fails. A missing parent is an error
unless the same declaration explicitly creates its Environment policy first;
then no earlier variables are inferred and unexpected deletion candidates are
rejected. Variable-only intent never creates or deletes an Environment, changes
its policy/patterns or accesses secrets. [Environment-object deletion](#environment-pruning)
uses the separate `environments` scope; standalone `all` selects both scopes,
while retaining every listed parent, including variable-only entries.

API evidence: [Actions variable lists/deletion](https://docs.github.com/rest/actions/variables)
and [Agents variable lists/deletion](https://docs.github.com/rest/agents/variables).

## Agents repository variables

`repository.settings.agents.variables` corresponds to **Secrets and variables → Agents → Variables → Repository variables**. This current-source feature manages only explicitly named, non-sensitive repository Agents variables:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    agents:
      variables:
        - name: PROJECT_CONTEXT
          value: public-documentation
```

The [name/value, omission and upsert rules above](#actions-repository-variables) also apply to Agents variables: names are canonical uppercase and values are required nonempty strings. Without prune, omission and `[]` select nothing and unlisted variables remain unchanged. `--prune agents-variables` adds unlisted deletions only for a present list; [] selects all, omission stays unmanaged. There is no YAML deletion/rename member. Without prune, plan reads only selected names; apply reads fresh state, conditionally creates or updates, then verifies the exact value. Failed reads do not become empty state, and failed mutations or verification can have unknown effects without rollback.

Ordinary `export` lists all supported Agents variables through its separate paginated inventory API. An unavailable/incomplete/unsupported inventory emits info and retains prior intent; it never supplies an empty fallback. The same canonical name can exist with different values in Actions and Agents. Values remain **plaintext** in configuration, exports and detailed output; secrets are outside scope.

The [Agents Variables REST API](https://docs.github.com/en/rest/agents/variables) uses `/repos/{owner}/{repo}/agents/variables` and requires the repository **Agent variables** permission at read level for acquisition and write level for mutation with fine-grained/App tokens. This is separate from the **Variables** permission for Actions variables. No permission or token request is added automatically. The Free/personal/public UI and API persistence were verified separately; the probe does not prove actual agent consumption or access for an untested credential. See the [behavior specification](features/repository-variables.feature.md) for evidence and its limits.

## Excluded private Actions controls

As of 2026-11-09, `repository.settings.actions.general.access` and `.fork_pull_request_workflows` are unsupported, including empty mappings and explicit false. The former shares a private repository's actions/workflows; the latter controls private fork workflow execution, tokens, secrets and approval. Neither is a configurable control for the target public repository; the exclusion follows visibility, not account plan. See GitHub's [private Actions sharing documentation](https://docs.github.com/en/actions/how-tos/reuse-automations/share-with-your-organization) and [fork workflow settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository).

The old `--actions-access` and `--private-fork-workflows` export options are rejected before repository/API/file work. No alias or migration is provided. Public contributor approval, Actions execution, workflow permissions, retention and cache controls remain separate. Removing these YAML management paths does not change live settings: their dedicated endpoints are not selected by supported public intent, so no private nested state is read-modify-written. Internal raw identities remain available for validation; an attempted private-only export projection fails instead of silently dropping values.

## Actions event policies

`repository.settings.actions.policies` manages repository-owned policies in
**Settings → Actions → Policies**, with **Target workflows** inclusion/exclusion
patterns and the **Restrict events** and **Restrict actors** rules. These are separate objects from the
[Actions execution policy](#actions-execution-policy). Without prune, omission or
`policies: []` selects nothing, makes no policy API request and deletes nothing.
Unlisted policies remain untouched; removing an entry stops managing it.
Explicit `--prune actions-policies` adds deletions for a present list as described
below.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      policies:
        - policy_name: Restrict workflow events
          enforcement_status: disabled
          target_workflows:
            include: ['.github/workflows/ci*.yml']
            exclude: ['.github/workflows/ci-experimental.yml']
          restrict_events:
            allowed_events: [pull_request, push, workflow_dispatch]
```

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive integer REST ID of an existing policy owned by the target repository; omission means create-only |
| `policy_name` | Required nonempty/non-whitespace string without control characters; identity and collisions use exact spelling |
| `enforcement_status` | Required `disabled` or `active`; neither is inferred |
| `target_workflows` | Optional object with required `include` and `exclude` string arrays; declares the complete workflow selection |
| `restrict_events` | Optional object with `allowed_events`, or `enabled: false` to turn the rule off |
| `restrict_events.allowed_events` | Required when the event rule is enabled; nonempty array of unique supported event strings, compared as a set and exported sorted |
| `restrict_actors` | Optional object with `allowed_actors`, or `enabled: false` to turn the rule off |
| `restrict_actors.allowed_actors` | Required when the actor rule is enabled; array of unique actor identities, including `[]` for checked with no allowed actors |

Each allowed actor selects exactly one identity: `{user: 155234749}` for a
positive REST user ID, or `{role: write}`, `{role: maintain}` or
`{role: repository_admin}`. These roles map to the verified `RepositoryRole`
IDs 4, 2 and 5. User IDs identify existing accounts; they are not login strings.
No access is granted by adding an allowed actor. Other actor kinds and role IDs,
including Bot, App, installation and team identities, are unsupported.

A restriction object without `enabled`, or with `enabled: true`, requires its
allowed array. `enabled: false` forbids that array and removes that rule.
An empty actor array keeps **Restrict actors checked with no allowed actors**;
it is not the same as turning the rule off. Empty event arrays remain invalid.
Nulls, wrong types, unknown keys, duplicate identities and mixed actor identities
fail locally.

On creation, omitted restrictions are off and at least one rule must be enabled.
On ID-selected updates, an omitted restriction preserves the freshly acquired
rule, including its absence. Metadata-only updates may omit both restrictions.
Explicitly disabling the final rule fails before writing. The complete effective
policy must contain at least one supported restriction. For example:

```yaml
restrict_events:
  enabled: false
restrict_actors:
  allowed_actors:
    - role: repository_admin
    - user: 155234749
```

This changes only the supported restriction selections on that policy; omitted
workflow targeting still follows the preservation contract below.

Supported events are `branch_protection_rule`, `check_run`, `check_suite`,
`create`, `delete`, `deployment`, `deployment_status`, `discussion`,
`discussion_comment`, `fork`, `gollum`, `image_version`, `issue_comment`, `issues`,
`label`, `merge_group`, `milestone`, `page_build`, `project`, `project_card`,
`project_column`, `public`, `pull_request`, `pull_request_review`,
`pull_request_review_comment`, `pull_request_target`, `push`, `registry_package`,
`release`, `repository_dispatch`, `schedule`, `status`, `watch`, `workflow_call`,
`workflow_dispatch` and `workflow_run`.

Duplicate IDs or exact names, missing required fields, nulls, unknown keys and
unsupported event/enforcement values fail locally. A no-ID entry never adopts
an existing name: a collision fails and identifies the existing ID. An ID must
exist in this repository; missing or foreign IDs never fall back to creation.
An ID-bound declaration replaces the complete supported event object and may
rename it to a noncolliding name. Renames cannot rely on another planned rename
freeing the name. There is no delete declaration or name-based update.

`target_workflows` corresponds to **Include by pattern** and **Exclude by
pattern** in the UI. Both arrays are required when the object is supplied, and
at least one must contain a pattern. Empty `include` means every workflow except
the exclusions. `~ALL` represents **All workflows** and is valid only as the sole
include entry, never in exclusions. For example, `include: [~ALL]` with
`exclude: []` explicitly resets targeting to All workflows. An exclude-only
selection uses `include: []` and one or more exclusion patterns.

Patterns must be nonempty/non-whitespace strings without control characters;
duplicates within either array are rejected. Case and exact pattern spelling
are preserved. Comparisons treat each array as a set and exports sort it. All
workflows normalize to `[~ALL]` with empty exclusions; exclude-only targeting
uses an empty include array. The tool passes patterns to GitHub and does not
emulate workflow matching.

Omitting `target_workflows` creates an All workflows policy, but preserves an
existing policy's freshly acquired targeting on update. It does not reset or
broaden an existing selection. To change targeting, declare both arrays. Remote
omitted/null/empty `conditions`, or `workflow_path` with empty exclusions and
empty includes or `[~ALL]`, represent All workflows. A present `workflow_path`
requires both typed arrays. Unknown conditions, unsupported actor identities,
Preview rules, unknown rules/fields and unsupported event/enforcement values are not
lossily converted. A complete unsupported object's ID/name is retained so
unlisted objects stay unmanaged without prune; selecting it for update, as a
prune candidate, or including it in a full collection export fails. Incomplete or inconsistent reads always fail.
Currently unsupported actor types and Preview lockfile controls require target
UI/API assessment and implementation; Preview status alone is not an exclusion.
[OIDC subject configuration](#actions-oidc-subject-template) has separate support.
Workflow and other feature configuration-file authoring remain excluded.

`--prune actions-policies` adds deletion of unlisted repository-owned IDs only
when `policies` is present; `[]` selects every current policy as a candidate.
An omitted list remains unmanaged. Deleting a policy removes its workflow
restriction, so future workflow execution may be permitted. All candidates must
be completely supported; unknown/Preview rules, unadopted actor types or targeting,
wrong ownership, incomplete reads and missing permissions fail before any write.
Apply fixes the candidate set, performs creates/updates first, rechecks exact
identity/state before each DELETE, requires HTTP 204 and confirms ID absence in
a complete fresh inventory. Earlier successful changes survive a later failure.
There is no policy `delete` YAML field.

Plan acquires the complete repository collection and details before showing
creates/ID-bound updates and explicitly requested prune deletions. Apply freshly acquires and preflights the whole
collection before its first policy write, skips matches and rechecks each
selected update's detail immediately before mutation. Drift or newly unsupported
state fails rather than overwriting it. Create uses POST and update uses PUT at
the [official repository Actions policies API](https://docs.github.com/en/rest/actions/policies).
Changed explicit targeting is sent as `conditions.workflow_path`; omitted
targeting uses All workflows on create and preserves the preflight-verified
selection on update. Unchanged targeting need not be resent. Complete write acknowledgment and
an independent final GET must both match the requested identity, name,
enforcement, event/actor rule presence and identity sets, and supported targeting.

Failures stop later policy writes; earlier successful changes remain applied.
A write/verification failure may leave the remote effect uncertain. Known new
IDs remain available for recovery, including acknowledgment/readback failures;
uncertain creation is never automatically retried. There is no transaction,
rollback or concurrency lock. Inspect remote state and record the correct ID
before retrying. Local apply records confirmed creations using
[guarded ID writeback](#apply-id-writeback), with `policy_name` as selector;
`--no-writeback` and `GITHUB_ACTIONS=true` suppress it.

Ordinary `export` independently observes the full repository collection,
including IDs and explicit canonical `target_workflows`, sorted by exact policy
name with sorted events, actor identities and patterns. Export explicitly emits
`enabled: false` for absent restriction rules, so a snapshot does not accidentally
preserve an existing rule. All workflows exports as `include: [~ALL]`
and `exclude: []`, so a saved snapshot owns its complete selection. All pages and details must be complete and representable before adoption;
a failure emits info and retains the prior collection while other reads continue.
Fine-grained and GitHub App tokens require **Administration write even for
list/GET, export and plan**, as well as writes, according to the official API.
The client never grants permissions or substitutes empty state for denied reads.

The Free/personal/public UI and disabled-policy API storage, including
include/exclude patterns, omitted-target preservation, explicit All workflows
reset, User and the three RepositoryRole identities, actor-only/combined rules,
checked-empty actors and unchecked-rule removal, were verified;
local fixtures cover active/disabled metadata transitions. Actual enforcement
and workflow execution were not tested. Storage readback does not establish
workflow blocking or a particular token's grants. See the
[behavior specification](features/actions-event-policies.feature.md) and
[event-policy adoption evidence](https://github.com/daiksud/gh-settings/issues/647)
and [workflow-targeting evidence](https://github.com/daiksud/gh-settings/issues/649).

## Actions execution policy

All paths below are relative to `repository.settings.actions.general.actions_permissions`:

| Field | Choice / type |
| --- | --- |
| `policy` | `allow_all_actions_and_reusable_workflows`, `disable_actions`, `allow_owner_actions_and_reusable_workflows`, `allow_owner_and_select_non_owner_actions_and_reusable_workflows` |
| `require_actions_to_be_pinned_to_a_full_length_commit_sha` | boolean |
| `allow_actions_created_by_github` | boolean; selected-policy companion |
| `allow_actions_by_marketplace_verified_creators` | boolean; selected-policy companion |
| `allow_or_block_specified_actions_and_reusable_workflows` | ordered string array; selected-policy companion |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        actions_permissions:
          policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows
          require_actions_to_be_pinned_to_a_full_length_commit_sha: true
          allow_actions_created_by_github: true
          allow_actions_by_marketplace_verified_creators: false
          allow_or_block_specified_actions_and_reusable_workflows: ['my-org/*', '!my-org/blocked@*']
```

Selecting an allow policy explicitly enables Actions and selects its policy; `policy: disable_actions` declares only disabled execution, leaving latent allowed policy unmanaged. There are no separate public API-shaped enabled/allowed fields. Omitting `policy` does not implicitly enable Actions. SHA pinning is independent; reusable workflows can still use tags.

Companions can accompany the selected non-owner policy or an omitted policy, not another explicit policy. An omitted policy requires an already selected observed policy for companion updates. Transition from a nonselected policy requires all three companions: inactive values cannot be acquired or invented. Plan shows them as unavailable before the transition. Existing selected policy permits partial declarations merged with complete fresh companions.

Patterns are at most 1000 nonempty strings. An explicit array replaces the entire sequence, and `[]` clears it. Case, order, duplicates, wildcards and negative entries are preserved; the tool does not emulate matching. Removing a negative entry may permit more actions. GitHub documents pattern applicability limits; configuring a list alone does not prove enforcement on private repositories.

Ordinary `export` includes the execution policy, SHA control and visible acquired companions in the same section. Disabled/other policy hides selected companions; complete raw acquisition is still required. Failed or incomplete acquisition retains this family’s prior declaration with info, without erasing independently acquired Actions siblings. Correctly typed unknown execution policies follow the observation policy below.

Apply reads complete fresh state and skips matching intent. A needed base PUT includes the endpoint-required enabled value, using the fresh companion when omitted, and only changed optional fields. A needed selected PUT preserves omitted companions from the complete active read. Policy changes precede selected writes. When enabling a disabled repository with selection, it selects policy while disabled and installs companions only as needed, then enables. Independent readback verifies the original complete requested state and preserved raw companions, including initially matching declared values. Repeated matching apply performs no PUT.

Operations are not transactional: an already enabled target can temporarily use remembered/default selection between writes. Earlier policy changes survive later refusal; failed writes/readback have uncertain effects. There is no rollback, saved-plan execution, automatic retry, lock or permission expansion. Existing Administration read/write and organization/enterprise policy remain prerequisites; policy changes can stop future workflows. See [execution policy](features/actions-execution-policy.feature.md).

## Sponsor button visibility

Use `repository.settings.general.features.sponsorships`. This boolean controls the Sponsor button, not funding destinations. Omission is unmanaged; false disables. `.github/FUNDING.yml` is never created, read, changed or deleted by this control. Ordinary export includes the observed value through the existing shared GraphQL path. Apply compares the declared boolean with fresh state, skips a match and independently verifies a changed value after mutation acknowledgment. See [Sponsorships](features/sponsorships.feature.md).

### Execution-policy observation normalization

A complete, correctly typed unknown `allowed_actions` string (including empty or whitespace) retains its exact raw value and projects to the named `allow_all_actions_and_reusable_workflows` fallback. Both acquired booleans remain unchanged. With execution enabled, export warns and emits that fallback; a declared policy keeps one UI decision actionable even at fallback equality. Disable owns one native field; enabling owns two, or five with all three selected companions. Managed counts reflect native fields internally.

With execution disabled, the visible choice is the known `disable_actions`, so the latent unknown policy produces no projection warning. Export omits latent choices, and a matching Disable plan has no differences. Enabling requires an explicit canonical policy; selected intent still needs all three companions and configures them before enabling. Private radio reports retain raw provenance for closed report validation/presentation but detailed output does not disclose hidden policy; status-only output suppresses all policy values.

A needed base update sends only changed declared policy/SHA members with the endpoint-required enabled value. Disable and SHA-only updates preserve the exact raw policy; SHA-only updates do not warn about omitted policy intent. Fresh raw readback must confirm declared canonical policy or preserved raw policy, both booleans and any managed selected companions. An unchanged unknown string cannot confirm a declared fallback, and a different unknown token cannot confirm preservation. Failed verification retains uncertainty about writes and earlier completed resource work. Direct repeated apply skips raw matches, including an already disabled target with unknown latent policy. Missing/null/wrong types and failed acquisition never become fallbacks.

## Creation-policy observation normalization

Complete successfully acquired Issue/PR creation-policy strings outside their two UI choices project to the named `all_users` fallback with native value `ALL`. Empty and whitespace strings remain exact acquired evidence. One scalar record retains its YAML path, original token and fallback. Missing/null members, wrong types, failed/partial reads and incomplete PR availability remain errors. Repository IDs/names retain strict identity validation.

Export warns only for visible normalized policies and saves UI choices; disabled feature controls hide their latent policy without warnings or replay. Plan retains one write-required field decision even when desired intent equals the projected default. Omitted desired policies remain unmanaged. Apply acquires identity and only the originally declared PR availability, PR/Issue creation policies and Sponsorships members. Exact native matches require no mutation or second read; otherwise one shared mutation sends only changed declared values. Its response and an independent readback must confirm every original declared member against the original repository ID, including matching members omitted from the payload. Unknown raw policy strings remain write-required even at fallback equality. Checkbox-only intent does not acquire or warn about omitted policies; policy-only intent does not acquire or enable an omitted feature checkbox. Missing/null/wrong-type selected values and failed or partial acquisition stop before mutation. An unchanged unknown token, changed ID or declared-member drift after mutation fails with uncertain write effects. Only actual successful writes count toward later partial progress; earlier completed REST writes remain applied if access fails. No rollback occurs.

## Issue creation policy

Use `repository.settings.general.features.issue_permissions.creation_allowed_by`: `all_users` or `collaborators_only`. It can be declared independently with the Issues checkbox omitted; it never implicitly enables Issues. Explicit Issues=false plus this choice is invalid. Export omits it behind disabled Issues. Existing declared-only GraphQL writes and independent observation remain; it does not manage existing issues or access grants. See [Issue permissions](features/issue-creation-policy.feature.md).

## Pull request access

Use boolean `repository.settings.general.features.pull_requests` and independent choice `features.pull_request_permissions.creation_allowed_by` under the same General root. Choices are `all_users` and `collaborators_only`. Explicit false plus a permission choice is invalid; omitted enabling checkbox is independent. Export hides its policy when Pull requests is disabled. Shared identity-checked GraphQL reconciliation skips fresh matches and sends only changed declared values. Mutation responses and independent same-ID readback confirm all original declared members, including the checkbox. No collaborator invitation/grant is implied. See [PR access](features/pull-request-access.feature.md).

## Default branch

`repository.settings.general.default_branch` selects an existing branch using a valid literal name; it does not create, rename or move a branch. Empty, whitespace/ref-syntax-invalid or wrong-type values fail local validation. Existing existence/readback and target checks remain authoritative. See [default branch](features/default-branch.feature.md).

## Template repository

`repository.settings.general.template_repository` is an optional boolean. It controls template availability without adding repository generation or copying. See [template repository](features/template-repository.feature.md).

## Web commit sign-off

`repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits` is an optional boolean. It controls the web-based commit sign-off requirement, not signature verification or existing commits. See [sign-off](features/web-commit-signoff.feature.md).

## Discussions

`repository.settings.general.features.discussions` is an optional boolean, independent from discussion categories and individual discussions. Complete typed observation is required; absent values cannot become false. See [Discussions](features/discussions.feature.md).

## Merge methods

Use General's `pull_requests.allow_merge_commits`, `allow_squash_merging` and `allow_rebase_merging` booleans. These independent optional checkboxes do not create individual merges. Explicit false remains intent. See [merge methods](features/merge-methods.feature.md).

## Automatic branch deletion

General's `pull_requests.automatically_delete_head_branches` controls GitHub's automatic deletion after merge; it does not manage or delete individual branches directly. See [branch deletion](features/delete-branch-on-merge.feature.md).

## Commit-message observation normalization

A complete successful GraphQL read with two strings outside a merge/squash menu's UI choices projects to that menu's `default_message`. One provenance record retains both exact raw strings, its YAML menu path and fallback. Missing/null members, wrong types and failed or partial acquisition remain errors. Hidden disabled menus are omitted from export; visible normalized menus warn and save only the UI choice. Omitted desired menus remain unmanaged without warnings or replay.

Plan retains one warning and one menu-level write-required decision even when the declaration equals the projected default. That projection does not prove remote equality. Apply selectively reads declared menus, sends only declared native members in PATCH and verifies both raw members through a fresh GraphQL read. Unchanged out-of-UI readback fails with possible write effects retained. A failed preflight makes no repository PATCH; a later verification failure does not roll back other members in that PATCH.

## Squash commit defaults

General's `pull_requests.squash_merge.default_commit_message` accepts `default_message`, `pull_request_title`, `pull_request_title_and_commit_details` or `pull_request_title_and_description`. One UI choice translates to both internal API members; partial API pairs are not public YAML. Explicitly disabling squash while declaring the menu fails. An omitted enabling checkbox remains omitted. See [squash defaults](features/squash-commit-defaults.feature.md).

## Auto-merge and branch updates

General's `pull_requests.allow_auto_merge` and `always_suggest_updating_pull_request_branches` are independent optional booleans. Existing platform prerequisites still apply. See [PR options](features/pull-request-options.feature.md).

## Merge commit defaults

General's `pull_requests.merge_commit.default_commit_message` accepts `default_message`, `pull_request_title` or `pull_request_title_and_description`. It translates to a complete API pair. Explicitly disabling merge commits with this menu fails locally. Existing runtime dependency validation also rejects menu intent on an observed disabled merge method unless the declaration enables it. It never enables the method implicitly. See [merge defaults](features/merge-commit-defaults.feature.md).

## Topics

`repository.about.topics` manages the complete topic set. It is a direct array, not a mapping. Omission leaves it unmanaged; `[]` explicitly clears all topics; `{}` and null are invalid. Names are unique lowercase letters/numbers/hyphens, 1–50 characters each, at most 20 entries. Names are public metadata even on private repositories. Export, comparison and PUT use sorted set order without rewriting existing input YAML. Apply reads complete, validated current membership and skips PUT when the set matches, regardless of order. Changed membership uses one full-set PUT followed by an independent GET; the PUT response alone is not confirmation. A failed prerequisite read makes no Topics write. A rejected update or failed/mismatching readback stops later resources with uncertain effects and no automatic retry or rollback. Metadata read and Administration write must already exist. See [Topics and ordering](commands.md#apply-behavior).

## Deploy keys

Current source manages the **Title**, public **Key** and **Allow write access**
controls for a personally owned public repository. [GitHub documents these UI
controls](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/managing-deploy-keys)
and [list, get, create and delete REST operations](https://docs.github.com/en/rest/deploy-keys/deploy-keys).
Administration read is needed for observation and write for creation/deletion.
The client never generates keys, reads private keys, uses an SSH agent or tests SSH
access. API acceptance and a stored configuration do not prove actual key usability.

```yaml
repository:
  settings:
    deploy_keys:
      - id: 456
        title: build-server
        key: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB
        allow_write_access: false
```

The example public material is a fixture, not a usable deployment credential.
Every entry requires native string `title` and `key` and boolean
`allow_write_access`. Empty titles are accepted; titles may repeat. Reject title
control characters, nulls, duplicate positive IDs and duplicate parsed key material.
`key` must contain one valid authorized-public-key line, optionally ending in one
LF/CRLF and a trailing comment. Options, private formats, internal newlines,
multiple keys and mismatched algorithm tokens fail locally without echoing input.
A maintained SSH parser determines syntax; it does not promise GitHub algorithm
support. Parsing is bounded to 64 KiB. Export and creation use canonical public
material without comments, and `allow_write_access` is the inverse of REST
`read_only`, including explicit false.

Omission is unmanaged. `deploy_keys: []` preserves existing keys during ordinary
apply. Export requires complete target identity and paginated, typed inventory;
unavailable reads emit bounded info and retain valid prior declarations, or omit
this family on first export. Missing fields never become empty inventories or
false permissions. Export includes IDs. The optional server `enabled` field is
private evidence: missing means unknown, null is invalid, and neither state is
converted into declared access.

An ID-bound exact match is a no-op. A missing ID or changed title, material or
access flag fails: there is no update endpoint, implicit replacement or fallback
creation. IDless entries create only when their canonical key does not exist;
existing material requires explicit ID binding. Titles and comments cannot cause
adoption. Before creating, apply checks the complete target/inventory again. A
201 acknowledgment, independent GET by the new ID and complete inventory readback
must confirm the configuration and preservation of previous keys. Only then may
local CLI writeback add the ID to the unique canonical-key entry, preserving its
original key scalar/comment. Existing IDs, ambiguous/aliased entries and concurrent
file changes block writeback. Action and `--no-writeback` leave YAML unchanged;
record the reported fingerprint and ID against the matching key before repeating.

A positive new ID from a successful creation response may be reported separately
for recovery when confirmation fails. It is not written into YAML, does not prove
successful creation verification and never comes from an error response or reused
ID. Later work stops; earlier verified writes remain. Reports show title,
SHA256 fingerprint and grant, with numeric ID when selected; raw key material and
private snapshots stay out of plan reports. Status-only also hides this identity.
Raw transport logging is suppressed even with `GH_DEBUG=api`.

### Deploy key pruning

`--prune deploy-keys` and standalone `all` add deletion only for a present list.
Selected IDs are retained. Complete target and all key snapshots are frozen before
any write, including unrelated changes and unarchive. Before each DELETE, complete
inventory and candidate GET must still match. Expected state includes only this
run's verified creations/deletions and known own-unarchive transition; new external
keys never become inferred candidates. ID, parsed material, title, grant, creation
time, canonical API URL and exact unknown/false/true `enabled` state are guarded.
Dynamic `verified`, usage time and creator attribution are typed but do not trigger
identity drift. The server URL is validated against the fixed GitHub.com target
route and is never followed. Unknown fields or incomplete pages block acquisition.

Deletion requires a 204 empty acknowledgment and fresh complete inventory proving
absence and preserved remaining keys. GET404 alone proves nothing. A failed
readback can leave a deleted key and uncertain remote effects; no rollback or retry
is attempted. Deletion revokes that repository credential. Replace immutable
configuration by explicitly pruning the old ID, then creating the replacement in
a later apply. A distinct new key can be created before pruning an old key, with
temporary overlapping access. This is not automatic rotation. Removing a creator
or uninstalling an App is not deploy-key cleanup.

See the [behavior and verification specification](features/deploy-keys.feature.md).

## Personal collaborators

`repository.settings.collaborators` manages people on Settings > Collaborators
for personally owned public repositories. Each entry has a required `username`
and optional positive immutable GitHub **user** `id`:

```yaml
repository:
  settings:
    collaborators:
      - username: existing-person
        id: 123456
      - username: new-person
```

An ID identifies the person, not an invitation, membership or proof of accepted
access. Export includes supported active people and pending invitees with their
user IDs, excluding the owner by owner ID. Personal collaborators have GitHub's
fixed write permission; no role or permission field is accepted. Email-only
invitations have an unconfirmed complete API roundtrip and remain unsupported.

Omission is unmanaged. Without prune, `[]` and unlisted people cause no access
mutation. Explicit [`--prune collaborators`](#collaborator-pruning), including
standalone `all`, adds bounded invitation cancellation and active-grant removal
for a present collection.
An ID-bound person must still be active or pending; disappearance fails without
reinviting. A username-only entry is initial-add intent. Existing active access
or a matching pending invitation makes no PUT; apply records the verified user ID
locally. Otherwise it sends one bodyless invitation PUT and independently reads
back the complete active and pending inventories. A 201 must match the exact
invitation or the same user's now-active access; a 204 must verify active access.
A successful invitation is not acceptance. CLI plan and detailed Action Summary
show pending acceptance even when no new invitation is planned.

A username-only binding is a local change, distinct from a remote notification.
After confirmation, normal CLI apply records the user ID using the source
snapshot and safe-file checks shared with created-resource IDs. It never records
an unverified ID. Earlier confirmed bindings remain if a later operation fails;
local binding alone is not reported as a completed remote write. `--no-writeback`
or `GITHUB_ACTIONS=true` suppresses file changes and reports the verified ID for a
reviewed declaration update. Persist that ID before relying on repeated scheduled
runs. No Git commit or push is made. Output or file-write failure can occur after
remote effects; diagnostics preserve verified user IDs and reported invitation IDs
for deliberate recovery, without claiming acceptance or inviting again.

Saved user IDs prevent ordinary reinvitation after expiry, decline, cancellation
or revocation. Deliberately removing an ID restores initial-add intent and can
notify that person again. There is no hidden journal: if an ambiguous request was
sent, no verified binding was saved and the invitation later disappears, another
username-only invocation cannot prove the earlier request happened. Resolve access
before retrying. No automatic retry, polling, acceptance or rollback occurs.
Cancellation and removal require explicit prune intent.

Both inventories must be fully paginated and typed. Missing/null invitees,
unsupported actors/roles, malformed state, explicit `expired: true`, duplicate
identities and active/pending overlap make the family unavailable. Missing expiry
metadata does not promise an expiry date. Export omits an initially unavailable
family or retains its complete valid prior declaration; it never combines fresh
active rows with retained pending rows. Declared plan/apply observations fail
strictly. Denial or 404 is not an empty access list.

Case-only usernames match; export uses acquired spelling. ID-bound entries resolve
the user by ID and reject genuine renames with the current username for deliberate
correction. Reused usernames never silently select a different ID. Apply resolves
all declared people before the first collaborator operation, then rechecks the
repository/owner IDs, personal-public status, complete inventories and current
username identity before each request. A prepared pending invitation that vanished
is not resent. Without prune, an explicit own unarchive occurs before these reads; archived
repositories cannot receive a new invitation. These reads are not atomic, and the
username-addressed PUT has no atomic expected-user-ID condition, leaving a narrow
concurrent rename/reassignment race. Later failures stop subsequent operations
without undoing earlier changes.

The [Collaborators REST API](https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2026-03-10),
[Invitations REST API](https://docs.github.com/en/rest/collaborators/invitations?apiVersion=2026-03-10)
and [user identity API](https://docs.github.com/en/rest/users/users?apiVersion=2026-03-10)
define these operations. Complete observation requires collaborator-list access
and Administration read for invitations; mutation requires Administration write.
Installation-token support is documented, but local fixtures do not establish a
real App's grants or a recipient's notification/acceptance. See the
[behavior specification](features/repository-access.feature.md).

### Collaborator pruning

`plan --prune collaborators` and `apply --prune collaborators` use the existing
`repository.settings.collaborators` list. Omission stays unmanaged even under
`all`; `[]` selects every supported non-owner active grant and pending invitation.
A nonempty list retains the union of ID-bound people and resolved username-only
people, including initial-add intent. The owner is always protected. No deletion
marker, role expansion, email invitation, recipient acceptance or installation
removal is introduced.

Before **any** remote write, including unarchive and unrelated settings, apply
acquires complete supported active/pending inventories and freezes selected user
IDs, active removal candidates and invitation-ID cancellation candidates. Failed,
partial, duplicate, unsupported or ambiguous evidence blocks the whole preflight.
The same credential must positively identify the personal repository owner, or
prove installation target access through a complete authenticated
[installation repository inventory](https://docs.github.com/en/rest/apps/installations?apiVersion=2026-03-10#list-repositories-accessible-to-the-app-installation)
and complete admin-only invitation visibility. A failed `/user` alone is not
installation evidence. Ambiguous positive identities and a non-owner user fail.
No App/installation ID is invented, no second credential is requested, and token
prefixes or `GITHUB_ACTOR` do not determine the management path. Administration
write remains a mutation prerequisite; successful reads do not prove that grant.

Inviter identities are private dependency evidence, never YAML or report values.
An active removal is refused if a retained invitation has unknown inviter identity
or was sent by that candidate. Only initially selected invitations are cancelled;
a late invitation cannot be cancelled to unblock removal. Ordinary selected
reconciliation and verified local bindings run first, followed by fixed invitation
cancellations, then fixed active removals, before Environment work. A selected
pending person's acceptance may advance to active; a cancellation candidate's
acceptance, disappearance or replacement fails instead of becoming a grant removal.
Frozen selected IDs prevent later username reassignment from changing retention.

Every DELETE freshly checks the exact repository/owner, management path and complete
remaining inventories, accounting only for confirmed operations. Only a verified
own unarchive may change expected archive state. After cancellations, active
removal requires no remaining invitation dependency and a last immutable-ID and
username-to-ID check. A bodyless repository invitation-ID or collaborator-username
DELETE must return empty 204, followed by independent complete readback. It must
confirm absence and unchanged remaining access; 204 alone is not verified success.
CLI/detailed Summary distinguish cancellation from removal. Apply diagnostics
separate attempted, acknowledged and verified effects, retain earlier bindings and
verified deletions, and stop later work without retry, rollback or failed-readback
adoption. Local binding alone is not a remote partial write.

GitHub documents additional collaborator-removal effects on issue assignments,
stars, package access, invitations and forks. These may settle asynchronously;
verified list absence does not certify their completion. Removing a grant does
not make public content inaccessible. Inventories are not atomic, the DELETE is
username-addressed without an expected-ID condition, and invitation creation can
race the final dependency check. Removing and re-adding an identical active grant
between reads is indistinguishable because there is no membership incarnation ID.
Review these effects before apply; there is no transaction or hidden journal.

## Labels

`repository.labels` selects whole label objects by positive REST `id`; no ID means create-only. Without prune, omission or `[]` selects nothing and unlisted objects remain untouched. Explicit `--prune labels` changes present-list ownership: [] selects all current labels and nonempty lists retain declared IDs. Omission always stays unmanaged. Deletion loses Issue/PR assignments; see [collection pruning](#collection-pruning).

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive integer; selects an existing label in the target repository; missing ID errors, never create fallback |
| `name` | Required nonempty/non-whitespace string without control characters; spaces, punctuation and emoji allowed |
| `color` | Required `#` plus six hex digits; quote the string; canonical lowercase, no guessed default |
| `description` | Optional string, at most 100 Unicode characters; omission defaults to `''` on create/update, clearing an existing description |

Duplicate IDs and Unicode-simple-folded names fail without trimming. ID-less collisions show the existing ID for explicit adoption. A valid ID can rename to a noncolliding name while preserving assignments; case-only differences keep remote spelling. Renames cannot collide even with another planned rename; apply is not a rename transaction. Export requires complete IDs/settings and pagination, includes only writable supported values and sorts labels consistently. Incomplete reads fail rather than becoming empty state.

Apply compares a fresh complete collection and skips matching selected objects.
Updates check the selected ID at its observed name before writing. After each
create/update response passes validation, apply independently reads the resulting
name and requires the same positive ID, name, color and description. Null API
descriptions normalize to empty strings. A failed or mismatching readback stops
later operations, including prune deletions, with uncertain write effects; a
matching acknowledgment alone does not establish success. These checks do not
lock the label against concurrent changes.

### Apply ID writeback

Normal CLI apply records each confirmed created Label/Ruleset/Actions policy ID in the original current-schema collection after creation verification, before another write. Label verification includes independent readback. Rulesets use `ruleset_name` and Actions policies use `policy_name` as their selectors. Existing IDs and unrelated values remain; comments are retained where supported and formatting may normalize. Regular `.github` and file entries plus the initial byte snapshot and earlier ID additions are required. Aliased/merged collection entries are not rewritten.

`apply --no-writeback` disables this local update. `GITHUB_ACTIONS=true` always disables it, including public Action use. Created IDs remain in output/recovery diagnostics; no Git commit or push is performed. Plan never writes. A confirmed creation survives writeback/later remote failure; add its ID before retrying. Unverified responses do not supply a confirmed writable ID. There is no automatic create retry, delete or rollback. Guarded file replacement is atomic, not a concurrent-editor lock. Label reads/mutations need the existing Issues or Pull requests grants; Administration alone does not replace them.[^github-labels] See [Labels](features/labels.feature.md).

A positive Label ID from the creation response remains in recovery diagnostics
even if later validation or readback fails. It is not automatically written to
YAML on that failure. Retain the reported ID and inspect the remote object before
retrying; earlier confirmed IDs already recorded in YAML remain intact.

## Configuration semantics

The version marker is independent of the CLI release. Only one YAML mapping document is accepted. `repository` is optional; version-only, `repository: {}` and empty settings mappings are local no-ops without origin/authentication/API requirements. A present collection entry must still satisfy its complete object rules. Empty ordinary section objects add no scalar intent; they do not enable a parent or fill a policy. A parameterized rule object within a selected Ruleset still selects that rule, and a name-only Environment selects creation only when missing.

Unknown/duplicate/non-string keys, unsupported tags, malformed/multiple documents, wrong shapes, invalid choices and wrong scalar types fail before remote work. Aliases/merges are validated after resolution. Strings must remain strings, booleans must be true/false and integers must fit their declared range. Explicit null is invalid except the nullable Pages custom-domain scalar. Empty strings, zero, false and empty arrays retain their distinct contracts. No desired defaults are populated from observation metadata or planned #480 fallbacks.

Scalar omission is unmanaged. Labels/Rulesets/Actions policies own only listed whole objects with their stated field rules. Existing Environment built-in omissions are preserved from fresh state; creation omissions remain API-controlled. Topics and explicitly declared inner lists have their authoritative collection semantics. These distinctions remain the same through CLI and public Action use.

## Exported values

Export completes acquisition and conversion before writing. All independent families follow the [automatic export omission/retention contract](features/readable-settings-export.feature.md); checkout/origin resolution, invalid prior files, family-info output and destination failures remain fatal. About/General observation groups are independent too; unavailable settings retain valid prior declarations only where compatible with positively acquired settings, with bounded info. No incomplete family becomes plausible empty/false YAML. Incomplete data remains an acquisition failure. Correctly typed out-of-UI Security statuses follow their [Security policy](#secret-scanning-and-push-protection), and public-fork approval strings follow their [fork policy](#fork-pr-workflow-policies). Complete workflow permission strings, merge/squash message pairs and creation-policy strings follow their [workflow](#default-workflow-permissions), [menu](#commit-message-observation-normalization) and [creation-policy](#creation-policy-observation-normalization) fallback policies. Complete Pages modes and legacy folders follow their [raw-source policy](#pages-configuration); Actions execution follows its [visible/latent policy](#execution-policy-observation-normalization). Ruleset enforcement follows its [ID-bound policy](#ruleset-enforcement-observation-normalization); Environment deployment modes follow their [raw-pair policy](#environment-deployment-mode-observation-normalization), and pattern ref types follow their [raw-identity policy](#environment-ref-type-observation-normalization). Only the explicitly documented typed fallback policies normalize observations; unsupported identities and incomplete values remain errors. Ordinary export attempts every supported family without selectors; excluded controls and secrets remain out of scope. Hidden conditional controls are omitted without replaying latent API values as selected intent. IDs, false/zero/empty values and nullable Pages domains retain their declared meanings.

An export followed by validate/plan roundtrips exact supported acquired values, subject to the resource-specific ownership and normalization limits below. A normalized fallback does not establish raw equality. Export does not claim live entitlement, effective protection or transactional apply. It overwrites existing declaration content; review before adoption.

## Rulesets

`repository.settings.rulesets` selects whole repository-owned branch or tag Rulesets by optional positive numeric ID. No ID means create-only; a collision requires explicit ID adoption. Unknown/wrong-target/type IDs fail without create fallback. Without prune, unlisted objects are not deleted and omission/`[]` selects nothing. With `--prune rulesets`, present lists own retention as described below. Names compare exactly without trimming/case folding; duplicate names/IDs and colliding renames fail.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 123
        ruleset_name: release-tags
        enforcement_status: active
        target_tags: {include_by_pattern: ['v*']}
        rules: {restrict_deletions: true}
```

Deletion uses the command option `--prune rulesets`, not YAML tombstones. Without
prune, only listed objects are created/updated. With prune, an omitted Rulesets
list stays unmanaged, `[]` selects all acquired Rulesets for deletion, and a
nonempty list retains its declared IDs while pruning unlisted objects. Declare
all Rulesets to retain before pruning. The [prune contract](#collection-pruning)
owns preflight, ordering and partial-failure behavior.

The `delete` member is removed: old `delete: true`/`false` declarations fail with
migration guidance. Remove the unwanted entry only after listing every other
Ruleset to keep; inspect `plan --prune rulesets`, then use `apply --prune rulesets`.
This is broader than the old single-object selector. Complete unsupported,
inherited/Push or normalized policy remains a deletion blocker. A complete supported
PR rule is eligible, including either additional-approval value.
Exact identity/policy preflight and HTTP 204 plus complete absence readback are
required; no retry, substitution or rollback is introduced.

Deletion uses [GitHub's repository Ruleset DELETE endpoint](https://docs.github.com/en/rest/repos/rules#delete-a-repository-ruleset) with Administration write permission. Complete bypass visibility is required before any deletion. Explicit deletion removes the whole selected Ruleset. Any acquired unsupported protection prevents deletion; an omitted API field does not establish that its underlying protection is disabled. Organization/Enterprise-level policy, legacy branch protection, grants, releases and bypass configuration are outside this operation. Implementation verification uses local fixtures and does not delete live protection.

Every present-policy entry requires `ruleset_name`, `enforcement_status` (`active` or `disabled`) and exactly one of `target_branches`/`target_tags`. Branch targets support `include_default_branch`, `include_all_branches`, `include_by_pattern` and `exclude_by_pattern`; tag targets support `include_all_tags` and the pattern arrays. Patterns use the UI spelling without internal ref prefixes. An explicit empty target object selects empty include/exclude lists; omitting all target selectors is invalid. No protected scope is guessed.

Omitted `bypass_list` and unselected `rules` become empty policy on an owned object. Basic booleans include `restrict_creations`, `restrict_deletions`, `block_force_pushes`, `require_linear_history` and `require_signed_commits`; false/omission leaves that rule unselected. Parameterized groups include `restrict_updates`, `require_a_pull_request_before_merging`, `require_status_checks_to_pass`, `require_code_scanning_results`, `require_code_quality_results`, `restrict_code_coverage`, `require_deployments_to_succeed`, `automatically_request_copilot_code_review`. Omitting a group removes it on the owned object; it does not preserve an existing group. Target-specific applicability remains required.

`restrict_creations: true` allows only bypass actors to create matching branches or tags. `restrict_updates: {allow_fetch_and_merge: false}` allows only bypass actors to update them; selecting an empty object supplies a declaration-only false default. Explicit true allows pulling upstream changes. Both rules support branch and tag targets. Export preserves the acquired upstream-sync flag; an absent, null or malformed flag is an acquisition error. Updates adding, changing or removing either rule require independent whole-policy GET verification after mutation acknowledgment. These repository rules use the standard Rulesets plan availability and Administration write permission; no organization-wide Ruleset is managed. See [GitHub's rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) and [REST parameter contract](https://docs.github.com/en/rest/repos/rules).

`automatically_request_copilot_code_review: {review_new_pushes: true, review_draft_pull_requests: false}` selects automatic Copilot reviews for a branch Ruleset. Selecting an empty object defaults both options false only in the declaration; omitting the group removes the rule. Export requires both acquired booleans with their actual values; missing/null/malformed options fail rather than inventing a policy. Updates and removals require independent whole-policy GET verification after the mutation response. Automatic review requires author access to Copilot code review and available AI credits, subject to GitHub's applicable organization policies. This repository rule neither grants licenses nor configures Copilot approval eligibility. See [GitHub's automatic review controls](https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review) and [repository Rulesets API](https://docs.github.com/en/rest/repos/rules).

A bypass entry selects exactly one actor: `role` (`repository_admin`, `maintain`, `write`), positive `team`, `app` or `user`, or `deploy_keys: true`. `bypass_mode` is required and must be `always`, `exempt` or `for_pull_requests_only`; existing branch/actor applicability constraints apply. No actor/grant/credential is created. Organization-only actors and unadopted rules are unsupported.

### Ruleset enforcement observation normalization

A complete, correctly typed enforcement string outside Active/Disabled, except the unsupported `evaluate` value, including unknown, empty or whitespace strings, projects to the named `disabled` observation fallback. Preserve the exact raw token and its positive Ruleset ID outside desired policy. Acquired `evaluate` fails instead of projecting to Disabled; it cannot be exported or overwritten as a normalized policy. Export warns with the UI field, fallback, ID and name, in exported name order. Missing/null/wrong types, incomplete nested policy, ownership/ID failures and incomplete pagination remain acquisition errors without partial export.

A selected normalized ID requires one whole-object update even when desired Disabled equals the projection. Plan/detailed Summary show the contextual warning, a UI enforcement decision marked normalized/write-required and the complete effective supported policy. The effective object is not a raw observation. Unlisted IDs remain unmanaged, undisclosed and unwritten; desired whole-object defaults and name/ID collision guards are unchanged. Same-version private reports carry closed validation/presentation evidence and never supply apply input. A selected Rulesets collection counts once.

Apply uses fresh collection preflight and exact desired whole-object payloads. Normalized updates always require an independent post-write GET, including basic Rulesets without review/analysis rules. Both mutation response and GET must retain canonical desired policy, correct ID and repository ownership; neither verification normalizes unknown values into success. A lying canonical acknowledgment followed by retained unknown GET fails with uncertain effects. Visible fresh diagnostics never supply verification values. Canonical equal objects remain no-ops; earlier completed resource writes remain reported on later failure.

### Ruleset Quality severity observation normalization

For a present, complete code-quality rule, a correctly typed severity outside the four mapped UI choices projects to `errors`, with exact raw provenance bound to the positive Ruleset ID. Unknown, empty, whitespace and UI-spelling API strings follow this policy; missing/null/wrong-type required parameters remain acquisition errors. An absent rule stays absent. Neither observation fallback invents desired intent or makes an incomplete policy usable.

Enforcement and Quality keep independent records on one whole-object change. Export warnings follow Ruleset name order and retain independent field provenance within each object. Selected normalized equality still requires a write; unlisted IDs remain private. Plan and detailed Summary show normalized/write-required UI choices, or `inactive (rule removed; write required)` when desired policy removes the Quality group, followed by one effective policy. Same-version private reports validate each concrete record independently and retain status-only privacy; existing collection count stays one.

Apply sends only the complete desired policy and verifies the original mutation response plus an independent fresh GET. Unknown severity cannot prove an Errors match, and removing the rule requires actual absence: a retained canonical or unknown rule fails with uncertain effects. Warning projection never supplies verification values. Canonical equal policy remains a no-op; acquisition failures preserve export destinations and make no Ruleset write.

### Ruleset Required approvals observation normalization

A complete present PR rule acquires `required_approving_review_count` as an integer. UI choices are 0–10. Other integers, including -1 and API-only 11, project to the named 0 fallback while retaining exact integer provenance per positive Ruleset ID. No string, fraction, overflow, null or missing member supplies a count. An absent PR rule stays absent. Native transport validation remains separate from the UI range.

A selected normalized count requires one whole-object update, including desired 0 matching the projection. Plan and detailed Summary show unquoted integers marked normalized/write-required. Enforcement and Required approvals retain independent concrete records and warn in that order within name-sorted objects. Unlisted IDs stay unmanaged and private. Existing collection counts stay one; ordinary canonical equality stays a no-op.

Private `required_approvals_normalization` metadata requires a string path and integer raw/fallback members, including explicit zero; present-zero, forged and ID-conflicting records fail closed. Report values never supply mutation input. Apply verifies original acknowledgment and independent fresh GET against exact desired whole policy: retained 11/-1 cannot confirm 0, and a retained canonical PR rule cannot confirm removal. Incomplete acquisition makes no Ruleset write and preserves export destinations; later failure retains earlier resource effects.

### Ruleset Allowed merge methods observation normalization

A complete present pull-request rule requires a nonempty, unique list of strings. If any acquired `allowed_merge_methods` member is outside `merge`, `squash` and `rebase`, project the entire list to the named default: Merge, Squash, Rebase. Retain the exact original list and its order, including empty or whitespace strings, as owned ID-bound provenance. Missing/null lists, null or non-string members, empty lists, duplicates, incomplete policies and failed pagination remain acquisition errors. An absent PR rule stays absent.

Export warns and saves only effective choices. Plan and detailed Summary show the original list, the normalized default and the selected desired choices in named UI order; fallback-equal selection still requires one whole-object update. Existing declared-object defaults apply when the methods field is omitted within a selected PR rule. Removing the PR rule removes its owned controls, including additional approval, and requires verified absence. Unlisted IDs and omitted Ruleset collections remain undisclosed and unwritten.

Apply verifies the original mutation acknowledgment and an independent fresh GET against the desired whole policy. A retained unsupported list cannot confirm the full default, and a retained canonical PR rule cannot confirm removal. Fresh warnings are diagnostic evidence only. Incomplete acquisition preserves export destinations and makes no Ruleset write; later failures retain earlier or uncertain effects.

Private `merge_methods_normalization` metadata requires a string path and present typed raw/fallback lists, rejects empty, forged and ID-conflicting evidence even beside valid companion records, and never supplies apply input or proves acquisition completeness. Status-only Summary hides policy/provenance; the existing collection count stays one.

### Ruleset scanning Alerts observation normalization

Each tool in a complete code-scanning rule acquires `alerts_threshold` as a string. Exact `none`, `errors`, `errors_and_warnings` and `all` choices remain unchanged. Other strings, including empty and whitespace values, project to the named `errors` fallback while retaining the exact raw value under the positive Ruleset ID and exact tool name. Tool names are identities: no trimming, case folding, substitution or synthetic tool is introduced. Canonical tools and each tool's Security alerts choice remain intact.

Missing/null/non-string thresholds or names, invalid/duplicate names, malformed companion fields and incomplete acquisition remain errors. Security alerts retain their own independent [typed normalization policy](#ruleset-scanning-security-alerts-observation-normalization). An absent scanning rule stays absent; a present empty tool list stays empty. Each declared tool still requires its own Alerts and Security alerts choices; observed fallbacks do not fill omitted declarations.

Requested export saves effective policy and warns after complete acquisition. Scanning Alerts warnings precede Security alerts, ordered by exact tool name. Each includes Ruleset ID/name and `tool="name"` context. Selected fallback-equal or different choices require one whole-object update. Removing a tool or replacing its name retains the old tool's warning and `inactive (tool removed; write required)` decision; removing the scanning rule shows `inactive (rule removed; write required)`. Plan and detailed Summary show one effective policy. Unlisted Ruleset IDs stay unmanaged and undisclosed.

Original acknowledgment and an independent fresh GET must confirm the exact desired whole policy. A retained unknown Alerts string cannot confirm Errors; a retained canonical tool or scanning rule cannot confirm removal. Contextual fresh warnings never supply successful verification values. Failed acquisition makes no Ruleset write and preserves export destinations; later failure retains honest reporting of earlier or uncertain effects. Private `scanning_alerts_normalizations` records are presentation evidence, not apply input or acquisition proof; status-only output stays private and collection count remains one.

### Ruleset scanning Security alerts observation normalization

Each complete tool's `security_alerts_threshold` string maps exactly to `none`, `critical`, `high_or_higher`, `medium_or_higher` or `all`. Other strings, including empty and whitespace values, project to the named `high_or_higher` fallback. Keep the exact raw string under the positive Ruleset ID and exact tool name. No identity default, trimming or case folding applies. The [scanning Alerts contract](#ruleset-scanning-alerts-observation-normalization) also governs complete acquisition, tool/rule ownership, removal, replacement and honest failure; Security alerts retain a separate record and choice.

Both thresholds may need normalization on the same tool. Retain both independently, keep canonical companions exact and preserve absent rules and explicit empty tool lists. Missing/null/non-string thresholds, invalid or duplicate tool identities and incomplete acquisition remain errors rather than fallback values. Desired Alerts and Security alerts remain explicitly required on every declared tool.

Warnings and UI decisions order Alerts records first, then Security alerts records, each by exact tool name, before bypass modes. Selected fallback-equal intent stays write-required. Original acknowledgment and independent fresh GET must confirm the exact desired thresholds or actual removal: retained unknown strings cannot confirm High or higher, and retained canonical tools or rules cannot confirm absence. Plan and detailed Summary keep one effective policy; private `scanning_security_alerts_normalizations` records never supply apply input or prove acquisition. Status-only hides context and values, unlisted Rulesets stay unmanaged, and collection count remains one.

### Ruleset bypass-mode observation normalization

A complete acquired `bypass_mode` string outside the native `always`, `exempt` and `pull_request` choices projects to the UI default `always`. Keep its exact raw string, including empty/whitespace values, bound to the Ruleset ID and the actor's exact type and optional numeric ID. Canonical companions keep their own identities and choices. Missing/null modes, malformed identities, duplicate actors, incomplete acquisition and unsupported actor/target applicability remain errors; normalization never invents an actor, role, grant or ID.

Requested export warns after complete acquisition and saves only effective UI policy. Warnings follow Ruleset name order, then enforcement, Required approvals, Allowed merge methods, scanning Alerts and Security alerts (each in tool-name order), bypass actors ordered by type and numeric ID. Actor diagnostics include `actor_type` and include `actor_id` only when present. These are identity context, not public YAML aliases. Unlisted Ruleset IDs stay unmanaged and undisclosed in plan/apply.

Selected normalized equality still requires one whole-object update. Plan and detailed Summary show the raw warning and `always` marked as a normalized observation, followed by the desired UI mode marked write-required. Removing an actor, explicitly or through existing whole-object declaration defaults, shows `inactive (actor removed; write required)`. Each operation has one effective-policy line; canonical equal objects remain no-ops. Observation fallback does not populate omitted desired resources or change whole-object ownership.

Apply verifies the original mutation response and an independent fresh GET against the desired observable policy. A canonical acknowledgment followed by the same unknown raw mode cannot confirm `always`; a retained canonical actor cannot confirm removal. Fresh warnings require complete owned readback and never supply verification values. Incomplete acquisition makes no Ruleset write and preserves export destinations; a later failure reports possible and earlier completed effects without implying rollback.

Private same-version reports retain actor-bound `bypass_mode_normalizations`, validate each record independently beside scalar evidence, and omit nil actor IDs rather than accepting JSON null. They preserve status-only privacy and the existing collection count one; records never enter public YAML or apply input and cannot prove acquisition completeness.

### PR review and required check parameters

`rules.require_a_pull_request_before_merging` is a branch-only group:

| Field | Default / accepted value |
| --- | --- |
| `required_approvals` | 0; UI integers 0–10 |
| `require_an_additional_approval_for_unattributed_copilot_pull_requests` | true; explicit false disables the additional approval |
| `allowed_merge_methods` | all three: `merge`, `squash`, `rebase`; nonempty unique selected choices |
| `dismiss_stale_pull_request_approvals_when_new_commits_are_pushed`, `require_review_from_code_owners`, `require_approval_of_the_most_recent_reviewable_push`, `require_conversation_resolution_before_merging` | false |

The additional-approval checkbox maps to the REST PR-rule parameter
`require_extra_approval_for_unattributed_changes`. It adds one approval for
unattributed Copilot pull requests when `required_approvals` is nonzero; it has no
effect when the count is 0. GitHub documents this Preview control as enabled by
default for new and existing Rulesets. [Official Ruleset behavior](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#additional-approval-for-unattributed-copilot-pull-requests).

Selecting the PR group owns this boolean: omission within the group declares
true, while explicit false remains false. Export writes the actual acquired
boolean. Missing, null or incorrectly typed observations fail complete acquisition;
declaration defaults never fill unread values. Omitting the PR group removes the
whole rule, including additional approval. Existing explicit Ruleset prune can
remove its whole containing object after the normal complete-candidate checks.

Every update compares the fresh additional-approval value and PR-rule presence
against the original acquisition before PUT; drift stops the write. An intended
true/false change is compared against desired state only in the mutation response
and independent GET verification. Fresh checks do not lock concurrent edits;
post-write mismatches remain uncertain and no rollback occurs. Creation and
deletion retain complete-policy and identity checks.

Independent supported collections canonicalize by identity.

`rules.require_status_checks_to_pass` contains `required_status_checks` (`[{check: name, source: {any_source: true}}]` or `{source: {app: positive-id}}`), `require_branches_to_be_up_to_date_before_merging` and `do_not_require_status_checks_on_creation`. Defaults are empty checks and false flags. Omitted source means any source, not a retained App. Explicit source must select true Any source or App, not both. YAML null is invalid.

### Required deployments

`rules.require_deployments_to_succeed` selects a branch or tag rule and requires an explicit `required_deployment_environments` string list, for example `{required_deployment_environments: [staging, production]}`. Names retain exact case and whitespace; blank names, control characters and duplicate names are invalid. Lists compare as sets and export in stable order. Explicit `[]` retains an empty selected rule; omission removes the rule. No environment is created, renamed or deployed by this setting. GitHub accepts the rule on tag Rulesets and displays the selected environments in its UI. Successful storage and readback confirm the saved policy, not that deployment protection has been exercised; enforcement depends on GitHub and the Ruleset enforcement status.

[Repository Rulesets REST](https://docs.github.com/en/rest/repos/rules#create-a-repository-ruleset) defines the `required_deployments` transport and requires repository Administration write permission for mutations. [Ruleset availability](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) depends on repository visibility and plan: public repositories support Free; private repositories require Pro, Team or Enterprise Cloud. Deployment environments have their own [availability requirements](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments). This is a repository-level rule; organization and Enterprise rulesets remain outside ownership. Mutations adding, changing or removing it require independent post-write GET confirmation in addition to the mutation response. Unknown or incomplete acquired policy stops export and updates rather than dropping protection.

### Code scanning parameters

`rules.require_code_scanning_results` requires `code_scanning_tools`, each with `tool_name`, `alerts` (`none`, `errors`, `errors_and_warnings`, `all`) and `security_alerts` (`none`, `critical`, `high_or_higher`, `medium_or_higher`, `all`). Tool names are case-sensitive identities; duplicates fail and collection order canonicalizes by name. Explicit `[]` remains empty; no threshold default is inferred.

### Code quality parameters

`rules.require_code_quality_results` requires `severity`: `errors`, `warnings_and_higher`, `notes_and_higher` or `all`. The UI choices map to the public API values `errors`, `warnings`, `notes` and `all`; API tokens are not YAML aliases. This is a managed rule on the selected whole Ruleset: omitting the group removes it, so include every supported protection that must remain. Exact canonical equality is a no-op. Changes and removals require strict mutation-response confirmation plus an independent GET.

The scope correction on 2026-11-09 follows the user's observed availability of **Require code quality results** in a personal repository. The [official Rulesets reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-code-quality-results) documents the rule and Free public Rulesets, and the [public REST API](https://docs.github.com/en/rest/repos/rules#update-a-repository-ruleset) documents its severity field. Product entitlement descriptions are not evidence that this individual control is unavailable. The rule requires configured Code Quality analysis; managing it does not enable the product, start analysis, create workflows or change billing. Coverage uses the separate [coverage threshold rule](#code-coverage-thresholds). See [Code Quality normalization](#ruleset-quality-severity-observation-normalization).

### Code coverage thresholds

`rules.restrict_code_coverage` selects a branch rule with two numeric controls:

```yaml
rules:
  restrict_code_coverage:
    minimum_line_coverage_percentage: 80.5
    maximum_line_coverage_drop: 2.25
```

Both accept finite numbers from 0 through 100, including fractions. The minimum
is a line coverage percentage; the maximum drop is measured in percentage
points relative to the default branch. **Zero disables that threshold.**
A selected group defaults each omitted declaration member to 0 and sends both
numbers. An empty mapping selects a rule with both thresholds disabled; it is
distinct from omitting the group, which removes the rule on an owned Ruleset.
These are declaration defaults, never defaults for unread GitHub state.

The group must be a mapping. Booleans, null, quoted numbers, nonfinite or
out-of-range values, API field aliases and tag targets are rejected. Include
every supported protection that must remain on the selected whole object.
Unlisted Rulesets remain untouched by ordinary apply; explicit Ruleset prune
can delete a complete supported coverage Ruleset under its normal safety checks.

Export requires both acquired numeric thresholds and writes their actual values,
including zero. The official response schema makes them optional without
specifying omission defaults; missing/null members therefore make acquisition
unavailable rather than becoming disabled thresholds. Initial export omits the
unavailable Rulesets family, and re-export retains its valid prior intent. No
partial protection is exported. Exact matching objects cause no write. Before
PUT, a fresh coverage presence/value snapshot must still match the original
acquisition, alongside the Code Quality and additional-approval drift checks.
The mutation response and independent GET must confirm the complete requested
policy. Failed confirmation retains uncertain effects and creation-ID recovery;
only verified IDs are written back to local YAML.

The [official coverage guide](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/restrict-code-coverage)
and [Rules REST API](https://docs.github.com/en/rest/repos/rules) define the
control. It requires Code Quality and uploaded coverage data for evaluation;
it does not wait for uploads. Configure the associated required status checks
when merging must wait for coverage results. Managing these thresholds does not
enable Code Quality, upload data, author workflows, add checks, expand grants or
change billing. Configuration acceptance, execution entitlement and effective
merge enforcement are separate: fixture verification does not prove live
account acceptance or enforcement. See the [coverage evidence](contributing/settings-coverage.md)
and [behavior specification](features/rulesets.feature.md#code-coverage-thresholds).

### Unimplemented branch controls

Public YAML does not accept `require_a_pull_request_before_merging.require_review_from_specific_teams`, including empty collections and explicit false. The personal-public Ruleset editor observed on 2026-11-13 visibly offered the **Require review from specific teams** checkbox, despite official documentation describing the control as unavailable for user-owned repositories. The checkbox was not expanded, so complete usable team selection and the personal-repository API contract remain unconfirmed; this discrepancy does not establish a policy exclusion or implemented support. Organization team ownership and access management remain outside the target. The [official rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) and [coverage documentation](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/restrict-code-coverage) provide the control context. This policy change neither makes those keys valid nor disables live settings.

Acquired nonempty PR `required_reviewers` and unknown or malformed rules still fail complete acquisition without dropping protection. Ordinary code-scanning merge protection, Code Quality thresholds, PR approval counts, public fork approval and automatic Copilot code review remain separate controls. All existing Ruleset updates independently recheck ownership, identity and complete policy before PUT; changes to additional approval, PR-rule presence, Code Quality presence or raw severity, and Coverage presence or either threshold since acquisition stop the update. Preflight is not an atomic lock: concurrent changes between GET and PUT can still race, and post-write failures report that writes may have applied without rollback.

### Unsupported Push Rulesets

`target_pushes` and the `restrict_file_paths`, `restrict_file_path_length`, `restrict_file_extensions` and `restrict_file_size` controls are not accepted, including false or empty declarations. [GitHub limits Push Rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository) to paid-plan private/internal repositories and their forks, outside the [personal-public target](#feature-adoption-scope) because of repository visibility, not price. No compatibility alias or automatic migration is provided.

An acquired repository-owned Push Ruleset or file-restriction rule fails complete acquisition. Export preserves its destination; plan, updates and explicit deletion stop before a Rulesets write. Unsupported policy is never silently omitted or converted into a branch/tag policy. Inherited organization/enterprise policy remains outside ownership.

### Plan, apply, and export

Complete supported observations, including bypass visibility, are required. Unknown writable fields/rules, missing values, invalid types, unreadable details and failed pages stop mutation/export. Inherited organization/enterprise Rulesets remain outside mutation ownership. Repository-owned Push Rulesets are unsupported and stop complete acquisition before any Rulesets mutation or export. Other protection can still affect effective behavior.

Without prune, plan shows only listed creates/ID updates and default-derived reductions. Explicit prune additionally shows its acquired deletion candidates. A nonempty collection counts once as managed. Apply reads fresh state after earlier families, prepares selected operations before the first Rulesets write and writes in stable name order. Fresh comparable matches make no write.

POST/PUT responses must confirm identity, ownership and requested comparable policy. Every create/update, including basic and empty-rule policies, then requires an independent detail GET with `includes_parents=false` to confirm the same ID, repository ownership and complete requested policy. Missing, denied, redirected or mismatching readback returns nonzero and stops later operations, including prune deletions: writes may already have applied. No retry or rollback occurs. A reported positive creation ID remains in recovery diagnostics when verification fails, but is not automatically written to YAML. Only verified creations follow [guarded writeback](#apply-id-writeback).

Unsupported protection is never omitted to produce a usable policy. Acquired Evaluate enforcement, merge queue, metadata restrictions, license compliance, or a present review-dismissal restriction stop Ruleset export, comparison and mutation. Review-dismissal absence in an API response does not prove the stored restriction is disabled. These controls are not accepted configuration. See the [Rulesets specification](features/rulesets.feature.md) and [UI mapping](contributing/ui-ruleset-mapping.md).

## Autolink references

`repository.settings.autolinks` manages the documented **Autolink references**
controls: Reference prefix, Target URL and reference identifier format.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    autolinks:
      - id: 123
        reference_prefix: TICKET-
        target_url: https://tickets.example.com/<num>?view=full
        reference_format: numeric
```

Every entry requires all three controls. `reference_format` is `numeric` or
`alphanumeric`; omission is invalid. Prefix and URL spelling are retained exactly.
Prefixes must be nonblank, contain no control characters and must not overlap
another prefix. No case folding is inferred. URLs must be absolute HTTP(S), have
no userinfo and contain literal `<num>`. These are product input constraints;
they do not describe every URL GitHub might accept. The placeholder is substituted
only for local URL parsing; gh-settings never fetches the destination. Use
non-secret URLs. Nulls, aliases for field names, unknown fields, `delete: true`,
nonpositive/duplicate IDs and missing controls are rejected before API work.

An optional positive `id` binds an existing immutable reference. A matching bound
entry is unchanged and sends no mutation. A missing ID or differing configuration
fails; the API has no documented in-place update. An IDless entry is create-only
and cannot adopt a live overlapping prefix, including an object selected for prune.
Replace a reference explicitly: remove its declaration and prune it, then declare
the replacement without an ID in a later apply. This leaves a temporary
configuration gap; there is no automatic replacement, retry or rollback.

Omission is unmanaged. `autolinks: []` preserves existing references without prune.
`--prune autolinks`, its comma-separated combinations, and standalone `--prune all`
make a present list authoritative for retention. All deletion candidates and the
complete private inventory are frozen before any remote write, including unrelated
settings and unarchive. Candidates never expand. Before each deletion, complete
collection and individual-object reads must match the expected identity,
configuration and optional timestamp presence. Expected state changes only for
this invocation's verified creations/deletions and its verified unarchive.
Unknown additions or changed, missing or replaced retained/candidate objects stop
the operation. Timestamp equality alone is never identity evidence.

Creation sends all three native fields and requires exact HTTP 201, a positive
previously unseen identity, a complete matching acknowledgment, independent GET by
ID and a complete inventory check. Only verified creations receive guarded local
ID recording. All declared creations and required recording finish before deleting
the fixed candidates. Deletion requires an empty HTTP 204 and a successful complete
inventory proving absence and retained-state preservation. A failed/denied GET or
isolated 404 does not prove absence. Races after these checks remain possible.

Only a positive unseen ID from the intended successful creation response is kept
as recovery information; it is not proof of a completed creation. Readback or
recording failure stops later work. Earlier verified changes remain applied and
uncertain effects remain uncertain. CLI `--no-writeback` and GitHub Actions leave
the file unchanged. Record confirmed IDs through the existing
[creation-ID procedure](contributing/repository-automation.md#repository-settings-changes)
before repeating an IDless declaration; repeated IDless creation will collide.

Automatic export independently reads the complete collection and exact
personal-public repository/owner identity. Confirmed empty state exports `[]`.
Unavailable, partial, malformed or unsupported state is omitted initially or
retains valid previous intent with bounded information. No required acquired
value is defaulted. Optional `updated_at` is private evidence, not YAML or report
configuration. Detailed CLI plans and Action Summary can contain prefix, URL and
format; status-only plans hide them. HTTP debug payloads and upstream error bodies
are suppressed. A mixed create/delete plan counts this family once.

The [UI guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/configuring-autolinks-to-reference-external-resources)
documents the controls and Pro availability; plan names do not exclude the
personal-public target. The current account's missing sidebar entry does not
establish global unavailability. The
[Autolinks REST API](https://docs.github.com/en/rest/repos/autolinks)
supports App installation tokens with Administration read for acquisition and
write for creation/deletion. This implementation uses list-all, individual GET,
POST and DELETE; it grants no permission and performs no external-service work.
No fresh authenticated form, live write or rendered-link behavior was verified.
See the [behavior specification](features/autolinks.feature.md).

## Classic branch protection

`repository.settings.branch_protection_rules` manages exact classic rule patterns
and six independent checkboxes plus complete PR-review, required status-check,
deployment and force-push groups. It is separate
from Rulesets and does not clone all branch protection. Current source supports personally owned public repositories;
GitHub documents public availability, including Free, in its
[protected branch guide](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches).

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    branch_protection_rules:
      - id: 123456
        branch_name_pattern: 'release/**'
        require_conversation_resolution_before_merging: true
        require_signed_commits: true
        require_linear_history: true
        allow_deletions: false
        lock_branch: false
        do_not_allow_bypassing_the_above_settings: true
        require_a_pull_request_before_merging:
          enabled: true
          require_approvals: true
          required_number_of_approvals_before_merging: 2
          dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false
          require_review_from_code_owners: false
          require_approval_of_the_most_recent_reviewable_push: true
        require_deployments_to_succeed_before_merging:
          enabled: true
          environments: [staging, production]
```

`id` is an optional positive numeric GraphQL `databaseId`. Without it an entry is
create-only: an existing exact pattern is a collision, even when its controls
match. With ID, a missing or replaced rule fails instead of creating or adopting.
An ID-selected pattern change updates that identity; it never deletes and
recreates the rule. A rename to another rule's exact pattern is rejected. Existing
same-pattern rules remain independently selectable by ID. Duplicate IDs and
conflicting requested patterns fail before this family's writes.

The pattern is required, nonblank and free of control characters. Its exact case,
spaces and slash syntax are preserved. GitHub owns pattern matching and syntax;
`*` does not cross `/`, and `qa/*` differs from `qa/**/*`. A matching branch need
not exist. Only one classic rule applies to a branch, and creation age affects
priority. New rules execute in YAML order; existing rules retain identity and
creation age. Export sorts by numeric ID for stable presentation, without claiming
that this is priority order. See GitHub's
[pattern and priority guide](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule).

Each checkbox is optional. Omission preserves its latest observed value; explicit
`false` requests false. Null, native GraphQL aliases and unlisted controls are
rejected. Without explicit classic prune, omitted collections and `[]` manage no
objects, and unlisted rules are preserved.
[Classic rule pruning](#classic-branch-protection-pruning) adds whole-rule deletion
through a command option; there is no YAML deletion marker or reordering operation.

Plan compares only the requested pattern and controls. Apply first acquires all
selected identities and rejects collisions. Each existing rule is read again by
its acquired opaque node ID before comparison; matching intent sends no mutation.
A changed request sends only changed declared inputs. Enabling linear history
requires fresh squash or rebase availability, after any earlier repository updates;
it does not implicitly enable a merge method. Creation sends the exact pattern
and declared controls; other defaults are controlled by GitHub.

Lock branch makes matching branches read-only. It does not enable fork syncing;
the acquired fork-sync setting stays unowned and must survive updates unchanged.
GitHub documents that locked branches cannot be deleted, even when deletion is
otherwise allowed. This checkbox does not rewrite the separate deletion setting
or promise that it will take effect while locked. Readback verifies stored
settings, not an attempted push, deletion or effective enforcement.

`do_not_allow_bypassing_the_above_settings` matches the UI checkbox: true applies
this rule's protections to administrators; false removes this flag's enforcement.
It maps directly to GraphQL `isAdminEnforced`. Omission preserves the acquired
boolean, and neither direction changes actor allowance lists, other rule controls
or repository Rulesets. Overlapping policies can still affect effective access.
The unsaved target form was inspected on 2026-11-14. The
[maintained provider's direct read/write mapping](https://github.com/integrations/terraform-provider-github/blob/1b9c5c13eb840a72c99d088f3bf3b5e00b950c04/github/resource_github_branch_protection.go)
corroborates this polarity despite the ambiguous GraphQL description. This is
mapping evidence, not a live test of administrator bypass or effective enforcement.

`require_a_pull_request_before_merging` owns one complete active policy. Omission
and `{}` leave it unmanaged; `enabled: false` forbids all child fields. With
`enabled: true`, explicitly set `require_approvals` and all three review checkboxes
shown above. Approval true requires an explicit count from 1 through 6; approval
false forbids that count and requests native count 0. Do not omit active children
or use null/native aliases. No default approval or parent enablement is inferred.
Code Owner, stale-dismissal and latest-push choices remain independent of generic
approval count. Stored configuration does not establish applicable Code Owners,
review eligibility or effective merge enforcement.

Creation and disabled-to-enabled transitions send all five native policy fields,
including false booleans and count 0. Within an already-enabled policy, only changed
fields are sent. Disabling sends only `requiresApprovingReviews: false`, without
promising storage of inactive count or review booleans. Export hides inactive
children; active count 0 exports approval false, while 1 through 6 exports approval
true with its count. Unknown active counts are unavailable rather than clamped.
Omission preserves every raw flag, count nullness and review boolean exactly.

Plan rejects enabled-to-disabled transitions with `restrictsReviewDismissals`,
review-dismissal actors or PR bypass actors. Apply repeats this check on the fresh
rule before mutation. This is a conservative boundary for unowned dependent
protections, not evidence that GitHub was observed deleting them. Matching disabled
state remains a no-op even with latent actors. Changing an active approval count
to zero is allowed while those unowned settings are independently preserved. The
client never adds or removes bypass, review-dismissal or push actors.

The personal-public unsaved form inspected on 2026-11-14 offered approval counts
1 through 6 and left the other review checkboxes visible when Require approvals
was unchecked. The
[maintained provider's independent parent/count mapping](https://github.com/integrations/terraform-provider-github/blob/1b9c5c13eb840a72c99d088f3bf3b5e00b950c04/github/util_v4_branch_protection.go#L155-L204)
and the official GraphQL read/create/update fields corroborate PR-required with
zero generic approvals. No live rule or merge enforcement was tested.

`require_deployments_to_succeed_before_merging` is a complete active group.
Omission and `{}` leave it unmanaged. `enabled: true` requires an explicit
`environments` list, including `[]`; `enabled: false` forbids that list. Names are
an exact-spelling set: duplicates, blank names, controls and null values fail.
No trimming, case folding or Environment creation is performed. Export sorts
active names and emits only `enabled: false` for disabled groups. The unsaved
personal-public form offered an empty selection on 2026-11-14; this is not proof
that the server accepts it or that it enforces useful protection.

A fresh matching rule sends no mutation and makes no Environment request. Before
a needed mutation with an explicitly enabled nonempty group, the writer completely
paginates identity-only Environment observations and resolves each exact name to
its node ID. This also applies when another declared control caused the write.
The [GraphQL Environment schema](https://docs.github.com/en/graphql/reference/deployments#environment)
permits a null numeric database ID; the node ID remains the identity anchor.
After independent rule readback, selected names and node IDs are checked again.
Missing, ambiguous or foreign references fail before mutation; failed or changed
post-write observations stop later work with uncertain effects. New unselected
Environments do not cause drift. Disabling, omission and an empty active set do
not acquire Environment identities. No-op equality does not establish Environment
existence or deployment success.

Referenced Environments must already exist at the Classic write stage, which
precedes Environment creation and updates. Create a missing Environment in a
separate apply. The client does not create, rename or deploy to a reference.
Exact spelling is a conservative selection rule, not a claim about GitHub name
matching. Replacement before fresh acquisition and later remote races are not
ruled out. When Environment prune is selected, an explicitly enabled reference
to a fixed deletion candidate fails global preflight before any writes or
unarchive, even for a matching rule. Retain that Environment explicitly or change
the requirement; references are never silently added to retained names. Conflict
detection conservatively uses Environment deletion's existing name equivalence;
reference selection remains exact. This check does not adopt omitted requirements or certify their viability after an
intentional Environment deletion.

Omitting the deployment group preserves its raw flag, nullness, list order and
inactive names exactly. Selecting it owns the complete active configuration;
disabling sends only `requiresDeployments: false` and makes no promise about
hidden inactive-list storage. Complete known PR, check/App binding, deployment,
fork-sync, force-push and actor-allowance state is acquired as preservation
evidence. Omitted groups stay unowned. Null stays distinct from zero or empty; complete disabled-group raw values are not rewritten.
Missing, malformed, partial, foreign or ambiguous observations fail. All nested
actor connections are paginated. After mutation acknowledgment, an independent
query verifies original requested controls and all omitted known companions.
Collateral changes or failed readback stop later work with uncertain effects;
earlier verified writes remain applied. No retry, rollback or atomic lock is
promised. Stored-value readback does not establish effective branch enforcement.

`allow_force_pushes` owns a complete active group. Omission or `{}` leaves it
unmanaged. `enabled: false` forbids children. `enabled: true` requires
`who_can_force_push: everyone` or `specified_actors`. Everyone forbids `actors`;
specified actors require a nonempty authoritative set:

```yaml
allow_force_pushes:
  enabled: true
  who_can_force_push: specified_actors
  actors:
    - type: user
      id: "<opaque User node ID>"
    - type: app
      id: "<opaque App node ID>"
```

Each actor requires exactly `type: user|app` and an opaque nonblank string ID.
Nulls, unknown members/types, case variants, controls and duplicate IDs fail.
Logins, numeric IDs and installation IDs are not substitutes. Actor ordering is
insignificant; export sorts identities. No App installation or access grant is
performed. User and ChatGPT Codex Connector App selection were observed on the
unsaved personal-public form on 2026-11-14. No rule was saved or force push tested.
The [pinned provider mapping](https://github.com/integrations/terraform-provider-github/blob/1b9c5c13eb840a72c99d088f3bf3b5e00b950c04/github/util_v4_branch_protection.go)
corroborates the native mapping alongside the official GraphQL schema.

Everyone maps to `allowsForcePushes: true`. Specified actors map to false plus the
complete `bypassForcePushActorIds` set. False with no actors means disabled;
false alone does not disable existing allowances. Explicit disabling sends false
and an empty actor list together. Selecting a specified list replaces the complete
membership, including previously acquired Team allowances. Everyone changes only
the boolean and preserves all latent actor identities; omission preserves both
native controls. Unexpected latent changes fail verification, rather than implying
that GitHub always retains inactive storage.

Before any needed rule mutation with specified actors selected, the client checks
every desired node ID and User/App type, even if the actor set matches and another
flag caused the write. Only a wholly matching rule skips these extra lookups.
The scalar mutation acknowledgment does not prove actor membership; independent
complete paginated readback verifies actors and omitted raw state. Other protections
and repository access permissions still determine effective force-push access.

Team is a known acquired identity, but its personal-target selection remains
unconfirmed and it is not accepted in desired YAML. Other Classic updates and
whole-rule prune can preserve or inspect that known state. Detailed plans may show
an active Team in the acquired before-policy, never as a desired input. An active
specified set containing Team makes automatic Classic export unavailable, retaining
compatible prior intent instead of dropping it. Everyone can export normally while
latent Team identities remain private. Unknown actor kinds or incomplete reads fail.

Only independently confirmed creations receive guarded numeric-ID writeback,
using the original unique pattern. The existing file/symlink/concurrent-edit
checks, `--no-writeback` and GitHub Actions suppression apply. If confirmation or
local recording fails, retain the reported numeric or opaque recovery identity
and inspect the remote rule before retrying; never guess or decode node IDs.

Automatic export projects every readable rule's ID, pattern, six checkboxes and
active PR-review, status-check, deployment and force-push groups. It does not export or adopt the other controls.
Unavailable acquisition retains compatible prior declarations with bounded info; it never fabricates an
empty or partial collection. The complete known query follows the inspected
[GraphQL branch-protection schema](https://docs.github.com/en/graphql/reference/branches).
GraphQL cannot reveal future unselected schema fields. Actor restrictions,
fork-sync and additional actor controls remain separate work;
this is not complete classic UI coverage. Queries and mutations use existing
[GraphQL authentication](https://docs.github.com/en/graphql/guides/forming-calls-with-graphql)
without granting permissions. The REST concrete-branch endpoint is not used for
wildcard rules, and its permission table does not prove every GraphQL App grant.
See the [behavior and verification specification](features/classic-protection.feature.md).

### Required status checks

The optional `require_status_checks_to_pass_before_merging` group owns the complete
active policy when selected:

```yaml
require_status_checks_to_pass_before_merging:
  enabled: true
  require_branches_to_be_up_to_date_before_merging: false
  status_checks:
    - name: screening
      accept_updates_from: APP_opaque_node_ID
    - name: legacy
      accept_updates_from: any
```

Omission or `{}` leaves the group unmanaged. A nonempty group requires `enabled`.
With `true`, the strictness boolean and complete `status_checks` list are required;
`[]` is valid, but does not establish useful check enforcement. With `false`, child
controls are forbidden: this requests deactivation only. Null, wrong types, unknown
members and native GraphQL aliases are rejected.

Check names retain exact case and bytes and must be nonblank and free of control
characters. Names must be unique, even when sources differ; this is a conservative
product boundary. Distinct names may share one App. `accept_updates_from` is either
the exact string `any` or an opaque GraphQL App node ID. Numeric App/installation
IDs, slugs and inferred providers are not substitutes. The writer always sends
`appId`; GitHub's [input contract][classic-check-input] otherwise infers the most
recent source. gh-settings does not discover recently run checks, grant access or
require App installation.

For an active observed policy, both native check lists must be present and
nonnull, names must be unique, and the legacy context list must contain exactly
the same names as the App-bound list. This coherence check is a supported-observation
restriction, not a guarantee about every GitHub rule. A malformed, missing or
partial observation makes the whole classic family unavailable. An explicit null
App means `any`; a missing App member is unknown. Inactive export contains only
`enabled: false`. Its typed latent lists, including nullness, order and source
bindings, remain private preservation evidence when the group is omitted.

Creation or enabling sends all three writable members, including explicit false
strictness and an empty list. Already active updates send changed members only;
check order does not cause a write. Disabling sends only the false gate and permits GitHub to reset latent child values. The legacy context alias
is never sent. Omitted groups must retain their exact raw state across other
updates; selecting an active group owns its complete semantic policy.

A matching fresh rule sends no mutation or App lookup. Before a needed write
with a selected active group, each distinct non-`any` source must resolve through
the same credential to the exact [App node][classic-check-app]. This applies even
when only another declared checkbox changes. After the complete mutation
acknowledgment, independent rule/actor readback verifies the entire original
request and omitted state, then the same App identities are checked again.
Prewrite failures stop before mutation; failed acknowledgment or subsequent
verification leaves uncertain effects and creation IDs are recovery information
only. Readback proves stored settings, not check execution or effective protection.

[classic-check-input]: https://docs.github.com/en/graphql/reference/branches#requiredstatuscheckinput
[classic-check-app]: https://docs.github.com/en/graphql/reference/apps#app

## Classic branch-protection pruning

`--prune branch-protection-rules` and standalone `all` add unlisted whole-rule
removal when `repository.settings.branch_protection_rules` is present. An omitted
collection stays unmanaged. `[]` selects all initially acquired rules; a nonempty
list retains declared numeric IDs and newly created rules. All retained rules must
be listed. ID-less exact-pattern collisions, missing/foreign IDs and conflicting
renames fail before any remote write, including unrelated settings or unarchive.
Ordinary create/update retains partial-control ownership and complete active
PR-review, status-check, deployment and force-push ownership.

Deletion removes the entire rule, including unowned protections. An overlapping
rule may then take effect; recreation cannot restore creation-age priority. The plan shows
this consequence and each complete supported before-policy, without raw private
companions. It does not delete branches. Inspect the complete plan before apply.

Before any remote write, acquire the complete rule/actor inventory and freeze each
unlisted candidate's repository ID, numeric ID, opaque node ID, exact pattern,
six flags, active PR-review/status-check/deployment/force-push projections and all known raw companions,
including inactive PR-review, status-check, deployment and force-push values. Missing, denied, partial,
unsupported or inconsistent evidence blocks the whole preflight. A retained rule's unreadable
state can also block this complete inventory. Before this family's work, recheck
the initial candidates: disappearance, replacement or state drift stops execution.
Newly observed unlisted rules remain untouched, including when the initial
candidate set was empty. They never expand the frozen selection. Only this
invocation's confirmed unarchive adjusts the acquired archive state; unrelated
repository merge-method changes do not constitute deletion-policy drift.

All declared classic creates/updates and required creation-ID writeback precede
fixed deletions. Immediately before each deletion, independently read its selected
node and complete actor state and require an exact snapshot match. The official
[GraphQL deletion contract](https://docs.github.com/en/graphql/reference/branches#deletebranchprotectionrule)
accepts the acquired opaque ID. Its nullable payload has no deleted-ID echo:
require an error-free `DeleteBranchProtectionRulePayload` acknowledgment, then an
independent complete repository inventory proving both old node and numeric ID
absent. A distinct retained same-pattern rule is allowed. A null node alone is not
absence confirmation.

Mutation errors, partial GraphQL errors, malformed acknowledgments and incomplete
or mismatching readback stop later work with uncertain effects and the fixed
recovery IDs. Earlier confirmed writes remain applied. No retry, rollback,
recreation or deletion-ID writeback occurs. The final read/delete race remains,
and future unselected GraphQL fields cannot be detected. Private companion
snapshots stay out of reports and inherited `GH_DEBUG` logs. No live deletion or
credential-grant sufficiency was established by the local fixture tests.

## Environments

`repository.settings.environments` selects supported policies and variables by `name`, without numeric Environment/pattern ID writeback. Names are nonempty, have no leading/trailing whitespace/control characters and use Unicode simple folding for client identity. Spelling remains; slash/percent are URL-encoded as one segment. Without prune, omission/`[]` selects nothing and unlisted Environments are preserved. Explicit [Environment pruning](#environment-pruning) adds bounded parent deletion; omission remains unmanaged. Deletion/rename markers and secret values remain unsupported. Already installed custom protection Apps use the separate checkbox ownership below; installation and grants remain excluded. Selected non-sensitive variables have the separate ownership described below.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: production
        deployment_protection_rules:
          wait_timer: 0
          required_reviewers: []
          prevent_self_review: false
          allow_administrators_to_bypass_configured_protection_rules: false
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules:
          - {ref_type: branch, name_pattern: 'release/*'}
          - {ref_type: tag, name_pattern: 'v*'}
```

For an existing policy-selected Environment, only declared built-in controls are owned. Matching partial and name-only entries are no-ops. When a built-in change needs PUT, apply overlays the original partial request onto the latest complete supported timer, reviewers, self-review, administrator-bypass and deployment-mode observation. It sends and independently verifies those preservation companions without adding them to desired intent. Plan and Summary show the partial request, not an invented final state. Creation still sends only declared controls; omitted creation values remain API-controlled, and a missing name-only entry sends an empty create-or-update body.

`deployment_protection_rules` contains optional `wait_timer` (0–43200 minutes), up to six unique `required_reviewers` (each `{person: positive-id}` or `{team: positive-id}`), `prevent_self_review` and `allow_administrators_to_bypass_configured_protection_rules`. True prevent-self-review cannot accompany an explicitly empty reviewer set. An existing partial update that conflicts with freshly preserved reviewers or self-review fails before writing; apply never clears an unowned dependency implicitly. No null is accepted. Reviewer eligibility, grants and entitlement remain GitHub prerequisites; principals are not invited.

`deployment_branches_and_tags` is `no_restriction`, `protected_branches_only` or `selected_branches_and_tags`. There is no public raw boolean-pair/null alias. Protected mode may permit all branches when no branch protections exist; configured intent does not prove protection.

`deployment_branch_or_tag_rules` is an authoritative list of `{ref_type: branch|tag, name_pattern: pattern}`. Type+pattern is case-sensitive identity; branch and tag may share a pattern. Omission makes no pattern CRUD; `[]` removes all observed patterns. Omitted mode plus a list requires existing selected mode; creating with patterns needs explicit selected mode. Explicit incompatible modes fail. Patterns are a separate API collection, not PUT members. When selected mode stays selected and the list is omitted, complete raw pattern IDs, types and names are retained and verified without pattern CRUD or unowned warnings, including unsupported ref types and projection collisions. Changing from a raw custom mode to No restriction or Protected branches only is rejected while any raw pattern exists and the list is omitted. To remove those patterns deliberately, first apply `deployment_branch_or_tag_rules: []` while Selected, then change mode. A complete known-empty collection permits the transition. GitHub can discard the patterns when custom mode is disabled; the client does not probe inactive state by temporarily changing mode or treat an inaccessible pattern endpoint as empty.

Reads require complete inventory, built-in/custom-protection details and typed paginated patterns when applicable; missing type is not assumed branch. Unknown or malformed protection fails instead of being erased. Apply preflights and reconciles in stable name order. A pattern-only change skips policy PUT, verifies existing custom mode, creates missing patterns and deletes obsolete patterns by verified IDs. Retained pairs keep IDs. Complete fresh readback verifies original requested values/list and the preserved built-in snapshot; after a policy PUT it also verifies every sent companion. Duplicate-create 303, unavailable reads and mismatches fail; earlier/uncertain effects remain without retry, rollback or an intermediate-enforcement guarantee.

Administrator bypass is a real writable setting. Export requires/retains its complete boolean, even false. A prior temporary public-Environment experiment confirmed explicit false/true and retained preceding values on omission with other members fixed; this is not a platform-wide omission or entitlement guarantee. No default true is inserted.

Ordinary `export` observes supported Environment policy, active selected patterns and enabled custom App IDs, without Environment/pattern/binding IDs. Variables are inventoried independently after complete name acquisition; policy failure does not prevent listing accessible variables. Existing Actions read, Administration write, custom-protection visibility and account eligibility still apply. See [Environment behavior](features/environments.feature.md).

### Custom protection Apps

`deployment_protection_rules.custom_rules` is an optional set of up to six unique
positive GitHub App IDs. It selects the already installed Apps checked under
**Deployment protection rules** in the Environment UI:

```yaml
repository:
  settings:
    environments:
      - name: production
        deployment_protection_rules:
          custom_rules: [12345, 67890]
```

Omission preserves every freshly acquired binding. A present list owns the full
checkbox set; `[]` explicitly disables every supported binding in that Environment
without a prune flag. Matching and reordered sets make no binding write. App IDs
identify integrations; rule IDs identify individual enabled bindings and never
enter YAML or ID writeback. This does not install or uninstall Apps, grant access,
approve deployments or configure webhooks. Enabling a rule affects later deployment
gates according to that App. [GitHub documents the UI and prerequisites](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/configure-custom-protection-rules).

Export reads enabled bindings independently of App availability and emits `[]`
only after a complete known-empty observation. Missing/null fields, unsupported
shapes, ambiguous identities or failed reads retain the whole previous Environment
policy family, or omit it on first export. A retained enabled App does not need to
appear in the available list. Export never enables an App or follows integration URLs.

For an existing parent, apply validates all additions against complete available-App
pagination before the first Environment write. Parent creation can precede that
availability read; later failure reports the confirmed parent progress. App-only
changes send no parent PUT. Built-in and pattern changes preserve exact omitted
binding identities, and each App write preserves complete built-ins and patterns.
A changed repository, parent, App or binding identity stops later operations.

Apply adds before removing. A temporary custom-App union above six fails before
any Environment write: first apply an explicit reduced set, then apply the desired
replacement set, accounting for the protection change between those runs. The
published six-rule limit does not establish how built-ins count; no combined
capacity formula is invented. A server capacity refusal never triggers deletion
as recovery. This sequence is nontransactional and has no intermediate-enforcement
or atomic drift guarantee.

Enable sends only `integration_id`, requires 201 with matching identities, then
independently verifies the exact binding and complete enabled inventory. Disable
uses the fixed rule ID, requires an empty 204 and complete fresh absence with all
retained bindings unchanged. A recreated binding for the same App is drift, never
a fallback deletion target. Failed verification returns nonzero, retains any
positive acknowledged rule ID in recovery diagnostics and stops later operations;
no binding is written to YAML and no retry or rollback occurs. Read actual state
before retrying uncertain effects. See the [REST contracts](https://docs.github.com/en/rest/deployments/protection-rules).

Enabled reads need Actions read where authentication is required. Additions also
need Administration read for availability; changes need Administration write.
Fixture checks do not establish actual token grants or live nonempty App behavior.
Parent pruning still requires an explicitly known-empty custom protection list;
ordinary App support does not expand parent-deletion ownership.

### Environment variables

`repository.settings.environments[].variables` selects explicit non-sensitive values in **Settings → Environments → the selected Environment → Environment variables**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: production
        variables:
          - {name: DISPLAY_NAME, value: 'Production'}
```

The [variable name/value rules](#actions-repository-variables) apply: canonical uppercase names, unique ignoring ASCII case within each Environment, required nonempty string values and no YAML delete/rename member. The same name in different Environments or repository variable families remains independent. Without prune, omitted `variables` and `variables: []` make no variable requests and unlisted variables are preserved. Explicit `--prune environment-variables` follows the [variable pruning contract](#variable-pruning); omission stays unmanaged, while [] selects all variables in that listed Environment.

An entry containing `name` and `variables` alone selects **only variables**, including when the variable list is empty. It never sends an Environment policy PUT, changes protection or secrets, or creates a missing Environment implicitly. A nonempty variable selection requires that Environment to exist. To manage policy and variables together, include the desired policy controls in the same entry; policy upserts run first, then selected variable upserts. Existing name-only entries select no policy mutation; a missing name-only entry still selects creation. Combined declarations retain the existing-policy preservation and creation-omission rules above.

Without prune, plan reads selected names only. Prune additionally acquires complete selected inventories. When the same plan explicitly creates a missing Environment policy, its selected variables are shown as creates without requesting nonexistent variable endpoints. Apply checks the selected Environment identity, reads each variable fresh, creates an absent variable, updates a changed value, skips equality and verifies independent readback. Writes run in stable Environment/name order. Failed reads do not become empty state; later failures stop execution without rolling back earlier writes, and a failed write or verification can have uncertain effects.

Ordinary `export` inventories variables for every completely listed Environment. Each parent variable list is atomic; failed acquisition retains that parent’s prior declarations with info and other parents continue. Unknown parent inventory retains all prior scopes. Variable-only entries do not invent deployment policy. Values are **plaintext** in YAML, exports and detailed output; secret APIs are not used.

The [Environment Variables REST API](https://docs.github.com/en/rest/actions/variables#get-an-environment-variable) uses name-addressed `/repos/{owner}/{repo}/environments/{environment_name}/variables` endpoints. Fine-grained/App tokens need **Environments** read for variables and write for mutations, plus **Actions** read for the Environment identity check. No access is granted automatically. UI creation/update/deletion and API persistence were verified on a Free, personally owned public test repository on 2026-11-11; both rejected an empty value. This is bounded adoption evidence, not a runtime entitlement check or proof of workflow consumption. See the [behavior specification](features/environments.feature.md#selected-environment-variables).

### Environment deployment mode observation normalization

Complete typed equal policy booleans `(false,false)` or `(true,true)` are outside the three UI choices. Project the named `no_restriction` fallback while retaining both acquired booleans separately from effective policy. A valid API null remains exact No restriction without provenance. Missing policy, missing/null/wrong-type boolean members, unknown controls, failed reads and incomplete pages remain acquisition errors.

Raw `custom_branch_policies:true` still requires complete paginated patterns and custom-protection checks before publishing an observation, including a projected equal pair. Keep those latent pattern IDs/values in separate acquired state; effective No restriction never exports or replays them as selected patterns. Acquired empty patterns differ from an incomplete read. Pattern ref-type observations follow the policy below; unrepresentable principal identities are not invented.

Export warns in folded environment-name order and saves only effective policy. Plan/detailed Summary label every Environment before-object as **effective before policy**. An explicitly selected normalized mode adds a contextual raw-pair warning and normalized/write-required UI decision, including fallback-equal No restriction. Only selected names are compared; unlisted names stay undisclosed and unwritten. Private reports retain closed typed pair metadata, actual partial requests, status-only privacy and the existing collection count one; they never supply apply input.

Apply verifies an explicit mode against the original nullable pair in the acknowledgment and an independent complete fresh read. Retained equal booleans cannot confirm canonical No restriction. Pattern intent under an unknown mode needs explicit Selected; neither fallback nor the raw custom flag proves active applicability. A canonical mode acknowledgment alone does not permit pattern CRUD if the intermediate fresh read retains an unknown mode. Writes remain ordered create-before-delete with verified pattern identities and honest partial effects.

An omitted mode remains unowned and emits no mode warning. A needed existing-policy PUT preserves its freshly observed canonical nullable pair as a companion. Equal raw booleans cannot be safely replayed through the supported PUT contract: when another built-in needs changing, apply fails before writing and requires an explicit supported mode. Matching partial or name-only intent may still skip without changing that raw pair. No effective fallback is used as preservation evidence. Malformed acquisition preserves export destinations and makes no Environment write; later failures retain actual prior resource and uncertain effects.

## Unsupported settings

Only the listed UI controls are accepted by current source. Remaining classic branch-protection controls are adopted but unimplemented; other Actions/security gaps are tracked separately from API limitations in the [coverage ledger](contributing/settings-coverage.md). Organization settings and team access, organization-only custom properties, inherited organization Rulesets and private-only controls are outside the personal-public target. Repository rename/visibility/transfer/deletion, feature configuration-file authoring and secret values are explicit exclusions. Pages visibility and Environment rename remain unsupported without asserting that a usable target control/API is absent. See the [architecture](contributing/architecture.md) and [access implementation boundary](features/repository-access.feature.md); future work is not valid configuration today.

## Pages configuration

`repository.settings.pages` manages site creation, existing-site configuration and explicit deletion. Site visibility is not managed.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    pages:
      build_and_deployment:
        source: deploy_from_a_branch
        branch: main
        select_folder: docs
      custom_domain: null
      enforce_https: true
```

`build_and_deployment.source` is `deploy_from_a_branch` or `github_actions`. Branch mode uses a complete literal `branch` plus `select_folder: root|docs`; partial tuples fail except the explicit deletion selector below. Switching from workflow to branch requires that tuple; workflow mode rejects branch/folder intent. No branch/content/workflow is created. Omitted Pages/empty mapping manages nothing; other omitted members remain unmanaged.

`custom_domain` is a nullable scalar, never a `{value: ...}` mapping. Null (including a bare empty scalar) removes it. Omission leaves it unmanaged; empty string and quoted `null` are invalid. Strings must be ASCII DNS hostnames with at least two labels and normal length limits, without trailing dot, URL, wildcard, port or whitespace. Comparison/export lowercase without IDNA or DNS verification. `enforce_https` is boolean; null is invalid.

A complete declaration with explicit `source: deploy_from_a_branch`, a real branch and folder creates a missing site and reconciles a present one. `source: github_actions` creates workflow publishing when absent, without creating or running a workflow. On an absent site, domain/HTTPS-only intent, branch/folder without explicit Source, and branch Source without a complete tuple fail before any write. On a present branch site, source alone may retain the existing branch/folder; a workflow-to-branch transition still requires both.

Explicit deletion uses the UI's Branch None choice:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    pages:
      build_and_deployment:
        source: deploy_from_a_branch
        branch: none
```

Only exact lowercase `none` is reserved; other branch names retain their case (`None` is a literal branch). The accepted collision with a real branch named `none` is intentional. Deletion requires the explicit Source above and rejects any `select_folder`, `custom_domain` (including null), or `enforce_https` (including false). It directly deletes either publishing mode; it never sends the sentinel to POST/PUT. Omission, null, empty groups and `custom_domain: null` never delete a site. A present site using the reserved literal branch cannot be exported as a publishable declaration; export fails rather than writing contradictory YAML.

Under the required-permissions precondition, target Pages GET 404 means absent. Deleting an absent site is a no-op. ordinary `export` emits the deletion declaration above for absence; applying that export later intentionally deletes any site then present. Present-site export includes mode, active branch/folder, nullable domain and HTTPS after complete acquisition. Visibility/readiness metadata are excluded. Other failures and missing/null/wrongly typed required fields emit info and retain prior Pages declarations; independent families continue.

Creation sends only source fields with POST, requires 201, then a fresh complete matching GET 200 and repository identity verification. Declared domain/HTTPS controls are reconciled afterward with PUT as needed, preserving observed server defaults for omissions. A later failure retains and reports confirmed creation, with no rollback. Updating requires PUT 204 and complete readback; deletion requires DELETE 204 and GET 404. A concurrent DELETE 404 requires a new GET 404 before success. POST/PUT followed by 404, unexpected statuses, redirects, explicit denials, timeouts and readback mismatch fail; writes are never retried automatically. Creation observes server-selected visibility without requesting a visibility change.

A complete legacy-mode response with a folder string outside `/` and `/docs`, including an empty string, projects to `select_folder: root`. Its exact raw folder, UI path and named default remain observation provenance; the branch is preserved and never invented. Export warns and saves only the effective UI choice. A declared branch/folder pair retains a warning and a write-required folder decision even when both declared values equal the effective projection. The pair still counts as two managed native fields. Omitted source remains unmanaged and does not disclose its raw folder. Inactive workflow source is not acquired.

A correctly typed publishing mode outside `legacy` and `workflow` projects to `source: deploy_from_a_branch` only with a complete typed branch/folder tuple and valid literal branch. Missing/null/wrongly typed source members remain errors; no branch is inferred. Empty and whitespace mode strings retain exact provenance. Mode and folder records remain independent, and export warns in that order. A declared publishing mode counts one native field and retains a write-required decision even when it equals the projection. Explicit source intent with an unknown raw mode requires `source: deploy_from_a_branch`; the fallback alone cannot prove applicability. Omitted publishing mode remains unmanaged and is not disclosed.

Plan is read-only; an inactive source is not invented as an observed tuple. Apply reads fresh state, skips exact matches, uses the lifecycle operations above or sends declared-only PUT values and verifies requested values, omitted applicable companions, public visibility and repository identity. Verification checks exact raw mode and branch/folder strings: an unchanged unknown token cannot confirm a requested canonical value, and omitted active mode/source must retain their original raw values. Domain canonicalization remains unchanged. Apply warnings use only fresh acquired provenance, independently scoped to declared publishing mode and source intent. Explicit unarchive precedes it; final archive follows. Failure can leave earlier writes or uncertain effects, without retry/fallback/rollback.

Convergence does not prove DNS, certificate or deployment readiness. Pending metadata alone is not drift. GitHub can publish or create a CNAME-file commit as a side effect; the tool does not trigger builds/deployments or edit DNS/content. Pages read/write, Administration write and actor authorization are prerequisites; public GET does not prove write grants. See [Pages behavior](features/pages.feature.md).

[^ui-inventory]: Read-only UI wording inventory inspected 2026-10-28; internal conversion details and the other family inventories are linked there. This reference changes the public YAML composition, not the recorded UI inspection history.
[^github-update-repository]: Repository update request fields; original inspection 2026-10-11. This does not expand the supported UI inventory.
[^github-repository-openapi]: Fixed GitHub OpenAPI evidence for nullable observed description/homepage; original inspection 2026-10-12.
[^github-labels]: Labels listing/mutation and existing permission boundary.

### Environment ref-type observation normalization

A complete unsupported string `type`, including empty or whitespace, projects to the named `branch` fallback. Retain the exact original type, positive API ID and case-sensitive pattern name separately from effective policy. Missing/null/wrong-type fields, invalid names/IDs, duplicate raw identities, unstable counts and incomplete pages remain acquisition failures; they do not produce defaults or partial warnings.

In canonical Selected mode, export warns with the Environment name, pattern ID, exact name and original/fallback type, then saves effective `ref_type`/`name_pattern` values without API IDs or provenance. Warnings follow folded Environment-name order, mode warning first, then ascending pattern ID. Canonical companions stay silent. Distinct raw identities that project to the same branch/name are retained separately: export warns and fails without replacing its destination, rather than collapsing them. Raw patterns under effective No restriction remain latent and produce neither ref-type warnings nor exported rules.

An explicit pattern list owns the observed collection, including `[]`; an omitted list owns none of its values or diagnostics. Plan and detailed Summary retain one actionable collection update at fallback equality, exact raw evidence and the requested fields of the effective before policy. A genuine collision is labeled unavailable effective pattern data, not an empty list. Only selected Environment names are observed. Same-version private reports retain closed per-ID evidence and canonical companions; status-only output hides these details. Reports never supply apply input, and the existing managed collection count remains one.

Apply sends declared fields in the Environment PUT and canonical types in separate pattern requests. It confirms fresh canonical Selected mode, creates required pairs before deleting obsolete raw identities, and retains the IDs of existing canonical companions. Before deletion, an individual GET must match the exact original type, case-sensitive name and ID. Creation acknowledgments remain strict; unsupported or malformed acknowledgments stop before deletion. Independent complete readback must confirm the requested canonical collection: a retained unsupported type cannot prove success merely because its fallback matches. Fresh owned warnings expose the acquired record, not a verification substitute.

Acquisition failures make no Environment write. Later failures retain earlier or uncertain effects, report failure and do not trigger automatic retry or rollback. Caller YAML is unchanged. A complete canonical match is a silent no-op on repeat; existing omitted built-ins are preserved. These source behaviors do not apply live settings or publish a release.

## Persistent pull request limits

Settings → Interaction limits → Pull request limits maps to:

```yaml
repository:
  settings:
    interaction_limits:
      pull_request_limits:
        limit_open_pull_requests_from_users_without_write_access: true
        maximum_open_pull_requests_per_user: 2
        count_draft_pull_requests_toward_the_limit: true
```

The enable and draft controls are optional booleans. The maximum is an optional
integer from 1 through 1000. Omission leaves that control unmanaged, including
latent maximum/draft values while disabled. Empty groups manage nothing. Null,
quoted booleans/integers, fractions and API-shaped aliases are rejected.
Moderation is a navigation category, not a configuration group.

Plan selects the cap endpoint only when a cap control is declared. Apply
acquires fresh complete cap state, skips
exact matches and PATCHes required `enabled` plus only declared optional members.
When enable is omitted, the required member carries the freshly observed value.
Independent GET confirms the complete requested/preserved three-field state.
Only a complete response equal to the pre-write baseline can be retried: at most
three confirmation GETs, with one second between those stale observations. This
is a client budget, not a GitHub propagation guarantee. Writes are never retried.
Denied, malformed or other mismatching readback fails immediately. Failure after
PATCH reports uncertain effects; earlier successful operations remain applied.
Apply is not atomic and does not roll back.

Ordinary `export` independently includes all three stored values,
including disabled/false and latent values. Missing `include_drafts` is an acquisition error, even though
it is optional in the published response schema; it never means false. Failed
acquisition emits info and retains prior cap intent. CLI and detailed Action reports
use these UI paths and native booleans/integers; status-only output hides values.

The [official creation-cap API](https://docs.github.com/rest/interactions/repos#update-pull-request-creation-cap-for-a-repository)
requires administrator access. Credentials must already authorize the endpoint;
no grants are expanded. Adoption evidence covers GitHub Free personal public
repositories with these visible controls, not other plans/owners/visibility.
The cap limits users without write access and does not grant write access.
Temporary Interaction restrictions and automatic exemption rules remain unmanaged.
The separately selected Exemption list is described below; ordinary cap updates
make no exemption request. Companion preservation in a cap update means only its
three known fields. Configuration
readback does not establish enforcement against external actors. See the
[behavior specification](features/pull-request-limits.feature.md).

### Pull request limit Exemption list

`repository.settings.interaction_limits.pull_request_limits.exemption_list`
manages the complete selected list of trusted User logins, independently of the
three cap controls:

```yaml
repository:
  settings:
    interaction_limits:
      pull_request_limits:
        exemption_list: [trusted-user, another-user]
```

Omission leaves the list unmanaged and makes no list request. An explicit list
owns all membership, so omitted users are removed; `[]` explicitly clears the
list. Membership is case-insensitive and exported in sorted lowercase order.
At most 100 distinct User logins are accepted. Logins are 1–39 ASCII letters,
digits or single interior hyphens; surrounding whitespace, consecutive hyphens,
nulls, duplicate names and API-shaped user objects are rejected. Bot and
Organization identities are unsupported and stop acquisition rather than being
dropped from the list. Numeric user IDs are acquired identity evidence, not YAML
or writeback fields.

Plan reads only the complete list. Apply reads fresh inventory and skips an
exact match. Before any mutation, newly added logins are resolved through the
public Users API and must identify supported User accounts. It removes obsolete users first and independently verifies the
remaining membership and retained User IDs before adding users, avoiding a
100-user list's temporary overflow during replacement. It then independently
verifies exact desired membership and both retained and newly resolved User IDs. Each mutation uses only its
computed user difference. A failure stops further work; confirmed removals can
remain applied when addition fails. Writes are not retried and no rollback or
concurrency protection is promised. This selected-list operation runs before
separately declared cap controls; it never changes the cap, automatic exemption
rules, temporary restrictions, collaborator grants or repository permissions.

Ordinary `export` independently includes the complete list, including `[]`.
A failed, incomplete, ambiguous or unsupported inventory emits info and retains
prior membership without stopping independent reads. CLI
and detailed Action reports/Summary expose the selected membership; status-only
output omits it. The collection counts as one managed value.

The [official bypass-list API](https://docs.github.com/en/rest/interactions/repos#get-pull-request-creation-cap-bypass-list-for-a-repository)
requires **Administration write**, including for reads. Its GET returns User
objects; PUT adds and DELETE removes up to 100 logins per request. An unexpected
pagination Link is rejected rather than treating a partial list as complete.
Existing credentials must authorize these operations. Exemption allows listed
contributors to exceed the cap without granting write access; callers must
review the complete desired list before applying it.

On 2026-11-11, the Free, personally owned public test repository exposed the
Exemption list and Add users control after its cap was temporarily enabled.
The existing empty list was acquired through the API. The owner was not offered
as a UI search candidate, so no user was added and no live add/remove persistence
or external-actor enforcement is claimed. The cap was restored to its disabled
baseline and the list remained empty. Automated mutation evidence uses local
fixtures. See the [behavior specification](features/pull-request-limits.feature.md).

### Automatic Security export and retained intent

The [automatic export contract](features/readable-security-export.feature.md)
replaces per-family `--security` selection. Failed Security observations remain
unavailable and never become false or an empty collection. Valid existing
declarations for those families are retained; this is not proof of remote state.
Plan/apply keep their strict declared-field rules and may still fail when those
retained values cannot be read. Existing invalid files and concurrent local edits
are not overwritten. Re-export can reformat comments/YAML while preserving these
declared values; it is not a text-preserving editor or a concurrency lock.
