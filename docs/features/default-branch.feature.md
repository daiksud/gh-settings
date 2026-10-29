---
type: Feature
title: Default branch setting
description: Select an existing default branch through the current UI declaration.
---

## Status

supported and implemented in current source. The [settings reference](../settings.md#default-branch) owns usage and adoption guidance.

## Behavior

- The optional string is `repository.settings.general.default_branch` in the current versioned UI document. Omission is unmanaged; a declared valid name counts as one managed field, including when it already matches.
- Empty, null, non-string or invalid literal names fail before origin/authentication/API work. The validator implements literal Git branch-name syntax without spawning Git, expanding revisions or checking local branches. Slash-separated and Unicode names are preserved; `HEAD`, leading dashes, invalid ref characters/components and `@{-n}` expressions are rejected.
- A complete repository REST observation requires a valid non-null name. Missing, empty, invalid or wrongly typed data fails the read, discarding partial comparison/export instead of becoming an inferred default.
- Export includes the observed branch after all reads succeed. Plan remains read-only and needs only the repository REST observation for this field. The string appears after About Website and before booleans; existing fields keep their relative order.
- Direct apply sends only declared fields in one repository PATCH, with no preliminary read, branch creation/rename or local Git mutation. The selected branch must already exist; GitHub handles existence and policy rejection at update time.
- A declared branch requires a returned valid string equal to the request. Missing/null/invalid/mismatched responses fail after PATCH, warn that repository writes may have applied, and stop later resources without rollback. Undeclared branch values do not alter existing PATCH response requirements.
- CLI and rendered Summary use `repository.settings.general.default_branch` with native strings. The serialized private same-version report retains raw internal identifiers and bounds pending the #481 reporting/count audit.
- Protections, Rulesets and individual PR bases are not edited. Policies that target the default branch can apply to a different branch after this setting changes.

## API and evidence boundaries

The [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents `default_branch` and Administration write. [Git branch-name syntax](https://git-scm.com/docs/git-check-ref-format) informs local validation. No branch endpoint, new persistent permission, or mutation probe is added.

Local fixtures establish exact requests, response verification, rejection and operation ordering. They do not prove a live branch change, every organization policy, or availability in a published package. Real repository settings and live branches are not changed by these tests.

## Test mapping

- [Model](../../internal/settings/default_branch_test.go): literal names, typed setter behavior, omission, managed count and comparison.
- [Codec](../../internal/workspace/ui_general_test.go): strings, omission, invalid types/names and encode/decode round trips.
- [REST](../../internal/github/default_branch_test.go): strict reads, declared-only and mixed PATCH, returned value/type/completeness, permission and missing-branch errors, with exactly one PATCH and no branch operations.
- [Plan](../../internal/app/default_branch_test.go) and [export](../../internal/app/export_test.go): requested fields, native string comparison and complete export.
- [Summary](../../internal/actions/default_branch_test.go): string rendering and invalid before/after values.
- [Process plan](../../tests/acceptance/plan_test.go), [apply](../../tests/acceptance/apply_test.go) and [export](../../tests/acceptance/export_validate_test.go): actual executable output/requests, no-difference cases, invalid-input ordering, missing reads, post-write failure before topics, and export round trips.
