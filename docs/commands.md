# Settings commands

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines supported repository fields, topics and configuration semantics.

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
| `plan` | Validate, then read managed resources and show differences for declared fields only. |
| `apply` | Validate, then send declared values directly to GitHub without reading current settings. |

There are no target override, configuration-path or output-format flags. Use `gh settings --help` for usage and `gh settings --version` for the installed version. Unsupported commands or arguments fail with exit status 1.

## Repository and authentication

Export and nonempty plan/apply operations resolve the checkout-scoped `origin` remote. Supported origins directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or a full SSH URL. Rewrite-only aliases and username-less `github.com:owner/repo` are rejected; configure a supported direct URL instead.

Only local/worktree origin values, including repository-specific included configuration, can select the target. System, global and command Git configuration cannot supply it. A direct GitHub origin retains its repository identity when Git `insteadOf` changes the transport destination. `GH_REPO` is ignored; missing or unsupported origins fail before API access.

API operations use GitHub CLI authentication. Check access with `gh auth status`; authenticate with `gh auth login` when needed. Reads require access to the target's metadata, while updates require Administration write for a fine-grained or App installation token.[^github-update-repository] Local validation and valid declarations with no managed fields do not require an origin or authentication.

## Topics command behavior

The [topics contract](settings.md#topics) defines names, omission and explicit clearing. Export includes the observed topics; plan compares the managed set; apply replaces that set. A configuration containing only `settings.topics.names` is a valid managed declaration. An omitted or empty topics mapping is a local no-op only when no repository fields are managed either.

## Export behavior

Export uses the [repository and authentication rules](#repository-and-authentication) above. It reads repository settings, then topics, and serializes the complete supported configuration before opening the destination. Unsupported fields and read-only metadata are excluded. Export preserves false and empty strings; the [settings reference](settings.md#exported-values) explains response normalization and invalid-response handling.

Export creates `.github` when missing and always overwrites an existing regular `.github/settings.yml`, without a confirmation or overwrite flag. Symlinks at `.github` or `settings.yml`, including dangling links, and other non-regular destinations are rejected. A read or serialization failure leaves the destination untouched. A write failure can leave it truncated or partially written; use Git to restore tracked configuration.

Success is silent and exits 0. Export changes no GitHub settings. Failures exit 1 and report diagnostics on stderr, with no stdout. Read the saved file after success; do not redirect export stdout, because the shell can truncate the destination before the command runs:

```sh
gh settings export
cat .github/settings.yml
```

## Validate behavior

Validate reads `.github/settings.yml` and checks the [configuration contract](settings.md#configuration-semantics). It requires one YAML root mapping with integer `schemaVersion: 1`, accepts partial declarations and empty mappings, and rejects unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple documents, and wrong shapes or value types. Alias and merge values are checked after YAML resolves them.

Success is silent and exits 0. Failures exit 1 and report a diagnostic on stderr, with no stdout. Validation requires a Git checkout but no GitHub origin or authentication. It makes no API requests and changes neither the file nor GitHub settings. Passing local validation does not guarantee that GitHub will accept an update.

## Plan behavior

Plan validates the configuration before resolving origin, configuring authentication or reading GitHub. A valid declaration with no managed repository fields or declared topic set prints `No managed repository settings.` and exits 0 locally without an origin, credentials or API calls. `names: []` is a managed declaration, so it still requires a comparison.

Plan reads only resources with declared fields, in repository-then-topics order. It uses the same target resolution and response handling as export. Only declared values are compared; false, empty strings and an empty topics list remain explicit values. The GitHub client may sanitize some API control characters to caret notation before comparison.

Changed repository fields appear in this order: description, homepage, has_issues, has_projects, has_wiki, followed by topics. Lines use `settings.repository.FIELD: CURRENT → DESIRED` or `settings.topics.names: CURRENT → DESIRED`:

```text
settings.repository.has_issues: true → false
settings.topics.names: ["old"] → ["github-cli","settings-as-code"]
```

Strings are double-quoted with Go string escaping, including quotes, backslashes, newlines and control characters. Booleans use `true` or `false`; topics use sorted JSON arrays. Equal managed values are omitted. When all managed values match, output is `No differences for managed repository settings.`

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding and output errors exit 1 with a diagnostic on stderr. Failures before output leave stdout empty. Plan changes neither configuration nor GitHub settings. Its output is an observation, not a saved execution plan or drift protection for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) separately requires target/commit metadata and explicit plan/apply tokens, including for empty declarations.

## Apply behavior

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no fields are managed across either repository settings or topics, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `names: []`, performs an update.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `settings.repository` | `PATCH /repos/{owner}/{repo}` with only declared repository fields.[^github-update-repository] |
| `settings.topics.names` | `PUT /repos/{owner}/{repo}/topics` with the complete declared topic set in `names`; `names: []` clears it. |

The version and outer YAML mappings are not API fields. False and empty strings are sent explicitly; omitted fields remain outside the request. Both update clients reject redirects; update origin when a repository moves.

Apply does not read current settings, reconcile a diff or consume a saved plan. It sends the YAML values even when they already match GitHub or when managed values changed remotely after plan. Review those overwrite semantics before applying.

Success prints `Applied repository settings.` and exits 0, including for a topics-only update. File, validation, origin, authentication, API/response and output failures exit 1 with a diagnostic on stderr. The first failure stops later work; earlier successful updates remain applied. An output failure can occur after an update succeeds. Apply does not edit the configuration.

## Failures and limits for mutation

Local validation cannot establish remote permissions or server-side acceptance. Permission and API failures return a nonzero status.[^github-update-repository]

Updates have no transaction, automatic retry or rollback. A failed command can leave repository settings updated even when a later topics update fails. After a failed/cancelled apply or output error, inspect current GitHub settings before deciding whether to retry; retrying sends the declared values again.

Contributors can find the [behavior specifications](features/README.md), [testing guide](contributing/testing.md) and [internal Action interfaces](contributing/actions-internals.md) separately.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.
