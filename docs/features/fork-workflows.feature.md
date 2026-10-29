---
type: Feature
title: Fork PR workflow policies
description: Explicit contributor approval and private fork execution and credential choices.
---

## Status and scope

Implemented in current source, **supported**, for [#342](https://github.com/daiksud/gh-settings/issues/342). The [configuration contract](../settings.md#fork-pr-workflow-policies) owns values, usage and security consequences. No release, live policy, workflow dispatch or grant change is part of this implementation.

Public contributor approval is `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`. It accepts `require_approval_for_first_time_contributors_who_are_new_to_github`, `require_approval_for_first_time_contributors` or `require_approval_for_all_external_contributors`. Private controls are under `repository.settings.actions.general.fork_pull_request_workflows`: `run_workflows_from_fork_pull_requests`, `send_write_tokens_to_workflows_from_fork_pull_requests`, `send_secrets_and_variables_to_workflows_from_fork_pull_requests` and `require_approval_for_fork_pull_request_workflows`. They translate internally to existing API values; old aliases are rejected. `pull_request_target` is outside the public contributor approval gate. Internal repositories are unsupported by this adopted scope; 404/403 never mean disabled. These are distinct from execution enablement, default workflow token permissions and approving PR reviews. Repository settings cannot override organization policy.

## Declaration and complete observation

Omission/empty mappings are unmanaged; false is preserved and null/wrong/unknown/duplicate values fail locally. Each scoped adapter verifies current exact repository visibility before its dedicated GET and acquires complete typed values. Missing/null/type/member-case errors discard the observation. Complete public approval strings outside UI choices retain raw provenance and use the named default; private booleans remain strict. Ordinary export does not add requests; explicit `--fork-approval` and `--private-fork-workflows` retain separate scope and compose with other choices. Inapplicable or incomplete selected export preserves the destination. Both fork choices cannot succeed for a fixed target because their scopes differ. Export omits private companions hidden behind a false run-workflows checkbox only after complete raw acquisition, without replaying latent values as intent. Exact supported unchanged export has zero differences; a normalized export retains an actionable raw-versus-fallback distinction.

Matching private partial intent needs no required run parameter because it sends no PUT. A changed private policy requires explicitly declared `run_workflows_from_fork_pull_requests`; no observed value is synthesized. Enabling previously disabled execution requires all four values, exposing latent credential/approval choices as intent. Disabling does not invent resets of stored companions.

Test mapping: [model acquisition/prerequisites](../../internal/settings/fork_workflows_test.go), [Actions UI codec](../../internal/workspace/ui_actions_test.go), [complete applicable API reads](../../internal/github/fork_workflows_test.go), [independent opt-ins](../../internal/cli/fork_workflows_test.go).

## Reconciliation, failures and reports

Plan is read-only and discards earlier comparisons on a later failure. Direct apply follows retention, with contributor approval then private fork before security, and rereads complete applicability/state before a changed PUT. Fresh exact raw matching state skips writing; effective fallback equality does not. Payloads contain only declared values; private readback verifies requested plus unowned companions without sending unowned keys. Refused, ignored or redirected writes and malformed/drifted readback fail without retry or rollback. Pre-write failures retain confirmed no-write certainty; successful reads and skips do not invent prior writes. A later failure retains earlier actual changes, including when the later family is inapplicable.

The private native comparison registry retains existing identifiers, enums, boolean values and bounds. Normalized public approval decisions use the UI path/choices with escaped raw warnings and a write-required note, including fallback equality. Status-only Summary suppresses these values. Exact observations and other CLI/credential-free Summary decisions still use native component paths/tokens pending #481; public UI choice names are not serialized as a new report protocol here. Unknown paths/types/enums, unavailable before values and no-ops are rejected. Reports remain ephemeral presentation data.

Test mapping: [narrow PUT/fresh prerequisites/readback and certainty](../../internal/github/fork_workflows_test.go), [app isolation and earlier progress](../../internal/app/fork_workflows_test.go), [closed native comparisons](../../internal/settings/fork_workflows_test.go), [Summary](../../internal/actions/fork_workflows_test.go), [built-product export/plan/apply/Summary](../../tests/acceptance/fork_workflows_test.go). Process fixtures cover public/private scope, selected export→zero plan→edit→apply→zero plan, preserved omitted credentials, missing required run flag, file preservation, complete production GET/PUT redirect rejection, refusal/ignored-write errors and earlier public-policy progress before private applicability failure.

All mutations use local fixtures and dummy credentials. A harmless GET on the actual public repository returned contributor policy `first_time_contributors`. No actual private-policy GET, PUT, credential content read, workflow run, live grant or policy update was performed. Fixtures establish modeled optional-key preservation and client behavior; they do not prove live PUT compatibility, approval enforcement or safety of arbitrary workflow code. The [official API](https://docs.github.com/en/rest/actions/permissions#set-private-repo-fork-pr-workflow-settings-for-a-repository) supplies the required/optional request keys; the [repository guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository) supplies the visibility and credential boundaries.

## Feature: Retain raw public-fork approval observations

### Rule: Complete successful policy strings outside UI choices use the named default

#### Scenario: A public repository returns a future or empty approval policy

- Given public visibility and a complete successful policy GET are acquired
- When the correctly typed policy string is outside the three adopted UI choices
- Then the effective choice is `require_approval_for_first_time_contributors`
- And the native effective value is `first_time_contributors`
- And one record retains the exact raw string, YAML field and named fallback
- And empty and whitespace strings remain exact acquired values
- But unsupported visibility, missing/null members, wrong types and failed reads remain errors

#### Scenario: An unacquired observation stays unacquired

- Given no complete public-fork approval observation was acquired
- When the observation is queried
- Then neither an effective value nor raw evidence is available
- And omission never becomes a request for the default

### Rule: Normalized equality remains a managed write decision

#### Scenario: A declared approval choice equals or differs from the fallback

- Given the public-fork approval policy required a named fallback
- When the declaration selects any of the three adopted UI choices
- Then plan retains one actionable policy decision with original provenance
- And fallback equality does not establish independently observed zero drift
- And an omitted approval policy remains unmanaged without a policy decision
- But malformed normalization cannot produce a partial comparison

### Rule: Apply warnings and verification follow complete managed reads

#### Scenario: A normalized raw policy survives a requested write

- Given a managed public-fork approval declaration and complete unknown raw policy
- When apply observes it, rereads its prerequisites and sends only the declared policy
- Then each successful normalized read emits its original projection warning
- And fresh raw readback must match the requested native policy
- And an unchanged unknown string fails with possible write effects
- And earlier successful writes remain applied without rollback
- But failed or incomplete acquisition emits no fallback warning and performs no policy write

Test mapping: [pure and HTTP normalization](../../internal/github/normalization_fork_approval_test.go), [closed provenance](../../internal/settings/fork_approval_normalization_test.go), [comparison](../../internal/settings/fork_approval_comparison_test.go), [private reports and presentation](../../internal/actions/normalization_fork_approval_test.go), [scoped export](../../internal/app/normalization_test.go), and [executable commands](../../tests/acceptance/normalization_fork_approval_test.go).
