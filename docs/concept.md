---
type: Concept
title: Repository settings as code
description: Define the purpose, ownership model, and future direction of gh settings.
---

## Purpose

`gh settings` is a new tool inspired by [babarot/gh-infra](https://github.com/babarot/gh-infra), [noirbizarre/gh-settings](https://github.com/noirbizarre/gh-settings), and [integrations/terraform-provider-github](https://github.com/integrations/terraform-provider-github). It is not a fork implementation of those projects.

Repository maintainers should be able to record intended settings in version control, review changes, and apply declared values. Configuration lives in `.github/settings.yml` and targets only the repository containing that file. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are also outside repository scope.

The tool currently implements read-only `export`. The other commands and GitHub Action delivery remain future work. Accepted ADRs record agreed decisions; current implementation status is described in the [commands page](commands.md).

## Design philosophy

Written fields are managed, and YAML is authoritative for their values. Omitted fields at any nesting depth are unmanaged. This supports gradual adoption without claiming ownership of settings that maintainers have not declared. Null is not omission; its treatment in future YAML input remains undecided. Read-side description/homepage null normalization is described in the [settings page](settings.md).

Follow the corresponding GitHub update API request schema for field keys and types, including any enum values, as recorded in [ADR 0006](adr/0006-update-api-schema.md). [ADR 0020](adr/0020-group-settings-by-resource.md) places the current fields under `settings.repository`; this configuration grouping is separate from the API request body. [ADR 0021](adr/0021-require-schema-version.md) requires root `schemaVersion: 1` for the file format. Do not turn a repository response, including read-only metadata, into the configuration schema. The [settings page](settings.md) defines exactly five fields supported in the initial design. The [commands page](commands.md) defines their proposed command behavior.

Separate observation from mutation: `plan` informs the maintainer about observed differences, while `apply` sends declared settings directly. A plan is not an execution artifact or a guarantee about later changes. Failures must be visible, without promises of transactional behavior or automatic recovery.

## Current decisions

Each ADR records one independently reversible decision:

- [Deliver as a GitHub CLI extension](adr/0001-github-cli-extension.md)
- [Manage only declared fields](adr/0002-managed-fields.md)
- [Apply declared values directly](adr/0003-direct-apply.md)
- [Validate strictly before mutation](adr/0004-strict-validation.md)
- [Do not roll back automatically](adr/0005-no-automatic-rollback.md)
- [Follow the update API request schema](adr/0006-update-api-schema.md)
- [Implement in Go](adr/0007-go-implementation.md)
- [Use the official GitHub CLI Go library](adr/0008-official-go-gh-library.md)
- [Use the YAML organization's Go YAML v4 library](adr/0009-yaml-v4-library.md)
- [Group repository fields within settings](adr/0020-group-settings-by-resource.md)
- [Require a configuration schema version](adr/0021-require-schema-version.md)

Go is the agreed implementation language, and `github.com/cli/go-gh/v2` is selected for API access and GitHub CLI authentication integration. Prefer the Go standard library where sufficient; no command framework is mandated. The YAML organization's `go.yaml.in/yaml/v4` is selected for configuration reading and writing in [ADR 0009](adr/0009-yaml-v4-library.md), accepting release-candidate risk; it is not a Go team-maintained library. The [release design](releases.md) records separate distribution decisions and their ADRs.

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

Explicit null handling in future YAML input, remaining authentication details, toolchain and dependency versions, and output options have not been decided. Future composition across multiple update endpoints, new-object required fields, and inner rules identity are also unresolved. The current format requires `schemaVersion: 1` and places the five fields within `settings.repository`; this does not settle those future composition choices or extend supported fields. Release decisions and open implementation mechanics are tracked on the [release page](releases.md).
