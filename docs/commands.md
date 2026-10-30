# Settings commands

## Environment export

`gh settings export --environments` includes all supported named Environment policies and composes with other export options. It requires complete Environment/custom-protection reads and preserves the existing output-file safety contract. Ordinary export does not read Environments.

`validate`, `plan` and `apply` accept the [Environment configuration contract](settings.md#environments); names select ordinary create-or-update, not numeric IDs. No Environment writeback occurs, regardless of `--no-writeback` or GitHub Actions mode. Unlisted objects remain untouched. Name-only entries send an empty upsert body. Omitted built-in values remain API-controlled; the client promises neither preservation nor reset. Inspect plan before apply.

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

```text
gh settings export [--environments] [--pages] [--archive] [--rulesets] [--actions] [--workflow-permissions] [--retention] [--fork-approval] [--private-fork-workflows] [--security FAMILY]...
gh settings validate
gh settings plan
gh settings apply [--no-writeback]
```

| Command | Contract |
| --- | --- |
| `export` | Read ordinary and explicitly selected settings and save `.github/settings.yml`. Existing configuration is not required. |
| `validate` | Check the configuration locally against the supported schema. |
| `plan` | Validate, then read managed resources and show comparisons/selected-object operations. |
| `apply` | Validate, then execute the declared families in order with their existing reconciliation/preflight rules. |

There are no target override, configuration-path or output-format flags. Use `gh settings --help` for usage and `gh settings --version` for the installed version. Unsupported commands or arguments fail with exit status 1.

## Repository and authentication

Export and plan/apply operations with managed resources resolve the checkout-scoped `origin` remote. Supported origins directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or a full SSH URL. Rewrite-only aliases and username-less `github.com:owner/repo` are rejected; configure a supported direct URL instead.

Only local/worktree origin values, including repository-specific included configuration, can select the target. System, global and command Git configuration cannot supply it. A direct GitHub origin retains its repository identity when Git `insteadOf` changes the transport destination. `GH_REPO` is ignored; missing or unsupported origins fail before API access.

The extension uses the authentication selected by GitHub CLI, normally the user's own authority. Check access with `gh auth status`; authenticate with `gh auth login` when needed. gh-settings has no separate credential store, does not mint credentials, and does not retry a denied API operation as another identity. Standard `gh` authentication selection (including explicitly configured environment credentials) remains GitHub CLI behavior, not an independent fallback in this extension. `validate` performs no API work and needs no authentication. For automation, use the separate [single-App Actions policy](github-actions.md#authentication-and-app-permissions), not a required App/PAT combination. Credentials need permissions for every resource the command reads or writes:

| Resource | Read permission | Write permission |
| --- | --- | --- |
| Repository fields and topics | Metadata read | Administration write |
| Labels | Issues or Pull requests read | Issues or Pull requests write |
| Merge methods; automatic branch deletion; squash commit defaults; auto-merge and branch updates; normal merge commit defaults | GraphQL query; ordinary Actions Contents read verified | Administration write through REST PATCH |

Administration write alone does not grant label write permission.[^github-labels] Source export always reads labels as well as repository fields/topics; source apply uses its apply credential for the fresh label read and any required mutations. Public REST reads may be unauthenticated; GraphQL reads use an authenticated read token. Local validation and valid declarations with no managed resources do not require an origin or authentication.

Plan does not mutate settings, but complete Ruleset bypass observation requires write access to the Ruleset. A read-only token can therefore be insufficient. Feature-specific API permissions and unverified App-integration boundaries are listed in the [permission table](github-actions.md#authentication-and-app-permissions); denied/missing reads remain errors, never empty/default/success.

## Topics command behavior

The [topics contract](settings.md#topics) defines names, omission and explicit clearing. Export includes the observed topics; plan compares the managed set; apply replaces that set. A configuration containing only `repository.about.topics` is valid managed intent. Omitted topics skip the resource; `[]` clears it and `{}` is invalid. The command is a local no-op only when no other resource is managed.

## Labels command behavior

The [labels contract](settings.md#labels) uses ID-based listed-object ownership: only listed objects are managed, and omitted/empty collections or unlisted objects cause no label writes. An ID selects an existing label; no ID means create-only and a name collision reports the existing ID.

Plan shows creates and ID-bound updates, including default-derived values. Color is required; omitted description defaults to empty on create and update. A supported rename keeps the ID. Apply records confirmed new IDs in the original YAML after each creation; `--no-writeback` or `GITHUB_ACTIONS=true` disables this. Created IDs remain visible for recovery, including when later remote operations or local writeback fail. There is no automatic commit, push, create retry or rollback.

## Export behavior

`export --retention` for the [repository retention policy](settings.md#check-run-status-artifact-and-log-retention). It composes with the other selections; duplicates fail before reads or file changes. Both configured days and the read-only maximum must be acquired completely; only days are exported. Retention reads occur after workflow permissions and before security families. Other export selections compose with it.

Current source adds `export --security secret-scanning`, selecting the two [Secret scanning controls](settings.md#secret-scanning-and-push-protection), and `export --security private-vulnerability-reporting` for the independent [public-repository reporting toggle](settings.md#private-vulnerability-reporting). Repeat `--security` with distinct families to include both. It composes with the existing selections; duplicate/missing/unknown families fail before reads or destination changes. Implemented families are `secret-scanning`, `private-vulnerability-reporting` and `immutable-releases`: `all`, Dependabot and product-level families are not accepted. Selected security reads occur after existing opt-in resources and must complete before saving. Ordinary export does not acquire security policy.

`export --workflow-permissions` independently includes complete default workflow-token/PR-approval policy. It composes with `--actions` and `--rulesets` in any order; duplicate/unknown options fail before export. All reads precede saving.

`export --actions` additionally reads complete repository Actions execution policy; active selected policy includes its complete selection. It can be combined with `--rulesets` in either order. Extra resources are read after labels (Rulesets before Actions when both are requested), and all reads finish before saving. Inactive selected values are omitted, never guessed. See the [Actions policy contract](settings.md#actions-execution-policy) for required credentials and transition limits.

`export --rulesets` additionally reads complete repository-owned branch/tag Rulesets. Ordinary export retains its existing requests/permissions and does not include Rulesets. Complete bypass visibility is required; unsupported/hidden data fails before saving. Excluded push Rulesets produce a diagnostic on stderr; export success otherwise stays silent. See the [Rulesets contract](settings.md#rulesets) before adoption.

Export uses the [repository and authentication rules](#repository-and-authentication) above. It reads repository settings, then topics, then every labels page, and serializes the complete supported configuration before opening the destination. Label reads use pages of 100 until a short page; a failed later page fails the operation before any file replacement. Unsupported fields and read-only metadata are excluded. Export preserves false and empty strings; the [settings reference](settings.md#exported-values) explains response normalization and invalid-response handling.

Export creates `.github` when missing and always overwrites an existing regular `.github/settings.yml`, without a confirmation or overwrite flag. Symlinks at `.github` or `settings.yml`, including dangling links, and other non-regular destinations are rejected. A read or serialization failure leaves the destination untouched. A write failure can leave it truncated or partially written; use Git to restore tracked configuration.

Success writes no stdout and exits 0. Explicit observation-normalization warnings and excluded push Ruleset diagnostics can appear on stderr; see the [workflow](settings.md#default-workflow-permissions), [commit-message](settings.md#commit-message-observation-normalization), [creation-policy](settings.md#creation-policy-observation-normalization), [public-fork](settings.md#fork-pr-workflow-policies) [Security](settings.md#secret-scanning-and-push-protection) and [Pages](settings.md#existing-pages-configuration) field semantics. Export changes no GitHub settings. Failures exit 1 and report diagnostics on stderr, with no stdout. Read the saved file after success; do not redirect export stdout, because the shell can truncate the destination before the command runs:

```sh
gh settings export
cat .github/settings.yml
```

## Validate behavior

Validate reads `.github/settings.yml` and checks the [configuration contract](settings.md#configuration-semantics). It requires one YAML root mapping with the exact string `version: daiksud/gh-settings@v0`, accepts partial UI declarations and empty repository/settings mappings, and rejects unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple documents, and wrong shapes or value types. Alias and merge values are checked after YAML resolves them. Old envelopes/API-shaped aliases are rejected without compatibility or migration. Public paths are defined by the current-source reference; published packages keep their own contracts.

Success is silent and exits 0. Failures exit 1 and report a diagnostic on stderr, with no stdout. Validation requires a Git checkout but no GitHub origin or authentication. It makes no API requests and changes neither the file nor GitHub settings. Passing local validation does not guarantee that GitHub will accept an update.

## Immutable release policy commands

`export --security immutable-releases` independently includes the repository
[immutable release policy](settings.md#immutable-release-policy), requiring
Administration read. Omitted selection adds no policy request. GET 404 is
interpreted as disabled without a permission discriminator; an authorization
404 can consequently be misreported as false. Other failures remain errors and
preserve the export destination. Owner enforcement is never exported as intent.

Plan compares only declared values without writes and rejects an owner-enforced
disable. Apply follows private vulnerability reporting, uses fresh preflight,
body-free PUT/DELETE, exact 204 and fresh readback; matching state is a no-op.
Administration write is required. A 404 readback uses the same explicit tradeoff.
Write/readback failures can leave earlier changes applied; no retry or rollback
occurs. Individual releases, tags and assets are outside this command's scope.

## Plan behavior

Plan validates the configuration before resolving origin, configuring authentication or reading GitHub. A valid declaration with no managed resources prints `No managed repository settings.` and exits 0 locally without an origin, credentials or API calls. `repository.about.topics: []` remains a managed clearing declaration. In current source, `labels: []` selects no objects and requires no label comparison.

Plan reads only declared resources, in repository-then-topics-then-labels order, completing every page of a declared label collection. It uses the same target resolution and response handling as export. Declared values and collection membership are compared; false, empty strings and empty topic/label lists remain explicit values. The GitHub client may sanitize some API control characters to caret notation before comparison.

About and General comparisons retain the existing stable field order, followed by Topics and selected-object operations. Unambiguous scalar/Topics paths use the current UI declaration paths:

```text
repository.settings.general.features.issues: true → false
repository.about.topics: ["old"] → ["github-cli","settings-as-code"]
```

One exact UI menu choice can produce two internal component comparisons. An out-of-UI merge/squash pair instead retains one warning and menu-level write-required decision with both raw strings, even if the declared choice equals its projected default. Complete UI rendering of compound choices and other families is deferred to [#481](https://github.com/daiksud/gh-settings/issues/481). Normalized Actions execution uses one UI radio decision with visible-only projection warnings; an observed disabled repository remains known Disable and keeps latent raw policy private. Their existing internal display paths/values are not accepted YAML aliases. Actions execution/workflow permissions, public-fork approval, Security status strings, creation policies, merge/squash menu pairs and complete Pages modes and legacy folders have explicit normalization policies; remaining families stay strict under [#480](https://github.com/daiksud/gh-settings/issues/480).

Strings are double-quoted with Go string escaping, including quotes, backslashes, newlines and control characters. Booleans use `true` or `false`; topics use sorted JSON arrays. Equal exact managed values are omitted; normalized projections retain their explicit write-required decisions. When all managed values match, output is `No differences for managed repository settings.`

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

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no resources are managed, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `repository.about.topics: []`, performs an update. A nonempty label selection reads current labels and preflights ID-selected object updates and ID-less creates. `labels: []` selects nothing.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `repository.about` and `repository.settings.general` scalars | Declared REST members use repository PATCH.[^github-update-repository] Declared merge/squash message menus additionally require selective complete GraphQL preflight and fresh raw readback; failed verification can retain other members in that PATCH. PR access, Issue permissions and Sponsorships share identity-checked, declared-only GraphQL updates; declared creation policies also use selective preflight and fresh same-ID raw readback. Earlier REST success remains partial progress if the later access substep fails; GraphQL-only intent skips PATCH. |
| `repository.about.topics` | Topics PUT with the complete sorted set; `[]` clears it. |
| `repository.labels` | Complete label reads, then required ID-selected PATCH or create-only POST.[^github-labels] |
| `repository.settings.rulesets` | Complete supported reads, then selected-object PUT/create-only POST; no unlisted deletion. |
| `repository.settings.environments` | Stable name-order upserts, then explicit typed pattern reconciliation; omitted built-ins are API-controlled. |
| `repository.settings.actions.general` | Execution policy, workflow permissions, retention, public-fork approval, then private-fork workflows. Each uses its adopted preflight/readback contract. |
| `repository.settings.security_and_quality.advanced_security` | Secret protection/push protection in dependency order, then private vulnerability reporting. |
| `repository.settings.general.releases` | Immutable release policy after private reporting and before Pages. |
| `repository.settings.pages` | Existing-site declared-only update with identity/visibility/readback checks. |

An explicit archive lifecycle surrounds these operations: unarchive first, final archive last. The [archive contract](settings.md#repository-archive-lifecycle) owns preflight and failure behavior.

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

Plan and the credential-free Action Summary retain native credential-exposure booleans and the existing internal approval tokens pending #481; public YAML uses the full UI choices. Matching apply skips these PUTs. A differing private partial declaration must explicitly include the required run flag; enabling run also requires the three companion choices. Incomplete/denied/inapplicable observations fail visibly, and direct-apply partial effects remain possible. See the [configuration contract](settings.md#fork-pr-workflow-policies) for exact values, applicability and failure behavior.

`export --pages` for [existing Pages configuration](settings.md#existing-pages-configuration). It is explicit, composable and rejects duplicate selection. Ordinary export adds no Pages requests. Complete Pages read access is required; a 404 or incomplete response fails without replacing the output. Site creation/deletion and visibility remain unsupported under #350.
