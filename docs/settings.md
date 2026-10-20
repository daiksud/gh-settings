# Repository settings

## Scope and status

Configuration lives in `.github/settings.yml` at the Git checkout root and manages only that checkout's GitHub.com repository. v0.15.0 supports sixteen repository fields, topics and labels, using the same file-format version. v0.14.0 supports fifteen repository fields, excluding Discussions. v0.13.0 supports thirteen repository fields, excluding normal merge commit defaults. v0.12.0 supports eleven repository fields, excluding auto-merge and branch-update options. v0.11.0 supports nine repository fields, excluding squash commit defaults. The v0.10.0 package supports eight repository fields, excluding automatic branch deletion. The v0.9.0 package supports the original five repository fields, topics and labels. See the [command reference](commands.md) for behavior and the [Action package boundary](github-actions.md#published-package-and-consumer-pins) before adopting a configuration.

## Supported fields

The keys inside `settings.repository` follow GitHub's repository update request body parameters.[^github-update-repository] Discussions additionally uses the parameter sent by GitHub's official CLI. The unreleased PR-access fields use the explicit GraphQL-to-YAML mapping below.

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The booleans accept `true` or `false`; the string fields accept strings, including empty strings. Explicit null and other value types are invalid. All other repository fields and read-only response metadata are excluded, even when GitHub exposes them as writable parameters.

## Pull request access (unreleased)

Current source adds two independent `settings.repository` values. Published v0.16.0 and older CLI/Action packages reject them; use a source-built CLI until a supporting package is published. Checking out newer Action source still installs its selected published CLI.

| YAML field | Type / accepted values | GraphQL update input |
| --- | --- | --- |
| `has_pull_requests` | boolean | `hasPullRequestsEnabled` |
| `pull_request_creation_policy` | `ALL`, `COLLABORATORS_ONLY` | `pullRequestCreationPolicy` |

```yaml
schemaVersion: 1
settings:
  repository:
    has_pull_requests: true
    pull_request_creation_policy: COLLABORATORS_ONLY
```

