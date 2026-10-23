# Repository access ownership design

This is the design for #355 and [ADR0031](../adr/0031-repository-access-ownership.md), not implemented behavior. Current source and published packages reject all proposed access keys/options. #356 and #357 require separate adoption PRs, complete observations and the gates below. No live collaborator, invitation, team, membership or permission change is authorized by this document.

## Scope and ownership

Begin with organization-owned repositories and existing identified users/teams. Personal repositories, organization/enterprise membership, owners/base permissions, team creation/rename/deletion, hierarchy changes, bot/service-principal onboarding, deploy keys, App installation grants and credentials are outside this design. Removing a direct grant does not promise removal of all effective access, and never justifies deleting an inherited source.

Reserved shape, **not valid configuration today**:

```yaml
settings:
  access:
    collaborators:
      - login: example-user
        id: 123
        role: read
    teams:
      - organization: example-org
        slug: documentation
        id: 456
        role: read
```

Names are locators and review labels; positive immutable IDs are expected identities. Resolve both before comparison and immediately before mutation. Case-only login differences may canonicalize to the observed login; a renamed login/slug, reused name, wrong organization or different ID fails. Do not follow redirects, retarget a declaration or create a missing actor. Duplicate IDs or equivalent locators are errors. IDs in this example are illustrative, never targets for a live action.

| Declaration/observation | Ownership and result |
| --- | --- |
| Access mapping omitted or `{}` | No access reads or writes |
| One collection omitted | That collection remains unmanaged |
| Explicit collaborator `[]` | Remove proven direct user grants and cancel owned pending invitations; never organization/team/enterprise access; recovery gates still apply |
| Explicit team `[]` | Remove proven direct team grants only; inherited grants stay outside ownership |
| Listed existing direct entry, role omitted | Retain observed direct membership and its complete direct role; do not substitute effective role |
| Listed pending invitation, role omitted | Retain its requested role and report pending acceptance |
| New entry, role omitted | Error before any access mutation; never use an API default |
| Null collection/entry/role, unknown field or empty role | Invalid; removal is collection membership intent, not null |
| Matching effective access without a direct grant | Distinct from a matching direct grant; never invent or silently adopt one |
| Incomplete role/provenance, hidden actor or ambiguous 404 | Error, not absent, inherited or no-difference |

Use explicit, independent future export selection for collaborators and teams; ordinary export must retain its existing access requirements. Opted-in export includes all and only the supported owned entries, with complete direct roles and pinned identity. Pending entries retain pending metadata in presentation and must not be exported as completed grants. The implementation must define an executable invitation-aware declaration before exporting pending invitations; until then a selected collection containing one fails export and preserves the old file. Never silently omit unsupported owned entries from an authoritative export.

## Role and source evidence

The product vocabulary for the initial candidate is `read`, `triage`, `write`, `maintain`, `admin`. Translate explicitly to endpoint-specific values (`read`→`pull`, `write`→`push` for collaborator/team grants; invitation PATCH uses `read`/`write`). Never send an omitted permission and inherit an endpoint default. `triage_plus` and custom roles are separate capabilities and remain gated until every required read/write/invitation path supports exact identity and semantics. Unknown or unsupported existing direct roles block an authoritative operation rather than being downgraded to a base role.

For collaborators, investigate paginated GraphQL `Repository.collaborators` edges with complete, non-null `permissionSources`. A source must include its organization, immutable source identity, permission and `roleName`; the matching Repository source is a candidate direct grant, while Organization, Team and EnterpriseTeam sources remain inherited. The edge's effective permission and REST `role_name` cannot replace the source role. Prove the chosen query is complete for the supported token/organization shape, cross-check actor coverage and pagination, reject partial GraphQL errors and conflicting duplicate sources, and verify custom-role behavior before claiming adoption. The nullable source list or role name is an observation failure, not an empty set or a standard-role fallback.

For teams, a repository permission check includes parent inheritance. A parent graph and an effective permission alone do not prove whether a redundant direct grant exists. User permission sources cannot enumerate grants to empty teams. `TeamRepositoryEdge.permission` likewise does not document direct provenance. #357 needs a documented, complete source for the direct team relation and exact role before authoritative clearing, export or update can ship. If no such observation is available, keep that capability gated; do not infer direct access from stronger/equal/lower comparisons with the parent.

