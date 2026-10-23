# Feature: Declarative Pages configuration

Status: accepted design in [ADR 0028](../adr/0028-declarative-pages-settings.md), Issue #349. Implementation is #350. All keys/options below are planned and remain rejected by current source and published v0.17.0. No deployment or live setting is changed by this design.

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

## Planned declaration and ownership

```yaml
schemaVersion: 1
settings:
  pages:
    build_type: legacy
    source:
      branch: main
      path: /docs
    cname: null
    https_enforced: true
```

This example is a planned contract, not a currently executable settings file.

| Input | Meaning |
| --- | --- |
| Omitted `pages`, or `pages: {}` | Unmanaged; no Pages read or write |
| `pages: null`, scalar or sequence | Invalid locally; not deletion |
| Omitted member | Preserve that configured value; never send a default for it |
| `build_type` | Exactly `legacy` or `workflow`; null/empty/other enum invalid |
| `source` | If present, a mapping containing exactly nonempty `branch` and `path`; path `/` or `/docs`; no aliases to default branch or branch creation |
| Omitted `source` | Preserve an existing legacy source; inactive workflow source is not managed |
| Empty/partial/null `source` | Invalid locally; no guessed tuple companion or source removal |
| `cname: null` (including an explicit empty YAML scalar) | Manage removal of the custom domain; distinct from omission |
| String `cname` | Nonempty ASCII DNS hostname, compared case-insensitively and exported lowercase; no URL, port, slash, wildcard, whitespace or automatic IDNA conversion |
| `cname: ""`, quoted `"null"`, wrong type | Invalid domain; empty string is not a second clearing syntax |
| `https_enforced: false` | Explicit request to disable HTTPS enforcement; never equivalent to omission |
| `public`, `enabled`, `state`, unknown keys | Unsupported; no visibility mutation or implicit lifecycle operation |

Accept case-insensitive ASCII domain labels with normal DNS label/hostname length limits; reject a trailing dot rather than silently changing intent. Domain ownership, DNS validity and certificate eligibility are server/operational checks, not guarantees of local syntax validation. An unknown response field is unowned metadata; it never becomes accepted YAML automatically. Strict duplicate/merge/alias checks and schemaVersion handling retain existing workspace rules.

### Publishing mode

- For an observed legacy site, omitted mode/source preserves the publishing tuple. An explicit source is a complete replacement tuple.
- For an observed workflow site, any declared source is incompatible unless an explicit transition to legacy also declares the full source. Source metadata returned for workflow mode is inactive, not evidence to select legacy.
- An explicit transition to legacy requires both source members, even if a stale tuple is returned. An explicit workflow mode rejects a source declaration. Missing source for a known legacy-to-workflow transition is intentional deactivation of branch-source ownership, not an inferred empty branch.
- Do not create branches, commit site content, edit workflows or call a build/deployment endpoint. GitHub itself can publish after configuration changes. Domain setup in branch publishing can involve a GitHub-created CNAME commit; do not promise a settings-only remote effect or attempt to undo that commit.

## Observation, export and comparison

Use one API-neutral acquired observation for a positively observed existing site, with separate configured values and read-only operational metadata. Its zero value is unknown. Require valid `build_type`, `cname` (a present string or explicit null), `https_enforced` and `public`; require a complete source in legacy mode. Missing mode is not inferred from a source or build status. Missing/null booleans and unsupported enums fail the read. Workflow mode may omit or return null for its inactive source; do not export or compare that source as an active branch. Validate returned strings/tuples before export.

`export --pages` composes with other explicit selections and rejects duplicate options. Include build mode, legacy source when active, explicit nullable cname, and the HTTPS boolean. Exclude visibility, endpoint/site URLs, build state, certificate state, timestamps and DNS verification metadata. Fail an opted-in 404/incomplete read without replacing the settings file; ordinary export gains no Pages request. A supported complete existing export must produce zero configured differences when immediately planned against unchanged fixture state.

Use up to five managed leaf fields: `pages.build_type`, `pages.source.branch`, `pages.source.path`, `pages.cname`, `pages.https_enforced`. A declared complete source manages both leaves. Native comparisons retain booleans, strings and the cname-only null; stable display order follows that list. Render null distinctly from empty string/omission. Count configured differences only. Changing a build/certificate status alone must not produce a desired setting or a perpetual plan difference.

