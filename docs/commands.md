# Settings commands

## v0.17.0 package

Published [v0.17.0](https://github.com/daiksud/gh-settings/releases/tag/v0.17.0) includes repository Rulesets (basic, PR review and required checks), PR/Issue access policies, Actions execution and default workflow permissions, Secret scanning/push protection, public private-vulnerability reporting, retention and fork workflow policies. Its native binaries and published Action passed the [release workflow](https://github.com/daiksud/gh-settings/actions/runs/39508410269). Dependabot, paid-product candidates and immutable-release policy remain unsupported. v0.16.0 already includes web commit sign-off, template repository status and default branch settings. The maintained examples select v0.17.0; adopting its pin does not expand caller grants or change settings by itself.

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

This reference describes v0.17.0 and current source, supporting repository fields, topics and labels. The [pinned Action and examples](github-actions.md#published-package-and-consumer-pins) select v0.17.0; older v0.8.0 packages do not support labels.

v0.10.0 adds [merge-method booleans](features/merge-methods.feature.md); published v0.9.0 does not accept those keys. Their GraphQL query runs only for declared fields during plan and always during export. Existing repository PATCH behavior is retained.

v0.11.0 also supports [`delete_branch_on_merge`](features/delete-branch-on-merge.feature.md). It joins the fixed merge-related GraphQL query, and v0.11.0 export saves nine repository fields. Published v0.10.0 retains its eight-field export and rejects this new key.

v0.12.0 also supports the [squash commit title/body defaults](features/squash-commit-defaults.feature.md). v0.12.0 export saves eleven repository fields; published v0.11.0 retains nine and rejects those two keys.

v0.13.0 adds [auto-merge and branch-update booleans](features/pull-request-options.feature.md). v0.13.0 export includes thirteen repository fields; published v0.12.0 retains eleven and rejects these keys.

v0.14.0 adds [merge commit title/body defaults](features/merge-commit-defaults.feature.md). v0.14.0 export includes fifteen repository fields; published v0.13.0 retains thirteen and rejects these keys.

v0.15.0 adds [`has_discussions`](features/discussions.feature.md). It adds a sixteenth exported repository field; published v0.14.0 rejects it. Apply verifies the returned boolean when this field is declared; verification errors can occur after repository writes.

Published v0.16.0 and current source also support the [`is_template` setting](settings.md#template-repository) and [`default_branch` setting](settings.md#default-branch). Source export includes twenty-three repository fields; published v0.15.1 and older packages reject the new key.

Current source also supports [Rulesets with basic, PR review and required check rules](settings.md#rulesets), including explicit `export --rulesets` opt-in. Published v0.16.0 and older packages reject the group and option. Incomplete or unsupported Rulesets are never partially exported or reconciled.

Published v0.17.0 also adds [PR access controls](settings.md#pull-request-access): `has_pull_requests` and `pull_request_creation_policy`. Published v0.16.0 and older packages reject them. Plan uses a separate complete GraphQL query; apply resolves repository identity and sends only declared PR values through `updateRepository`, after any existing REST update. No current-policy comparison is added to direct apply.

Published v0.17.0 also adds [`issue_creation_policy`](settings.md#issue-creation-policy). It has an independent narrow GraphQL plan read and shares declared-only repository mutations with PR access; it never enables Issues implicitly. That published package exports twenty-two repository fields. Published v0.16.0 and older packages reject the new key.

Published v0.17.0 also adds [Actions execution policy](settings.md#actions-execution-policy) and explicit `export --actions`. Normal export keeps its existing permissions; selected policy reads require Administration read. Published v0.16.0 and older packages reject this group and option.

Published v0.17.0 adds [default workflow permissions](settings.md#default-workflow-permissions). Its two fields are independent of execution policy; explicit `export --workflow-permissions` acquires them without expanding normal export or `--actions`. Published v0.16.0 and older packages reject the keys/option.

## Interface and implementation status

Current source also supports [`archived`](settings.md#repository-archive-lifecycle-unreleased-source) as an explicitly managed lifecycle boolean. Ordinary export excludes it; use `export --archive` to opt in. Unarchive precedes other writes and archive follows them, with identity and state verification. Published v0.17.0 and older packages reject the key and option.

Current source additionally supports unreleased [`has_sponsorships`](settings.md#sponsor-button-visibility-unreleased), increasing full source export to twenty-three repository fields. Plan reads its independent GraphQL boolean only when declared; apply shares the declared-only repository mutation and never edits funding destinations. Published v0.17.0 and older packages reject the key.

Published v0.16.0 and current source additionally support [`web_commit_signoff_required`](settings.md#web-commit-sign-off), an optional repository boolean. Published v0.15.1 and older packages reject it. It uses the existing repository REST GET/PATCH and appears after `has_discussions` in source plan output; the remaining field order is unchanged.

```text
gh settings export [--forking] [--custom-properties] [--pages] [--archive] [--rulesets] [--actions] [--workflow-permissions] [--retention] [--fork-approval] [--private-fork-workflows] [--security FAMILY]...
gh settings validate
gh settings plan
gh settings apply
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

The [labels contract](settings.md#labels) makes a declared list authoritative: labels absent from that list are deleted, and `labels: []` deletes all labels. Deletion removes issue/PR associations; recreating the label does not restore them. Omitted `labels` is unmanaged.

Plan shows explicit create/update/delete operations. Apply reads fresh labels after any repository/topics updates and performs only the needed operations. Omitted label attributes retain existing values; creating a label with omitted attributes delegates those values to GitHub. Changing a name to a different identity is create plus delete, without rename inference.

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

Plan validates the configuration before resolving origin, configuring authentication or reading GitHub. A valid declaration with no managed resources prints `No managed repository settings.` and exits 0 locally without an origin, credentials or API calls. Both `names: []` and `labels: []` are managed declarations and require a comparison.

Plan reads only declared resources, in repository-then-topics-then-labels order, completing every page of a declared label collection. It uses the same target resolution and response handling as export. Declared values and collection membership are compared; false, empty strings and empty topic/label lists remain explicit values. The GitHub client may sanitize some API control characters to caret notation before comparison.

Changed repository fields appear in this order: description, homepage, default_branch (v0.16.0), has_issues, has_projects, has_wiki, has_discussions (v0.15.0), web_commit_signoff_required (v0.16.0), is_template (v0.16.0), allow_merge_commit, allow_squash_merge, allow_rebase_merge, delete_branch_on_merge, allow_auto_merge, allow_update_branch, squash_merge_commit_title, squash_merge_commit_message, merge_commit_title, merge_commit_message, has_pull_requests, pull_request_creation_policy and issue_creation_policy (v0.17.0), then has_sponsorships, archived and allow_forking (all unreleased source), followed by topics and then label operations. Lines use `settings.repository.FIELD: CURRENT → DESIRED` or `settings.topics.names: CURRENT → DESIRED`:

```text
settings.repository.has_issues: true → false
settings.topics.names: ["old"] → ["github-cli","settings-as-code"]
```

Strings are double-quoted with Go string escaping, including quotes, backslashes, newlines and control characters. Booleans use `true` or `false`; topics use sorted JSON arrays. Equal managed values are omitted. When all managed values match, output is `No differences for managed repository settings.`

Label creates and updates appear in identity order, followed by deletes in identity order. Names and attribute values are quoted and escaped; deletes include the association-loss warning. For example, source plan can print:

```text
settings.labels: create "bug" color="d73a4a"
settings.labels: update "help wanted" description: "Old description" → ""
settings.labels: delete "obsolete" (removes issue/PR associations)
```

A declared label collection counts as one managed item, including an empty list. Label attributes omitted from an existing definition are not compared or updated.

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding and output errors exit 1 with a diagnostic on stderr. Failures before output leave stdout empty. Plan changes neither configuration nor GitHub settings. Its output is an observation, not a saved execution plan or drift protection for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) separately requires target/commit metadata and explicit plan/apply tokens, including for empty declarations.

## Apply behavior

v0.17.0 workflow permissions are applied after execution policy as a separate operation. Partial declarations send only their own keys; fresh complete readback verifies desired and preserved values. Failure retains earlier execution/resource writes and does not broaden existing credentials. See the [workflow-permission contract](settings.md#default-workflow-permissions).

v0.17.0 Actions policy is applied after existing repository/topics/labels/Rulesets operations. It uses fresh preflight and readback, preserves API-required and selected companion values, and enables a currently disabled repository only after any requested selected-list update. Endpoints are not transactional; see the [Actions policy ordering and failure contract](settings.md#actions-execution-policy).

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no resources are managed, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `names: []`, performs an update. A labels-only declaration, including `labels: []`, reads current labels and reconciles membership and declared attributes.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `settings.repository` | Existing fields use `PATCH /repos/{owner}/{repo}`.[^github-update-repository] PR access, Issue creation policy, and unreleased Sponsorships then use a shared ID lookup and declared-only GraphQL `updateRepository`; GraphQL-only declarations skip PATCH. |
| `settings.topics.names` | `PUT /repos/{owner}/{repo}/topics` with the complete declared topic set in `names`; `names: []` clears it. |
| `settings.labels` | Read all pages of `GET /repos/{owner}/{repo}/labels`, then perform required label POST/PATCH/DELETE operations.[^github-labels] |

The version and outer YAML mappings are not API fields. False and empty strings are sent explicitly; omitted repository fields and label attributes remain outside update requests. Apply rejects redirects, including on its label reads; update origin when a repository moves.

Repository fields and topics are sent without a preliminary comparison read, even when the YAML values already match GitHub. v0.17.0 PR access and Issue creation policy apply add only the required repository-ID lookup; it does not infer companion values. A later GraphQL failure leaves earlier REST writes applied. Declared labels use a fresh read: creates/updates run in identity order, then deletes in identity order. Create sends the name and only declared attributes; update sends only changed declared attributes; delete removes each undeclared remote label. Names are escaped as individual URL path segments, and no rename field is sent. The opt-in `allow_forking` field uses its own complete policy reads, declared-only PATCH and readback after other resources; it cannot compose with `archived`. The separately declared `archived` control is written in an isolated verified lifecycle PATCH, never merged into the ordinary repository update.

Apply never consumes a saved plan. Remote changes after review can alter label operations and be overwritten for managed values or membership. Label deletion removes issue/PR associations, which are not restored by recreating the name. Review these semantics before applying.

Success prints `Applied repository settings.` and exits 0 for a managed declaration, including a topics-only update or a labels-only reconciliation with no required writes. File, validation, origin, authentication, API/response and output failures exit 1 with a diagnostic on stderr. The first failure stops later work; earlier successful updates remain applied. An output failure can occur after an update succeeds. Apply does not edit the configuration.

## Failures and limits for mutation

Local validation cannot establish remote permissions or server-side acceptance. Permission and API failures return a nonzero status.[^github-update-repository]

Updates have no transaction, automatic retry or rollback. A failed command can leave repository settings or topics updated when a later label read or mutation fails, and earlier label changes can remain applied. After a failed/cancelled apply or output error, inspect current GitHub settings before deciding whether to retry; retrying resends declared repository/topic values and computes label operations from another fresh read.

Contributors can find the [behavior specifications](features/README.md), [testing guide](contributing/testing.md) and [internal Action interfaces](contributing/actions-internals.md) separately.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including repository label listing, create/update/delete endpoints and fine-grained token permissions.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.

## Fork workflow export choices

Current-source `gh settings export --fork-approval` adds the public-repository contributor approval enum. `gh settings export --private-fork-workflows` adds the four private-repository execution/token/secrets/approval booleans. Both require existing Administration read credentials and preserve ordinary export requests when omitted. They compose with `--rulesets`, `--actions`, `--workflow-permissions`, `--retention` and distinct `--security` selections; duplicate flags fail before export. Selecting both fork families fails because their adopted visibility scopes differ, preserving an existing destination. Published v0.16.0 and older packages reject both options.

Plan and the credential-free Action Summary show native credential-exposure booleans and the exact approval enum. Matching apply skips these PUTs. A differing private partial declaration must explicitly include the required run flag; enabling run also requires the three companion choices. Incomplete/denied/inapplicable observations fail visibly, and direct-apply partial effects remain possible. See the [configuration contract](settings.md#fork-pr-workflow-policies) for exact values, applicability and failure behavior.

Current unreleased source adds `export --pages` for [existing Pages configuration](settings.md#existing-pages-configuration-unreleased-source). It is explicit, composable and rejects duplicate selection. Ordinary export adds no Pages requests. Complete Pages read access is required; a 404 or incomplete response fails without replacing the output. Published v0.17.0 rejects the option.

## Custom property commands (unreleased source)

`export --custom-properties` explicitly selects the [bounded property-value contract](settings.md#custom-property-values-unreleased-source). It composes with existing export choices; duplicates and use on other commands fail. Published v0.17.0 rejects the option and YAML group. Ordinary export retains its existing reads and permissions.

Local validation checks value syntax without API access. Plan/apply additionally require complete organization schema and repository values, including authenticated GraphQL string constraints. Values use Metadata read; organization schema uses Custom properties read. Updates need repository Custom properties write plus schema editability. Incomplete or unsupported observations fail instead of producing partial output. No grants or organization schemas are changed.

Differences use `settings.custom_properties.NAME: CURRENT → DESIRED`, sorted by name, with JSON strings/arrays/null. A nonempty map counts as one managed collection. Apply uses fresh state rather than a saved plan, sends only declared names, and checks readback. Failed export preserves the destination; failed apply preserves earlier successful writes and reports an uncertain attempted update without rollback.

Current-source `gh settings export --forking` selects the [private repository forking policy](settings.md#private-repository-forking-unreleased-source). It requires complete matching organization-policy visibility in addition to private repository metadata. Ordinary export is unchanged; duplicate flags and composition with `--archive` fail locally. Published v0.17.0 rejects the option.