Examples: a user with direct `read` and team `admin` exports direct `read`, and removing that grant leaves team access. A user whose only source is an organization base role is never a collaborator removal candidate. A child team whose parent grants `write` does not become a direct `write` entry merely because its check succeeds. A custom role inheriting `read` remains that custom role; it cannot round-trip as `read`.

## Invitations and notifications

Read every page of repository-specific open invitations with repository identity, invitation ID, invitee ID/login, requested role and any lifecycle fields required by the adopted adapter. Email-only, null/unresolved invitees, duplicate/conflicting invitations, wrong repository and missing role are unsupported observations. Do not use the authenticated user's personal invitation inbox as an authoritative repository collection.

An existing matching invitation is pending and is not resent. A supported role change updates the observed invitation ID and verifies it remains bound to the same repository/invitee; do not revoke/reinvite to work around a role limitation. A no-longer-declared pending entry is cancelled only after all collection and recovery checks. An invitation disappearing between reads may have been accepted, declined, cancelled or expired: reread both grants and invitations and stop that operation if the transition cannot be established. Never interpret disappearance as permission to resend.

New collaborator PUT can either create an invitation or immediately establish a direct grant. Creation therefore needs explicit future intent accepting GitHub's notifications and policy constraints, with a declared role and fresh absence evidence. That intent is an adoption gate, not an accepted YAML flag here. A 201 response reports a verified pending invitation; a 204 still requires direct-grant readback. Neither response proves recipient acceptance. No auto-accept, separate notification API, organization enrollment or repeated invitation on retry is permitted. An ambiguous attempted write stops with uncertain outcome; the next user-invoked run begins by observing both states.

## Preflight, ordering and recovery

All requested collections are validated and completely read before access writes. Bind repository ID/owner, organization ID, actor identities, direct roles, invitations, execution principal and existing administrative recovery paths. Plan performs no writes and renders grants, role changes, revocations, invitation creation/update/cancellation and pending states separately. A pending invitation cannot produce a clean active-access success or a zero-difference assertion that all requested access is effective. Private reporting must suppress actor/role details when selected.

The initial reconciliation sequence is: fresh full preflight; safe additions and role increases; verified in-place non-reducing changes; fresh full observation; role reductions, invitation cancellations and direct-grant removals in stable identity order; final complete readback. Custom roles without a proven privilege ordering cannot be classified as a safe increase and remain gated. Any failed/unverified addition stops before reductions. An unaccepted invitation or a newly established admin path cannot justify removing an existing recovery path in the same invocation.

Reject self-removal/self-demotion and changes that remove a team path needed by the current execution principal. A GitHub App installation token is not the human who requested a run; do not invent a human identity from its bot name. Require an independently verified, retained human administrative recovery path before reductions, and reject unknown principal/access evidence. Do not count a token permission flag as proof of a recoverable human administrator or fabricate an organization owner. If this cannot be established without broader access, keep reduction operations gated rather than issuing a stronger token.

Recheck the target and affected identity/provenance before each write. Recheck management/recovery access before each destructive operation. Concurrent changes fail rather than being reconciled against a stale plan. Identity and state observations are not atomic locks; the name-addressed APIs still have a race window. Define allowed lifecycle composition before implementation; until an access-specific contract is adopted, reject combining access changes with archive transitions. Never automatically unarchive to gain write access.

Report completed mutations, pending invitations, failed pre-write reads and uncertain attempted writes separately. Earlier writes stay applied after a later failure. Do not compensate by recreating grants/invitations or changing organization policy. Reruns re-observe state and skip confirmed matches; they do not consume saved plans or a hidden local ownership database. Removing access may delete private forks; restoring a grant is not a promised recovery of deleted forks or local clones.

## Official contracts and permission boundaries

Checked against GitHub's documented 2026-03-10 REST version and current GraphQL reference; no live access mutation was performed.

