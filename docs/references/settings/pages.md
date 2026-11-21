---
type: Reference
title: "Pages settings"
description: "Pages settings declaration, ownership and API contracts."
---

# Pages configuration

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

## Source, branch and custom domain

`build_and_deployment.source` is `deploy_from_a_branch` or `github_actions`. Branch mode uses a complete literal `branch` plus `select_folder: root|docs`; partial tuples fail except the explicit deletion selector below. Switching from workflow to branch requires that tuple; workflow mode rejects branch/folder intent. No branch/content/workflow is created. Omitted Pages/empty mapping manages nothing; other omitted members remain unmanaged.

`custom_domain` is a nullable scalar, never a `{value: ...}` mapping. Null (including a bare empty scalar) removes it. Omission leaves it unmanaged; empty string and quoted `null` are invalid. Strings must be ASCII DNS hostnames with at least two labels and normal length limits, without trailing dot, URL, wildcard, port or whitespace. Comparison/export lowercase without IDNA or DNS verification. `enforce_https` is boolean; null is invalid.

A complete declaration with explicit `source: deploy_from_a_branch`, a real branch and folder creates a missing site and reconciles a present one. `source: github_actions` creates workflow publishing when absent, without creating or running a workflow. On an absent site, domain/HTTPS-only intent, branch/folder without explicit Source, and branch Source without a complete tuple fail before any write. On a present branch site, source alone may retain the existing branch/folder; a workflow-to-branch transition still requires both.

## Deleting a Pages site

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

Only exact lowercase `none` is reserved; other branch names retain their case (`None` is a literal branch). The accepted collision with a real branch named `none` is intentional. Deletion requires the explicit Source above and rejects any `select_folder`, `custom_domain` (including null), or `enforce_https` (including false). It directly deletes either publishing mode; it never sends the sentinel to POST/PUT. Omission, null, empty groups and `custom_domain: null` never delete a site. If a present site publishes from a branch literally named `none`, export reports Pages as unavailable. First export omits Pages and re-export retains compatible prior Pages intent. Independent settings can still be saved.

## Export and verification

Under the required-permissions precondition, target Pages GET 404 means absent. Deleting an absent site is a no-op. On an absent site, `export` emits the deletion declaration above for absence; applying that export later intentionally deletes any site then present. Present-site export includes mode, active branch/folder, nullable domain and HTTPS after complete acquisition. Visibility/readiness metadata are excluded. Other failures and missing/null/wrongly typed required fields emit info and retain prior Pages declarations; independent families continue.

Creation sends only source fields with POST, requires 201, then a fresh complete matching GET 200 and repository identity verification. Declared domain/HTTPS controls are reconciled afterward with PUT as needed, preserving observed server defaults for omissions. A later failure retains and reports confirmed creation, with no rollback. Updating requires PUT 204 and complete readback; deletion requires DELETE 204 and GET 404. A concurrent DELETE 404 requires a new GET 404 before success. POST/PUT followed by 404, unexpected statuses, redirects, explicit denials, timeouts and readback mismatch fail; writes are never retried automatically. Creation observes server-selected visibility without requesting a visibility change.

A complete legacy-mode response with a folder string outside `/` and `/docs`, including an empty string, projects to `select_folder: root`. Its exact raw folder, UI path and named default remain observation provenance; the branch is preserved and never invented. Export warns and saves only the effective UI choice. A declared branch/folder pair retains a warning and a write-required folder decision even when both declared values equal the effective projection. The pair still counts as two managed native fields. Omitted source remains unmanaged and does not disclose its raw folder. Inactive workflow source is not acquired.

A correctly typed publishing mode outside `legacy` and `workflow` projects to `source: deploy_from_a_branch` only with a complete typed branch/folder tuple and valid literal branch. Missing/null/wrongly typed source members remain errors; no branch is inferred. Empty and whitespace mode strings retain exact provenance. Mode and folder records remain independent, and export warns in that order. A declared publishing mode counts one native field and retains a write-required decision even when it equals the projection. Explicit source intent with an unknown raw mode requires `source: deploy_from_a_branch`; the fallback alone cannot prove applicability. Omitted publishing mode remains unmanaged and is not disclosed.

Plan is read-only; an inactive source is not invented as an observed tuple. Apply reads fresh state, skips exact matches, uses the lifecycle operations above or sends declared-only PUT values and verifies requested values, omitted applicable companions, public visibility and repository identity. Verification checks exact raw mode and branch/folder strings: an unchanged unknown token cannot confirm a requested canonical value, and omitted active mode/source must retain their original raw values. Domain canonicalization remains unchanged. Apply warnings use only fresh acquired provenance, independently scoped to declared publishing mode and source intent. Explicit unarchive precedes it; final archive follows. Failure can leave earlier writes or uncertain effects, without retry/fallback/rollback.

Convergence does not prove DNS, certificate or deployment readiness. Pending metadata alone is not drift. GitHub can publish or create a CNAME-file commit as a side effect; the tool does not trigger builds/deployments or edit DNS/content. Pages read/write, Administration write and actor authorization are prerequisites; public GET does not prove write grants. See [Pages behavior](../../development/features/pages.feature.md).

[Settings index](README.md) · [GitHub App permissions](permissions.md)
