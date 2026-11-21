# Feature: Export readable settings while retaining unobserved intent

Ordinary `export` attempts every implemented family without selection flags.
The [settings reference](../../references/settings/README.md) lists supported
declarations, including the retained v0 Autolinks and classic branch-protection
contracts. Private-only controls, product switches and organization settings
are outside the target. Export neither enables prerequisites nor reads secret values.
Unimplemented controls remain unavailable regardless of Preview status.

## Rule: Replace each family only after complete observation

Each independent family is converted completely before adoption. Readable false,
empty strings and empty collections retain their supported meanings. A failed,
partial, malformed or unsupported observation emits bounded `info: export` and
independent reads continue. Initial export omits unavailable families. Re-export
retains valid prior declarations for them as unverified intent where compatible
with positively acquired settings. Failed pages do not supply partial collections or empty defaults. Projection failure also retains
the family; another successfully read Actions field must not erase unread Actions
siblings. Acquired families are validated independently from retained intent.
Valid partial declarations remain partial when their family is unavailable; they
must neither reject unrelated acquired values nor invent omitted controls.
Existing normalization and API-specific 404 decisions are unchanged.

About/basic General, merge controls, PR access, Issue permissions and Sponsorships
are independently acquired groups. Each response must be complete before its
owned fields replace prior intent. All five can be unavailable without stopping
later families; partial or absent About/General is valid. Confirmed disabled
Issues or Pull requests suppress incompatible creation-permission details even
when the detail read failed. Unknown gates do not imply enablement or disablement.
Retained intent never supplies fresh normalization provenance.

Checkout/origin target resolution remains required. Family-specific remote
identity checks remain independent acquisitions, not a mandatory global gate. Successful local output does not certify repository
existence when remote observations are unavailable. Invalid prior YAML, target/file
safety, family-info output, normalization-warning output and write failures remain
fatal. Diagnostic delivery errors are kept separate from unavailable observations
and prevent saving even when later family reads or info writes succeed. An initial
destination stays absent; an existing declaration remains byte-identical. File
snapshot and guarded replacement rules remain as specified by [export](export.feature.md).
Plan/apply continue to validate and strictly observe declared intent.

### Scenario: Preserve an unread family and update an independently readable one

- Given the existing file declares a repository description and topics
- When export reads a new description but cannot acquire the topics
- Then it writes the new description and retains the prior topics with an informational diagnostic
- And a first export with the same observations omits topics

## Rule: Acquire variable inventories independently for each Environment

Actions, Agents and Environment variable inventories use the existing complete
paginated list APIs. Names and values are not guessed. These are plaintext
variables; secrets and organization values are never requested. Info never logs
values, credentials, URLs or upstream messages. Output files can contain variable
values and must be handled accordingly.

An inventory containing a value unsupported by the current declaration schema
is unavailable as a family: emit info and retain its prior declaration. In
particular, the existing nonempty desired-variable constraint is not relaxed or
worked around by inventing another value or silently deleting that entry.

Environment names/IDs are inventoried completely independently of protection
policy. Variables are then read per Environment, with existing parent identity
checks. One failed parent inventory retains its prior variables and does not stop
other parents. A complete parent inventory proves absent parents; unknown does
not. Policy and variable observations remain independent and do not manufacture
policy intent from variable-only output.

Complete observed collections replace their prior collection. Unavailable
collections are retained as a whole. Exported `[]` is a deliberate observed empty
collection and can be destructive input to a later explicit prune. The existing
Pages GET404 absence declaration can delete a site that appears before a later
apply. Export itself only reads; inspect YAML before applying it.

### Scenario: One unavailable Environment does not hide variables from another

- Given the complete Environment inventory includes `staging` and `production` and the file declares variables for both
- When the variables read for `staging` is denied and the complete `production` variable inventory is empty
- Then export retains the declared variables for `staging` and writes an empty list for `production`
- And the diagnostic contains no variable values

## CodeQL

Preserve [CodeQL's single-response decision](readable-security-export.feature.md):
confirmed `not-configured` exports false and removes incompatible old details with
info. Configured state exports true and readable details. Unknown state retains
prior intent. No automatic enablement or scan execution occurs.

## Verification

The [application matrix](../../../internal/app/export_readable_test.go) covers each
independent reader's explicit failure/unacquired observation, prior preservation,
projection failure, diagnostic IO and continued acquisition. It also covers full
and empty variable inventories, unsupported values and independent Environment
variables. Existing adapter tests own pagination and typed-response validation;
plan/apply tests retain strict failure and mutation ownership. The
[built-CLI About/General regressions](../../../tests/acceptance/repository_export_isolation_test.go)
cover denied groups, partial retained declarations, disabled-Issues suppression,
continued sibling acquisition, bounded diagnostics, local validation and strict
plan failures. The [built-CLI late-page regression](../../../tests/acceptance/readable_settings_export_test.go)
exercises ordinary export, literal YAML, partial Actions intent, info privacy and file safety separately
from live read-only verification. No fixture establishes live settings mutation.
