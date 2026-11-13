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

The supported scopes are `labels`, `rulesets`, `actions-policies`,
`actions-variables`, `agents-variables`, `environment-variables`, `environments`,
`collaborators`, `branch-protection-rules` and `autolinks`. Standalone `all` selects all currently recognized
scopes; future supported prune scopes must join the shared registry and tests.

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
  policy. A supported complete PR rule, including either additional-approval value,
  is owned and eligible. Prune grants no broader ownership.
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

## Environment deletion

Present `repository.settings.environments` owns retention when the explicit
`environments` scope or standalone `all` is selected. Retain the union of policy
and variable-only names, including `variables: []`, using Unicode folding. Empty
[] selects all acquired parents; omission never does. Retained policy upserts and
all selected Environment variables succeed before the fixed deletes run.

Global acquisition binds repository/archive identity, positive parent ID,
complete supported policy, active raw patterns, complete variables and secret
name/created/updated metadata. Explicit custom-protection endpoint acquisition
must be complete and known-empty at acquisition and each last recheck. Unsupported,
normalized, denied or incomplete candidates block all writes. Retained unsupported
parents need no deletion snapshot. Recheck the candidate set before Environment
work, then each parent's observable state/identity before DELETE. Only this
command's confirmed unarchive may adjust the acquired archive state. DELETE 204
requires fresh complete inventory absence of both old ID and folded name;
recreation or failed confirmation has uncertain effects and stops later deletes.

Parent deletion removes contained configuration. GitHub documents secret and
protection removal and failure of waiting jobs. Workflow references can recreate
an unprotected name; no automatic restoration, backup, retry or rollback is
promised. Secret values are never read. Secret metadata does not prove every
rotation and never enters reports, Summary or snapshot debug logs. Inactive
patterns are inaccessible under the custom-mode prerequisite; parent deletion
permits this documented limit without fabricating an empty acquired latent list.
The final name-addressed check/delete race remains. See the
[permission and ownership contract](../settings.md#environment-pruning).

Plan counts one Environment collection even for explicit [] with prune, shows
complete supported before-policy and absent after-policy, and rejects normalization
metadata on delete reports. Detailed output includes deletion consequences;
status-only hides names/policy. YAML and numeric-ID writeback remain unchanged.

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

## Environment-stage test mapping

- Model retention, parser expansion and complete before-only reports:
  `internal/settings/environment_prune_test.go`, `internal/actions/environments_test.go`.
- Child metadata pagination, custom-protection gates, late drift, escaped names,
  target/archive checks and deletion absence/recreation:
  `internal/github/environment_prune_test.go`.
- Global preflight, fixed candidates, retained policy/variable ordering, confirmed
  unarchive and partial progress: `internal/app/environment_prune_test.go`.
- Supplied CLI present/empty/omitted lists, variable-only retention, combined variable
  prune, repeat no-op, late protection and unchanged YAML:
  `tests/acceptance/environment_prune_test.go`.
- Actual supplied-product Action detailed/status-only plan and `all` apply:
  `tests/github-actions/environment_prune_acceptance_test.go`. The candidate gate
  runs this against the producer executable; no live parent deletion is performed.

Personal access uses the same `collaborators` selector and standalone `all`, with
present-list ownership, global fixed-candidate preparation and independent
readback. The [access specification](repository-access.feature.md#explicit-personal-access-pruning)
owns its retained-user union, invitation dependency, operator and progress cases.

## Classic branch-protection deletion

The [classic pruning contract](../settings.md#classic-branch-protection-pruning)
adds whole-rule deletion independently of partial supported-control updates. Present
lists retain numeric IDs and successful creations; omission stays unmanaged.
Complete private candidate snapshots precede any write. Recheck only the frozen
initial candidates before family work and before each mutation; late unlisted
rules are not adopted, even when the initial selection was empty. Own verified
unarchive is the only archive-state exception. All declared classic writes and
required creation-ID recording precede deletion. Disappearance, replacement,
raw nullable/check/App/actor drift or incomplete observations stop work.

The exact GraphQL payload type acknowledges an error-free mutation but carries no
deleted ID. Independent complete inventory must then exclude both acquired IDs;
a same-pattern different-ID rule may remain. Unknown mutation effects stop later
writes with recovery identities. No branch deletion, ordering rewrite, automatic
retry, recreation or rollback is provided. Detailed plan/Summary discloses removal
of all configured protections and possible overlapping-rule activation. Status-only
hides identities/policy; raw snapshots never enter reports or inherited debug logs.

Model, adapter, sequencing and report cases are in
`internal/settings/classic_prune_test.go`, `internal/github/classic_prune_test.go`,
`internal/app/classic_prune_test.go` and `internal/actions/classic_protection_test.go`.
`tests/acceptance/classic_prune_test.go` runs the actual CLI through empty/omitted
lists, whole deletion, retained duplicate patterns, creation/writeback ordering,
late arrivals, complete preflight refusal, partial ACK failure, privacy and repeat
no-op. Its unsupported-scope baseline fails before this feature.
`tests/github-actions/classic_prune_acceptance_test.go` checks the actual supplied
product's detailed/status-only plan and standalone `all` apply/repeat behavior;
the candidate artifact gate includes it. All mutations are isolated fixtures.

## Autolink pruning

The `autolinks` scope and standalone `all` use the [Autolinks contract](../settings.md#autolink-references). Omitted collections remain unmanaged; present empty lists prune every acquired reference. Global preflight freezes all candidates and validates complete declared intent before any write. Expected inventory includes only verified own creations and excludes verified own deletions; only successful own unarchive changes the archive expectation. Complete inventory and individual candidate checks reject drift before DELETE, and empty 204 plus complete inventory absence verify each removal. Creation and required ID recording precede deletion. An immutable replacement needs explicit deletion followed by later creation; a current candidate cannot be silently adopted by prefix.
