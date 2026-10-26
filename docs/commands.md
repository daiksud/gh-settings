# Settings commands

## v0.18.0 package

Published [v0.18.0](https://github.com/daiksud/gh-settings/releases/tag/v0.18.0), built from immutable source `6d3146b30860692f84dc54e32d746e6fbbb55c66`, adds private plan reporting, Sponsor visibility, archive lifecycle, existing-site Pages configuration, bounded custom-property values and private repository forking. Its native binaries and published Action passed the [release workflow](https://github.com/daiksud/gh-settings/actions/runs/39979971859). The maintained examples select v0.18.0; adopting its pin does not expand caller grants or change settings by itself.

## Environment export (unreleased source)

`gh settings export --environments` includes all supported named Environment policies and composes with other export options. It requires complete Environment/custom-protection reads and preserves the existing output-file safety contract. Ordinary export does not read Environments. Published v0.19.0 and earlier reject this option.

`validate`, `plan` and `apply` accept the [Environment configuration contract](settings.md#environments) in current source. Names select ordinary create-or-update, not numeric IDs. No Environment writeback occurs, regardless of `--no-writeback` or GitHub Actions mode. Unlisted objects remain untouched. Name-only entries reset supported settings to documented defaults; inspect plan before apply.

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

This reference describes v0.18.0 and current source. The [pinned Action and examples](github-actions.md#published-package-and-consumer-pins) select v0.18.0; older v0.8.0 packages do not support labels.

v0.10.0 adds [merge-method booleans](features/merge-methods.feature.md); published v0.9.0 does not accept those keys. Their GraphQL query runs only for declared fields during plan and always during export. Existing repository PATCH behavior is retained.

v0.11.0 also supports [`delete_branch_on_merge`](features/delete-branch-on-merge.feature.md). It joins the fixed merge-related GraphQL query, and v0.11.0 export saves nine repository fields. Published v0.10.0 retains its eight-field export and rejects this new key.

v0.12.0 also supports the [squash commit title/body defaults](features/squash-commit-defaults.feature.md). v0.12.0 export saves eleven repository fields; published v0.11.0 retains nine and rejects those two keys.

v0.13.0 adds [auto-merge and branch-update booleans](features/pull-request-options.feature.md). v0.13.0 export includes thirteen repository fields; published v0.12.0 retains eleven and rejects these keys.

v0.14.0 adds [merge commit title/body defaults](features/merge-commit-defaults.feature.md). v0.14.0 export includes fifteen repository fields; published v0.13.0 retains thirteen and rejects these keys.

v0.15.0 adds [`has_discussions`](features/discussions.feature.md). It adds a sixteenth exported repository field; published v0.14.0 rejects it. Apply verifies the returned boolean when this field is declared; verification errors can occur after repository writes.

Published v0.16.0 and current source also support the [`is_template` setting](settings.md#template-repository) and [`default_branch` setting](settings.md#default-branch). Current ordinary export includes twenty-three repository fields; published v0.15.1 and older packages reject the new key.

Current source also supports [Rulesets with basic, PR review and required check rules](settings.md#rulesets), including explicit `export --rulesets` opt-in. Published v0.16.0 and older packages reject the group and option. Incomplete or unsupported Rulesets are never partially exported or reconciled.

Published v0.17.0 also adds [PR access controls](settings.md#pull-request-access): `has_pull_requests` and `pull_request_creation_policy`. Published v0.16.0 and older packages reject them. Plan uses a separate complete GraphQL query; apply resolves repository identity and sends only declared PR values through `updateRepository`, after any existing REST update. No current-policy comparison is added to direct apply.

Published v0.17.0 also adds [`issue_creation_policy`](settings.md#issue-creation-policy). It has an independent narrow GraphQL plan read and shares declared-only repository mutations with PR access; it never enables Issues implicitly. That published package exports twenty-two repository fields. Published v0.16.0 and older packages reject the new key.

Published v0.17.0 also adds [Actions execution policy](settings.md#actions-execution-policy) and explicit `export --actions`. Normal export keeps its existing permissions; selected policy reads require Administration read. Published v0.16.0 and older packages reject this group and option.

Published v0.17.0 adds [default workflow permissions](settings.md#default-workflow-permissions). Its two fields are independent of execution policy; explicit `export --workflow-permissions` acquires them without expanding normal export or `--actions`. Published v0.16.0 and older packages reject the keys/option.

## Interface and implementation status

Published v0.18.0 supports [`archived`](settings.md#repository-archive-lifecycle) as an explicitly managed lifecycle boolean. Ordinary export excludes it; use `export --archive` to opt in. Unarchive precedes other writes and archive follows them, with identity and state verification. Published v0.17.0 and older packages reject the key and option.

Published v0.18.0 supports [`has_sponsorships`](settings.md#sponsor-button-visibility), increasing ordinary export to twenty-three repository fields. Archive and private forking are separate explicit selections and cannot be combined. Plan reads the Sponsor setting's independent GraphQL boolean only when declared; apply shares the declared-only repository mutation and never edits funding destinations. Published v0.17.0 and older packages reject the key.

Published v0.16.0 and current source additionally support [`web_commit_signoff_required`](settings.md#web-commit-sign-off), an optional repository boolean. Published v0.15.1 and older packages reject it. It uses the existing repository REST GET/PATCH and appears after `has_discussions` in source plan output; the remaining field order is unchanged.

```text
gh settings export [--pages] [--archive] [--rulesets] [--actions] [--workflow-permissions] [--retention] [--fork-approval] [--private-fork-workflows] [--security FAMILY]...
gh settings validate
gh settings plan
gh settings apply [--no-writeback]
```

| Command | Contract |
| --- | --- |
| `export` | Read all supported settings and silently save `.github/settings.yml`. Existing configuration is not required. |
| `validate` | Check the configuration locally against the supported schema. |
| `plan` | Validate, then read managed resources and show field differences and label operations. |
| `apply` | Validate, send repository fields/topics directly, then reconcile declared labels against a fresh read. |

There are no target override, configuration-path or output-format flags. Use `gh settings --help` for usage and `gh settings --version` for the installed version. Unsupported commands or arguments fail with exit status 1.

## Repository and authentication

Export and plan/apply operations with managed resources resolve the checkout-scoped `origin` remote. Supported origins directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or a full SSH URL. Rewrite-only aliases and username-less `github.com:owner/repo` are rejected; configure a supported direct URL instead.

Only local/worktree origin values, including repository-specific included configuration, can select the target. System, global and command Git configuration cannot supply it. A direct GitHub origin retains its repository identity when Git `insteadOf` changes the transport destination. `GH_REPO` is ignored; missing or unsupported origins fail before API access.

API operations use GitHub CLI authentication. Check access with `gh auth status`; authenticate with `gh auth login` when needed. Fine-grained and GitHub App tokens need permissions for every resource the command reads or writes:

| Resource | Read permission | Write permission |
| --- | --- | --- |
| Repository fields and topics | Metadata read | Administration write |
| Labels | Issues or Pull requests read | Issues or Pull requests write |
| Merge methods (v0.10.0); automatic branch deletion (v0.11.0); squash commit defaults (v0.12.0); auto-merge and branch updates (v0.13.0); normal merge commit defaults (v0.14.0) | GraphQL query; ordinary Actions Contents read verified | Administration write through REST PATCH |

Administration write alone does not grant label write permission.[^github-labels] Source export always reads labels as well as repository fields/topics; source apply uses its apply credential for the fresh label read and any required mutations. Public REST reads may be unauthenticated; GraphQL reads use an authenticated read token. Local validation and valid declarations with no managed resources do not require an origin or authentication.

## Topics command behavior

The [topics contract](settings.md#topics) defines names, omission and explicit clearing. Export includes the observed topics; plan compares the managed set; apply replaces that set. A configuration containing only `settings.topics.names` is a valid managed declaration. An omitted or empty topics mapping skips topics; the command is a local no-op only when no other resource is managed.

## Labels command behavior

Current unreleased source uses the [ID-based labels contract](settings.md#labels): only listed objects are managed, and omitted/empty collections or unlisted objects cause no label writes. An ID selects an existing label; no ID means create-only and a name collision reports the existing ID. Published v0.19.0 and earlier retain authoritative-list deletion.

Plan shows creates and ID-bound updates, including default-derived values. Color is required; omitted description defaults to empty on create and update. A supported rename keeps the ID. Apply records confirmed new IDs in the original YAML after each creation; `--no-writeback` or `GITHUB_ACTIONS=true` disables this. Created IDs remain visible for recovery, including when later remote operations or local writeback fail. There is no automatic commit, push, create retry or rollback. Review the migration instructions before using old partial declarations.

## Export behavior

Published v0.17.0 adds `export --retention` for the [repository retention policy](settings.md#check-run-status-artifact-and-log-retention). It composes with the other selections; duplicates fail before reads or file changes. Both configured days and the read-only maximum must be acquired completely; only days are exported. Retention reads occur after workflow permissions and before security families. Existing export choices are unchanged; published v0.16.0 and older packages reject `--retention`.

Current source adds `export --security secret-scanning`, selecting the two [Secret scanning controls](settings.md#secret-scanning-and-push-protection), and `export --security private-vulnerability-reporting` for the independent [public-repository reporting toggle](settings.md#private-vulnerability-reporting). Repeat `--security` with distinct families to include both. It composes with the existing selections; duplicate/missing/unknown families fail before reads or destination changes. Implemented families are `secret-scanning` and `private-vulnerability-reporting`: `all`, Dependabot and product-level families are not accepted. Selected security reads occur after existing opt-in resources and must complete before saving. Ordinary export requests and scope are unchanged; published v0.16.0 and older packages reject this option.

`export --workflow-permissions` independently includes complete default workflow-token/PR-approval policy after any other requested resources. It composes with `--actions` and `--rulesets` in any order; duplicate/unknown options fail before export. All reads precede saving.

`export --actions` additionally reads complete repository Actions execution policy; active selected policy includes its complete selection. It can be combined with `--rulesets` in either order. Extra resources are read after labels (Rulesets before Actions when both are requested), and all reads finish before saving. Inactive selected values are omitted, never guessed. See the [Actions policy contract](settings.md#actions-execution-policy) for required credentials and transition limits.

`export --rulesets` additionally reads complete repository-owned branch/tag Rulesets. Ordinary export retains its existing requests/permissions and does not include Rulesets. Complete bypass visibility is required; unsupported/hidden data fails before saving. Excluded push Rulesets produce a diagnostic on stderr; export success otherwise stays silent. See the [Rulesets contract](settings.md#rulesets) before adoption.

Export uses the [repository and authentication rules](#repository-and-authentication) above. It reads repository settings, then topics, then every labels page, and serializes the complete supported configuration before opening the destination. Label reads use pages of 100 until a short page; a failed later page fails the operation before any file replacement. Published v0.8.0 export stops after topics. Unsupported fields and read-only metadata are excluded. Export preserves false and empty strings; the [settings reference](settings.md#exported-values) explains response normalization and invalid-response handling.

Export creates `.github` when missing and always overwrites an existing regular `.github/settings.yml`, without a confirmation or overwrite flag. Symlinks at `.github` or `settings.yml`, including dangling links, and other non-regular destinations are rejected. A read or serialization failure leaves the destination untouched. A write failure can leave it truncated or partially written; use Git to restore tracked configuration.

Success is silent and exits 0. Export changes no GitHub settings. Failures exit 1 and report diagnostics on stderr, with no stdout. Read the saved file after success; do not redirect export stdout, because the shell can truncate the destination before the command runs:

```sh
gh settings export
cat .github/settings.yml
```

## Validate behavior

Validate reads `.github/settings.yml` and checks the [configuration contract](settings.md#configuration-semantics). It requires one YAML root mapping with integer `schemaVersion: 1`, accepts partial declarations and empty settings/repository/topics mappings, and rejects unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple documents, and wrong shapes or value types. Alias and merge values are checked after YAML resolves them.

Success is silent and exits 0. Failures exit 1 and report a diagnostic on stderr, with no stdout. Validation requires a Git checkout but no GitHub origin or authentication. It makes no API requests and changes neither the file nor GitHub settings. Passing local validation does not guarantee that GitHub will accept an update.

## Plan behavior

Plan validates the configuration before resolving origin, configuring authentication or reading GitHub. A valid declaration with no managed resources prints `No managed repository settings.` and exits 0 locally without an origin, credentials or API calls. `topics.names: []` remains a managed clearing declaration. In current source, `labels: []` selects no objects and requires no label comparison.

Plan reads only declared resources, in repository-then-topics-then-labels order, completing every page of a declared label collection. It uses the same target resolution and response handling as export. Declared values and collection membership are compared; false, empty strings and empty topic/label lists remain explicit values. The GitHub client may sanitize some API control characters to caret notation before comparison.

Changed repository fields appear in this order: description, homepage, default_branch (v0.16.0), has_issues, has_projects, has_wiki, has_discussions (v0.15.0), web_commit_signoff_required (v0.16.0), is_template (v0.16.0), allow_merge_commit, allow_squash_merge, allow_rebase_merge, delete_branch_on_merge, allow_auto_merge, allow_update_branch, squash_merge_commit_title, squash_merge_commit_message, merge_commit_title, merge_commit_message, has_pull_requests, pull_request_creation_policy and issue_creation_policy (v0.17.0), then has_sponsorships and archived (v0.18.0), followed by topics and then label operations. Lines use `settings.repository.FIELD: CURRENT → DESIRED` or `settings.topics.names: CURRENT → DESIRED`:

```text
settings.repository.has_issues: true → false
settings.topics.names: ["old"] → ["github-cli","settings-as-code"]
```

Strings are double-quoted with Go string escaping, including quotes, backslashes, newlines and control characters. Booleans use `true` or `false`; topics use sorted JSON arrays. Equal managed values are omitted. When all managed values match, output is `No differences for managed repository settings.`

Current-source label creates and updates appear in stable identity order, with no implicit deletes. Names and attribute values are quoted and escaped. For example:

```text
settings.labels: create "bug" color="d73a4a" description=""
settings.labels: update "help wanted" (id 123) description: "Old description" → ""
```

A nonempty label selection counts as one managed item; an empty selection manages nothing. The effective whole supported object is compared, including default-derived description clearing.

Current unreleased Ruleset operations show only selected whole supported objects, with numeric IDs and fixed-default changes. Unlisted Rulesets never appear as deletes. ID updates can rename; ID-less collisions fail instead of adopting existing names. See the [Rulesets migration](settings.md#plan-apply-export-and-migration) before upgrading from v0.19.0 or earlier.

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding and output errors exit 1 with a diagnostic on stderr. Failures before output leave stdout empty. Plan changes neither configuration nor GitHub settings. Its output is an observation, not a saved execution plan or drift protection for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) separately requires target/commit metadata and explicit plan/apply tokens, including for empty declarations.

## Apply behavior

v0.17.0 workflow permissions are applied after execution policy as a separate operation. Partial declarations send only their own keys; fresh complete readback verifies desired and preserved values. Failure retains earlier execution/resource writes and does not broaden existing credentials. See the [workflow-permission contract](settings.md#default-workflow-permissions).

v0.17.0 Actions policy is applied after existing repository/topics/labels/Rulesets operations. It uses fresh preflight and readback, preserves API-required and selected companion values, and enables a currently disabled repository only after any requested selected-list update. Endpoints are not transactional; see the [Actions policy ordering and failure contract](settings.md#actions-execution-policy).

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

Current-source `gh settings export --fork-approval` adds the public-repository contributor approval enum. `gh settings export --private-fork-workflows` adds the four private-repository execution/token/secrets/approval booleans. Both require existing Administration read credentials and preserve ordinary export requests when omitted. They compose with `--rulesets`, `--actions`, `--workflow-permissions`, `--retention` and distinct `--security` selections; duplicate flags fail before export. Selecting both fork families fails because their adopted visibility scopes differ, preserving an existing destination. Published v0.16.0 and older packages reject both options.

Plan and the credential-free Action Summary show native credential-exposure booleans and the exact approval enum. Matching apply skips these PUTs. A differing private partial declaration must explicitly include the required run flag; enabling run also requires the three companion choices. Incomplete/denied/inapplicable observations fail visibly, and direct-apply partial effects remain possible. See the [configuration contract](settings.md#fork-pr-workflow-policies) for exact values, applicability and failure behavior.

Published v0.18.0 adds `export --pages` for [existing Pages configuration](settings.md#existing-pages-configuration). It is explicit, composable and rejects duplicate selection. Ordinary export adds no Pages requests. Complete Pages read access is required; a 404 or incomplete response fails without replacing the output. Site creation/deletion and visibility remain unsupported under #350. Published v0.17.0 and older packages reject the option.

## Removed organization-only commands

Current source rejects `export --custom-properties`, `export --custom-property NAME` and `export --forking`. It also rejects `settings.custom_properties` and `settings.repository.allow_forking` during validate, plan and apply before remote work. Remove these declarations/options before upgrading. Remote values are left unmanaged rather than cleared. Published v0.18.0 and v0.18.1 retain full custom-property export and private repository forking; named custom-property export was unreleased. See the [configuration migration](settings.md#custom-property-values).
