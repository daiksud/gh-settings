# Feature: Manage persistent pull request limits and exemptions

## Contract

The [configuration reference](../../references/settings/moderation.md#persistent-pull-request-limits) owns
UI paths, values, scope, permissions, export and confirmation semantics.

## Rule: Reconcile only declared cap controls

- The canonical YAML and reported control paths are under
  `repository.settings.moderation.interaction_limits.pull_request_limits`.
  Export places Moderation after Collaborators and preserves independent cap/list
  ownership. The obsolete direct path and declarations containing both paths
  fail before API requests or rewriting a configuration. Empty Moderation groups
  manage nothing (`internal/workspace/ui_moderation_test.go`,
  `tests/acceptance/pull_request_limits_test.go`).
- Omitted groups manage nothing; explicit false and integer boundaries survive
  strict YAML round trips. Null, wrong types, API aliases and invalid ranges
  fail before remote work (`internal/workspace/ui_pull_request_limits_test.go`).
- Only selected controls count toward managed state and produce typed changes;
  an unacquired observation never supplies defaults
  (`internal/settings/pull_request_limits_test.go`).
- Complete GET requires all three known fields. PATCH contains required enabled
  and requested optional controls only; omitted latent values are retained.
  Exact matches perform no PATCH. Stale-baseline then matching reads can confirm;
  three stale reads exhaust the client budget. Companion drift, denied or
  malformed reads fail without another write
  (`internal/github/pull_request_limits_test.go`). Duplicate members, case
  aliases, invalid UTF-8, trailing data and oversized responses cannot supply
  evidence. An invalid pre-write read prevents PATCH; invalid readback after
  PATCH reports uncertain effects. Archived matching rejects ambiguous responses
  without writes (`tests/github-actions/composite_archived_matching_acceptance_test.go`).
- Plan selects one family read. Apply distinguishes no-op, confirmed write,
  pre-write read failure and uncertain write effects
  (`internal/app/pull_request_limits_test.go`).
- Ordinary export attempts the cap independently, and reports validate
  each fixed typed field and the closed managed-count bound
  (`internal/cli/export_test.go`,
  `internal/actions/pull_request_limits_test.go`).

- Built-executable export, validate, plan, apply, private reports and Summary
  exercise actual command wiring, exact payloads, no-op/omission, bounded stale
  confirmation and destination-preserving export failure
  (`tests/acceptance/pull_request_limits_test.go`).

### Scenario: Change a cap while preserving its other controls

- Given the persistent limit is enabled and the current maximum is five open pull requests per user
- When apply selects only `maximum_open_pull_requests_per_user: 2`
- Then the PATCH sends the observed enabled state and the new maximum
- And confirmation requires the maximum to be two and the omitted controls to retain their observed values

## Evidence limits

Local fixtures use dummy credentials and establish client behavior, not effective
GitHub grants or enforcement. An authorized live storage experiment observed one
old immediate GET followed by a matching later GET and restored baseline; its
cause and maximum propagation delay remain unknown. No external-actor enforcement
or automatic exemption preservation was established. No temporary restriction or automatic exemption-rule management is implemented.
Selected list membership has the separate contract below.

## Rule: Replace exemption list membership independently of the cap

- A separate `exemption_list` is authoritative membership, including explicit
  empty removal. Omission makes no list request and does not change cap ownership
  (`internal/settings/pull_request_exemptions_test.go`,
  `internal/workspace/ui_pull_request_exemptions_test.go`).
- Complete acquisition requires supported User identities and rejects malformed,
  null, duplicate, over-limit, redirected or possibly paginated inventory. Apply
  resolves new logins to supported User identities before writing, removes first,
  verifies retained membership/IDs, adds next and verifies exact
  final state. A full 100-user replacement never exceeds the limit; later failure
  retains confirmed removals (`internal/github/pull_request_exemptions_test.go`).
- Independent plan, automatic export and direct apply distinguish no-op, pre-write
  failure, uncertain effects and confirmed partial replacement. An archived
  target retains the existing guard (`internal/app/pull_request_exemptions_test.go`).
- CLI presentation, whole-list reports, strict report reconstruction,
  escaped Summary and status-only privacy have owning tests in
  `internal/cli/pull_request_exemptions_test.go` and
  `internal/actions/pull_request_exemptions_test.go`.
- The supplied binary exercises replacement order, repeated no-op, explicit
  clearing, independent export, destination preservation and partial failure
  with local dummy API fixtures (`tests/acceptance/pull_request_exemptions_test.go`).

On 2026-11-11, the Free personal public test repository showed Exemption list,
Add users and its empty inventory while the cap was temporarily enabled. The
owner was not offered as an add candidate; no other user was substituted. GET
returned `[]`, and the cap was restored disabled with an empty list. Live user
addition/removal, effective enforcement and preservation of automatic exemption
rules are not established. Local fixtures verify the mutation client contract.

### Scenario: Clear the exemption list without changing the cap

- Given the complete exemption list contains one supported user
- When apply declares `exemption_list: []` and omits the cap controls
- Then it removes that user and verifies that the list is empty
- And it sends no cap update
