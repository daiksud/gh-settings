# CLI reference

The `gh settings` commands use `.github/settings.yml` at the root of the current
Git checkout. They work from a subdirectory of that checkout. See the
[settings reference](settings/README.md) for the fields you can manage and the
[support policy](settings/scope.md#feature-adoption-scope) for supported repositories.

## Synopsis

| Command | Result on success |
| --- | --- |
| [`gh settings export`](#export) | Save readable settings to the local YAML file. No stdout. |
| [`gh settings validate`](#validate) | Check the local file. No stdout. |
| [`gh settings plan [--prune SCOPES]`](#plan) | Print the comparison with GitHub. Both matching settings and proposed changes exit 0. |
| [`gh settings apply [--no-writeback] [--prune SCOPES]`](#apply) | Reconcile declared settings with GitHub and report the result. |

Use `gh settings --help` (`-h`) for usage and supported prune scopes, or
`gh settings --version` (`-v`) for the installed version. Successful commands
exit 0. Errors, including unsupported arguments and output failures, exit 1
with a diagnostic on stderr.

## Repository and authentication

For commands that contact GitHub, the target is the repository named by the
first checkout-scoped `remote.origin.url`. To inspect the configured URLs, run:

```sh
git config --show-scope --get-all remote.origin.url
```

Read the first `local` or `worktree` entry, including entries from
repository-specific config includes. The URL must directly name GitHub.com
using HTTPS, `git@github.com:OWNER/REPO.git`, or an `ssh://` URL. A transport
rewrite such as Git's `insteadOf` can change where Git fetches without changing
the selected repository. A global origin, rewrite-only alias or `GH_REPO` cannot
select a target. If the repository moves, update the checkout's direct origin.

The extension uses the credentials selected by GitHub CLI. Run `gh auth status`
to inspect your login or `gh auth login` to authenticate. Access depends on the
resources the operation reads or writes. Even a read-only plan can need write
permission to observe a complete Ruleset bypass list. For GitHub Apps, use the
[permission matrix](settings/permissions.md) to check each operation. The
[public Action](action.md) also requires checkout metadata and explicit tokens
for export, plan and apply, including when the YAML selects no resources.

## Export

`gh settings export` reads the supported settings for the selected repository
and saves a declaration. It creates `.github/settings.yml` on first use and
refreshes an existing regular file. Read the file after the command succeeds.
Redirecting stdout to the destination could truncate the existing configuration.

- Complete observations replace prior declarations, including false and empty
  values. If an independent group cannot be read completely, export continues
  and reports `info: export` on stderr. First export omits unavailable settings.
  Re-export retains compatible prior declarations for them. A retained value is
  unverified, and a successful export can contain no observed repository values.
- Info diagnostics identify the affected family and give a bounded reason.
  Settings with a documented fallback can produce a normalization warning.
  Review these diagnostics alongside the [exported values](settings/configuration.md#exported-values).
  The file may contain plaintext Actions, Agents and Environment variables.
  Export never reads secret values or changes settings on GitHub.
- Export validates an existing file before API reads. An invalid, unreadable
  or unsafe destination fails. Export checks for local changes before replacing
  the existing file and refuses a symlink or a file that appeared during a
  first export. If a required warning or info diagnostic cannot be written,
  export fails before saving. A write failure during first-file creation can
  leave a partial file.

Review the resulting ownership before applying. For example, exporting a
repository without a Pages site can yield `branch: none`. Applying that
declaration after someone creates a site can delete it. Exported collections
also become eligible for [explicit pruning](#pruning). See the [Pages lifecycle](settings/pages.md#pages-configuration).

## Validate

`gh settings validate` checks that the file in the checkout contains one valid
YAML declaration with the exact version string `daiksud/gh-settings@v0`. It
checks field names, types, choices and combinations. The [configuration
reference](settings/configuration.md#configuration-semantics) defines empty
declarations, omissions and explicit values. Validation is entirely local. It
requires a Git checkout and needs neither a GitHub origin nor authentication.
It cannot establish remote permissions, resource existence or server acceptance.

## Plan

`gh settings plan` validates the file, then reads the state needed to compare
the managed settings. It leaves both GitHub and the file untouched. A declaration
that selects no work, including under the requested prune scopes, succeeds
locally without resolving an origin or opening an API connection.

| Condition | Standard output |
| --- | --- |
| No managed settings | `No managed repository settings.` |
| Managed settings already match | `No differences for managed repository settings.` |
| Changes proposed | Differences at UI-oriented YAML paths and operations on selected objects. |

For example, the following lines show a scalar change, a cleared set, and a
creation request:

```text
repository.settings.general.features.issues: true → false
repository.about.topics: ["old"] → []
repository.labels: create "new" color="#abcdef" description=""
```

Strings use Go-style double quoting and escaping. Booleans are `true` or
`false`, and topics are sorted JSON arrays. Some settings also report status
for a matching selection, such as an invitation still awaiting acceptance.
A normalized observation can appear as requiring a write even when its
displayed fallback equals the declaration. Its warning identifies the observed
value. Detailed plans can expose plaintext variable values. Handle their
output with the same care as the configuration file.

## Apply

`gh settings apply` validates the declaration and reads current state again.
It takes no saved plan as input. A concurrent edit to a declared setting can
therefore affect the result after planning. With no managed work, the CLI
prints `No managed repository settings.` and succeeds locally. With managed
work, success prints `Applied repository settings.`, including when everything
already matched. Additional lines may report collaborator outcomes, created
object IDs or notices.

Apply processes settings in dependency order. For example, it updates selected
About and General repository controls before Topics and Labels. For REST-backed
repository controls, it skips exact matches, writes changed controls and
independently reads back the originally declared values. A changed menu can
require both underlying values. Topics own a complete set, while Labels own
selected objects with their supported defaults. Consult the [resource
reference](settings/README.md) for each setting's write and verification rules.

When archive state is explicitly managed, a necessary unarchive precedes
ordinary writes. A requested final archive runs only after earlier work succeeds.
Use an active repository for the ordinary export, plan and apply workflow.
See the [archive and unarchive contract](settings/general.md#repository-archive-lifecycle).

### Recording IDs

After independently verifying a creation, ordinary CLI apply records the
returned ID in the original YAML for resource types that support it. For
example, a created label can produce this line:

```text
Created repository.labels "new": id 123
```

This binds future applies to the existing object. The CLI checks that the file
still matches its original snapshot plus any IDs it has already recorded.
It cannot write an ID into an aliased collection or a list containing aliased
or merged entries. A remote creation may already have succeeded when local
recording fails. Verify the reported ID, record it explicitly, then plan again.
Writeback preserves existing IDs and can normalize formatting. Review and
commit the resulting file yourself.
`--no-writeback` disables local ID recording. Setting `GITHUB_ACTIONS=true`
has the same effect. Both still allow remote writes and report IDs for recovery.

Collaborator results distinguish a verified **user ID** from an invitation ID
and report whether access is active or acceptance is pending. Use the verified
user ID for the YAML binding. See [collaborator identity and invitations](settings/collaborators.md#personal-collaborators).

## Pruning

Use one `--prune` option with a comma-separated list of scopes. For example,
`gh settings plan --prune labels,rulesets` previews the declared work plus
eligible unlisted deletions. After reviewing that plan, use the same scopes with
`gh settings apply --prune labels,rulesets` to perform the operations. Apply
also accepts `--no-writeback` with pruning.

An omitted collection stays unmanaged. With ordinary apply, `repository.labels: []`
selects no labels. With `--prune labels`, that same empty list selects **every
current label for deletion**, including its issue and pull request assignments.
Standalone `all` selects every supported scope whose collection is present.
Unknown or duplicate scopes, empty elements, repeated flags and mixing `all`
with another scope fail locally. Export and validate reject `--prune`.

Apply checks the prune candidates before any remote write, including writes
for other settings. It rechecks candidates before deletion and verifies the
result as defined for each collection. The [pruning reference](settings/pruning.md#collection-pruning)
lists the scopes, identity rules and consequences for each deletion.

## Recovering from a failed apply

1. Read stderr for the affected resource, earlier confirmed writes and any
   operation marked `write effects are unconfirmed`. A read failure before a
   write and a failed readback after a write have different consequences.
2. Inspect the current state in GitHub. Earlier successful writes remain
   applied. A failed or cancelled command can leave partial changes, including
   when the final failure involved local ID writeback or stdout.
3. Retain reported IDs. If a creation or invitation was attempted, verify
   which object or user the ID identifies before recording it in the
   declaration. Repeating an ID-less request can collide with an existing
   object or send a new invitation.
4. Run plan against the resulting declaration and current state before deciding
   whether to apply again. Apply has no automatic retry, transaction or rollback.