For a workflow-to-legacy transition, the source's before side is explicitly inactive, not an unknown branch or an empty string. Represent that structural state separately from the configured value in the typed report; render it as inactive and require the declared after tuple. Only a verified workflow observation permits an inactive source. Do not reuse cname's nullable value to represent an inactive source or missing data. A legacy-to-workflow transition reports the mode change and its source-deactivation consequence without inventing source declarations or extra managed leaves.

CLI and Action reports need closed Pages-specific field/type validation and their managed-count bounds. Do not broaden existing repository string/boolean constructors to accept null. Credential-free Summary must reconstruct the same comparisons and escape data; private mode must not expose domain, branch or raw server diagnostics to stdout/stderr/Summary or outputs.

## Direct apply and failures

Plan is read-only. Apply validates locally, resolves the checkout target and, only for a managed Pages declaration, obtains fresh complete site state and repository identity. Reject redirects/identity changes and incompatible mode/source combinations before a Pages write. Repeat relevant reads immediately before update rather than consuming a saved plan. No write on a fresh match.

Use PUT with only declared values; a source declaration supplies the complete tuple, and a domain removal sends actual JSON null. Capture omitted applicable companions and read-only visibility for verification. After a successful response, perform bounded fresh readback of complete configured state, require all declared values to match and verify omitted applicable companions/visibility were preserved. Legacy-to-workflow source becoming inactive is the one explicitly modeled source consequence. A mismatch or missing readback is an uncertain update, not success. Do not widen permissions or send compensating settings.

A matching configured observation can succeed while DNS/certificate/build metadata is pending, but claim only configuration convergence, never deployed availability, certificate readiness or a successful build. If the server refuses HTTPS before readiness or rejects a combined update, return its failure, retaining earlier changes. Do not split a failed request into hidden fallback writes, disable HTTPS as a prerequisite, trigger verification/builds, poll indefinitely, retry a write or roll back. A later independent explicit invocation may reconcile after operational prerequisites are ready.

Sequence Pages after ordinary resource updates and before final archive; explicit unarchive remains first. Classify prerequisite reads/validation as no Pages write attempted, and PUT/verification errors as potentially applied writes. Report known earlier writes honestly. No transaction or atomic protection against external changes is promised.

## Lifecycle and remaining gates

Initial implementation manages existing sites only. Every GET 404 remains unknown/inaccessible for export, plan and apply. It is not an absent observation, a no-op for a deletion intent, or permission to POST. A 200 public value is read-only and is never synthesized from repository visibility. Do not adopt creation, deletion, public/private site mutation or access grants until their endpoint-specific evidence and exposure contract have a separately reviewed decision. No undocumented switch or user assertion may override the unknown state.

For #350, implement the accepted existing-site slice in one coherent PR, with the contracts above; keep creation/deletion visibly unresolved and do not close or describe the full lifecycle work as complete. If splitting implementation tracking, retain links and explicit held lifecycle acceptance criteria. A lifecycle gate must not block model/codec/report/API-update tests or other independent issues. No release or actual Pages operation is authorized by this design.

## Acceptance plan for #350

These are planned tests, not executed coverage:

| Layer | Required cases |
| --- | --- |
| Model/codec | Omission/group-empty; explicit false; null only for cname; source pair validation; invalid domains/mode/source combinations; unacquired observations; defensive copies |
| HTTP read/export | Legacy/workflow complete states; present null versus missing cname; incomplete/unknown mode; 401/403/404/redirect; failed export preserves file; opt-in composition; ordinary export unchanged |
| Reconciliation/update | Match no write; full source tuple; legacy/workflow transitions; declared-only PUT; real JSON null; fresh state changes; omitted companions/public preserved; 204 plus verified readback; 400/409/422 and uncertain/partial verification failures |
| Readiness | Changing pending build/DNS/certificate metadata produces no config diff; server refusal stops without fallback, polling, hidden build, rollback or HTTPS disabling |
| Lifecycle/isolation | No POST/DELETE/build/deployment/DNS/workflow/grant request; 404 never creates, deletes or claims absence; unarchive first/final archive last |
| Reports/process | Actual built CLI against local HTTP fixtures with dummy credentials; export-to-zero plan, edit/apply/idempotence; native null/bool/string CLI and credential-free Summary; private redaction; earlier successful writes retained on failure |

Use the existing CI and representative actual-binary acceptance harness; record executed commands/results, per-request timeout and any unexecuted/live cases in the implementation PR. Fixtures prove client behavior against modeled responses, not live permissions, billing, DNS, certificates or deployment. This design was checked against documentation, not a live successful Pages read or mutation. No live Pages mutation, domain change, token issuance or release was used as a probe.
