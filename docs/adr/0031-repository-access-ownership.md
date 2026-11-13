---
type: ADR
title: Separate direct repository grants from effective access
description: Define collaborator and team ownership, invitation states and revocation gates.
---

## Status

Historical organization-focused design for #355. The current [personal-public policy](../settings.md#feature-adoption-scope) adopts personal-repository collaborators and invitations; the earlier not-planned decision for #356 no longer defines that boundary. Organization and team management, including #357, remain excluded. This ADR's organization role model and collection-removal convention are not the adopted personal-repository contract; use the [current access implementation boundary](../features/repository-access.feature.md). Current source implements the separate [personal collaborator contract](../settings.md#personal-collaborators); determine packaged support from the selected immutable release. No live grants, invitations, revocations or credential changes are authorized by this design.

## Context

Effective access can combine a direct grant, organization base permissions, ownership and team inheritance. Exporting the highest effective role as a direct grant changes ownership. Treating a successful invitation request as accepted access hides asynchronous state. Removing access can also delete private forks and remove the operator's ability to finish or recover.

The [access specification](../features/repository-access.feature.md) records the official REST/GraphQL evidence. In particular, REST collaborator role summaries are effective roles; GraphQL permission sources offer a candidate direct-grant observation with per-source role names, while team permission checks include inherited access. Neither a successful endpoint nor its required-permissions header proves actual token grants or completeness of unrelated data.

## Decision

Reserve separate collaborator and team collections for organization-owned repositories. Each opted-in collection will own only its completely observed direct repository grants; collaborators additionally own repository-specific pending invitations to positively identified users. Inherited grants, organization ownership/base roles, enterprise access and team membership remain outside ownership. Do not mutate another resource to make desired effective access possible.

An omitted collection is unmanaged; an explicit empty collection requests removal of all owned entries subject to the same safety gates as a nonempty list. Omitted role preserves an existing direct grant or pending invitation, but cannot supply a role for a new grant. Null does not mean removal. Actor identity includes an expected immutable ID and a human-readable login or organization/team slug; stale or conflicting identity fails before mutation.

Keep pending invitations distinct from active grants. Reusing a matching invitation makes no new request, yet must still report pending acceptance. Never accept on another user's behalf or automatically reissue a vanished/expired invitation. Require a separate explicit creation intent for operations that can send a new invitation; the implementation must settle that syntax and notification contract before enabling creation.

Use complete fresh observations and deterministic nontransactional reconciliation. Preserve a verified existing administrative recovery path; do not rely on a newly invited or newly promoted actor to justify destructive changes in the same run. Apply all safe grants/updates before reductions, verify each, then recheck identity/access before each revoke. Reject requests that would remove the execution actor's management path or all verified human administrators. If the actor, inherited paths, role semantics or recovery path cannot be established, fail without access writes. No hidden privilege escalation, retry, fallback endpoint or rollback is introduced.

## Consequences

The historical organization-specific proposal would require a separate scope decision and reviewed implementation, with unsupported cases rejected explicitly. Its direct collaborator provenance, custom-role identity, complete team direct-versus-inherited evidence, invitation creation intent and operator/recovery identity remain unresolved. Personal-repository implementation must establish its own applicable contract rather than inherit organization-only requirements. Partial/null/denied observations never become empty grants or a no-difference result. A future custom role cannot be reduced to its base role, and changing a parent team cannot be hidden as a child-team correction.

The required-permissions precondition applies to this design without certifying actual grants, direct-source completeness or recovery identity. Endpoint-specific absence semantics would need an adoption decision: an authorization-related 404 rationale is not an extra permission-discrimination requirement, and another endpoint's 404 interpretation cannot establish absent direct access. Explicit denials, missing provenance/roles, incomplete pagination, timeouts and mismatching readback remain failures. This reconciliation neither resolves those observation gates nor reopens #356/#357.

The specification defines proposed ownership, observable outcomes, ordering and acceptance cases. It is not a new supported YAML schema. Future code must deliver validate/export/plan/apply, CLI/Action presentation, relevant user documentation and behavior/real-binary fixtures together, one implementation issue at a time. Fixture evidence cannot establish live privileges, notification delivery, policy applicability or fork recovery. No API mutations or operational configuration changes accompany this design.
