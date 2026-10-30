---
type: Feature
title: Declarative Pages configuration
description: Manage existing-site UI controls with explicit nullable domain and complete identity verification.
---

Status: accepted design in [ADR 0028](../adr/0028-declarative-pages-settings.md), Issue #349. The existing-site subset of #350 is supported. Creation/deletion and visibility mutation remain deferred and #350 remains open for those criteria. No live deployment or setting was used as release evidence.

## Official contracts

Reviewed against GitHub.com documentation on October 23, 2026:

| Contract | Official reference |
| --- | --- |
| GET site: Pages read; public data may be unauthenticated | [Get a Pages site](https://docs.github.com/en/rest/pages/pages#get-a-github-pages-site) |
| PUT existing site: Pages write and Administration write; nullable cname clears | [Update a Pages site](https://docs.github.com/en/rest/pages/pages#update-information-about-a-github-pages-site) |
| POST/DELETE lifecycle: Pages write and Administration write; not adopted here | [Create](https://docs.github.com/en/rest/pages/pages#create-a-github-pages-site), [delete](https://docs.github.com/en/rest/pages/pages#delete-a-github-pages-site) |
| A 404 can conceal authorization failure | [REST troubleshooting](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api#404-not-found-for-an-existing-resource) |
| Availability depends on repository/account plan | [About Pages](https://docs.github.com/en/pages/getting-started-with-github-pages/what-is-github-pages) |
| Private/internal repository does not guarantee a private site | [Create a site](https://docs.github.com/en/enterprise-cloud@latest/pages/getting-started-with-github-pages/creating-a-github-pages-site) |
| Private site access is a separate Enterprise Cloud capability | [Site visibility](https://docs.github.com/en/enterprise-cloud@latest/pages/getting-started-with-github-pages/changing-the-visibility-of-your-github-pages-site) |
| Legacy branch/folder versus workflow publishing | [Publishing source](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site) |
| Domain management and possible CNAME commits | [Custom domains](https://docs.github.com/en/pages/configuring-a-custom-domain-for-your-github-pages-site/managing-a-custom-domain-for-your-github-pages-site) |
| Certificate/DNS readiness is asynchronous | [HTTPS](https://docs.github.com/en/pages/getting-started-with-github-pages/securing-your-github-pages-site-with-https) |

GitHub Free supports public repositories; private repository eligibility requires an appropriate paid plan. Do not infer actual entitlement from plan names or purchase/enable a product. Pages writes also require the actor's applicable administrative/maintenance/custom-role authorization. Public GET success establishes observable site data, not that a token has Pages write or Administration write. Report the actual failure without asserting an unobserved cause.

Private publication requires an eligible Enterprise Cloud organization project site from a private/internal repository; organization sites do not support that access-control choice. Enterprise Managed Users impose additional private-only restrictions. Treat these as applicability constraints, not proof of the target site's visibility. Read the site's own `public` boolean and preserve it, including for a private repository whose site is public. The documented write schemas here do not expose `public` as a writable parameter.

## Declaration and ownership

All public paths below are relative to `repository.settings.pages`; the exact current version marker is required.

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

| Input | Meaning |
| --- | --- |
| Omitted Pages or `{}` | Unmanaged; no Pages request |
| `build_and_deployment.source` | `deploy_from_a_branch` or `github_actions`, internally legacy/workflow |
| `build_and_deployment.branch` and `select_folder` | Complete tuple when declared; literal nonempty branch and `root`/`docs` folder |
| Omitted branch/folder | Preserve applicable existing branch source; workflow source remains inactive |
| Empty/partial/null tuple | Invalid; no guessed companion or source deletion |
| `custom_domain: null`, including bare empty scalar | Explicit domain removal, distinct from omission |
| String `custom_domain` | ASCII DNS hostname with at least two labels, lowercase comparison/export; no URL/port/slash/wildcard/whitespace/IDNA conversion |
| Empty string, quoted `null`, internal `{value: ...}` | Invalid domain/shape |
| `enforce_https: false` | Explicit disable; not omission |
| API aliases, visibility/lifecycle/unknown keys | Rejected; no site creation/deletion/visibility mutation |

Accept normal ASCII DNS length limits and reject trailing dots. DNS ownership, certificate readiness and server eligibility are not guaranteed by local validation. Unknown response metadata never becomes accepted YAML. Existing strict duplicate/merge/alias/type validation applies.

### Publishing mode

- For an observed branch-publishing site, omitted source choice and branch/folder controls leave its tuple unmanaged. Declared branch/folder controls form a complete replacement tuple.
- For an observed workflow site, branch/folder intent requires an explicit `source: deploy_from_a_branch` transition with both controls. Inactive API source metadata never selects a mode.
- A transition to branch publishing needs the complete tuple even if stale source metadata exists. `source: github_actions` rejects branch/folder declarations and intentionally deactivates branch-source ownership without inventing an empty branch.
- Do not create branches, commit site content, edit workflows or call a build/deployment endpoint. GitHub itself can publish after configuration changes. Domain setup in branch publishing can involve a GitHub-created CNAME commit; do not promise a settings-only remote effect or attempt to undo that commit.

## Observation, export and comparison

Use one API-neutral acquired observation for a positively observed existing site, with separate configured values and read-only operational metadata. Its zero value is unknown. Require valid `build_type`, `cname` (a present string or explicit null), `https_enforced` and `public`; require a complete source in legacy mode. Missing mode is not inferred from a source or build status. Missing/null booleans and unsupported modes fail the read. Workflow mode may omit or return null for its inactive source; do not export or compare that source as an active branch. Validate returned strings/tuples before export.

A complete legacy source with a valid literal branch and a correctly typed folder outside `/` and `/docs` projects to the named `root` default. Empty/whitespace strings are acquired raw values; missing/null/wrongly typed source members remain errors. Own the exact raw folder, UI path and default alongside the effective source. Never infer a branch. Export warns for selected acquired normalized folders; ordinary export and pure reads remain silent. A declared complete source pair retains a write-required folder comparison even if effective root and branch already match. Source still counts as two managed native fields. Private reports retain closed normalization evidence; detailed CLI/Summary render the UI warning and decision, while status-only output suppresses raw values.

Apply warns only for declared source intent using each fresh successful acquisition. Verify declared source against exact raw branch/folder values; unchanged unknown readback is an uncertain write. When source is omitted and remains active, preserve its original raw source while verifying other declared values. Switching to workflow leaves source inactive. Preserve existing domain canonicalization, HTTPS, visibility and identity verification. These rules do not activate unknown publishing modes or complete #481's reporting audit.

`export --pages` composes with other explicit selections and rejects duplicate options. Project the UI source choice, active branch/select-folder controls, scalar nullable `custom_domain` and `enforce_https`. Exclude visibility, endpoint/site URLs, build state, certificate state, timestamps and DNS verification metadata. Fail an opted-in 404/incomplete read without replacing the settings file; ordinary export gains no Pages request. A supported complete existing export must produce zero configured differences when immediately planned against unchanged fixture state.

The existing private protocol uses up to five internal managed leaves: `pages.build_type`, `pages.source.branch`, `pages.source.path`, `pages.cname`, `pages.https_enforced`. A declared complete source manages both leaves. Native comparisons retain booleans, strings and the cname-only null; stable display order follows that list. Render null distinctly from empty string/omission. Count configured differences only. Changing a build/certificate status alone must not produce a desired setting or a perpetual plan difference.

For a workflow-to-legacy transition, the source's before side is explicitly inactive, not an unknown branch or an empty string. Represent that structural state separately from the configured value in the typed report; render it as inactive and require the declared after tuple. Only a verified workflow observation permits an inactive source. Do not reuse cname's nullable value to represent an inactive source or missing data. A legacy-to-workflow transition reports the mode change and its source-deactivation consequence without inventing source declarations or extra managed leaves.

Current CLI/Summary and serialized private reports retain these internal component paths/values pending #481; they are not YAML aliases. Reports keep closed Pages-specific field/type validation and existing bounds. Do not broaden existing repository string/boolean constructors to accept null. Credential-free Summary must reconstruct the same comparisons and escape data; private mode must not expose domain, branch or raw server diagnostics to stdout/stderr/Summary or outputs.

## Direct apply and failures

Plan is read-only. Apply validates locally, resolves the checkout target and, only for a managed Pages declaration, obtains fresh complete site state and repository identity. Reject redirects/identity changes and incompatible mode/source combinations before a Pages write. Repeat relevant reads immediately before update rather than consuming a saved plan. No write on a fresh match.

Use PUT with only declared values; a source declaration supplies the complete tuple, and a domain removal sends actual JSON null. Capture omitted applicable companions and read-only visibility for verification. After a successful response, perform bounded fresh readback of complete configured state, require all declared values to match and verify omitted applicable companions/visibility were preserved. Legacy-to-workflow source becoming inactive is the one explicitly modeled source consequence. A mismatch or missing readback is an uncertain update, not success. Do not widen permissions or send compensating settings.

A matching configured observation can succeed while DNS/certificate/build metadata is pending, but claim only configuration convergence, never deployed availability, certificate readiness or a successful build. If the server refuses HTTPS before readiness or rejects a combined update, return its failure, retaining earlier changes. Do not split a failed request into hidden fallback writes, disable HTTPS as a prerequisite, trigger verification/builds, poll indefinitely, retry a write or roll back. A later independent explicit invocation may reconcile after operational prerequisites are ready.

Sequence Pages after ordinary resource updates and before final archive; explicit unarchive remains first. Classify prerequisite reads/validation as no Pages write attempted, and PUT/verification errors as potentially applied writes. Report known earlier writes honestly. No transaction or atomic protection against external changes is promised.

## Lifecycle and remaining gates

Initial implementation manages existing sites only. Every GET 404 remains unknown/inaccessible for export, plan and apply. It is not an absent observation, a no-op for a deletion intent, or permission to POST. A 200 public value is read-only and is never synthesized from repository visibility. Do not adopt creation, deletion, public/private site mutation or access grants until their endpoint-specific evidence and exposure contract have a separately reviewed decision. No undocumented switch or user assertion may override the unknown state.

For #350, implement the accepted existing-site slice in one coherent PR, with the contracts above; keep creation/deletion visibly unresolved and do not close or describe the full lifecycle work as complete. If splitting implementation tracking, retain links and explicit held lifecycle acceptance criteria. A lifecycle gate must not block model/codec/report/API-update tests or other independent issues. No release or actual Pages operation is authorized by this design.

## Acceptance criteria and test mappings

The existing-site implementation verifies these criteria with local fixtures. Lifecycle mutations remain prohibited and unimplemented:

| Layer | Required cases |
| --- | --- |
| Model/codec | Omission/group-empty; explicit false; null only for the public custom-domain scalar; source pair validation; invalid domains/mode/source combinations; unacquired observations; defensive copies |
| HTTP read/export | Legacy/workflow complete states; present null versus missing cname; incomplete/unknown mode; 401/403/404/redirect; failed export preserves file; opt-in composition; ordinary export unchanged |
| Reconciliation/update | Match no write; full source tuple; legacy/workflow transitions; declared-only PUT; real JSON null; fresh state changes; omitted companions/public preserved; 204 plus verified readback; 400/409/422 and uncertain/partial verification failures |
| Readiness | Changing pending build/DNS/certificate metadata produces no config diff; server refusal stops without fallback, polling, hidden build, rollback or HTTPS disabling |
| Lifecycle/isolation | No POST/DELETE/build/deployment/DNS/workflow/grant request; 404 never creates, deletes or claims absence; unarchive first/final archive last |
| Reports/process | Actual built CLI against local HTTP fixtures with dummy credentials; export-to-zero plan, edit/apply/idempotence; native null/bool/string CLI and credential-free Summary; private redaction; earlier successful writes retained on failure |

Use the existing CI and representative actual-binary acceptance harness; record executed commands/results, per-request timeout and any unexecuted/live cases in the implementation PR. Fixtures prove client behavior against modeled responses, not live permissions, billing, DNS, certificates or deployment. This design was checked against documentation, not a live successful Pages read or mutation. No live Pages mutation, domain change, token issuance or release was used as a probe.

### Executed existing-site test mappings

- Model ownership, defensive copies, nullable-domain comparison, mode/source transitions and inactive-source validation: `TestPagesOwnershipTransitionsAndCopies` in [model tests](../../internal/settings/pages_test.go).
- YAML omission/empty mapping, null-only domain clearing, false preservation, complete source and strict unknown/duplicate/type checks: `TestUIConfigPagesControls`, `TestUIConfigPagesDomainPresence` and `TestUIConfigPagesRejectsInvalidSyntax` in [UI workspace tests](../../internal/workspace/ui_pages_test.go).
- Complete reads, denied/ambiguous404 responses, declared-only PUT, repository identity, preserved companions/public visibility, no retries, matching state and uncertain writes: [API tests](../../internal/github/pages_test.go).
- Independent plan, honest earlier-write progress, no unmanaged reads, explicit unarchive/Pages/final-archive order: [application tests](../../internal/app/pages_test.go).
- Closed native null/boolean/string report handling, missing-versus-null JSON fields and inactive-source structure: [Summary tests](../../internal/actions/pages_test.go); explicit composable export: [CLI tests](../../internal/cli/pages_test.go).
- The supplied actual CLI binary exercises ordinary/opted-in export, export-to-zero-plan, clearing/false, idempotent apply, both publishing-mode transitions, private status-only Summary without credentials, failed-export preservation, redirects and partial writes: `TestPagesExistingSiteExportPlanApplyAndPrivateSummary` in [acceptance tests](../../tests/acceptance/pages_test.go).

These checks use local modeled responses and dummy credentials. They do not establish live Pages permissions, token grants, entitlement, API mutation compatibility, DNS or certificate readiness. No POST/DELETE, build/deployment request, live mutation or release was performed.

Folder normalization is covered by the [raw HTTP matrix](../../internal/github/pages_folder_http_test.go), [opaque model](../../internal/settings/pages_folder_observation_test.go), [atomic-source comparison](../../internal/settings/pages_folder_comparison_test.go), [closed private report](../../internal/actions/normalization_pages_folder_test.go), [scoped export](../../internal/app/normalization_pages_folder_export_test.go) and [executable workflow](../../tests/acceptance/normalization_pages_folder_test.go). The executable cases verify export/default provenance, fallback-equal write intent, two-field source counts, detailed/status-only reports, ignored-write uncertainty, canonical success/no-op, omitted raw-source preservation, earlier REST effects and null acquisition without saving or writing Pages.
