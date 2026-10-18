# Settings commands

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

This reference describes v0.10.0 and current source, supporting repository fields, topics and labels. The [pinned Action and examples](github-actions.md#published-package-and-consumer-pins) select v0.9.0; older v0.8.0 packages do not support labels.

v0.10.0 adds [merge-method booleans](features/merge-methods.feature.md); published v0.9.0 does not accept those keys. Their GraphQL query runs only for declared fields during plan and always during export. Existing repository PATCH behavior is retained.

## Interface and implementation status

```text
gh settings export
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
| Merge methods (v0.10.0) | GraphQL query; ordinary Actions Contents read verified | Administration write through REST PATCH |

Administration write alone does not grant label write permission.[^github-labels] Source export always reads labels as well as repository fields/topics; source apply uses its apply credential for the fresh label read and any required mutations. Public REST reads may be unauthenticated; GraphQL reads use an authenticated read token. Local validation and valid declarations with no managed resources do not require an origin or authentication.

## Topics command behavior

The [topics contract](settings.md#topics) defines names, omission and explicit clearing. Export includes the observed topics; plan compares the managed set; apply replaces that set. A configuration containing only `settings.topics.names` is a valid managed declaration. An omitted or empty topics mapping skips topics; the command is a local no-op only when no other resource is managed.

## Labels command behavior

The [labels contract](settings.md#labels) makes a declared list authoritative: labels absent from that list are deleted, and `labels: []` deletes all labels. Deletion removes issue/PR associations; recreating the label does not restore them. Omitted `labels` is unmanaged.

Plan shows explicit create/update/delete operations. Apply reads fresh labels after any repository/topics updates and performs only the needed operations. Omitted label attributes retain existing values; creating a label with omitted attributes delegates those values to GitHub. Changing a name to a different identity is create plus delete, without rename inference.

## Export behavior

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

Changed repository fields appear in this order: description, homepage, has_issues, has_projects, has_wiki, followed by topics and then label operations. Lines use `settings.repository.FIELD: CURRENT → DESIRED` or `settings.topics.names: CURRENT → DESIRED`:

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

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no resources are managed, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `names: []`, performs an update. A labels-only declaration, including `labels: []`, reads current labels and reconciles membership and declared attributes.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `settings.repository` | `PATCH /repos/{owner}/{repo}` with only declared repository fields.[^github-update-repository] |
| `settings.topics.names` | `PUT /repos/{owner}/{repo}/topics` with the complete declared topic set in `names`; `names: []` clears it. |
| `settings.labels` | Read all pages of `GET /repos/{owner}/{repo}/labels`, then perform required label POST/PATCH/DELETE operations.[^github-labels] |

The version and outer YAML mappings are not API fields. False and empty strings are sent explicitly; omitted repository fields and label attributes remain outside update requests. Apply rejects redirects, including on its label reads; update origin when a repository moves.

Repository fields and topics are sent without a preliminary read, even when the YAML values already match GitHub. Declared labels use a fresh read: creates/updates run in identity order, then deletes in identity order. Create sends the name and only declared attributes; update sends only changed declared attributes; delete removes each undeclared remote label. Names are escaped as individual URL path segments, and no rename or archive fields are sent.

Apply never consumes a saved plan. Remote changes after review can alter label operations and be overwritten for managed values or membership. Label deletion removes issue/PR associations, which are not restored by recreating the name. Review these semantics before applying.

Success prints `Applied repository settings.` and exits 0 for a managed declaration, including a topics-only update or a labels-only reconciliation with no required writes. File, validation, origin, authentication, API/response and output failures exit 1 with a diagnostic on stderr. The first failure stops later work; earlier successful updates remain applied. An output failure can occur after an update succeeds. Apply does not edit the configuration.

## Failures and limits for mutation

Local validation cannot establish remote permissions or server-side acceptance. Permission and API failures return a nonzero status.[^github-update-repository]

Updates have no transaction, automatic retry or rollback. A failed command can leave repository settings or topics updated when a later label read or mutation fails, and earlier label changes can remain applied. After a failed/cancelled apply or output error, inspect current GitHub settings before deciding whether to retry; retrying resends declared repository/topic values and computes label operations from another fresh read.

Contributors can find the [behavior specifications](features/README.md), [testing guide](contributing/testing.md) and [internal Action interfaces](contributing/actions-internals.md) separately.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including repository label listing, create/update/delete endpoints and fine-grained token permissions.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.
