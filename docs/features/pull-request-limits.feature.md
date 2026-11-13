# Persistent pull request limits

## Contract

The [configuration reference](../settings.md#persistent-pull-request-limits) owns
UI paths, values, scope, permissions, export and confirmation semantics.

## Scenarios and owning tests

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
  (`internal/github/pull_request_limits_test.go`).
- Plan selects one family read. Apply distinguishes no-op, confirmed write,
  pre-write read failure and uncertain write effects
  (`internal/app/pull_request_limits_test.go`).
- Export selection is independent, duplicate flags fail, and reports validate
  each fixed typed field and the closed managed-count bound
  (`internal/cli/export_test.go`,
  `internal/actions/pull_request_limits_test.go`).

- Built-executable export, validate, plan, apply, private reports and Summary
  exercise actual command wiring, exact payloads, no-op/omission, bounded stale
  confirmation and destination-preserving export failure
  (`tests/acceptance/pull_request_limits_test.go`).

## Evidence limits

Local fixtures use dummy credentials and establish client behavior, not effective
GitHub grants or enforcement. An authorized live storage experiment observed one
old immediate GET followed by a matching later GET and restored baseline; its
cause and maximum propagation delay remain unknown. No external-actor enforcement
or automatic exemption preservation was established. No temporary restriction or automatic exemption-rule management is implemented.
Selected list membership has the separate contract below.

## Selected Exemption list

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
- Independent selected plan/export and direct apply distinguish no-op, pre-write
  failure, uncertain effects and confirmed partial replacement. An archived
  target retains the existing guard (`internal/app/pull_request_exemptions_test.go`).
- Explicit export selection, whole-list reports, strict report reconstruction,
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
