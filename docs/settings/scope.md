---
type: Reference
title: "Support scope"
description: "Support scope declaration, ownership and API contracts."
sources:
  - id: ui-inventory
    resource: ../contributing/ui-repository-mapping.md
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Scope and status

Current source reads and exports UI-oriented YAML in `.github/settings.yml` at the Git checkout root. That checkout's directly configured GitHub.com `origin` selects the repository. Public configuration follows pages and sections, using English Web UI terms and meaningful control keys in snake_case; REST and GraphQL names are translated internally.[^ui-inventory]

```yaml
version: daiksud/gh-settings@v0
repository:
  about:
    description: My project
    website: https://example.com
    topics: [github-cli, settings-as-code]
  labels:
    - name: bug
      color: '#d73a4a'
      description: Something is not working
  settings:
    general:
      features: {issues: true}
```

All existing and future gh-settings specifications assume the caller already has the required permissions. Endpoint-specific documented 404 semantics apply under that precondition; explicit denials, malformed/incomplete responses, timeouts, policy enforcement and readback mismatches remain failures. This product precondition does not establish a live credential's grants or authorize access changes.

The exact string `version: daiksud/gh-settings@v0` is required. Superseded envelopes, navigation-category wrappers, scalar permission sections and API-shaped aliases are rejected. There is no compatibility decoder, migration or legacy-format fallback.

This reference describes current source, not the contract of every published CLI or pinned Action. Read the [published package boundary](../github-actions.md#published-package-and-consumer-pins) before adoption. Current-source output uses UI names and field-specific normalization warnings. Out-of-UI Actions execution/workflow permission strings, public-fork approval policies, Security status strings, creation policies, complete merge/squash message pairs, complete Pages modes and legacy folders, Ruleset enforcement/bypass-mode/scanning Alerts/scanning Security alerts strings, Required approvals integers, Allowed merge methods lists and Environment deployment-mode pairs/ref-type strings use the explicit observation policies below; unsupported identities and malformed or incomplete observations still fail. Observed fallbacks never populate omitted desired fields. CLI and detailed Action Summary use UI paths and choices for exact and normalized decisions. Same-version private reports retain native identities and are not apply input.

## Feature adoption scope

The [v1.0.0 goal (#668)](https://github.com/daiksud/gh-settings/issues/668)
targets **personally owned public repositories**. A setting is eligible when it
is configurable in that target's GitHub Web UI. Account plan, Copilot subscription,
feature execution entitlement and Preview status are separate facts; none is a
blanket exclusion. A writable API-only field does not create a target UI control.

Already-archived repositories are outside v1 support, including the ordinary
`export` → `plan` → `apply` contract through either CLI or public Action. Use an
active repository for that workflow. This is a product-scope decision, not a claim
that GitHub APIs cannot read or operate on archived repositories. Existing
readable observations, defensive guards and limited read-only matching paths do
not establish supported management of an already-archived target. The explicit
[archive/unarchive transition](general.md#repository-archive-lifecycle) is a separate
contract; its declaration, sequencing and failure behavior are unchanged.

Autolink references are outside v1 support under the confirmed Free-plan scope.
The [official UI guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/configuring-autolinks-to-reference-external-resources)
lists Pro, Team and Enterprise availability, without Free. The existing v0
[Autolinks configuration and command contract](legacy-autolinks.md#autolink-references) remains
implemented. API availability does not establish plan entitlement, and this
specific exclusion does not reclassify other setting families.

Classic branch protection is outside v1 support by product choice. Use
[Rulesets](rulesets.md#rulesets), the only supported branch-protection mechanism in v1.
This is not a claim of API absence or plan unavailability. Existing v0 classic
implementation, accepted schema and export/reconciliation contracts remain.
The scope decision does not migrate, replace or remove live protection; omitting
a classic declaration leaves those rules unmanaged.

Temporary interaction restrictions are an explicit product exception. They expire
and automatically restore normal access, so they are outside the managed settings
scope despite their known repository read/set/remove APIs. gh-settings provides
no declaration, export, plan or apply support for these finite restrictions.
Persistent pull-request limits and their Exemption list have separate ownership.

Eligibility, technical feasibility and implementation are separate dimensions.
Complete YAML/export/plan/apply support requires the applicable official API
observation and update paths. A read endpoint alone does not prove update support,
and a write endpoint alone does not prove export support. An unconfirmed API
contract remains unconfirmed, rather than being labeled unavailable. The
[coverage ledger](../contributing/settings-coverage.md) records these distinctions
and the limits of fixture and live evidence.

Personal-repository collaborators and invitations are supported through the
[identity and invitation contract](collaborators.md#personal-collaborators). Existing Environment
protection App selection follows the [custom protection contract](environments.md#custom-protection-apps).
Additional configurable custom Ruleset rules remain adopted future work when
supported by the target UI and applicable APIs. Classic controls are outside
v1 scope, including unimplemented controls. Organization-owned repositories and organization
settings, repository rename/visibility/transfer/deletion, authoring GitHub feature
configuration files and secret values are excluded. This file-authoring boundary
does not exclude gh-settings' own declaration file. Existing App selection does
not authorize installing an App or increasing its grants.

This supersedes the Free-only policy recorded on 2026-11-08 and blanket Preview
and custom-rule exclusions. Historical investigations remain evidence with their
original dates, not current scope rules. The contracts below describe **current
source**: newly eligible work does not make an unimplemented key valid or change
current export, reconciliation, preservation or pruning behavior. The v1 goal is
not a claim that current source or a pinned published package is complete.

## Unsupported settings

Only the listed UI controls are accepted by current source. Classic branch protection is outside v1 support by product choice; its existing partial v0 implementation remains documented above, while remaining classic controls are unimplemented and outside v1 scope. Other Actions/security gaps are tracked separately from API limitations in the [coverage ledger](../contributing/settings-coverage.md). Organization settings and team access, organization-only custom properties, inherited organization Rulesets and private-only controls are outside the personal-public target. Repository rename/visibility/transfer/deletion, feature configuration-file authoring, secret values and finite temporary interaction restrictions are explicit exclusions. Pages visibility and Environment rename remain unsupported without asserting that a usable target control/API is absent. See the [architecture](../contributing/architecture.md) and [access implementation boundary](../features/repository-access.feature.md); future work is not valid configuration today.

[^ui-inventory]: Read-only UI wording inventory inspected 2026-10-28; internal conversion details and the other family inventories are linked there. This reference changes the public YAML composition, not the recorded UI inspection history.
