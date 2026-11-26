---
type: Reference
title: "Moderation settings"
description: "Moderation settings declaration, ownership and API contracts."
---

# Persistent pull request limits

Settings → Moderation → Interaction limits → Pull request limits maps to:

```yaml
repository:
  settings:
    moderation:
      interaction_limits:
        pull_request_limits:
          limit_open_pull_requests_from_users_without_write_access: true
          maximum_open_pull_requests_per_user: 2
          count_draft_pull_requests_toward_the_limit: true
```

When `limit_open_pull_requests_from_users_without_write_access` is explicitly
`false`, only the checkbox is managed. The maximum, draft-counting option and
`exemption_list` are ignored even when present. For example,
`exemption_list: []` alongside `false` leaves the existing exemptions intact.
If any of these keys are present, `validate`, `plan`, `apply` and a re-export
of the existing file print one `info:` message on stderr. It names the ignored
keys and explains that the checkbox is `false`, without displaying their
values. The detailed Action plan shows this message in the job log.
A status-only plan suppresses it.

When the checkbox is `true` or omitted, each declared control is managed
independently. The enable and draft controls take booleans. The maximum takes
an integer from 1 through 1000. Validation rejects nulls, quoted booleans or
integers, and fractions for managed controls. Unknown keys and API-shaped
aliases are rejected even when the checkbox is `false`. Empty groups manage
nothing. Check the [CLI or Action version used by your workflow](../../guides/github-actions.md#published-package-and-consumer-pins)
for support for these semantics.

## Updating and verifying a cap

Plan selects the cap endpoint only when a cap control is declared. Apply
acquires fresh complete cap state, skips
exact matches and PATCHes required `enabled` plus only declared optional members.
When enable is omitted, the required member carries the freshly observed value.
Independent GET confirms the complete requested/preserved three-field state.
Only a complete response equal to the pre-write baseline can be retried: at most
three confirmation GETs, with one second between those stale observations. This
is a client budget, not a GitHub propagation guarantee. Writes are never retried.
Denied, malformed or other mismatching readback fails immediately. Failure after
PATCH reports uncertain effects; earlier successful operations remain applied.
Apply is not atomic and does not roll back.

When GitHub reports the cap disabled, `export` writes only the checkbox as
`false` and skips the exemption list request. When GitHub reports it enabled,
`export` writes all three cap values and reads the exemption list. A missing
`include_drafts` makes the cap observation incomplete. If the cap cannot be
read, export emits an info message and retains the prior cap declaration. It
can still read the exemption list unless the retained checkbox is `false`.
CLI and detailed Action reports use the UI paths and native booleans/integers.
Status-only output hides values.

The [official creation-cap API](https://docs.github.com/rest/interactions/repos#update-pull-request-creation-cap-for-a-repository)
requires administrator access. Credentials must already authorize the endpoint;
no grants are expanded. Adoption evidence covers GitHub Free personal public
repositories with these visible controls, not other plans/owners/visibility.
The cap limits users without write access and does not grant write access.
Temporary interaction restrictions are explicitly excluded by the [adoption policy](scope.md#feature-adoption-scope). Automatic exemption rules remain unimplemented.
The separately selected Exemption list is described below; ordinary cap updates
make no exemption request. Companion preservation in a cap update means only its
three known fields. Configuration
readback does not establish enforcement against external actors. See the
[behavior specification](../../development/features/pull-request-limits.feature.md).

## Pull request limit Exemption list

`repository.settings.moderation.interaction_limits.pull_request_limits.exemption_list`
manages the complete selected list of trusted User logins when the checkbox is
`true` or omitted. To manage just the list, omit the cap controls.

```yaml
repository:
  settings:
    moderation:
      interaction_limits:
        pull_request_limits:
          exemption_list: [trusted-user, another-user]
```

Omission leaves the list unmanaged and makes no list request. An explicit list
owns all membership, so omitted users are removed; `[]` explicitly clears the
list. Membership is case-insensitive and exported in sorted lowercase order.
At most 100 distinct User logins are accepted. Logins are 1–39 ASCII letters,
digits or single interior hyphens; surrounding whitespace, consecutive hyphens,
nulls, duplicate names and API-shaped user objects are rejected. Bot and
Organization identities are unsupported and stop acquisition rather than being
dropped from the list. Numeric user IDs are acquired identity evidence, not YAML
or writeback fields.

Plan reads only the complete list. Apply reads fresh inventory and skips an
exact match. Before any mutation, newly added logins are resolved through the
public Users API and must identify supported User accounts. It removes obsolete users first and independently verifies the
remaining membership and retained User IDs before adding users, avoiding a
100-user list's temporary overflow during replacement. It then independently
verifies exact desired membership and both retained and newly resolved User IDs. Each mutation uses only its
computed user difference. A failure stops further work; confirmed removals can
remain applied when addition fails. Writes are not retried and no rollback or
concurrency protection is promised. This selected-list operation runs before
separately declared cap controls; it never changes the cap, automatic exemption
rules, temporary restrictions, collaborator grants or repository permissions.

When the cap is enabled, `export` includes the complete list, including `[]`.
A failed, incomplete, ambiguous or unsupported list read emits info and retains
prior membership without stopping independent reads. CLI
and detailed Action reports/Summary expose the selected membership; status-only
output omits it. The collection counts as one managed value.

The [official bypass-list API](https://docs.github.com/en/rest/interactions/repos#get-pull-request-creation-cap-bypass-list-for-a-repository)
requires **Administration write**, including for reads. Its GET returns User
objects; PUT adds and DELETE removes up to 100 logins per request. An unexpected
pagination Link is rejected rather than treating a partial list as complete.
Existing credentials must authorize these operations. Exemption allows listed
contributors to exceed the cap without granting write access; callers must
review the complete desired list before applying it.

On 2026-11-11, the Free, personally owned public test repository exposed the
Exemption list and Add users control after its cap was temporarily enabled.
The existing empty list was acquired through the API. The owner was not offered
as a UI search candidate, so no user was added and no live add/remove persistence
or external-actor enforcement is claimed. The cap was restored to its disabled
baseline and the list remained empty. Automated mutation evidence uses local
fixtures. See the [behavior specification](../../development/features/pull-request-limits.feature.md).

[Settings index](README.md) · [GitHub App permissions](permissions.md)
