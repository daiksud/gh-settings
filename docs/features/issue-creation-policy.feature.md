---
type: Feature
title: Issue creation policy
description: Declarative Issue creation eligibility without changing issue availability.
---

## Contract

Current source supports `settings.repository.issue_creation_policy` with exact values `ALL` and `COLLABORATORS_ONLY`. Omission leaves it unmanaged. Null, empty strings, wrong types, case variants and unknown policies are invalid. Declaring a policy never enables or disables `has_issues`.

A separate narrow authenticated GraphQL query reads this policy only when requested; PR-only and existing repository declarations retain their reads. Export includes the complete observed policy after all resource reads succeed. Missing/null/unsupported data and partial GraphQL errors fail rather than becoming `ALL`, including when issues are disabled. Failed export preserves the old file. Reads and ID lookup disable rename following to reject stale origins.

Apply reuses the repository-ID lookup, selected-only `updateRepository` mutation and target/response verification from PR access. Send the Issue policy only when declared, without synthesizing `hasIssuesEnabled` or PR settings. When both PR and Issue controls are declared, use one lookup and one mutation. Existing REST fields, including explicitly declared `has_issues`, run first, before GraphQL and later resources. Stop on first error and report earlier successful writes; there is no rollback, retry or saved-plan execution.

Verification uses isolated HTTP/files/Git fixtures with dummy credentials. A read-only disabled-Issue observation does not prove live mutation compatibility, permissions or publication. Caller grants are unchanged and never expanded automatically. No live settings or release changes are included.

## Acceptance

Test both policies, independent omission, disabled-Issue observations, invalid local types/enums, strict missing/null/unknown/partial-error reads, selective API access, exact combined and policy-only mutation payloads, target and response failures, redirects and mixed REST/GraphQL partial failure. Exercise export/validate/plan/apply, native Action report/Summary and failure file preservation with the built CLI.

## Existing test mapping

| Boundary | Tests |
| --- | --- |
| Presence and acquired observations | [Issue policy model](../../internal/settings/issue_creation_policy_test.go) |
| YAML types, omission and round trip | [Issue policy codec](../../internal/workspace/issue_creation_policy_test.go) |
| Selective reads, shared exact mutation and response failures | [Issue policy API](../../internal/github/issue_creation_policy_test.go) |
| Native enum report and Summary validation | [Issue policy presentation](../../internal/actions/issue_creation_policy_test.go) |
| Built CLI disabled-Issue lifecycle, exact writes and failure sequencing | [Issue policy acceptance](../../tests/acceptance/issue_creation_policy_test.go) |

The existing PR-access tests continue to check their unchanged selective reads, identity guards and failure semantics through the shared writer. The current-source report limit is fifty-one across the complete adopted resource set. The read-only disabled-Issue sample returns policy `ALL`; fixtures cover explicit edits while availability remains false without asserting live write preservation.
