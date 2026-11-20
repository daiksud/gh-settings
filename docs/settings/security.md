---
type: Reference
title: "Advanced Security settings"
description: "Advanced Security settings declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Private vulnerability reporting

Use boolean `repository.settings.advanced_security.private_vulnerability_reporting`. Omission is unmanaged; false explicitly disables. It is independent from Secret protection. Ordinary `export` automatically attempts this family, with info and omission/prior-declaration retention when unavailable.

This control applies to public repositories. Fresh visibility and complete status are required; private/internal applicability failures, malformed data, permission errors and redirects do not become disabled state. Apply reads fresh prerequisites, skips matching state, performs the narrow enable/disable operation and verifies readback. Earlier writes remain on later failure. No rollback, permission expansion or automatic retry occurs. See [private reporting](../features/security.feature.md).

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

A fresh matching value skips its write, including enabled=true/paused=true. An enable requires enabled readback. A disable returning DELETE 204 followed by matching complete family readback, including target GET 404, completes under required permissions. Pause remains unknown on 404; the operation reports enabled intent only. There is no permission-risk warning or accepted-unverified notice. GET alone does not establish a write; non-204 writes, explicit 401/403, timeouts, malformed/incomplete reads or target/companion mismatch fail. Later failures retain earlier completed progress without retry or rollback. No Dependabot configuration files, individual alerts, grouping, auto-triage, credentials or organization settings are managed. See the [decision and fixture contract](../features/security.feature.md#dependabot-state-reading-decision).

## CodeQL default setup

Use the optional boolean `repository.settings.advanced_security.code_security.codeql_analysis.default_setup` for a personally owned public repository. Omission leaves default setup unmanaged; false explicitly disables it. Null and other types fail. Ordinary `export` automatically attempts this enablement, independently of other Security reads. Unavailable acquisition emits info and retains prior intent when present.

The public [default setup API](https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2026-03-10#update-a-code-scanning-default-setup-configuration) requires Administration read/write. Reads require public visibility, a User owner and a complete `configured` or `not-configured` state; missing/null/unknown states, 403 and 404 fail without fallback. Plan is read-only. Apply rereads before writing, skips a fresh matching state, and sends only `state`. Language selection, runner and query suite are separate [configuration controls](#codeql-default-configuration); they are not part of this enablement control. No product, Actions prerequisite or advanced workflow configuration is automatically enabled or edited. Ineligible repositories fail through the public API.

Changing this setting can start CodeQL validation/analysis and consume Actions resources. Disabling stops default-setup scanning and may affect features that depend on its results, including Autofix and required scanning checks. Existing analyses are not deleted. HTTP 202 accepts asynchronous work; fresh readback confirms configuration state only, never completion or success of a scan. A mismatching readback may indicate pending validation and fails with uncertain write effects. No automatic retry or rollback occurs; inspect state before retrying. Local fixtures do not establish live entitlement, analysis results or consumption.

## CodeQL default configuration

The optional controls below belong under `repository.settings.advanced_security.code_security.codeql_analysis`, alongside the independent `default_setup` boolean:

```yaml
languages: [github_actions, go, python]
query_suites: extended
threat_model: remote_sources
runner:
  type: labeled_runner
  label: codeql-pool
```

| Control | Accepted values and ownership |
| --- | --- |
| `languages` | A nonempty unique set: `github_actions`, `c_cpp`, `csharp`, `go`, `java_kotlin`, `javascript_typescript`, `python`, `ruby`, `swift`. Order is immaterial; export/report order is canonical. Unsupported or undetected selections may be refused by GitHub. |
| `query_suites` | One scalar: `default` or `extended`; never a list. Maps to REST `query_suite`. |
| `threat_model` | `remote_sources` or `remote_and_local_sources`; maps exactly to API `remote` or `remote_and_local`. Raw API spellings are not YAML choices. |
| `runner` | Owns the type/label pair. `type: standard_github_runner` forbids `label` and explicitly clears an old label through API null. `type: labeled_runner` requires a nonblank `label`; it can select an existing labeled GitHub or self-hosted runner. No runner/label/capacity is created. |

The [initial setup guide](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/configure-code-scanning/configure-code-scanning) uses “Query suites”, while the
[existing-configuration editing guide](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/manage-your-configuration/edit-default-setup#customizing-your-existing-configuration-of-default-setup) uses “Query suite”.
This contract follows the initial-setup spelling `query_suites`, still one scalar.
The former `query_suite` YAML key is rejected, including when both names are
present. The REST field and native report identity remain `query_suite`.
Published v0.45.0 uses the former YAML spelling; see the
[operational package boundary](../contributing/repository-automation.md#plural-ui-keys-and-operational-package).

GitHub documents local sources as [public preview](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/manage-your-configuration/edit-default-setup#including-local-sources-of-tainted-data-in-default-setup), currently supported for Java/Kotlin and C#. A repository's supported choices and source languages remain GitHub-controlled; gh-settings does not enable languages or infer UI availability from the selected analysis-language list. A complete read requires exactly the API `remote` or `remote_and_local` value. Missing, null, mistyped or unknown values fail acquisition without a fallback to remote-only scanning.

Omission leaves each control unmanaged. Null, invalid choices, duplicate/empty language lists and incomplete runner pairs fail before remote work. Declare a runner object to change the pair; omitting it preserves both members. This configuration requires an **already configured** default setup in a personally owned public repository. It never enables CodeQL or its prerequisites. If currently disabled, enable default setup in a separate operation and confirm it before configuring these controls. Configuration plus explicit `default_setup: false` is invalid; declaring true does not waive the already-configured requirement.

Ordinary `export` attempts all four controls as a complete configuration family. State and details share one response. Confirmed `not-configured` exports `default_setup: false` and removes existing languages/query suite/runner/threat model from the output with info. Unknown state (including denied, failed or malformed reads) emits info and omits the family on first export or retains its existing declaration. Configured state exports true; complete details replace prior values, while unavailable details emit info and retain existing details only. Reads require Administration read and complete exact state/languages/query suite/runner type and nullable label, plus the exact threat model. Unknown, missing, mistyped, denied or inaccessible observations are errors, not inferred defaults. Standard-runner null/empty labels represent the inactive label; a nonempty label with standard runner is rejected.

Plan only reads and reports declared controls. Apply fresh-reads and sends a single PATCH containing only declared configuration members, never `state`. It verifies the resulting configuration **and every omitted configuration companion** against the fresh pre-write state; concurrent changes can make verification fail. Changing only the threat model preserves languages, query suite and runner; omitting threat model preserves its acquired choice. An unchanged declaration writes nothing. Configuration changes may launch asynchronous validation/analysis and consume Actions resources. HTTP 202 and matching readback confirm configuration only; pending/mismatching state, companion drift or denied verification retain unknown effects and fail without retry or rollback. No analysis completion, live grant, runner availability or billing outcome is established by fixtures. See the [public API](https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2026-03-10#update-a-code-scanning-default-setup-configuration).

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

## AI Scan for pull requests

Use the optional boolean `repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_pull_requests` for a personally owned public repository. Omission leaves it unmanaged; false disables it. Null/other types fail. Ordinary `export` automatically attempts this toggle, with info and omission/prior-declaration retention when unavailable. The separate [repository-wide toggle](#ai-scan-for-repositories) is supported; AI-application scanning remains unimplemented pending its API assessment.

The [public repository AI Scan API](https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2026-03-10#get-ai-scan-enablement-for-a-repository) requires Code scanning alerts read and Administration write. Acquisition requires public visibility, a User owner and exact `pr_scan` enabled/disabled. Missing/null/unknown values, 403/404 and incomplete applicability remain errors. Plan only reads. Apply rereads, skips a matching value, sends only `pr_scan` and confirms it with fresh readback. Other AI scans, CodeQL and product/prerequisite controls are not sent or automatically enabled.

Enabling permits GitHub to process pull requests with AI; applicable billing and AI credits can apply. Disabling affects future AI findings. Neither configuration readback nor a successful PATCH establishes a completed scan, coverage, entitlement, grants or actual consumption. Write/readback errors retain unknown effects; there is no automatic retry or rollback. No alert fix, Copilot assignment, scan-execution or credits-purchase operation is performed.

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

Existing entitlement, token grants and appropriate admin/organization-owner/security-manager visibility are required. API visibility alone does not prove writable access or licensing. No product is purchased, credential expanded, alert/pattern/bypass managed or scanning workflow created. Unadopted security product controls remain unsupported. See the [security specification](../features/security.feature.md).

## Security UI settings not managed by gh-settings

The investigation recorded on **2026-11-08** did not establish a formal public API path for complete settings reads and updates for the controls below. They remain unimplemented; that historical result does **not** prove that no API exists today. Reassess target Web UI availability and the applicable official API contract under the [personal-public adoption policy](scope.md#feature-adoption-scope), recording fresh evidence in the [coverage ledger](../contributing/settings-coverage.md). UI availability can depend on prerequisites.

| UI control | Unconfirmed settings API boundary |
| --- | --- |
| Independent Dependency graph enablement | Public dependency comparison, SBOM and snapshot APIs operate on dependency data; organization/enterprise configuration APIs do not establish a personal-repository setter. |
| Automatic dependency submission | Submitting a dependency snapshot does not configure GitHub's automatic submission setting. |
| Dependabot malware alerts | Existing vulnerability-alert enablement does not manage this separate toggle. |
| Dependabot grouped security updates | Existing security-update enablement does not manage grouping. |
| Dependabot GitHub preset ON/OFF | No complete public preset identity/read/update contract was confirmed for the development-dependency or package-malware dismissal presets. API access to auto-dismissed alerts does not configure the presets. |
| Copilot Autofix / Agentic Autofix settings | Alert-specific fix generation, commits and assignment execute remediation rather than configure these repository controls. |
| Global code-scanning check-run failure thresholds | [Ruleset scanning thresholds](rulesets.md#code-scanning-parameters) are supported separately; they do not configure these global check-run settings. |
| Private vulnerability reporting details: Require a CWE assignment, Daily advisory limit, Trusted reporters | The public enablement endpoint exposes only enabled state, not complete detail settings. |

The existing [Dependabot alerts and security updates](#dependabot-alerts-and-security-updates) and [private reporting enablement](#private-vulnerability-reporting) remain supported. The alerts API intrinsically enables/disables Dependency graph too; this side effect is not independent graph management. Individual alert operations, advisory operations, Rulesets and organization/enterprise configuration are not substitutes for the missing settings contracts.

Configuration-file creation/editing remains excluded. Custom Dependabot auto-triage rules are eligible for assessment under the target UI/API policy but are not implemented. GitHub preset ON/OFF above is distinct from custom rules. [Code Security / Advanced Security product switches (#346)](https://github.com/daiksud/gh-settings/issues/346) are not planned because the target Web UI has no corresponding product-wide switch; they are not API-unconfirmed individual controls. The implemented [CodeQL default setup](#codeql-default-setup), [CodeQL configuration](#codeql-default-configuration) and [AI Scan for pull requests](#ai-scan-for-pull-requests) remain separate supported controls.

Sources reviewed on 2026-11-08: [public REST definition](https://github.com/github/rest-api-description/blob/2f8f80e47ea4d3292cf4e37ba9d9b25475e06140/descriptions/api.github.com/api.github.com.json), [public GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql), [GitHub preset documentation](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependabot-auto-triage-rules), and the [historical personal-public inventory](https://github.com/daiksud/gh-settings/issues/323#issuecomment-6719839333) with its [API recheck evidence](https://github.com/daiksud/gh-settings/issues/323#issuecomment-6728386507), both historical evidence rather than current eligibility rules. The public [Dependabot runner settings API](https://docs.github.com/en/rest/dependabot/runner-settings) was also assessed: labeled runners apply only to private/internal repositories with Actions enabled; public repositories report `standard`, so this does not add a configurable personal-public control.