| Source | Relevant contract |
| --- | --- |
| [Collaborators REST](https://docs.github.com/en/rest/collaborators/collaborators) | Lists mix grant sources and report highest effective role; `affiliation=direct` is not direct-role evidence. Metadata read and endpoint actor conditions apply to reads; mutations require Administration write. PUT defaults to push, not a safe omitted role; 201 is an invitation and 204 requires readback. Organization policies/base roles and invitation limits apply; personal-repository role behavior differs. |
| [Invitations REST](https://docs.github.com/en/rest/collaborators/invitations) | Repository-specific open invitations are paginated; Administration read or the documented invitation permission applies, with repository-admin actor conditions. Update/cancel require Administration write. Invitation PATCH has its own closed role vocabulary. Recipients decide acceptance; this tool cannot claim it occurred from a create response. |
| [Teams REST](https://docs.github.com/en/rest/teams/teams#check-team-permissions-for-a-repository) | Team lookup requires Members read. Permission checks require Members read plus Metadata read and include parent inheritance; private-resource authorization can yield 404. Repository grant mutations require Administration write, Members read and Metadata read, plus applicable organization/actor conditions. |
| [GraphQL user edges](https://docs.github.com/en/graphql/reference/users#repositorycollaboratoredge), [permission sources](https://docs.github.com/en/graphql/reference/repos#permissionsource), [source union](https://docs.github.com/en/graphql/reference/orgs#permissiongranter) | Per-user source list is nullable; per-source role name is nullable. Organization, Repository, Team and EnterpriseTeam are distinct source kinds. These fields are candidate evidence, not proof that a particular credential sees a complete collection. |
| [Team access effects](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/managing-team-access-to-an-organization-repository) | Parent changes affect descendants; revocation can delete private forks. Inherited access requires changing the parent, which is not an implicit child operation in this design. |
| [Organization roles](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/repository-roles-for-an-organization) | Owners retain repository admin access, organization base roles can contribute access, and custom roles have separate availability/semantics. No role definition, paid-plan enrollment or base-policy change is part of reconciliation. |

Complete authorized visibility is required independently of listed token scopes. Missing organization/team membership, SSO/enterprise restrictions, hidden sources, unsupported role mappings, rate limits and denied responses fail without broadening permissions. No team membership/hierarchy, organization policy or role-definition write is used to repair a repository access failure.

## Implementation acceptance gates and fixture plan

| Gate/case | Required behavior/evidence before adoption |
| --- | --- |
| Local declaration | Omitted collection versus `[]`, role omission versus explicit role, null/unknown/duplicates and immutable identity mismatch; no API before local validation |
| Direct provenance | Direct read plus effective admin, inherited-only users, redundant team grants, empty teams, custom base-role collision, null sources/role names, all pages, conflicting IDs and partial GraphQL errors |
| Invitation lifecycle | 201 pending versus 204 direct grant, matching rerun with no notification, supported role update, cancelled/expired/accepted between reads, unresolved invitee, ambiguous write and no automatic resend |
| Collection clearing | Complete authoritative direct set only; inherited access untouched; unknown roles and hidden grants block all access writes |
| Operator and recovery | Execution user versus App identity, self/team-path loss, last verified human admin, unavailable owners, pending replacement admin and changes between reduction steps |
| API boundaries | Exact declared payloads/role translations, redirects denied, unknown/403/404/422/rate-limit outcomes, no fallback org/member/team mutation and no retry after uncertainty |
| Sequencing and rerun | Grant verification precedes reductions; first failure stops later operations; prior success remains; fresh rerun reconciles confirmed state without rollback |
| Export/presentation | Explicit per-family reads, failed export preserves file, direct-role round trip, invitation-aware export gate, native operation/state reports and credential-free private Summary |
| Real binary | Local isolated fixtures with dummy credentials exercise a complete adopted validate/export/plan/apply cycle, representative inherited/pending failures and no unrelated requests |

These are planned acceptance cases, not existing tests or evidence of live behavior. #356 may proceed only after its direct-source, role, invitation-intent and operator gates are resolved in a reviewed implementation. #357 independently requires complete direct team provenance; completing #355 or #356 does not resolve it. Any live verification requires separate explicit operational authorization and a bounded target, while independent code/design work can continue.
