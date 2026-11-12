---
type: Feature
title: Explicit collection pruning
description: Add bounded deletions to normal reconciliation through command options.
---

## Contract and staged delivery

`plan --prune labels,rulesets` and `apply --prune labels,rulesets` add deletion
of undeclared independent objects; they do not limit other declared updates.
Use one comma-separated option. Unknown names, empty elements, duplicates and
repeated flags fail locally. `all` must stand alone and must never silently omit
an unsupported scope.

The intended scopes are `labels`, `rulesets`, `actions-policies`,
`actions-variables`, `agents-variables`, `environment-variables` and
`environments`. Current source implements labels/Rulesets, Actions Policies and
all three variable scopes. Environment-object deletion remains undecided and is
explicitly rejected before target resolution or remote access.
`all` remains unavailable while any intended scope is unimplemented or undecided.
This specification is not a claim that every stage has shipped.

Omitted collections remain unmanaged even when selected for prune. An explicit
empty list selects all current objects for deletion; a nonempty list retains its
declared identities and selects other objects. Ordinary reconciliation remains
create/update only for these independent collections. Topics, PR exemption lists
and deployment branch/tag pattern replacement retain their existing semantics.
Environment-variable pruning is restricted to listed Environments whose
`variables` list is explicitly present; it does not select other Environments or
their policy, secrets or omitted variables.

## Safety and acceptance conditions

- Acquire every page and complete identity/policy/value state. Missing, denied,
  malformed or partial acquisition never becomes an empty collection.
- Prepare and validate all selected prune candidates before the first remote
  write, including writes to other resources. A candidate that cannot safely be
  deleted fails the operation rather than being skipped.
- Preserve Ruleset rejection of inherited/Push objects, unsupported/normalized
  policy and unowned Preview or PR-rule data. Prune grants no broader ownership.
- Create/update before deleting, with a fixed candidate set. A newly created
  object cannot become a deletion candidate. Detect changed candidates before
  their write, and never expand deletion to concurrently introduced objects.
- Recheck target identity and the observed state immediately before DELETE;
  require the documented acknowledgment and fresh absence readback. Drift,
  denied/incomplete verification or mismatching readback fails without retry or
  rollback. Earlier completed writes and uncertain current effects remain in
  failure reports.
- Plan is read-only and exposes each deletion. Label deletion removes Issue/PR
  assignments; automatically created Dependabot labels are candidates too if not
  declared. Recreating a label cannot restore the lost assignments.
- Rulesets `delete: true` is removed. Any `delete` member is a local validation
  error with guidance to declare **all Rulesets to retain**, remove the unwanted
  object from the list, inspect `plan --prune rulesets`, then apply that scope.
  This is broader than the old single-ID tombstone; merely removing that old
  entry is not sufficient if other retained Rulesets were never declared.
- CLI, detailed Action report/Summary and executable acceptance retain typed
  deletion identities, honest counts and partial-failure status. No real target
  deletion or settings apply is part of the automated tests.

## Environment deletion decision pending

Environment-object deletion is not adopted. GitHub documents deletion of its
secrets and protection rules and automatic failure of jobs waiting on those
rules. Secret values cannot be read back for restoration. Effects on variable
ownership, workflow references and any recreation require a separate explicit
adoption decision; do not infer reversibility. Until resolved, requests for
`environments` or `all` fail clearly rather than promising full support.

Primary sources: [deleting an Environment](https://docs.github.com/actions/deployment/targeting-different-environments/using-environments-for-deployment#deleting-an-environment),
[Environment REST API](https://docs.github.com/rest/deployments/environments#delete-an-environment),
[Labels REST API](https://docs.github.com/rest/issues/labels#delete-a-label),
and [Rulesets REST API](https://docs.github.com/rest/repos/rules#delete-a-repository-ruleset).

## First-stage test mapping

- Scope parsing, omitted/empty retention and deletion refusal:
  `internal/settings/prune_test.go`, `internal/settings/ruleset_delete_test.go`.
- Global preflight, changed candidates, fixed creates/deletes and partial progress:
  `internal/app/prune_test.go`, `internal/app/rulesets_test.go`.
- Strict label identity/state/204/absence/pagination verification:
  `internal/github/label_prune_test.go`; Rulesets retain
  `internal/github/ruleset_delete_test.go`.
- CLI flag boundaries: `internal/cli/prune_test.go`. Public Action propagation and
  old-package refusal: `tests/github-actions/prune_test.go`.
- Real-binary lifecycle and unsupported-protection rejection:
  `tests/acceptance/prune_labels_test.go`,
  `tests/acceptance/ruleset_delete_test.go`,
  `tests/acceptance/ruleset_unsupported_controls_test.go`.

All use isolated repositories/local HTTP fixtures with dummy credentials. They
do not establish actual credential grants, live protection or deletion effects.

## Variable-stage test mapping

`internal/settings/variable_prune_test.go` covers presence and canonical name
identity. `internal/github/variable_prune_test.go` covers 30-entry pagination,
exact totals, values, status/readback and Environment identity races.
`internal/app/variable_prune_test.go` covers all-candidate preflight, fixed
upsert/delete ordering, scoped parents, explicit parent creation and partial
progress. `internal/actions/variable_prune_test.go` and
`tests/acceptance/prune_variables_test.go` cover typed absent values, both Summary
modes, actual CLI partial failure/restart and omitted-scope preservation.

Actions Policy pruning additionally refuses unsupported candidate policy state,
uses repository-only inventory and exact detail rechecks, and requires confirmed
absence after DELETE 204. Removing a restriction can permit future workflows.
