# Settings commands

## Environment export

`gh settings export --environments` includes all supported named Environment policies and composes with other export options. It requires complete Environment/custom-protection reads and preserves the existing output-file safety contract. Ordinary export does not read Environments.

`validate`, `plan` and `apply` accept the [Environment configuration contract](settings.md#environments); names select ordinary create-or-update, not numeric IDs. No Environment writeback occurs, regardless of `--no-writeback` or GitHub Actions mode. Unlisted objects remain untouched. Name-only entries reset supported settings to documented defaults; inspect plan before apply.

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

```text
settings.repository.has_issues: true → false
settings.topics.names: ["old"] → ["github-cli","settings-as-code"]
```

Strings are double-quoted with Go string escaping, including quotes, backslashes, newlines and control characters. Booleans use `true` or `false`; topics use sorted JSON arrays. Equal managed values are omitted. When all managed values match, output is `No differences for managed repository settings.`

Label creates and updates appear in stable identity order, with no implicit deletes. Names and attribute values are quoted and escaped. For example:

```text
settings.labels: create "bug" color="d73a4a" description=""
settings.labels: update "help wanted" (id 123) description: "Old description" → ""
```

A nonempty label selection counts as one managed item; an empty selection manages nothing. The effective whole supported object is compared, including default-derived description clearing.

Ruleset operations show only selected whole supported objects, with numeric IDs and fixed-default changes. Unlisted Rulesets never appear as deletes. ID updates can rename; ID-less collisions fail instead of adopting existing names.

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding and output errors exit 1 with a diagnostic on stderr. Failures before output leave stdout empty. Plan changes neither configuration nor GitHub settings. Its output is an observation, not a saved execution plan or drift protection for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) separately requires target/commit metadata and explicit plan/apply tokens, including for empty declarations.

## Apply behavior

Workflow permissions are applied after execution policy as a separate operation. Partial declarations send only their own keys; fresh complete readback verifies desired and preserved values. Failure retains earlier execution/resource writes and does not broaden existing credentials. See the [workflow-permission contract](settings.md#default-workflow-permissions).

Actions policy is applied after existing repository/topics/labels/Rulesets operations. It uses fresh preflight and readback, preserves API-required and selected companion values, and enables a currently disabled repository only after any requested selected-list update. Endpoints are not transactional; see the [Actions policy ordering and failure contract](settings.md#actions-execution-policy).

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no resources are managed, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `names: []`, performs an update. A nonempty label selection reads current labels and preflights ID-selected object updates and ID-less creates. `labels: []` selects nothing.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `settings.repository` | Existing fields use `PATCH /repos/{owner}/{repo}`.[^github-update-repository] PR access, Issue creation policy and Sponsorships use a shared ID lookup and declared-only GraphQL `updateRepository`; GraphQL-only declarations skip PATCH. |
| `settings.topics.names` | `PUT /repos/{owner}/{repo}/topics` with the complete declared topic set in `names`; `names: []` clears it. |
| `settings.labels` | Read all pages of `GET /repos/{owner}/{repo}/labels`, then perform only required label POST/PATCH operations.[^github-labels] |
| `settings.rulesets` | Read complete supported branch/tag state, then POST ID-less creates or PUT ID-selected whole objects; never implicitly DELETE unlisted objects. |

The version and outer YAML mappings are not API fields. False and empty strings are sent explicitly; omitted repository fields remain outside update requests. Selected labels send complete supported values, including description defaults. Apply rejects redirects, including on its label reads; update origin when a repository moves.

Repository fields/topics retain their direct-write behavior. Nonempty label selections use fresh observations and stable create/update order. A valid ID fixes update identity; a name collision without an ID fails before label writes. Updates may rename the same object. Current source performs no implicit label deletion.

Apply never consumes a saved plan. Concurrent remote changes can affect or invalidate an update; fresh checks and response verification are not a lock. Earlier writes remain applied after a later failure, with no automatic rollback/retry.

Success prints `Applied repository settings.` for managed work, followed by created IDs when applicable. Normal CLI apply adds new IDs to the original settings file; `--no-writeback` or `GITHUB_ACTIONS=true` disables this. Existing IDs are not replaced, comments are preserved as practicable and formatting may normalize. A changed/unsafe file or persistence failure after remote creation is a nonzero partial failure that reports recoverable IDs. Retain those IDs before retrying; no Git commit or push is made. Output errors can occur after remote/local changes succeed; created label/Ruleset IDs remain in stderr recovery diagnostics. Rulesets use the same writeback/opt-out mechanism as labels.

## Failures and limits for mutation

Local validation cannot establish remote permissions or server-side acceptance. Permission and API failures return a nonzero status.[^github-update-repository]

Updates have no transaction, automatic retry or rollback. A failed command can leave repository settings or topics updated when a later label read or mutation fails, and earlier label changes can remain applied. After a failed/cancelled apply or output error, inspect current GitHub settings before deciding whether to retry; retrying resends declared repository/topic values and computes label operations from another fresh read.

Contributors can find the [behavior specifications](features/README.md), [testing guide](contributing/testing.md) and [internal Action interfaces](contributing/actions-internals.md) separately.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including repository label listing, create/update/delete endpoints and fine-grained token permissions.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.

## Fork workflow export choices

`gh settings export --fork-approval` adds the public-repository contributor approval enum. `gh settings export --private-fork-workflows` adds the four private-repository execution/token/secrets/approval booleans. Both require existing Administration read credentials and preserve ordinary export requests when omitted. They compose with `--rulesets`, `--actions`, `--workflow-permissions`, `--retention` and distinct `--security` selections; duplicate flags fail before export. Selecting both fork families fails because their adopted visibility scopes differ, preserving an existing destination.

Plan and the credential-free Action Summary show native credential-exposure booleans and the exact approval enum. Matching apply skips these PUTs. A differing private partial declaration must explicitly include the required run flag; enabling run also requires the three companion choices. Incomplete/denied/inapplicable observations fail visibly, and direct-apply partial effects remain possible. See the [configuration contract](settings.md#fork-pr-workflow-policies) for exact values, applicability and failure behavior.

`export --pages` for [existing Pages configuration](settings.md#existing-pages-configuration). It is explicit, composable and rejects duplicate selection. Ordinary export adds no Pages requests. Complete Pages read access is required; a 404 or incomplete response fails without replacing the output. Site creation/deletion and visibility remain unsupported under #350.

