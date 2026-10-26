---
type: Feature
title: Fork PR workflow policies
description: Explicit contributor approval and private fork execution and credential choices.
---

## Status and scope

Implemented in current source, **supported**, for [#342](https://github.com/daiksud/gh-settings/issues/342). The [configuration contract](../settings.md#fork-pr-workflow-policies) owns values, usage and security consequences. No release, live policy, workflow dispatch or grant change is part of this implementation.

Public contributor approval uses the three exact API enums under `actions.fork_pr_approval_policy`. Private repositories use four independent booleans under `actions.private_fork_workflows`. `pull_request_target` is outside the public contributor approval gate. Internal repositories are unsupported by this adopted scope; 404/403 never mean disabled. These are distinct from execution enablement, default workflow token permissions and approving PR reviews. Repository settings cannot override organization policy.

## Declaration and complete observation

Omission/empty mappings are unmanaged; false is preserved and null/wrong/unknown/duplicate values fail locally. Each scoped adapter verifies current exact repository visibility before its dedicated GET and acquires complete known values. Missing/null/type/enum/case errors discard the observation. Ordinary export does not add requests; explicit `--fork-approval` and `--private-fork-workflows` retain separate scope and compose with other choices. Inapplicable or incomplete selected export preserves the destination. Both fork choices cannot succeed for a fixed target because their scopes differ. Valid unchanged export has zero differences.

Matching private partial intent needs no required run parameter because it sends no PUT. A changed private policy requires explicitly declared `run_workflows_from_fork_pull_requests`; no observed value is synthesized. Enabling previously disabled execution requires all four values, exposing latent credential/approval choices as intent. Disabling does not invent resets of stored companions.

Test mapping: [model acquisition/prerequisites](../../internal/settings/fork_workflows_test.go), [strict YAML and round trip](../../internal/workspace/fork_workflows_test.go), [complete applicable API reads](../../internal/github/fork_workflows_test.go), [independent opt-ins](../../internal/cli/fork_workflows_test.go).

## Reconciliation, failures and reports

Plan is read-only and discards earlier comparisons on a later failure. Direct apply follows retention, with contributor approval then private fork before security, and rereads complete applicability/state before a changed PUT. Fresh matching state skips writing. Payloads contain only declared values; private readback verifies requested plus unowned companions without sending unowned keys. Refused, ignored or redirected writes and malformed/drifted readback fail without retry or rollback. Pre-write failures retain confirmed no-write certainty; successful reads and skips do not invent prior writes. A later failure retains earlier actual changes, including when the later family is inapplicable.

The native Actions comparison registry has fourteen fields including these five; the overall current managed bound is fifty-one. Booleans remain native and enums remain exact strings in CLI, JSON report and credential-free Summary. Unknown paths/types/enums, unavailable before values and no-ops are rejected. Reports remain ephemeral presentation data.

Test mapping: [narrow PUT/fresh prerequisites/readback and certainty](../../internal/github/fork_workflows_test.go), [app isolation and earlier progress](../../internal/app/fork_workflows_test.go), [closed native comparisons](../../internal/settings/fork_workflows_test.go), [Summary](../../internal/actions/fork_workflows_test.go), [built-product export/plan/apply/Summary](../../tests/acceptance/fork_workflows_test.go). Process fixtures cover public/private scope, selected export→zero plan→edit→apply→zero plan, preserved omitted credentials, missing required run flag, file preservation, complete production GET/PUT redirect rejection, refusal/ignored-write errors and earlier public-policy progress before private applicability failure.

All mutations use local fixtures and dummy credentials. A harmless GET on the actual public repository returned contributor policy `first_time_contributors`. No actual private-policy GET, PUT, credential content read, workflow run, live grant or policy update was performed. Fixtures establish modeled optional-key preservation and client behavior; they do not prove live PUT compatibility, approval enforcement or safety of arbitrary workflow code. The [official API](https://docs.github.com/en/rest/actions/permissions#set-private-repo-fork-pr-workflow-settings-for-a-repository) supplies the required/optional request keys; the [repository guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository) supplies the visibility and credential boundaries.
