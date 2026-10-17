---
type: Concept
title: Repository settings as code
description: Define the purpose, ownership model, and future direction of gh settings.
sources:
  - id: aggressive-v0-plan
    resource: https://github.com/daiksud/gh-settings/issues/133
---

## Purpose

`gh settings` is a new tool inspired by [babarot/gh-infra](https://github.com/babarot/gh-infra), [noirbizarre/gh-settings](https://github.com/noirbizarre/gh-settings), and [integrations/terraform-provider-github](https://github.com/integrations/terraform-provider-github). It is not a fork implementation of those projects.

Repository maintainers should be able to record intended settings in version control, review changes, and apply declared values. Configuration lives in `.github/settings.yml` and targets only the repository containing that file. Organization settings and multi-repository management are outside the tool's scope. Inherited Organization rulesets are also outside repository scope.

The CLI implements export, validation, planning, and direct apply, with export saving `.github/settings.yml`. The [public composite Action](adr/0023-public-composite-action.md) composes those operations; this repository's operational callers use the released v0.7.0 package with owner-approved, single-job main apply. Source changes, release publication, and consumer adoption are separate operations. The [commands page](commands.md) describes command behavior, and the [Actions guide](github-actions.md) identifies released consumer pins.

## Design philosophy

Written fields are managed, and YAML is authoritative for their values. Omitted fields are unmanaged, allowing gradual adoption. The [settings page](settings.md#configuration-semantics) defines the supported fields and the distinction among omission, explicit values, YAML null, and read-side null normalization.

Field keys and types follow the corresponding GitHub update request schema, as recorded in [ADR 0006](adr/0006-update-api-schema.md). Configuration grouping and versioning follow [ADR 0020](adr/0020-group-settings-by-resource.md) and [ADR 0021](adr/0021-require-schema-version.md). Repository response metadata does not become configuration. The [settings page](settings.md) gives the current five-field contract; the [commands page](commands.md) records command behavior and release status.

The mutation workflow separates observation from mutation: `plan` informs the maintainer about observed differences, while `apply` sends declared settings directly. A plan is not an execution artifact or a guarantee about later changes. Failures must be visible, without promises of transactional behavior or automatic recovery.

## v0 simplification contract

The [ordered refactoring series](https://github.com/daiksud/gh-settings/issues/133) establishes the requirements below. The shared Go implementation, thin adapters and deliberate behavior simplifications are published in [v0.7.0](https://github.com/daiksud/gh-settings/releases/tag/v0.7.0); source integration, release publication and consumer adoption remain separate operations.[^aggressive-v0-plan]

### Retained product requirements

Keep `schemaVersion: 1` → `settings` → `repository`, the existing field names, partial declarations, omission, explicit false and empty strings. Preserve currently accepted YAML syntax, including aliases and merge keys. Do not flatten the document, remove the version, add settings, or introduce a speculative resource framework.

Keep checkout-root configuration and target isolation: only the containing checkout's repository is managed, and undeclared fields stay absent from updates. Invalid input fails before remote work. Apply sends declared values directly without GET, saved-plan execution or reconciliation; write redirects remain rejected. Preserve credential boundaries, owner approval, target-scoped token issuance/revocation and verification of immutable release artifacts. No rollback, retry, drift or additional approval framework is introduced.

### Implementation ownership

Use one Go implementation shared by thin CLI and Action adapters. Keep three concepts distinct:

- A declaration records only values the maintainer chose to manage.
- Observed state records normalized values read from GitHub.
- A comparison describes differences for declared fields without owning terminal or review rendering.

The [glossary](glossary.md) records the current declaration and observed-state names and model boundaries.

Configuration storage, checkout/target resolution and GitHub communication serve the shared application operations. The CLI handles command selection, human-readable output and process exit. The composite retains necessary bootstrap and Actions wiring, while the verified Go executable owns the shared operations, metadata and typed plan/Summary processing. These are refactoring responsibilities, not a requirement for generic interfaces, registries or a package per concept.

Extend useful existing structure when a real setting is requested; no new setting or future resource implementation belongs to this series. Current interfaces may remain where useful, without compatibility shims for retired wrappers, old internal interfaces, migration modes or historical CLI/Action combinations. Old tests and fixtures do not create a product obligation merely by existing. Retain tests at the owner of a current requirement and remove coverage belonging exclusively to deleted behavior.

### Deliberate behavior simplifications

Current source implements both deliberate simplifications; the [command reference](commands.md) describes their exact behavior:

- [Empty declarations](https://github.com/daiksud/gh-settings/issues/149): after validating an empty declaration, both CLI plan and apply succeed locally before target resolution, authentication or API construction. False and empty strings remain managed values. The Action's checkout metadata and explicit-token audit requirements remain separate.
- [Origin support](https://github.com/daiksud/gh-settings/issues/158): a checkout-scoped origin must directly identify GitHub.com through HTTPS, SCP-style `git@github.com:owner/repo.git` or full SSH. Rewrite-only aliases and username-less SCP are rejected; transport rewrites cannot change a directly configured repository identity. No fallback mode is provided.

### Publication and operations

Published tags and assets remain immutable; removing source support does not rewrite those releases. Current source need not emulate historical Action/CLI combinations. Candidate-source tests and published consumer tests have different subjects and must say which they verify.

Refactoring does not publish a release or adopt an unreleased candidate. Existing active released Action pins, Environment `gh-settings`, App `answer26`, credential bindings and approval policy remain unchanged. Monitored source or documentation changes can trigger read-only plan/prepare and queue an approval-gated apply; they do not authorize approving it. Publication, coordinated consumer pin/verifier changes and live apply require their separate authorization.

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
- [Reject explicit null for the first five settings](adr/0019-reject-explicit-null.md)
- [Group repository fields within settings](adr/0020-group-settings-by-resource.md)
- [Require a configuration schema version](adr/0021-require-schema-version.md)
- [Gate Actions apply with owner approval](adr/0022-owner-approved-actions.md)
- [Expose a bounded public composite Action](adr/0023-public-composite-action.md)

Go is the agreed implementation language, and `github.com/cli/go-gh/v2` is selected for API access and GitHub CLI authentication integration. Prefer the Go standard library where sufficient; no command framework is mandated. The YAML organization's `go.yaml.in/yaml/v4` is selected for configuration reading and writing in [ADR 0009](adr/0009-yaml-v4-library.md), accepting release-candidate risk; it is not a Go team-maintained library. The [release design](releases.md) records separate distribution decisions and their ADRs.

## Future direction

The [public Action](adr/0023-public-composite-action.md) and owner-approved workflows already exist. The simplification contract above guides their source refactoring without extending supported settings. Labels, rulesets, Pages, and Actions settings remain outside current scope; this section does not commit to implementing all of them.

The following label semantics are agreed for a future extension of scope:

- Omit `labels`: the collection is unmanaged.
- Provide a list: membership is authoritative; delete remote labels not declared.
- Provide `labels: []`: delete all remote labels.
- On retained labels, omitted fields such as `description` preserve current values.
- Label names identify labels; ambiguous matches are errors. There is no rename inference, and explicit rename support is deferred.

Deleting a label loses its assignments. That consequence must be considered before future label support is applied.

Names also identify outer rulesets, ambiguous matches are errors, and there is no rename inference. Applying label-like membership and nested collection behavior to rulesets was proposed but **not confirmed**. Ruleset deletion semantics and inner rule identity remain open questions.

## Unresolved decisions

Null handling for future fields, new output options, composition across multiple update endpoints, new-object required fields and inner rules identity remain unresolved. The [current configuration contract](settings.md#configuration-semantics) does not settle those future choices or extend supported fields. Current authentication belongs to the [command](commands.md) and [caller](github-actions.md) contracts; development tools are pinned in the [development guide](development.md). The [release page](releases.md) records publication procedures and limits.

[^aggressive-v0-plan]: Issue #133 records the retained simplification contract and the ordered source-refactoring scope. Publication and operational adoption are excluded from that series.
