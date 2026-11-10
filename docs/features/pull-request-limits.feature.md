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
  (`internal/cli/pull_request_limits_test.go`,
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
or automatic exemption preservation was established. No temporary restriction or
exemption management is implemented.