Omission leaves each value unmanaged. Explicit false disables PR access; GitHub hides the PR tab and existing PRs and prevents new PR creation. `COLLABORATORS_ONLY` restricts creation to collaborators with write access, while existing PRs remain visible and commentable when PR access is enabled. These effects follow [GitHub's access controls](https://github.blog/changelog/2026-02-13-new-repository-settings-for-configuring-pull-request-access/). No individual PR is edited. Null, quoted booleans, empty strings, case variants and unknown policies are invalid. Declaring one setting never adds the other to a request, including when disabling PRs.

Plan reads a narrow authenticated GraphQL query only when either value is declared. Export includes both values after complete reads. Missing/null/wrong-type values, unsupported policies and any GraphQL errors reject the entire observation rather than inventing false or `ALL`. A disabled PR feature can still have an observed policy; it is preserved as an independent value. Hidden or unavailable data remains an error. Read failures preserve the prior export file.

Direct apply performs existing repository REST updates first, then resolves the checkout repository's opaque GraphQL ID and sends only declared PR values through `updateRepository`, before topics/labels/Rulesets. The ID is transport identity, never YAML configuration. A PR-only declaration performs no REST PATCH and does not read current policy for comparison. Repeated scalar apply still sends declared values even when already matching. The mutation response must retain the requested values and identify the same repository; failure can occur after writes have applied. Earlier successful REST writes remain applied if a later lookup or mutation fails; later resources stop without retry or rollback.

Reads require complete policy visibility; writes require a credential authorized to update the target repository. Authentication and caller grants are unchanged, and no permission is expanded automatically. Local fixtures and read-only observations do not prove live write compatibility, token permissions or published-package support. See the [feature specification](features/pull-request-access.feature.md) for test mappings and evidence limits.

## Default branch (unreleased)

Current source (planned for v0.16.0, not yet published) adds optional `settings.repository.default_branch`, a string selecting an existing branch as the repository default. Published v0.15.1 and older CLI/Action packages reject this key. It uses the existing `schemaVersion: 1` repository group. Source export includes twenty-one repository fields, topics and labels.

```yaml
schemaVersion: 1
settings:
  repository:
    default_branch: release/v2
```

Omission leaves the setting unmanaged. Empty strings, null, wrong types and invalid literal branch names fail local validation. Names follow Git branch naming rules, including slash-separated names; they are neither trimmed nor normalized. Revision expressions such as `@{-1}` are rejected, and no local branch or revision is resolved. Validation remains local without requiring an origin or authentication. GitHub handles branch existence and authorization when the update is requested; a missing branch or policy restriction remains an API error rather than causing branch creation.

The existing repository REST GET must contain a valid non-null branch name. Export includes the observed string after all reads succeed; missing or invalid data fails instead of becoming an empty string or an inferred `main`. Plan is read-only and compares declared values only, placing this string after homepage and before repository booleans. CLI and Actions display quoted string differences.

Apply sends only declared repository fields in one Administration-write PATCH, without a preliminary read. When `default_branch` is declared, the response must contain a valid string equal to the requested name. A missing, malformed or unequal response fails after the PATCH; repository writes may already have applied, later resources stop, and no rollback is attempted. Omission keeps existing PATCH behavior without requiring this response field.

This feature does not create or rename branches, change the local checkout/origin, retarget individual PRs, or edit protections. Existing policies targeting the default branch, including Rulesets using `~DEFAULT_BRANCH`, can follow the newly selected branch, so review those effects before applying. GitHub's [default-branch guidance](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-branches-in-your-repository/changing-the-default-branch) and [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) apply. The [feature contract](features/default-branch.feature.md) maps validation, requests and post-write verification to tests.

## Template repository (unreleased)

Current source (planned for v0.16.0, not yet published) adds optional `settings.repository.is_template`, a boolean controlling whether the repository can be used as a template. Published v0.15.1 and older CLI/Action packages reject this key. It uses the existing `schemaVersion: 1` repository group; no repository-generation command is added.

```yaml
schemaVersion: 1
settings:
  repository:
    is_template: true
```

Omission leaves the setting unmanaged; explicit `false` removes template status. Null, quoted booleans and other types are invalid. Plan compares a complete non-null boolean from the existing repository REST GET and remains read-only. Export includes the observed value after all reads succeed. Missing, null or malformed observations fail instead of becoming false.

Apply sends only declared fields through the existing Administration-write repository PATCH, with no preliminary read or GraphQL request. GitHub's [template prerequisites](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-template-repository), including the Git LFS restriction, still apply; API rejections remain errors rather than triggering workarounds. The tool does not create repositories, copy files, change template provenance, or manage template contents. See the [feature contract](features/template-repository.feature.md).

## Web commit sign-off (unreleased)

Current source (planned for v0.16.0, not yet published) adds optional `settings.repository.web_commit_signoff_required`, a boolean requiring sign-off on commits made through GitHub's web interface. It extends the repository group under `schemaVersion: 1`. Published v0.15.1 and older CLI/Action packages reject this key; use a source build to exercise it until a supporting package is released. Source export includes twenty-one repository fields, topics, and labels; published v0.15.x export retains sixteen repository fields.

```yaml
schemaVersion: 1
settings:
  repository:
    web_commit_signoff_required: true
```

Omission leaves the field unmanaged. Explicit `false` disables the requirement; null, quoted booleans and other types are invalid. Plan reads a complete non-null boolean from the existing repository REST GET, compares declared values only, and makes no GraphQL request for this field. Export includes the observed boolean after all reads succeed. Missing or wrongly typed values fail the read instead of becoming false.

Apply sends the field only when declared, in the existing repository PATCH with Administration write permission and no preliminary read. No new token grant is introduced. This is a sign-off requirement, not cryptographic commit-signature enforcement or a DCO check. It does not rewrite commits or change branch protections. The [feature contract](features/web-commit-signoff.feature.md) maps the request, failure and presentation tests. GitHub's [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents the exact field name.

## Discussions (v0.15.0)

v0.15.0 adds optional `settings.repository.has_discussions`, a boolean controlling GitHub Discussions. It extends the repository group; it does not manage discussion categories or individual discussions. Published v0.14.0 and older packages reject this key. The pinned v0.15.0 CLI/Action supports it; update older consumer pins before adopting the declaration.

```yaml
schemaVersion: 1
settings:
  repository:
    has_discussions: false
```

Omission leaves the value unmanaged; explicit `false` remains a declared value. Null, quoted booleans, and other types are invalid. Export includes the observed value only after complete reads; v0.15.0 exports contain sixteen repository fields, topics, and labels. Existing configurations that omit this field keep their previous managed count.

Plan reads the value from the existing repository REST GET without an additional GraphQL query. Apply sends it only when declared, through the existing Administration-write PATCH. When declared, a successful PATCH must return the same boolean; a missing, null, wrongly typed, or unequal value is an error. This check occurs after the request and does not roll back repository writes or prove transactional application. No prerequisite read, implicit enabling, or additional permission is introduced.

GitHub's [official CLI sends this field through repository PATCH](https://github.com/cli/cli/blob/6fcf2ea52718ad1ef8b12130f07430269e871352/pkg/cmd/repo/edit/edit.go#L76-L94), although the public REST update schema does not currently list it. The [feature contract](features/discussions.feature.md) describes strict reads, response verification, and test boundaries.

## Merge methods (v0.10.0)

v0.10.0 adds optional `allow_merge_commit`, `allow_squash_merge` and `allow_rebase_merge` booleans under `settings.repository`. Omission remains unmanaged, explicit false is retained, and null/wrong types fail validation. These fields extend the repository group; they are not a new resource family. Published v0.9.0 does not accept them.

The [merge-method contract](features/merge-methods.feature.md) defines their narrow GraphQL read path, strict complete-read handling, unchanged REST PATCH semantics and read-only Actions evidence. Plan reads them only when declared; v0.10.0 export includes all three after a complete read. Review exported configuration before using an older pinned CLI/Action.

## Automatic branch deletion (v0.11.0)

v0.11.0 adds optional `settings.repository.delete_branch_on_merge`, a boolean controlling GitHub's automatic deletion of head branches after pull requests merge. Omission leaves it unmanaged, explicit false is preserved, and null/non-boolean input is invalid. It extends the existing repository group; it does not manage or delete individual branches.

v0.11.0 export includes the value after a complete read. Plan uses the same fixed GraphQL read as merge methods; apply sends only declared fields through the existing Administration-write REST PATCH, with no preliminary read. Missing/null/invalid data or any GraphQL error fails the read instead of becoming false. See the [feature contract and read-only Actions evidence](features/delete-branch-on-merge.feature.md).

Published v0.10.0 does not accept this key. The pinned v0.15.0 CLI/Action includes it; older consumer pins must be updated before adopting this declaration.

## Squash commit defaults (v0.12.0)

v0.12.0 adds two optional string enums to `settings.repository`, using `schemaVersion: 1`:

| Field | Exact accepted values |
| --- | --- |
| `squash_merge_commit_title` | `PR_TITLE`, `COMMIT_OR_PR_TITLE` |
| `squash_merge_commit_message` | `PR_BODY`, `COMMIT_MESSAGES`, `BLANK` |

```yaml
schemaVersion: 1
settings:
  repository:
    squash_merge_commit_title: COMMIT_OR_PR_TITLE
    squash_merge_commit_message: COMMIT_MESSAGES
```

These select GitHub's default squash commit title/body, not whether squash merging is allowed. `PR_TITLE` uses the PR title; `COMMIT_OR_PR_TITLE` uses a single commit's title or a multi-commit PR's title. The message choices use the PR body, branch commit messages, or a blank body respectively. `BLANK` is a literal enum, not an empty YAML string.

Omission leaves each field unmanaged. Title-only declarations are valid; message declarations require an explicit title in the same repository mapping, matching the REST API contract. Message-only input fails local validation before target/authentication/API access. Null, empty strings, wrong types, case variants and unknown enum values are rejected. There is no implicit title, `allow_squash_merge` update, or extra requirement to declare that boolean.

Export includes both values only after a complete read. Plan uses the fixed merge-related GraphQL query, compares declared fields in title-then-message order after branch deletion, and presents string values. Apply includes only declared fields in the existing Administration-write repository PATCH, without a preliminary read or inferred companion values. Existing token permissions are sufficient for that PATCH; no additional resource family is introduced. The [feature contract](features/squash-commit-defaults.feature.md) records read validation, hosted evidence and test mappings.

Published v0.11.0 rejects these keys. The pinned v0.15.0 CLI/Action supports them; update older consumer pins before adopting these declarations.

## Auto-merge and branch updates (v0.13.0)

v0.13.0 adds optional `allow_auto_merge` and `allow_update_branch` booleans to `settings.repository`, using the existing `schemaVersion: 1` format:

```yaml
schemaVersion: 1
settings:
  repository:
    allow_auto_merge: true
    allow_update_branch: true
```

`allow_auto_merge` controls whether people may enable auto-merge for individual PRs in the repository. It does not schedule, enable or cancel auto-merge on a PR, merge a PR, or bypass its merge requirements. GitHub [plan availability](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-auto-merge-for-pull-requests-in-your-repository) applies: public repositories on Free, and public/private repositories on Pro, Team or Enterprise.

`allow_update_branch` permits updating a PR head branch that is behind its base even when up-to-date status is not required for merging. It does not itself update branches or alter protections.

Each field is independent: omission remains unmanaged, explicit `false` is retained, and null/wrong types fail local validation. Neither value is synthesized. Export includes both after a complete fixed GraphQL read; plan reads only when a merge-related field is declared and compares declared fields only. Missing/null/wrong-typed observations or partial GraphQL errors fail the complete read. Apply sends declared fields only through the existing Administration-write REST PATCH, with no preliminary read or additional permissions.

Published v0.12.0 rejects these keys. The pinned v0.15.0 CLI/Action supports them; update older consumer pins before adopting these declarations. The [feature contract](features/pull-request-options.feature.md) records exact API mapping, hosted proof and test coverage.

## Merge commit defaults (v0.14.0)

v0.14.0 adds optional `merge_commit_title` and `merge_commit_message` string enums to `settings.repository` under `schemaVersion: 1`:

| Field | Exact values |
| --- | --- |
| `merge_commit_title` | `PR_TITLE`, `MERGE_MESSAGE` |
| `merge_commit_message` | `PR_BODY`, `PR_TITLE`, `BLANK` |

```yaml
schemaVersion: 1
settings:
  repository:
    merge_commit_title: MERGE_MESSAGE
    merge_commit_message: PR_TITLE
```

These select default title/body formatting for normal merge commits. Title uses the PR title or GitHub's classic merge message; body uses the PR body, PR title, or a blank body. `BLANK` is an enum, not an empty YAML string. These keys do not enable `allow_merge_commit`, change squash defaults, or merge PRs. They can be managed while normal merge commits are disabled.

Omission remains unmanaged; title-only is valid. Message requires an explicit `merge_commit_title` in the same mapping, matching GitHub's REST contract; a squash title cannot satisfy that dependency. Null, empty strings, wrong types, case variants and unknown enums fail local validation before API access. The CLI never infers a missing title or toggles a merge method.

Export reads both values in the fixed complete GraphQL observation. Plan compares declared fields only; missing/null/invalid data or partial GraphQL errors fail the whole read. Apply sends declared values only through the existing Administration-write repository PATCH, with no preliminary read or implicit companion values. See the [feature contract](features/merge-commit-defaults.feature.md) for exact API mapping, hosted evidence and tests.

Published v0.13.0 rejects these keys. The pinned v0.15.0 CLI/Action supports them; update older consumer pins before adopting these declarations.

## Topics

`settings.topics.names` manages the complete repository topic set. Topics have been supported since [v0.8.0](https://github.com/daiksud/gh-settings/releases/tag/v0.8.0), using the same `schemaVersion: 1` format.

```yaml
schemaVersion: 1
settings:
  topics:
    names:
      - github-cli
      - settings-as-code
```

`names` follows the [replace-topics request body](https://docs.github.com/en/rest/repos/repos#replace-all-repository-topics): an array of unique lowercase strings using letters, numbers and hyphens, each 1–50 characters long, with at most 20 entries. Uppercase names, null, scalars, non-string elements and duplicates fail local validation. GitHub's [topic naming rules](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/classifying-your-repository-with-topics) apply to public and private repositories; topic names are public metadata even on private repositories.

| Declaration | Meaning |
| --- | --- |
| Omitted `topics`, or `topics: {}` | Topics unmanaged; plan and apply skip that resource. |
| `names: [github-cli]` | Manage the complete topic set, replacing any other remote topics. |
| `names: []` | Explicitly clear all topics. |
| `names: null` | Invalid; never interpreted as omission or clearing. |

Names are compared as a set. Export, plan and update requests use sorted lists without rewriting an existing input configuration. An empty observed set is exported as `names: []`, which becomes an explicit managed declaration.

Topic reads require Metadata read; public resources may be read without authentication. Writes require Administration write. The [command reference](commands.md#apply-behavior) explains request ordering and partial failures when repository settings and topics are declared together.

## Labels

`settings.labels` manages repository label definitions, supported since v0.9.0 and included in the pinned v0.15.0 Action examples. Older v0.8.0 packages do not support Labels. It does not manage individual issue/PR assignments, although deleting a definition removes its assignments.

```yaml
schemaVersion: 1
settings:
  labels:
    - name: bug
      color: "d73a4a"
      description: Something is not working
    - name: help wanted
```

A declared list owns the complete label membership:

| Declaration | Meaning |
| --- | --- |
| Omitted `labels` | Labels unmanaged; plan/apply make no label request. |
| A list of label mappings | Keep the declared identities, creating or updating as needed; delete every undeclared remote label. |
| `labels: []` | Delete all labels and their issue/PR associations. |
| `labels: null` or `labels: {}` | Invalid; neither represents omission or an empty collection. |

Deleting a label removes it from issues and pull requests. Recreating the same name does not restore those associations.[^github-delete-labels] Review the create/update/delete plan before applying authoritative membership.

| Attribute | Contract |
| --- | --- |
| `name` | Required string that is neither empty nor whitespace-only, with no control characters. Spaces, punctuation and emoji are allowed. |
| `color` | Optional string of six hexadecimal characters without `#`; normalized to lowercase. |
| `description` | Optional string of at most 100 Unicode characters; `description: ""` clears an existing description. |

Names use Unicode simple case folding for identity, without trimming. Duplicate or ambiguous identities in either configuration or remote observations fail. Case-only spelling changes preserve the remote spelling. A name changed to a different identity creates a new label and deletes the old label; there is no rename inference or `new_name` field, so old associations are lost.

Omitting `color` or `description` preserves an existing label's value. For a new label, omitted attributes are left for GitHub to choose; the tool synthesizes no defaults. Null, wrong types, unsupported attributes and duplicate keys fail validation. Labels follow the existing YAML alias and merge rules. Archive settings and read-only label metadata are not managed.

Listing labels needs Issues or Pull requests read, and mutations need either permission at write level for fine-grained or GitHub App tokens; public lists may be read unauthenticated.[^github-labels] Administration write alone is insufficient for label mutations. v0.9.0 export reads labels too. The existing example tokens and workflow permissions have not been expanded for label mutations. Public zero-difference adoption needs no label writes; later mutations require separately approved App/installation and token permissions.

The [command reference](commands.md#apply-behavior) explains fresh reads, create/update/delete ordering and partial failures. Contributors can find API mapping and test evidence in the [labels feature specification](features/labels.feature.md).

## Configuration semantics

The document requires integer `schemaVersion: 1`, a file-format version independent of the CLI release. The `settings` mapping is optional and can contain optional `repository` and `topics` mappings plus an optional `labels` sequence. Unknown root keys, unsupported versions and unsupported groups fail validation; there is no automatic migration or compatibility parsing.

Declared values are managed, and YAML is authoritative for them. Omitting a resource leaves it unmanaged; declared topics and labels collections own their complete membership. Omitted repository fields and omitted label attributes remain unmanaged. No defaults are injected by the tool. For example:

```yaml
schemaVersion: 1
settings:
  repository:
    has_issues: false
  topics:
    names:
      - github-cli
      - settings-as-code
```

This declaration manages `has_issues` and the complete topic set. Description, homepage, projects, wiki and labels remain unmanaged. The version and outer YAML mappings are not sent as API fields; the repository update body contains only `{"has_issues": false}` and the topics request contains the declared `names` array.

The supported repository fields distinguish omission from explicit values:

| YAML under `settings.repository` | Meaning |
| --- | --- |
| Field omitted | Unmanaged; apply leaves it out of the request. |
| `has_issues: false` | A declared boolean value. |
| `description: ""` | A declared empty string; single-quoted empty strings are equivalent. |
| `description: null` or `has_issues: null` | Invalid for these string and boolean fields. A bare `description:` is also YAML null and invalid. |

The same type rules apply to `homepage`, `has_projects`, `has_wiki` and `has_discussions` (v0.15.0). Quoted `'null'` is a string value, not YAML null. Null is invalid for the version, mappings, topic list and label declarations too.

A version-only document manages nothing:

```yaml
schemaVersion: 1
```

Empty `settings`, `repository` or `topics` mappings are also valid. When no resources are managed, plan and apply succeed locally without an origin, authentication or API requests. An explicit `names: []` still manages topics; `labels: []` still manages all label membership. These declarations require a comparison or update rather than a local no-op.

Validation rejects unknown or duplicate keys, non-string keys, wrong types, malformed YAML and multiple documents. YAML aliases and merge keys are resolved before supported values are checked. Partial declarations are accepted; unsupported root-level field names are rejected. See [validate behavior](commands.md#validate-behavior) for output and failure handling.

## Exported values

Export declares every supported observed repository field and the complete topic set. Export additionally reads all label pages and saves the complete label collection sorted by identity, with names, lowercase colors and descriptions, including `labels: []` when none exist. This makes the observed membership authoritative. Review the saved file and remove resources or attributes you want to leave unmanaged before applying it.

GitHub can return null for an unset description or homepage. Export normalizes those response values to empty strings and writes quoted YAML strings (`""`); it preserves existing empty strings and false booleans. Missing response fields and null booleans are errors. Topic names must be present as a valid array; missing, null, invalid or unreadable names fail export instead of becoming an empty set.

For labels, response descriptions may be null and are normalized to empty strings. Missing required label fields, invalid names/colors/descriptions, ambiguous identities or a failed later page fail the whole read. All reads and serialization complete before export opens the destination.

This response normalization does not make explicit YAML null valid input. The published read schema permits null for description/homepage, while the update schema defines optional strings without `nullable`.[^github-repository-openapi] The [export command](commands.md#export-behavior) documents file replacement and write-failure behavior.

## Rulesets (unreleased)

Current source adds `settings.rulesets` for repository-owned branch/tag Rulesets with `deletion`, `non_fast_forward`, `required_linear_history`, `required_signatures`, `pull_request` and `required_status_checks` rules. Published v0.16.0 and older CLI/Action packages reject this group and the new export option. Use a source-built CLI for these capabilities until a supporting package is published; checking out newer Action source still installs its selected published CLI.

```yaml
schemaVersion: 1
settings:
  rulesets:
    - name: release-tags
      target: tag
      enforcement: active
      conditions:
        ref_name:
          include: ["refs/tags/v*"]
          exclude: []
      bypass_actors: []
      rules:
        - type: deletion
```

This creates or reconciles the whole owned branch/tag collection. It is not an additive example to apply alongside undeclared existing Rulesets. Start with `gh settings export --rulesets`, review all exported members, and run plan before editing or applying.

| Declaration | Meaning |
| --- | --- |
| Omitted `rulesets` | Unmanaged; plan/apply make no Rulesets requests. |
| `rulesets: []` | Delete every owned branch/tag Ruleset after complete supported reads. |
| Nonempty list | Create missing names, update retained members, delete undeclared owned branch/tag members. |
| Omitted attribute of retained member | Preserve the fresh observed value. |
| Declared `rules` / `bypass_actors` | Authoritative inner membership, including removal; `[]` clears the collection. |
| Declared ref `include` / `exclude` | Replace that entire pattern list; omission preserves the other list. |
| `null`, unknown keys or wrong types at any depth | Invalid; null is neither omission nor clearing. |

Ruleset identity uses exact names without trimming, case folding or rename inference. Duplicate names are errors, including across targets. Changing a name means creating the new object and deleting the old one. Numeric Ruleset IDs are observed for PUT/DELETE and never declared in YAML. Inner rules use unique `type`; bypass actors use `(actor_type, actor_id)`. Empty mappings manage no child attributes and clear nothing. Basic rules have no parameters; `parameters: {}` is equivalent to omission, while nonempty parameters are unsupported.

New Rulesets require explicit name, branch/tag target, enforcement (`active`, `disabled`, or `evaluate`), both ref condition lists, bypass actor list and rule list. Existing members may declare only a name. A newly added bypass actor requires `bypass_mode`; omission preserves the mode of a retained actor. Modes are `always`, `pull_request` (branch, non-DeployKey actors only) and `exempt`. Actor types are `Integration`, `RepositoryRole`, `Team`, `User`, `OrganizationAdmin` and `DeployKey`; the first four require a positive `actor_id`, while the latter two omit it in YAML. `OrganizationAdmin` is unavailable for personal repositories. The adapter explicitly maps DeployKey IDs to API null. GitHub validates plan/account prerequisites; errors do not trigger bypass expansion, defaults or workarounds.

### PR review and required check parameters

`pull_request` rules require a branch target; required status checks can also target tags. A retained rule may omit `parameters` or declare only changed attributes; omitted siblings retain fresh observed values. Explicit `false`, `0` and valid empty lists are managed values. Declared reviewer, dismissal-actor and status-check lists own membership, so omitted members are removed. Null is invalid at every YAML depth.

| Rule | Parameters |
| --- | --- |
| `pull_request` | Required for a new rule: `dismiss_stale_reviews_on_push`, `require_code_owner_review`, `require_last_push_approval`, `required_approving_review_count` (nonnegative integer), and `required_review_thread_resolution`. Boolean values are explicit; no companion defaults are invented. |
| `pull_request` optional attributes | `allowed_merge_methods` (nonempty unique list of `merge`, `squash`, `rebase`), `dismissal_restriction`, `required_reviewers`, and boolean `require_extra_approval_for_unattributed_changes`. |
| `required_status_checks` | Required for a new rule: `required_status_checks` and boolean `strict_required_status_checks_policy`. Optional boolean `do_not_enforce_on_create` preserves omission. |

`dismissal_restriction` contains boolean `enabled` and `allowed_actors`, required when introducing the restriction. Each actor has a positive numeric `id` and `type` of `User`, `Team`, `IntegrationInstallation` or `RepositoryRole`. Its identity is `(type, id)`; retained restriction attributes may be omitted independently.

Each `required_reviewers` entry identifies `reviewer: {type: Team, id: 123}`. New entries require integer `minimum_approvals` from 0 through 10 and `file_patterns`; retained entries preserve either omitted attribute. At most 15 teams are accepted. Patterns preserve exact order and repetitions, including negations: changing order changes policy. Other independent collections are canonicalized by identity; allowed merge methods are a set. GitHub enforces account and team prerequisites.

Each required check has a nonempty `context` and optional positive `integration_id`. Identity is the pair `(context, integration_id)`: an absent ID selects any source and differs from a particular App. Replacing a check list with an entry lacking the previous ID changes that identity; it does not preserve the previous binding. API null integration IDs export as omitted YAML IDs, while YAML null remains invalid. `required_status_checks: []` clears the check list; `required_reviewers: []` clears required teams.

For example, this partial declaration preserves other attributes of the retained `default` Ruleset and PR/check rules, but owns rule membership and the whole required-check list. Include every rule type you intend to retain:

```yaml
settings:
  rulesets:
    - name: default
      rules:
        - type: deletion
        - type: non_fast_forward
        - type: required_linear_history
        - type: pull_request
          parameters:
            required_approving_review_count: 1
        - type: required_status_checks
          parameters:
            required_status_checks:
              - context: test
                integration_id: 123
            strict_required_status_checks_policy: true
```

The extra-approval flag is a known explicit boolean from [GitHub's documented public-preview control](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#additional-approval-for-unattributed-copilot-pull-requests) and observed REST policy, although the published REST schema does not yet list it. The adapter preserves an observed value and sends an explicit declaration without inventing an absent default. Local fixtures and read-only observations do not establish live write compatibility. POST/PUT responses involving PR/check rules in either the old or new policy are checked against the complete sent policy, including removal of both rule types: missing, ignored or reset values fail with a warning that repository writes may already have applied, and later operations stop. Optional attributes omitted from the request may acquire API defaults. This is response verification, not rollback, an additional GET, or a guarantee against subsequent drift.

Opted-in export reads all pages and each owned Ruleset detail, saving complete writable branch/tag state in canonical order. Empty collections remain explicit. Push Rulesets are excluded with a diagnostic; inherited organization/enterprise Rulesets and legacy branch protection remain outside ownership and are never changed. These policies can still affect effective protection. Missing bypass visibility, unsupported rules, unknown writable fields, null/incomplete values, failed pages or unreadable details stop export/plan/apply rather than producing partial or empty state. Failed export preserves the previous file. Explicitly clearing a collection still requires a complete supported observation.

Plan is read-only and shows complete before/after policies, operation names and protection-removal warnings. The collection counts once as managed. Apply runs after repository/topics/labels, reads fresh Rulesets with the apply credential, prepares all operations before the first Rulesets mutation, and preserves omitted siblings in complete PUT bodies. Creates/updates run in name order, then deletes in name order. An unchanged policy causes no Rulesets write; a repeat apply after successful reconciliation also causes no Rulesets write absent concurrent changes. Earlier resources retain their own direct-write behavior.

The first API failure stops later operations; earlier writes can remain applied, failed writes have uncertain effect, and there is no transaction, rollback, automatic retry or saved-plan execution. Review plan, including nested removals, before authorizing apply. Concurrent edits can change the fresh apply result.

[Rulesets REST](https://docs.github.com/en/rest/repos/rules) lists Metadata read for GET and Administration write for mutations, but complete detail GET also needs write visibility to return bypass actors. Missing actors are an error, not an empty list. No token grant or App permission is expanded automatically. Branch/tag Rulesets are available for public repositories on Free and applicable public/private repositories on Pro/Team/Enterprise; evaluate requires Enterprise. [GitHub availability and layering](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) and [available rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) apply. Other rule types remain unsupported; the [feature contract](features/rulesets.feature.md) records implementation scope and test evidence.

## Unsupported settings

Pages and Actions settings are not supported. Rulesets support is limited to the current-source subset above; push and inherited Rulesets, and legacy branch protection are not managed. Organization settings and multi-repository management are outside the tool's scope. Contributor [architecture and future direction](contributing/architecture.md) explains design choices without extending the configuration contract.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-delete-labels]: [GitHub — Deleting a label](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/managing-labels#deleting-a-label), including removal from associated issues and pull requests.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including listing and mutation permissions.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.
