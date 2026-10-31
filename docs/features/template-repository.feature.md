---
type: Feature
title: Template repository setting
description: Manage the Template repository UI checkbox without generating repositories.
---

## Status

supported and implemented in current source. The [settings reference](../settings.md#template-repository) owns usage and package-adoption guidance.

## Behavior

- The optional boolean is `repository.settings.general.template_repository` in the current versioned UI document. Omission is unmanaged; true enables template status, false explicitly disables it, and each declaration counts as one managed field.
- Null, quoted booleans and other types fail before target, authentication or API work. Encoding preserves omission and explicit false.
- The complete repository REST GET requires a non-null boolean `is_template`. Missing or invalid data fails instead of becoming false, including when another REST field requested the observation.
- Export includes the observed boolean only after all required reads complete. Plan compares declared values only, renders native booleans after web commit sign-off and before merge-related values, and performs no mutation or extra GraphQL query for this field.
- Direct apply sends only declared repository fields in one PATCH, including native false. It makes no preliminary read. Forbidden or rejected updates stop later resources without implying rollback or that earlier writes were undone.
- CLI and rendered Summary use `repository.settings.general.template_repository` with native booleans. The serialized private same-version report retains raw internal identifiers and bounds.
- This feature does not generate a repository from a template, copy files, change source-template provenance, or modify Git LFS settings.

## API and evidence boundaries

The [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents `is_template` and Administration write. GitHub's [template prerequisites](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-template-repository) still apply. No new permission or live mutation is introduced to verify this feature.

Local fakes and HTTP fixtures establish typed reads, exact requests and error handling. They do not establish live write permission, supported template conversion for every repository type, or availability in a published package.

## Test mapping

- [Codec](../../internal/workspace/ui_general_test.go) and [model](../../internal/settings/template_test.go): omission, explicit booleans, invalid types, round trips, managed count and comparison.
- [REST](../../internal/github/template_test.go): required boolean observations, exact omitted/true/false/mixed PATCH payloads and forbidden writes.
- [Plan](../../internal/app/template_test.go) and [export](../../internal/app/export_test.go): REST-only requests and complete observed declarations.
- [Summary](../../internal/actions/template_test.go): native boolean rendering and type rejection.
- [Process plan](../../tests/acceptance/plan_test.go), [apply](../../tests/acceptance/apply_test.go) and [export](../../tests/acceptance/export_validate_test.go): actual executable output/requests, incomplete reads, validation ordering, permission failure and export round trips.
