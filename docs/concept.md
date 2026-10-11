---
type: Concept
title: Repository settings as code
description: Define the purpose, ownership model, and future direction of gh settings.
---

## Purpose

`gh settings` is a new tool inspired by [babarot/gh-infra](https://github.com/babarot/gh-infra), [noirbizarre/gh-settings](https://github.com/noirbizarre/gh-settings), and [integrations/terraform-provider-github](https://github.com/integrations/terraform-provider-github). It is not a fork implementation of those projects.

Repository maintainers should be able to record intended settings in version control, review changes, and apply declared values. Configuration lives in `.github/settings.yml` and targets only the repository containing that file. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are also outside repository scope.

This documentation records an agreed design for an **unimplemented tool**. Accepted ADRs mean agreed decisions, not completed implementation.

## Design philosophy

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. This supports gradual adoption without claiming ownership of settings that maintainers have not declared. Null is not omission; its treatment remains undecided.

Keep configuration field names, types, and any enum values aligned with GitHub's update API. Do not turn a repository response, including read-only metadata, into the configuration schema. The [settings page](settings.md) defines exactly five fields supported in the initial design. The [commands page](commands.md) defines their proposed command behavior.

Separate observation from mutation: `plan` informs the maintainer about observed differences, while `apply` sends declared settings directly. A plan is not an execution artifact or a guarantee about later changes. Failures must be visible, without promises of transactional behavior or automatic recovery.

## Current decisions

Each ADR records one independently reversible decision:

- [Deliver as a GitHub CLI extension](adr/0001-github-cli-extension.md)
- [Manage only declared fields](adr/0002-managed-fields.md)
- [Apply declared values directly](adr/0003-direct-apply.md)
- [Validate strictly before mutation](adr/0004-strict-validation.md)
- [Do not roll back automatically](adr/0005-no-automatic-rollback.md)

## Future direction

A GitHub Action is an eventual delivery requirement, deferred from initial implementation. Its trigger, packaging, and authentication design are not decided. Labels, rulesets, Pages, and Actions settings are excluded from the MVP; this section does not commit to implementing all of them.

The following label semantics are agreed for a future extension of scope:

- Omit `labels`: the collection is unmanaged.
- Provide a list: membership is authoritative; delete remote labels not declared.
- Provide `labels: []`: delete all remote labels.
- On retained labels, omitted fields such as `description` preserve current values.
- Label names identify labels; ambiguous matches are errors. There is no rename inference, and explicit rename support is deferred.

Deleting a label loses its assignments. That consequence must be considered before future label support is applied.

Names also identify outer rulesets, ambiguous matches are errors, and there is no rename inference. Applying label-like membership and nested collection behavior to rulesets was proposed but **not confirmed**. Ruleset deletion semantics and inner rule identity remain open questions.

## Unresolved decisions

The exact YAML wrapper shape, null handling, implementation language, authentication details, and output options have not been decided. Future new-object required fields and inner rules identity are also unresolved. The configuration path and five MVP field names are agreed; examples must not imply that a complete YAML schema or these remaining choices are settled.
