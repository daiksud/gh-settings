# Autolink reference behavior

The [Autolinks reference](../settings.md#autolink-references) owns the public
fields, identity, permissions, explicit replacement and failure contract.
These scenarios use local fixtures; they do not establish live API persistence,
an authenticated form observation, or external-link rendering.

## Configuration and observation

- Literal YAML requires prefix, target URL and numeric/alphanumeric format. Exact
  URL bytes survive export and payloads. Nulls, wrong types, unknown keys,
  overlapping prefixes and duplicate/nonpositive IDs fail locally.
- Absolute HTTP(S), no userinfo and literal `<num>` are explicit product
  constraints. Parsing substitutes the placeholder locally without fetching URLs.
- Required API fields, personal-public target/owner identity and the complete
  list must be known. A missing value cannot become a default. Optional timestamp
  presence is private evidence. Unavailable export retains prior valid intent;
  first export omits it; confirmed empty state exports `[]`.
- ID-bound matching configuration is unchanged. Missing/differing bound entries
  and IDless collisions fail before mutation, including collisions with candidates
  the same apply would prune. No automatic update or replacement exists.

## Creation, retention and deletion

- POST contains all native controls. Exact 201, new positive ACK identity,
  independent GET-by-ID and complete inventory verification precede ID recording.
  Only a positive unseen ID in the intended successful response can be recovery
  evidence; malformed or unverified evidence is not a completed creation.
- Prune freezes every candidate before any remote write. Its expected inventory
  changes only for verified own creations/deletions and successful unarchive.
  Retained and candidate changes, unknown additions, and metadata drift fail.
- Finish declared creations and required local recording before fixed deletions.
  A file error stops deletion. Each delete rechecks both inventory and candidate,
  then requires empty 204 plus independent complete absence and retained state.
- Omission remains unmanaged with any scope. Empty lists retain resources without
  prune and remove all acquired entries with explicit `autolinks` or `all`.
  Comma-separated scopes use the common grammar. No tombstones are accepted.
- Earlier confirmed work remains partial progress when a later operation fails.
  Pre-attempt errors do not claim uncertain deletion; post-attempt failures do.
  There is no transactional rollback, retry or lock against later races.

## CLI and Action

- Supplied CLI lifecycle covers plan/create/ID recording/prune/repeat no-op,
  ordinary retention, omission/all, immutable collisions, readback failure and
  own unarchive ordering under a host `GITHUB_ACTIONS=true` environment.
- The actual composite uses explicit installation-token fixtures and current
  product bytes. It covers export, detailed/status-only plan, create/prune,
  matching repeat, empty/all and failed readback with no success outputs.
- Action apply preserves YAML and HEAD. Repeating an unbound successful creation
  fails collision; the user must record confirmed IDs through the documented
  follow-up procedure. Detailed reports may contain non-secret URLs; status-only
  hides them. Upstream/debug canaries do not reach logs.

## Verification mapping

The literal codec regression fails on pre-change source because `autolinks` is
unknown. Model ownership, strict URL/identity constraints and expected inventory
transitions are in `internal/settings/autolinks_test.go`. REST ACK/readback,
unknown observations, recovery and fixed deletion checks belong to
`internal/github/autolinks_test.go`. App tests own export isolation, global
preflight, unarchive, recording and partial progress. Workspace tests own literal
YAML and guarded identity insertion; Action report tests own closed presentation
and privacy. Supplied product checks are `tests/acceptance/autolinks_test.go` and
`tests/github-actions/autolinks_acceptance_test.go`.
