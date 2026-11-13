# Personal collaborator identities and invitations

The [configuration contract](../settings.md#personal-collaborators) owns the
public schema and lifecycle. This feature covers personally owned public
repositories; organization roles, teams, acceptance, cancellation and removal
are not implemented by this slice. Email-only roundtrip remains an explicit
technical gap. [ADR0031](../adr/0031-repository-access-ownership.md) is historical
organization-focused context, not a second schema.

```gherkin
Feature: Reconcile personal collaborators without repeating matching invitations
  Scenario: Export complete active and pending people
    Given a personally owned public repository with complete typed inventories
    When export reads every collaborator and invitation page
    Then each supported person is exported with username and immutable user ID
    And owner access is excluded by owner ID
    And pending invitations do not imply accepted access

  Scenario: Retain the entire family when acquisition is unavailable
    Given valid prior collaborator intent
    When a page is denied, malformed, incomplete, unsupported or ambiguous
    Then export retains the complete prior family with bounded information
    And a first export omits the family
    And no partial active/pending snapshot is saved as a complete list

  Scenario: Bind an already pending username without another notification
    Given username-only initial-add intent matching an open invitation
    When apply freshly confirms its user identity and pending state
    Then no collaborator PUT occurs
    And the verified user ID is recorded locally unless writeback is disabled
    And output says acceptance is pending
    And local binding is not reported as a completed remote write

  Scenario: Send an initial invitation and verify independently
    Given a resolved non-owner person absent from both complete inventories
    When apply rechecks target identity and current username identity
    Then exactly one bodyless collaborator PUT is attempted
    And a 201 requires its exact pending invitation or the same active user
    And a 204 requires active access
    And only confirmed user identity can be recorded in YAML

  Scenario: Refuse a stale selection without reinviting
    Given an ID-bound person no longer active or pending
    Or a prepared pending invitation that vanished before writing
    Or a renamed or reassigned username
    When apply reconciles the declaration
    Then it fails before sending an invitation
    And it does not accept, cancel, remove, retry or roll back access

  Scenario: Recover an uncertain invitation result
    Given an attempted request with a positive reported invitation ID
    When acknowledgment validation or independent readback fails
    Then apply stops later operations and preserves the original declaration
    And diagnostics retain user and invitation recovery evidence
    And neither acceptance nor verified local binding is claimed
    And earlier confirmed changes and bindings remain available

  Scenario: Omitted and empty collections cause no access mutation
    Given collaborators is omitted or explicitly empty
    When plan or apply runs
    Then no collaborator read or write is required
    And unlisted people remain untouched

  Scenario: Keep pending state visible without leaking private reports
    Given an ID-bound pending person with no new invitation planned
    When CLI plan or detailed Action Summary renders the result
    Then it reports matching intent and pending acceptance
    And status-only publication omits recipient names and IDs
```

## Evidence and test owners

- [Model](../../internal/settings/collaborators_test.go): identities, omission,
  duplicates, case matching, active/pending/absent reconciliation and local binding.
- [GitHub adapter](../../internal/github/collaborators_test.go): complete pagination,
  target/actor/role validation, fresh no-op races, bodyless requests, independent
  readback, recovery evidence and pre-write versus uncertain failures.
- [Application](../../internal/app/collaborators_test.go): all-actor preparation,
  local versus remote progress, later failure and complete export retention.
- [Workspace](../../internal/workspace/ui_collaborators_test.go): schema shape,
  guarded ID recording, concurrent edits and alias rejection.
- [Action reporting](../../internal/actions/collaborators_test.go): pending details,
  closed reports and status-only privacy.
- [Built CLI](../../tests/acceptance/collaborators_test.go) and
  [supplied-product Action](../../tests/github-actions/composite_collaborators_acceptance_test.go):
  actual command/lifecycle/file/output behavior against local fixture services.

Official [personal permissions](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/repository-access-and-collaboration/permission-levels-for-a-personal-account-repository),
[invitation UI](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/repository-access-and-collaboration/inviting-collaborators-to-a-personal-repository),
[Collaborators REST](https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2026-03-10),
[Invitations REST](https://docs.github.com/en/rest/collaborators/invitations?apiVersion=2026-03-10)
and [User REST](https://docs.github.com/en/rest/users/users?apiVersion=2026-03-10)
establish the supported target and endpoints. The implementation was checked
against the official [OpenAPI snapshot](https://github.com/github/rest-api-description/blob/bd6df26d2a0104874fe5025a25ebcde356413364/descriptions/api.github.com/api.github.com.json).
Fixture evidence does not establish real App grants, notification delivery or
recipient acceptance. No live invitation or access mutation is authorized by
these tests. Complete collections are not atomic snapshots; username-addressed
PUT has no atomic expected-user-ID guard. Without a saved verified binding, a
later run cannot durably deduplicate an ambiguous request whose invitation has
since disappeared; resolve state before retrying.
