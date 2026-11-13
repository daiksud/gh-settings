# Personal repository access implementation boundary

Personal-repository collaborators and invitations are adopted future work under
the [v1.0.0 goal](https://github.com/daiksud/gh-settings/issues/668) and
[feature adoption policy](../settings.md#feature-adoption-scope). They are **not
implemented**: current source and published packages do not accept access
configuration or export this family. The [coverage ledger](../contributing/settings-coverage.md)
tracks the required UI, API and implementation evidence.

## Target and historical design

The target is a personally owned public repository. Model only its configurable
collaborator and invitation controls; do not import organization roles or team
collections into its YAML. Organization/enterprise membership, team access,
owner/base permissions, App installation grants and credentials remain outside
this target. Repository rename, visibility, transfer and deletion are separate
explicit exclusions.

[#355](https://github.com/daiksud/gh-settings/issues/355) and
[ADR0031](../adr/0031-repository-access-ownership.md) record a historical
organization-focused design. The later closures of #356 and #357 recorded a
previous scope decision, not evidence that a personal-repository API is
unavailable. The v1 goal supersedes the personal collaborator/invitation
exclusion; it does not adopt that historical schema or reopen those Issues.
Historical distinctions between direct grants, effective access and pending
invitations remain useful where the target APIs expose them.

## Required implementation evidence

Before adding public YAML, establish the following for the target repository:

- Inventory the actual UI roles and lifecycle actions, and map them to complete
  official list/read/create/update/remove contracts. A successful setter does
  not establish export or readback support.
- Distinguish direct repository grants, owner access and pending invitations.
  Complete paginated observations must preserve immutable repository/actor and
  invitation identities; partial or hidden data must not become an empty set.
- Define omission, explicit collection contents and pruning through the common
  [collection contract](../settings.md#collection-pruning). Do not infer deletion
  from an omitted collection or introduce one-shot YAML tombstones.
- Treat invitation creation, existing pending state and accepted access as
  distinct outcomes. A matching pending invitation must not be resent, and a
  successful request must not be described as recipient acceptance. Never accept
  an invitation on another user's behalf or automatically resend after an
  ambiguous result.
- Verify identity and resulting state independently after writes. Preserve the
  operator's management path and the repository owner's access; incomplete
  identity or unsafe removal must fail before mutation. Later failures retain
  confirmed earlier changes and report unverified effects without rollback.
- Deliver automatic export, validation, plan, apply and CLI/Action reporting
  together. Initial unavailable export omits the family; re-export preserves
  valid prior intent with bounded information. Plan/apply remain strict about
  observations required for declared work.

No access YAML path, role enum, prune selector or notification option is reserved
by this page. Their exact contract belongs to the implementing change after UI
and API evidence is recorded. Merely installing or running gh-settings does not
acquire grants or send invitations.

## Official contracts and verification limits

The [Collaborators REST API](https://docs.github.com/en/rest/collaborators/collaborators)
and [repository Invitations REST API](https://docs.github.com/en/rest/collaborators/invitations)
are starting points for the personal-repository assessment. The historical
[GraphQL PermissionSource](https://docs.github.com/en/graphql/reference/objects#permissionsource)
investigation concerns direct-versus-effective roles; it is not proof of complete
personal-repository observation or a requirement to use organization APIs.

The caller is assumed to have required permissions. This does not certify
actual grants or complete observations. Adopt endpoint-specific absent meanings
only from the relevant contract; a 404 on another resource does not establish an
absent collaborator or invitation. Explicit denials, malformed/incomplete reads,
timeouts and readback mismatches remain failures.

Implementation tests must cover omitted/empty/nonempty collections, identity
conflicts, all pages, pending-versus-active transitions, matching reruns without
writes, uncertain invitation creation, removal guards and partial progress.
Fixture tests establish modeled behavior; live evidence must separately record
target UI availability, credential type, API behavior and any notification or
acceptance limitations. This policy page performs no live access change and does
not authorize one.
