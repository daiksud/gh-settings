---
type: Reference
title: "Pages settings"
description: "Pages settings declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

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

Convergence does not prove DNS, certificate or deployment readiness. Pending metadata alone is not drift. GitHub can publish or create a CNAME-file commit as a side effect; the tool does not trigger builds/deployments or edit DNS/content. Pages read/write, Administration write and actor authorization are prerequisites; public GET does not prove write grants. See [Pages behavior](../features/pages.feature.md).

### Environment ref-type observation normalization

A complete unsupported string `type`, including empty or whitespace, projects to the named `branch` fallback. Retain the exact original type, positive API ID and case-sensitive pattern name separately from effective policy. Missing/null/wrong-type fields, invalid names/IDs, duplicate raw identities, unstable counts and incomplete pages remain acquisition failures; they do not produce defaults or partial warnings.

In canonical Selected mode, export warns with the Environment name, pattern ID, exact name and original/fallback type, then saves effective `ref_type`/`name_pattern` values without API IDs or provenance. Warnings follow folded Environment-name order, mode warning first, then ascending pattern ID. Canonical companions stay silent. Distinct raw identities that project to the same branch/name are retained separately: export warns and fails without replacing its destination, rather than collapsing them. Raw patterns under effective No restriction remain latent and produce neither ref-type warnings nor exported rules.

An explicit pattern list owns the observed collection, including `[]`; an omitted list owns none of its values or diagnostics. Plan and detailed Summary retain one actionable collection update at fallback equality, exact raw evidence and the requested fields of the effective before policy. A genuine collision is labeled unavailable effective pattern data, not an empty list. Only selected Environment names are observed. Same-version private reports retain closed per-ID evidence and canonical companions; status-only output hides these details. Reports never supply apply input, and the existing managed collection count remains one.

Apply sends declared fields in the Environment PUT and canonical types in separate pattern requests. It confirms fresh canonical Selected mode, creates required pairs before deleting obsolete raw identities, and retains the IDs of existing canonical companions. Before deletion, an individual GET must match the exact original type, case-sensitive name and ID. Creation acknowledgments remain strict; unsupported or malformed acknowledgments stop before deletion. Independent complete readback must confirm the requested canonical collection: a retained unsupported type cannot prove success merely because its fallback matches. Fresh owned warnings expose the acquired record, not a verification substitute.

Acquisition failures make no Environment write. Later failures retain earlier or uncertain effects, report failure and do not trigger automatic retry or rollback. Caller YAML is unchanged. A complete canonical match is a silent no-op on repeat; existing omitted built-ins are preserved. These source behaviors do not apply live settings or publish a release.
