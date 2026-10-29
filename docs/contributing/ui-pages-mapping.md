---
type: Reference
title: Prepared Pages UI mappings
description: Internal existing-site Pages controls, API conversion and activation handoff.
sources:
  - id: pages-ui
    resource: https://github.com/daiksud/gh-settings/settings/pages
  - id: existing-pages-ui
    resource: https://github.com/daiksud/gh-qw/settings/pages
  - id: deployed-pages-ui
    resource: https://github.com/daiksud/docfw-cmp/settings/pages
---

## Preparation boundary

This reference covers the internal preparation for [Issue #486](https://github.com/daiksud/gh-settings/issues/486), limited to the currently supported existing-site subset. Public YAML, export, application and reports remain unchanged. [#479](https://github.com/daiksud/gh-settings/issues/479) activates the complete schema; [#480](https://github.com/daiksud/gh-settings/issues/480) owns unknown-choice normalization/warnings; [#481](https://github.com/daiksud/gh-settings/issues/481) owns reporting integration. No compatibility aliases, migration, site lifecycle or DNS management is introduced.

The [UI model](../../internal/settings/uimodel/pages.go) contains display terminology and typed choices without serialization tags. The [adapter](../../internal/github/ui_pages.go) translates desired fields into existing validated Pages inputs and projects acquired configured state directly. `PreparedPages.Public` retains unowned visibility metadata outside desired UI values.

## Verified UI inventory

English Pages controls were inspected on 2026-10-28. The Source menu was inspected on an existing gh-qw site, and Custom domain / Enforce HTTPS on gh-qw and the deployed docfw-cmp site. gh-settings currently has no configured Pages site; its branch form was used only to inspect the conditional folder menu. A local branch selection revealed the menu without saving; reloading confirmed None and disabled. None/disabled is inspection context, not a newly supported declaration or absent-site observation.[^pages-ui][^existing-pages-ui][^deployed-pages-ui]

The prepared prefix `P` means `repository.settings.pages`.

| UI label / location | Prepared path below `P` | Internal API conversion |
| --- | --- | --- |
| Build and deployment / Source | `build_and_deployment.source` | `github_actions` → `build_type=workflow`; `deploy_from_a_branch` → `build_type=legacy` |
| Build and deployment / Branch | `build_and_deployment.branch` | Literal branch to `source.branch` |
| Branch form / Select folder | `build_and_deployment.select_folder` | `root` for `/ (root)` → `source.path=/`; `docs` for `/docs` → `source.path=/docs` |
| Custom domain | `custom_domain` | Hostname to `cname`; explicit null represents Remove |
| Enforce HTTPS | `enforce_https` | Boolean to `https_enforced` |

These paths use UI wording, not API identifiers. `PagesCustomDomain.Value` is only an internal nullable-scalar representation: activation must expose `custom_domain: example.com` or `custom_domain: null`, never a `value` child key. No `build_type`, `cname`, `https_enforced` or API path token becomes an alias.

Branch and Select folder are declared together or omitted together, preserving the existing atomic source contract. Source=`github_actions` cannot declare either branch control. An omitted Source leaves the existing acquired-mode dependency check in charge. Transitioning from Actions to branch deployment requires an explicit complete branch/folder pair; no branch is inferred. Folder choices remain exactly root/docs, not arbitrary API paths. Switching to Actions hides/deactivates the old source according to the current observation contract.

## Presence, defaults and acquisition

Each independent desired control remains optional. Domain wrapper omission leaves the domain unmanaged; a present wrapper with a nil value explicitly removes it. Empty text is invalid and cannot silently clear the domain. Hostname validation and lowercase canonicalization remain with existing Pages state/preflight logic. False HTTPS remains explicit; converted declarations and acquired projections own newly created pointers.

Named `DefaultPagesSource=deploy_from_a_branch` and `DefaultPagesFolder=root` are separate per-field candidates for #480 unknown-enum normalization. They never populate omitted desired fields, missing source members or failed acquisition. A source fallback must not invent a required branch or imply a complete valid legacy source. Current successful-response parsing remains strict for unknown modes/folders and missing required fields. Custom domain/free-text branch and actual booleans need no unknown-enum choice fallback.

Acquired null domain means known no custom domain, not unacquired. Legacy mode requires complete valid source; Actions mode does not expose inactive source. Visibility remains acquired metadata and is verified during writes, never desired intent. The UI can disable HTTPS editing when the default domain requires it; the helper preserves configured booleans without claiming every desired value will be accepted by GitHub.

## Existing-site write and failure boundaries

The existing client reads repository identity/archive state and the existing site, validates the complete desired result, refreshes identity before a single PUT, verifies all configured values including omitted companions/Public, and confirms identity again. Only declared members enter the payload; domain clearing sends JSON null. Only 204 is accepted. Repository/site creation, deletion, unpublishing, build triggering, source workflow creation and DNS/certificate management remain unsupported.

HTTP 404 and permission failures mean unavailable, never an absent site or false/default controls. Malformed or incomplete successful responses fail acquisition. Failed preflight attempts no Pages write. Write/status/readback failures retain possible effects and earlier confirmed work; there is no retry, rollback or transaction. These rules are unchanged by UI preparation.

## Validation and activation handoff

Independent [choice/dependency fixtures](../../internal/github/ui_pages_test.go), [nullable-domain/HTTPS fixtures](../../internal/github/ui_pages_domain_test.go), [acquired projection fixtures](../../internal/github/ui_pages_observation_test.go) and [literal raw/exact PUT fixtures](../../internal/github/ui_pages_composition_test.go) cover every supported control and choice. They verify omission, explicit false/null, source atomicity, ownership, inactive source, raw read conversions, exact writes, unowned visibility preservation and unavailable/malformed data. Existing [Pages transport tests](../../internal/github/pages_test.go) retain archive, identity, companion mismatch and uncertain-effect matrices.

Issue #479 must connect all UI paths together without widening the existing-site scope. #480 must retain provenance when converting unknown choices and leave required-source/acquisition failures explicit. #481 must preserve inactive-source, nullable-domain and visibility diagnostics at reporting boundaries. User-facing docs intentionally describe the current public contract until activation.

[^pages-ui]: GitHub, gh-settings Pages settings, unsaved conditional-form inspection, 2026-10-28.
[^existing-pages-ui]: GitHub, gh-qw Pages settings and Source menu, read-only inspection, 2026-10-28.
[^deployed-pages-ui]: GitHub, docfw-cmp deployed Pages settings, read-only inspection, 2026-10-28.
