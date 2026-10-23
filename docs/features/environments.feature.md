# Feature: Environment ownership and deployment protection

Status: design for #351 and [ADR 0029](../adr/0029-environment-ownership-and-protection.md). All syntax below is reserved, unimplemented and rejected by current source and published packages. This document defines implementation acceptance criteria, not permission to modify any live Environment.

## Evidence and API boundaries

Reviewed against GitHub.com documentation on October 23, 2026:

| Subject | Primary reference and consequence |
| --- | --- |
| Environment list/read and create-or-update/delete | [REST Environments](https://docs.github.com/en/rest/deployments/environments): Actions read for reads, Administration write for mutations; public reads may be unauthenticated |
| Name, deletion and implicit creation | [Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments): case-insensitive names, maximum 255 characters; deletion loses secrets/protection and fails waiting jobs; a workflow reference can recreate an unprotected environment |
| Effective protection and entitlement | [Deployments and environments](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments): reviewer/timer restrictions vary by plan; protected-branches mode allows all branches when no branch protection exists |
| Separate branch/tag patterns | [REST deployment branch policies](https://docs.github.com/en/rest/deployments/branch-policies): custom mode is a prerequisite; pattern wildcards do not match slash; creation can return 303 for a duplicate |
| Reviewer identity and access | [Users](https://docs.github.com/en/rest/users/users#get-a-user), [Teams](https://docs.github.com/en/rest/teams/teams#get-a-team-by-name), [repository permissions](https://docs.github.com/en/rest/collaborators/collaborators#get-repository-permissions-for-a-user), [team repository permissions](https://docs.github.com/en/rest/teams/teams#check-team-permissions-for-a-repository) |
| Alternative protection update shape | [GraphQL deployments](https://docs.github.com/en/graphql/reference/deployments#updateenvironmentinput): documents reviewer IDs, timer and self-review inputs; this is not proof of REST omission behavior |
| Ambiguous 404 and token requirements | [REST troubleshooting](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api#404-not-found-for-an-existing-resource): do not infer exact token grants from an actor role, unrelated read or required-permissions header |

The Environment REST request documents `wait_timer` (0–43200 minutes), `prevent_self_review`, up to six typed reviewer IDs, and nullable `deployment_branch_policy`. A non-null policy requires opposite `protected_branches` and `custom_branch_policies` booleans. Null allows all refs. These facts establish request shape, not the safety of omitted fields or entitlement to every returned protection.

`can_admins_bypass` is a real UI concept, but neither the reviewed REST create-or-update parameters nor GraphQL `UpdateEnvironmentInput` documents that setter. Do not send an undocumented key, assume a UI default is observed state, or simulate the setting through workflow approval. Reserve it as unsupported pending documented read/write semantics and preservation evidence.

Public repository resources are generally available on current plans; private/internal Environments require an eligible paid plan. Timer and required-reviewer availability is narrower for Free/Pro/Team. A missing rule can therefore mean unavailable rather than disabled. Do not change a subscription, repository visibility, organization policy or credentials to make a declaration applicable. API acceptance alone does not prove that a deployment is protected.

## Reserved declaration and ownership

The proposed first implementation selects existing environments by name:

```yaml
schemaVersion: 1
settings:
  environments:
    - name: production
      wait_timer: 30
      prevent_self_review: true
      reviewers:
        - type: User
          id: 123
          login: release-owner
        - type: Team
          id: 456
          organization: example
          slug: release-team
      deployment_branch_policy:
        protected_branches: true
        custom_branch_policies: false
```

The example is design syntax only. No Environment support is available merely because this design is accepted.

| Declaration | Ownership and outcome |
| --- | --- |
| Omitted `environments` or `environments: []` | No selected environments; no Environment reads/writes, deletion or protection removal |
| Null, mapping instead of sequence, duplicate case-insensitive names | Invalid locally |
| Omitted name from a nonempty sequence | Existing environment remains unmanaged and is never deleted |
| Entry with only `name` | Select no protection fields; no create or existence assertion and no Environment API request for that entry |
| Different name | Select a different existing resource; never rename, copy secrets or delete the old resource |
| Omitted protection member | Unmanaged; preserve it, including any required write companions |
| `wait_timer: 0` | Explicitly remove the wait timer; positive integers add a delay; null, fractions and out-of-range values invalid |
| `prevent_self_review: false` | Explicitly allow self-review; false is not omission; null invalid |
| Omitted `reviewers` | Preserve membership and identities |
| `reviewers: []` | Explicitly remove required-reviewer protection; no Environment deletion or invitation |
| Nonempty `reviewers` | Own the complete typed reviewer set; require at most six unique `(type, id)` identities and verified references |
| `reviewers: null` | Invalid; choose one collection-clearing representation even though REST also permits null |
| `deployment_branch_policy: null` | Explicitly remove branch/tag restrictions; only this policy field accepts null |
| Non-null deployment policy | Complete pair of opposite booleans; partial/empty objects invalid; custom selection is gated on #353 |
| `can_admins_bypass`, lifecycle state, delete/rename markers, unknown keys | Unsupported, rejected before API work |

Reject empty, control-containing and surrounding-whitespace names. Preserve spelling for display; compare names case-insensitively and URL-encode a name as one path component, including slash and percent. Do not silently trim, split paths or follow redirects. Duplicate detection must use the same equivalence as identity checks. Before implementation, choose and test a documented case-folding policy for non-ASCII names; do not assume ASCII lowercasing proves GitHub equivalence. An initially restricted ASCII subset must be explicit in validation and export errors.

Self-review protection is active only with a nonempty reviewer set. Require explicit `prevent_self_review: false` alongside `reviewers: []`; reject true with an empty final set. When only self-review is declared, positively observe a nonempty retained reviewer set. A later reactivation must not infer a missing inactive flag. Clearing protection is reported with its consequence rather than normalized to an empty environment.

## Reviewer identity

Require exact tagged shapes: User has positive integer `id` and nonempty `login`; Team has positive integer `id`, `organization` and `slug`. Reject mixed keys, display-name-only references, unknown types, duplicate IDs within a type and conflicting references. A user and a team with the same numeric ID remain different identities.

Resolve User through the user-by-login endpoint and Team through the organization/slug endpoint. Require returned type, numeric ID and canonical identity to agree with the declaration; for teams verify organization ownership and require the repository owner's organization. Obtain applicable repository read-access evidence using the corresponding user/team permission read. A renamed identity, a new principal reusing an old name, an unresolved team or an inaccessible permission read fails before a protection write. Never guess by display name, invite a principal, grant repository access or issue a token.

User resolution does not require private user-profile scopes. Team resolution can require organization Members read and the endpoint's membership conditions; document those extra reads before implementation. Repository permission checks do not prove Environment write grants. Re-resolve managed principals before writing, pin IDs through readback, and compare set identity independent of presentation order. Export includes verified canonical references and IDs; inability to reconstruct a reference fails selected export instead of dropping a reviewer.

## Complete observation and export

The first implementation must use explicit acquired observations for repository identity, existing Environment numeric ID/name, and requested protection state. An unacquired observation is unknown. Reject missing/null containers, duplicate rule types/identities, malformed booleans/integers, unknown relevant rules, incomplete reviewers and partial pagination. Every selected environment requires an individual complete observation; a list entry alone is not proof of all protection values.

Do not convert an absent `wait_timer` or `required_reviewers` rule to zero/empty/false until endpoint-specific evidence distinguishes disabled protection from hidden or inapplicable protection. Similarly, missing policy is not explicit null. Store active/inactive states separately where the adopted contract supports them. If an endpoint returns an unknown/custom protection that a write could replace, block that update until preservation is established; do not discard it to complete a payload.

A reserved `export --environments` would select all existing environments, following every page and checking consistent IDs/names/counts before serialization. Ordinary export gains no Environment requests. Export only adopted configurable fields with canonical reviewer references; exclude secrets, variables, URLs, timestamps and deployment history. If any selected environment cannot be completely represented, fail without replacing the output. Do not silently omit unsupported protection and present the result as complete. An empty complete list exports `environments: []`, which remains non-authoritative and cannot delete environments created later.

Plan remains read-only. Use closed resource-specific comparisons with native integer, boolean, collection and policy-null types. Count owned leaf/protection changes consistently; stable environment order and typed reviewer ordering must not cause false drift. No lifecycle difference is synthesized. Credential-free Summary must reconstruct the same schema and reject forged missing-as-null or unknown-as-false values; private reporting must hide names, reviewers, patterns and server diagnostics.

## Apply, ordering and failure

Before each environment update, freshly verify repository identity/archive state, environment ID/name, adopted protections, retained companions and managed reviewer resolution. Skip a complete fresh match. Validate the whole local declaration first, but do not imply that later remote preflight prevents earlier resource writes. Sequence explicitly requested unarchive first, ordinary resources then Environment updates, and final archive last. Do not temporarily unarchive or lower protection implicitly.

Choose the exact documented update adapter in #352, with tests and evidence for its omission behavior. Send declared fields only when omission is proven to preserve the other fields. If that API requires companions, merge only completely observed companions and verify them after the write. Do not fabricate defaults. Failure to establish preservation of administrator bypass, custom protection or inactive fields blocks that affected update; an alternate documented API needs its own contract, not an automatic fallback.

After a response, perform bounded fresh readback of identity, requested values and affected retained protection. Return nonzero for mismatches, denied or incomplete readback, redirects, timeouts and server rejection. Record prerequisite failures as no Environment write attempted; classify write/readback failures as potentially applied. Retain known prior writes. No hidden retry, rollback, deployment approval, build trigger, secret/variable read or permission change is allowed.

REST PUT is create-or-update by name: fresh reads reduce but do not eliminate a concurrent deletion/recreation race. The existing-only contract must not claim an atomic no-create guarantee. Before adopting that adapter, explicitly accept and document this exposure or select a documented identity-addressed update with verified applicability. A 404 never triggers a create fallback. Positive readback of a different Environment ID is an uncertain failure, not successful reconciliation.

### Branch and tag policy boundary (#353)

Keep pattern ownership separate from the Environment protection document. Reserve `deployment_branch_policies` as a sequence of exact `{type: branch|tag, name: pattern}` entries: omission preserves the active set, `[]` owns an empty set, and null is invalid. It is valid only with a positively observed custom mode or an explicit transition to custom. Same-text branch and tag patterns are different identities; pattern strings are literal glob declarations, not branch creation requests. Policy IDs are observed operation identities, not portable YAML keys.

Require complete pagination and an explicitly observed type for every policy; the current REST examples omit type, so they do not justify assuming branch. Do not adopt #353 until a documented, tested typed observation is available. A 303 is not permission to follow a redirect or assume the requested typed policy exists; stop with the actual result and require fresh reconciliation on a later invocation.

For an already custom environment, reconcile removals before additions so obsolete permissions are removed before new permissions are granted; a failure may temporarily restrict deployment and must report the remaining state. Pattern rename/type change is remove/add by identity, with no heuristic rename. Verify each operation and the final typed set. Do not promise atomic replacement or uninterrupted deployment.

For entry into custom mode, require an explicit complete pattern set and evidence for the intermediate state's enforcement before enabling the mode; the prerequisite prevents simply prepopulating inactive patterns. Until that transition behavior is established, reject it before mutation. Leaving custom mode requires explicit policy intent and must explain that patterns become inactive; do not delete them as an undeclared cleanup or assume they will survive reactivation. Return to custom requires a newly explicit complete set and fresh observation. These mode transitions remain gated independently of updates that can be shown to preserve an already active mode.

## Lifecycle and implementation gates

No creation, deletion or rename syntax is adopted. Deletion would destroy secrets/protection and fail waiting jobs; workflow references may recreate an unprotected environment. An empty managed collection is never evidence of that destructive intent. A future decision needs separate explicit intent, immutable identity binding, exposure/entitlement checks, absence observation, recovery limits and handling of concurrent recreation. No endpoint mutation may be used as an existence or permission probe.

| Candidate | Required evidence before implementation adoption |
| --- | --- |
| #352 existing protection | Complete enabled/disabled/inactive observations; entitlement/hidden-read distinction; verified reviewer identity/access; exact update omission/companion behavior; bypass/custom-rule preservation; existing-resource race decision |
| Administrator bypass | Documented setter and complete observable state; cannot be adopted from a UI label or example field |
| #352 lifecycle | Separate accepted lifecycle decision; not enabled by closing #351 |
| #353 active pattern reconciliation | Typed complete read, identity-safe CRUD, explicit intermediate protection/partial-failure behavior |
| #353 custom-mode transitions | Verified inactive/empty-mode behavior and preservation rules; no inferred allow/deny state |

These are operation-specific gates. Keep unresolved acceptance criteria on #352/#353 rather than closing an entire issue after a partial slice. Continue independent design, model/report investigation or other issue work without accepting reserved keys prematurely. Live Environment/protection/credential operations, including #380–#382, require their separate approvals and are not authorized by this product design.

## Acceptance plan and evidence limits

No behavior tests are claimed for this documentation-only design. An adopted implementation must cover:

| Layer | Required scenarios |
| --- | --- |
| Model/codec | Omitted/empty selection; names-only entries; duplicate case identities; explicit zero/false/reviewer-empty/policy-null; invalid null and partial pair; typed pinned reviewers; unsupported lifecycle/bypass keys |
| API observation | Complete enabled and documented disabled forms; hidden/inapplicable/missing protections; malformed rules; all-page enumeration; typed same-name patterns; 401/403/404/redirect; no guessed defaults |
| Reviewer resolution | Same display name across User/Team, changed ID under reused name, renamed login/slug, foreign organization, no repository access and denied member reads; no invitation or access mutation |
| Apply | Fresh match no write; declared-only or proven companion body; protection reduction; preserved bypass/custom rules; identity replacement; server refusal; successful response with ignored field; failed readback and earlier writes; archive ordering |
| Pattern transitions | Branch/tag identity, explicit empty set, remove-before-add partial failure, 303 without redirect, gated custom entry/exit/reactivation, no branch or deployment creation |
| Process/reporting | Actual binary with local fixtures/dummy credentials; export-to-zero-plan; apply/idempotence; failed export preserves file; strict native CLI/Action report round-trip; credential-free Summary; private redaction |

Documentation and fixtures establish intended client behavior, not live entitlement, exact token grants, omitted-field preservation on GitHub, deployment enforcement or successful mutation. Any live evidence required to resolve a gate must be separately authorized for a specific disposable target; do not use this repository's protected Environment as a probe. No release, live Environment modification, secret inspection, token issuance or workflow dispatch is part of #351.
