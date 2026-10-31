---
type: Reference
title: Repository UI YAML
description: Define the current-source UI-oriented declaration, ownership, validation and API boundaries.
sources:
  - id: github-update-repository
    resource: https://docs.github.com/en/rest/repos/repos#update-a-repository
  - id: github-labels
    resource: https://docs.github.com/en/rest/issues/labels
  - id: github-repository-openapi
    resource: https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json
  - id: ui-inventory
    resource: contributing/ui-repository-mapping.md
---

## Scope and status

Current source reads and exports UI-oriented YAML in `.github/settings.yml` at the Git checkout root. That checkout's directly configured GitHub.com `origin` selects the repository. Public keys and finite choices use the English Web UI wording in snake_case; REST and GraphQL names are translated internally.[^ui-inventory]

```yaml
version: daiksud/gh-settings@v0
repository:
  about:
    description: My project
    website: https://example.com
    topics: [github-cli, settings-as-code]
  labels:
    - name: bug
      color: '#d73a4a'
      description: Something is not working
  settings:
    general:
      features: {issues: true}
```

The exact string `version: daiksud/gh-settings@v0` is required. The old `schemaVersion`/root `settings` envelope and API-shaped aliases are rejected. There is no compatibility decoder, migration or legacy-format fallback.

This reference describes current source, not the contract of every published CLI or pinned Action. Read the [published package boundary](github-actions.md#published-package-and-consumer-pins) before adoption. This change does not upgrade a pin or publish a release. [#480](https://github.com/daiksud/gh-settings/issues/480) still owns out-of-UI observation normalization/warnings; [#481](https://github.com/daiksud/gh-settings/issues/481) owns complete UI reporting. Out-of-UI Actions execution/workflow permission strings, public-fork approval policies, Security status strings, creation policies, complete merge/squash message pairs, complete Pages modes and legacy folders, Ruleset enforcement/Quality severity/bypass-mode/scanning Alerts strings, Required approvals and Team reviewer Approvals integers, Allowed merge methods lists and Environment deployment-mode pairs/ref-type strings use the explicit observation policies below; other out-of-UI observations still fail pending the remaining normalization work. Observed fallbacks never populate omitted desired fields. Unambiguous About/General scalars, Topics and normalized execution/workflow/fork/Security/menu/Pages decisions use current UI display paths. Compound choices and other reports can still expose the existing internal paths/values pending #481; the overall new reporting contract is not complete.

## Supported fields

`repository.about`, `repository.labels` and `repository.settings` are siblings. Settings contains `general`, `rulesets`, `actions.general`, `security_and_quality.advanced_security`, `pages` and `environments`. Unknown keys, including otherwise writable API fields, are rejected.

Under `repository.about`, `description` and `website` are strings, including `''`; `topics` is a direct string array. These translate to repository REST fields and the Topics endpoint. Nullable observed repository strings become empty strings through the existing reader; YAML null does not request that behavior.[^github-update-repository][^github-repository-openapi]

The following paths are relative to `repository.settings.general`:

| Path | Type |
| --- | --- |
| `template_repository` | boolean |
| `default_branch` | literal branch-name string |
| `features.wikis`, `features.issues`, `features.sponsorships`, `features.discussions`, `features.projects`, `features.pull_requests` | boolean |
| `features.issue_permissions.creation_allowed_by`, `features.pull_request_permissions.creation_allowed_by` | `all_users` or `collaborators_only` |
| `pull_requests.allow_merge_commits`, `pull_requests.allow_squash_merging`, `pull_requests.allow_rebase_merging` | boolean |
| `pull_requests.always_suggest_updating_pull_request_branches`, `pull_requests.allow_auto_merge`, `pull_requests.automatically_delete_head_branches` | boolean |
| `pull_requests.merge_commit.default_commit_message`, `pull_requests.squash_merge.default_commit_message` | UI menu choice below |
| `commits.require_contributors_to_sign_off_on_web_based_commits` | boolean |
| `danger_zone.archive_this_repository` | boolean; explicit export opt-in |
| `releases.enable_release_immutability` | boolean; explicit export opt-in |

Omitting a scalar leaves it unmanaged. Explicit false and empty strings remain intent. An explicitly disabled Issues/Pull requests feature cannot accompany its creation-permission choice. Likewise, an explicitly disabled merge method cannot accompany its commit-message menu. Omitting the enabling checkbox never implicitly enables it. Export hides choices behind explicitly disabled controls after validating the acquired raw values.

## Repository archive lifecycle

Use `repository.settings.general.danger_zone.archive_this_repository`. Omission makes no archive-specific request; false explicitly unarchives and true explicitly archives. Ordinary export omits it; `export --archive` includes the observed boolean only after complete reads.

```yaml
version: daiksud/gh-settings@v0
repository:
  about: {description: Active maintenance}
  settings:
    general:
      danger_zone: {archive_this_repository: false}
```

Plan validates positive repository ID, canonical owner/name and archive state, then completely observes other managed resources without writing. Apply validates current identity before writes. Explicit unarchive runs first; final archive runs only after every other update succeeds and a fresh identity/state check. Archive PATCH contains only the internal archive member, and response/readback must confirm ID, name and state. Matching archive-only intent makes no lifecycle write.

On an already archived target, archive=true combined with other managed settings fails apply before all writes, even if plan showed a match or difference. Explicitly unarchive before making other changes. Failed unarchive stops later work. A confirmed unarchive survives a later failure; final archive failure can leave earlier updates applied. A write/readback failure can leave lifecycle state uncertain. There is no rollback, automatic retry, temporary unarchive, transaction or identity lock. Keep an external recovery route if the archived repository's own Actions cannot run.

Existing Administration read/write grants and actor authorization are prerequisites. Local fixtures do not establish live permissions or lifecycle behavior. See the [lifecycle specification](features/repository-lifecycle.feature.md) and [ADR 0027](adr/0027-repository-lifecycle.md).

## Secret scanning and push protection

Use booleans at `repository.settings.security_and_quality.advanced_security.secret_protection` and `push_protection`.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    security_and_quality:
      advanced_security:
        secret_protection: true
        push_protection: true
```

Omission is unmanaged; false disables. Explicit secret-protection=false with push-protection=true fails locally. No parent is enabled implicitly. Push protection true with its parent omitted requires fresh observed secret protection=true. Disabling secret protection while push protection is active requires declaring push protection false too. Unknown raw companions cannot prove prerequisites; explicit intent must resolve an unsafe transition. Safely unmodified companions retain exact raw evidence for readback. Enable parent before child; disable child before parent.

Ordinary export omits this group. `export --security secret-scanning` reads both complete statuses. Plan is read-only. Apply preflights, makes narrow per-member repository updates in dependency order, rereads prerequisites and verifies requested values plus preserved companions. Exact canonical matching members are skipped. Upper policy can ignore an apparently successful update; readback mismatch fails. Later failure retains earlier confirmed writes and uncertain effects; no rollback or automatic retry occurs.

A complete successful read requires both status strings. Canonical `enabled` / `disabled` map to true / false; correctly typed unknown, empty or whitespace statuses use the named per-field boolean fallback `false` with exact original provenance. Missing/null members, invalid types and failed reads remain acquisition errors. An inconsistent projected false/true pair remains acquired but cannot be exported as a consistent declaration; selected export warns and preserves the destination.

Plan retains one write-required decision per declared normalized control, including false-to-false. Omitted controls remain unmanaged. Apply verifies fresh raw statuses for the requested member and preserved companion; unchanged unknown status cannot confirm a write, and a changed unknown companion fails verification. Each successful managed initial/preflight/readback observation can warn; unowned companion values are not disclosed. Detailed CLI/Summary use UI paths and native booleans, while status-only Summary suppresses values. Export saves effective booleans after complete conversion and never writes GitHub settings.

Existing entitlement, token grants and appropriate admin/organization-owner/security-manager visibility are required. API visibility alone does not prove writable access or licensing. No product is purchased, credential expanded, alert/pattern/bypass managed or scanning workflow created. Dependabot and unadopted security controls remain unsupported. See the [security specification](features/security.feature.md).

## Immutable release policy

Use `repository.settings.general.releases.enable_release_immutability`, a boolean. This General-page control is independent from Advanced Security. Ordinary export omits it; use `export --security immutable-releases`. Owner enforcement is read-only metadata, never YAML intent.

A complete successful observation requires both enabled and owner-enforced booleans. HTTP 404 is treated as disabled without an authorization-discrimination request. GitHub can also hide inaccessible resources behind 404, so this deliberate tradeoff can misreport an authorization-related 404 as disabled. It does not prove access and is not an exception for other endpoints.

An owner-enforced policy cannot be planned/applied as false. Apply runs after private vulnerability reporting and before Pages, reads fresh state, skips a match, sends body-free PUT/DELETE, requires 204 and verifies fresh readback. A post-DELETE 404 follows the same tradeoff. Failed writes/readback can already have changed the policy; no retry or rollback occurs. Individual releases, tags and assets are never mutated by this feature, and disabling the policy does not make existing immutable releases editable. Administration read/write and applicable role must already exist. See [the full contract and evidence limits](features/immutable-releases.feature.md).

## Private vulnerability reporting

Use boolean `repository.settings.security_and_quality.advanced_security.private_vulnerability_reporting`. Omission is unmanaged; false explicitly disables. It is independent from Secret protection. `export --security private-vulnerability-reporting` opts in; ordinary export does not read the endpoint.

This control applies to public repositories. Fresh visibility and complete status are required; private/internal applicability failures, malformed data, permission errors and redirects do not become disabled state. Apply reads fresh prerequisites, skips matching state, performs the narrow enable/disable operation and verifies readback. Earlier writes remain on later failure. No rollback, permission expansion or automatic retry occurs. See [private reporting](features/security.feature.md).

## Default workflow permissions

These fields belong to `repository.settings.actions.general`:

| Field | Choice / type |
| --- | --- |
| `workflow_permissions` | `read_and_write_permissions` or `read_repository_contents_and_packages_permissions` |
| `allow_github_actions_to_create_and_approve_pull_requests` | boolean |

Each is independent and optional. `export --workflow-permissions` includes both after complete acquisition without acquiring execution policy. Omission is unmanaged, false remains explicit, and unknown choices/null/wrong types fail locally. Apply uses declared-only workflow-policy members, complete fresh reads/readback and the existing direct-apply semantics. Organization refusal can leave an earlier execution-policy update applied. This declaration does not create tokens or waive a PR's approval requirements. See [workflow permissions](features/workflow-permissions.feature.md).

A complete successful workflow-policy GET with a correctly typed permission string outside the UI choices uses `read_repository_contents_and_packages_permissions` and emits a warning naming the YAML field, raw token and fallback. Missing/null members, wrong types and failed reads remain acquisition errors. Export saves the UI fallback without changing GitHub. Plan retains an actionable write-required comparison even if the desired choice equals that fallback; it does not claim independently observed zero drift. Apply sends only declared members and confirms the requested permission against a fresh raw readback. An unchanged out-of-UI token fails verification with possible write effects retained. An omitted permission remains unmanaged and is verified as an unchanged raw companion during approval-only writes, without replaying or printing the fallback.

## Fork PR workflow policies

Public-fork approval is `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`:

- `require_approval_for_first_time_contributors_who_are_new_to_github`
- `require_approval_for_first_time_contributors`
- `require_approval_for_all_external_contributors`

A complete successful public-fork policy GET with an out-of-UI string projects to `require_approval_for_first_time_contributors` (native `first_time_contributors`). Original unknown, empty or whitespace strings remain exact provenance. Missing/null members, wrong types, unsupported visibility and failed reads remain errors. Export warns and saves only the UI choice; plan retains one write-required decision even when desired intent equals the fallback. Omitted intent remains unmanaged.

Apply checks exact raw values before skipping a write and in fresh readback after PUT. A retained unknown value fails verification with possible write effects; a canonical exact match remains a no-op. Each successful managed apply read can warn: the initial application read, updater preflight and a normalized readback. Pure plan readers emit no stderr; detailed plan/Summary carry the decision warning, while status-only Summary suppresses values.

Use `export --fork-approval` for that public-repository control. Private repository fork controls are under `repository.settings.actions.general.fork_pull_request_workflows`; use `export --private-fork-workflows`:

| Checkbox | Type |
| --- | --- |
| `run_workflows_from_fork_pull_requests` | boolean |
| `send_write_tokens_to_workflows_from_fork_pull_requests` | boolean |
| `send_secrets_and_variables_to_workflows_from_fork_pull_requests` | boolean |
| `require_approval_for_fork_pull_request_workflows` | boolean |

Visibility/applicability and complete observations are required; internal repositories are not silently classified. Omitted controls remain unmanaged. A differing private companion needs an explicitly declared run-workflows value; enabling a currently disabled group requires all four values so latent settings are not guessed or replayed. Matching partial intent can remain a no-op. Export omits companions hidden behind disabled run-workflows while retaining their actual raw state internally.

Narrow payloads, readback, redirect rejection and partial-failure reporting remain required. Upper policy may refuse or ignore changes. No retry/rollback or credential expansion occurs. These settings affect execution security; successful configuration does not establish effective enforcement. See [fork workflows](features/fork-workflows.feature.md).

## Check, run, status, artifact and log retention

Use integer `repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention` (1–400 days). It translates to the existing retention endpoint. `export --retention` opts in; ordinary export does not acquire retention permissions. The server's maximum is read-only metadata and is never replayed as intent. Existing repository visibility, range and freshly observed cap checks apply; a plan/readback cannot invent missing cap information or prove licensing. Apply verifies the requested value and preserves failure/uncertainty reporting. See [retention](features/retention.feature.md).

## Actions execution policy

All paths below are relative to `repository.settings.actions.general`:

| Field | Choice / type |
| --- | --- |
| `actions_permissions` | `allow_all_actions_and_reusable_workflows`, `disable_actions`, `allow_owner_actions_and_reusable_workflows`, `allow_owner_and_select_non_owner_actions_and_reusable_workflows` |
| `require_actions_to_be_pinned_to_a_full_length_commit_sha` | boolean |
| `allow_actions_created_by_github` | boolean; selected-policy companion |
| `allow_actions_by_marketplace_verified_creators` | boolean; selected-policy companion |
| `allow_or_block_specified_actions_and_reusable_workflows` | ordered string array; selected-policy companion |

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        actions_permissions: allow_owner_and_select_non_owner_actions_and_reusable_workflows
        require_actions_to_be_pinned_to_a_full_length_commit_sha: true
        allow_actions_created_by_github: true
        allow_actions_by_marketplace_verified_creators: false
        allow_or_block_specified_actions_and_reusable_workflows: ['my-org/*', '!my-org/blocked@*']
```

Selecting an allow radio explicitly enables Actions and selects its policy; `disable_actions` declares only disabled execution, leaving latent allowed policy unmanaged. There are no separate public API-shaped enabled/allowed fields. Omitting the radio does not implicitly enable Actions. SHA pinning is independent; reusable workflows can still use tags.

Companions can accompany the selected non-owner radio or an omitted radio, not another explicit radio. An omitted radio requires an already selected observed policy for companion updates. Transition from a nonselected policy requires all three companions: inactive values cannot be acquired or invented. Plan shows them as unavailable before the transition. Existing selected policy permits partial declarations merged with complete fresh companions.

Patterns are at most 1000 nonempty strings. An explicit array replaces the entire sequence, and `[]` clears it. Case, order, duplicates, wildcards and negative entries are preserved; the tool does not emulate matching. Removing a negative entry may permit more actions. GitHub documents pattern applicability limits; configuring a list alone does not prove enforcement on private repositories.

`export --actions` includes the selected radio, SHA control and visible acquired companions. Disabled/other policy hides selected companions; complete raw acquisition is still required. Normal export does not read this family. Failed or incomplete acquisition preserves the destination. Correctly typed unknown execution policies follow the observation policy below.

Apply reads fresh state and preflights before writing. The base policy endpoint requires enabled state; omitted radio uses the freshly observed value. Policy changes precede selected writes. When enabling a disabled repository with selection, it first selects policy while disabled, installs companions, then enables. Requested values and sent companions are verified with fresh reads. Repeated direct apply can still write declared values.

Operations are not transactional: an already enabled target can temporarily use remembered/default selection between writes. Earlier policy changes survive later refusal; failed writes/readback have uncertain effects. There is no rollback, saved-plan execution, automatic retry, lock or permission expansion. Existing Administration read/write and organization/enterprise policy remain prerequisites; policy changes can stop future workflows. See [execution policy](features/actions-execution-policy.feature.md).

## Sponsor button visibility

Use `repository.settings.general.features.sponsorships`. This boolean controls the Sponsor button, not funding destinations. Omission is unmanaged; false disables. `.github/FUNDING.yml` is never created, read, changed or deleted by this control. Ordinary export includes the observed value through the existing shared GraphQL path. See [Sponsorships](features/sponsorships.feature.md).

### Execution-policy observation normalization

A complete, correctly typed unknown `allowed_actions` string (including empty or whitespace) retains its exact raw value and projects to the named `allow_all_actions_and_reusable_workflows` fallback. Both acquired booleans remain unchanged. With execution enabled, export warns and emits that fallback; a declared radio keeps one UI decision actionable even at fallback equality. Disable owns one native field; enabling owns two, or five with all three selected companions. Native counts remain internal pending #481.

With execution disabled, the visible choice is the known `disable_actions`, so the latent unknown policy produces no projection warning. Export omits latent choices, and a matching Disable plan has no differences. Enabling requires an explicit canonical policy; selected intent still needs all three companions and configures them before enabling. Private radio reports retain raw provenance for closed report validation/presentation but detailed output does not disclose hidden policy; status-only output suppresses all policy values.

Apply sends only declared policy/SHA members with the endpoint-required enabled value. Disable and SHA-only updates preserve the exact raw policy; SHA-only updates do not warn about omitted radio intent. Fresh raw readback must confirm declared canonical policy or preserved raw policy, both booleans and any managed selected companions. An unchanged unknown string cannot confirm a declared fallback, and a different unknown token cannot confirm preservation. Failed verification retains uncertainty about writes and earlier completed resource work. Direct repeated apply retains its existing write behavior. Missing/null/wrong types and failed acquisition never become fallbacks.

## Creation-policy observation normalization

Complete successfully acquired Issue/PR creation-policy strings outside their two UI choices project to the named `all_users` fallback with native value `ALL`. Empty and whitespace strings remain exact acquired evidence. One scalar record retains its YAML path, original token and fallback. Missing/null members, wrong types, failed/partial reads and incomplete PR availability remain errors. Repository IDs/names retain strict identity validation.

Export warns only for visible normalized policies and saves UI choices; disabled feature controls hide their latent policy without warnings or replay. Plan retains one write-required field decision even when desired intent equals the projected default. Omitted desired policies remain unmanaged. Apply sends only declared values, validates the mutation response and verifies fresh raw policy values against the original repository ID. An unchanged unknown token or changed ID fails verification with possible write effects retained. A failed access preflight makes no access mutation; earlier REST writes in the same repository operation can already be applied and remain partial progress. No rollback occurs. Checkbox-only access writes retain their existing two-request identity/mutation behavior.

## Issue creation policy

Use `repository.settings.general.features.issue_permissions.creation_allowed_by`: `all_users` or `collaborators_only`. It can be declared independently with the Issues checkbox omitted; it never implicitly enables Issues. Explicit Issues=false plus this choice is invalid. Export omits it behind disabled Issues. Existing declared-only GraphQL writes and independent observation remain; it does not manage existing issues or access grants. See [Issue permissions](features/issue-creation-policy.feature.md).

## Pull request access

Use boolean `repository.settings.general.features.pull_requests` and independent choice `features.pull_request_permissions.creation_allowed_by` under the same General root. Choices are `all_users` and `collaborators_only`. Explicit false plus a permission choice is invalid; omitted enabling checkbox is independent. Export hides its policy when Pull requests is disabled. Shared identity-checked GraphQL mutations contain only declared values. Mutation responses confirm declared members; declared policy strings also require fresh same-ID raw readback. No collaborator invitation/grant is implied. See [PR access](features/pull-request-access.feature.md).

## Default branch

`repository.settings.general.default_branch` selects an existing branch using a valid literal name; it does not create, rename or move a branch. Empty, whitespace/ref-syntax-invalid or wrong-type values fail local validation. Existing existence/readback and target checks remain authoritative. See [default branch](features/default-branch.feature.md).

## Template repository

`repository.settings.general.template_repository` is an optional boolean. It controls template availability without adding repository generation or copying. See [template repository](features/template-repository.feature.md).

## Web commit sign-off

`repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits` is an optional boolean. It controls the web-based commit sign-off requirement, not signature verification or existing commits. See [sign-off](features/web-commit-signoff.feature.md).

## Discussions

`repository.settings.general.features.discussions` is an optional boolean, independent from discussion categories and individual discussions. Complete typed observation is required; absent values cannot become false. See [Discussions](features/discussions.feature.md).

## Merge methods

Use General's `pull_requests.allow_merge_commits`, `allow_squash_merging` and `allow_rebase_merging` booleans. These independent optional checkboxes do not create individual merges. Explicit false remains intent. See [merge methods](features/merge-methods.feature.md).

## Automatic branch deletion

General's `pull_requests.automatically_delete_head_branches` controls GitHub's automatic deletion after merge; it does not manage or delete individual branches directly. See [branch deletion](features/delete-branch-on-merge.feature.md).

## Commit-message observation normalization

A complete successful GraphQL read with two strings outside a merge/squash menu's UI choices projects to that menu's `default_message`. One provenance record retains both exact raw strings, its YAML menu path and fallback. Missing/null members, wrong types and failed or partial acquisition remain errors. Hidden disabled menus are omitted from export; visible normalized menus warn and save only the UI choice. Omitted desired menus remain unmanaged without warnings or replay.

Plan retains one warning and one menu-level write-required decision even when the declaration equals the projected default. That projection does not prove remote equality. Apply selectively reads declared menus, sends only declared native members in PATCH and verifies both raw members through a fresh GraphQL read. Unchanged out-of-UI readback fails with possible write effects retained. A failed preflight makes no repository PATCH; a later verification failure does not roll back other members in that PATCH. This does not complete the broader reporting audit in #481.

## Squash commit defaults

General's `pull_requests.squash_merge.default_commit_message` accepts `default_message`, `pull_request_title`, `pull_request_title_and_commit_details` or `pull_request_title_and_description`. One UI choice translates to both internal API members; partial API pairs are not public YAML. Explicitly disabling squash while declaring the menu fails. An omitted enabling checkbox remains omitted. See [squash defaults](features/squash-commit-defaults.feature.md).

## Auto-merge and branch updates

General's `pull_requests.allow_auto_merge` and `always_suggest_updating_pull_request_branches` are independent optional booleans. Existing platform prerequisites still apply. See [PR options](features/pull-request-options.feature.md).

## Merge commit defaults

General's `pull_requests.merge_commit.default_commit_message` accepts `default_message`, `pull_request_title` or `pull_request_title_and_description`. It translates to a complete API pair. Explicitly disabling merge commits with this menu fails locally. Existing runtime dependency validation also rejects menu intent on an observed disabled merge method unless the declaration enables it. It never enables the method implicitly. See [merge defaults](features/merge-commit-defaults.feature.md).

## Topics

`repository.about.topics` manages the complete topic set. It is a direct array, not a mapping. Omission leaves it unmanaged; `[]` explicitly clears all topics; `{}` and null are invalid. Names are unique lowercase letters/numbers/hyphens, 1–50 characters each, at most 20 entries. Names are public metadata even on private repositories. Export, comparison and PUT use sorted set order without rewriting existing input YAML. Metadata read and Administration write must already exist. See [Topics and ordering](commands.md#apply-behavior).

## Labels

`repository.labels` selects whole label objects by positive REST `id`; no ID means create-only. Omission or `[]` selects nothing, makes no label request and deletes nothing. Unlisted objects and assignments remain untouched; removing an entry stops managing it.

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive integer; selects an existing label in the target repository; missing ID errors, never create fallback |
| `name` | Required nonempty/non-whitespace string without control characters; spaces, punctuation and emoji allowed |
| `color` | Required `#` plus six hex digits; quote the string; canonical lowercase, no guessed default |
| `description` | Optional string, at most 100 Unicode characters; omission defaults to `''` on create/update, clearing an existing description |

Duplicate IDs and Unicode-simple-folded names fail without trimming. ID-less collisions show the existing ID for explicit adoption. A valid ID can rename to a noncolliding name while preserving assignments; case-only differences keep remote spelling. Renames cannot collide even with another planned rename; apply is not a rename transaction. Export requires complete IDs/settings and pagination, includes only writable supported values and sorts labels consistently. Incomplete reads fail rather than becoming empty state.

### Apply ID writeback

Normal CLI apply records each confirmed created Label/Ruleset ID in the original current-schema collection immediately after creation, before another write. Rulesets use `ruleset_name` as their selector. Existing IDs and unrelated values remain; comments are retained where supported and formatting may normalize. Regular `.github` and file entries plus the initial byte snapshot and earlier ID additions are required. Aliased/merged collection entries are not rewritten.

`apply --no-writeback` disables this local update. `GITHUB_ACTIONS=true` always disables it, including public Action use. Created IDs remain in output/recovery diagnostics; no Git commit or push is performed. Plan never writes. A confirmed creation survives writeback/later remote failure; add its ID before retrying. Unverified responses do not supply a confirmed writable ID. There is no automatic create retry, delete or rollback. Guarded file replacement is atomic, not a concurrent-editor lock. Label reads/mutations need the existing Issues or Pull requests grants; Administration alone does not replace them.[^github-labels] See [Labels](features/labels.feature.md).

## Configuration semantics

The version marker is independent of the CLI release. Only one YAML mapping document is accepted. `repository` is optional; version-only, `repository: {}` and empty settings mappings are local no-ops without origin/authentication/API requirements. A present collection entry must still satisfy its complete object rules.

Unknown/duplicate/non-string keys, unsupported tags, malformed/multiple documents, wrong shapes, invalid choices and wrong scalar types fail before remote work. Aliases/merges are validated after resolution. Strings must remain strings, booleans must be true/false and integers must fit their declared range. Explicit null is invalid except the nullable Pages custom-domain scalar. Empty strings, zero, false and empty arrays retain their distinct contracts. No desired defaults are populated from observation metadata or planned #480 fallbacks.

Scalar omission is unmanaged. Labels/Rulesets own only listed whole objects with their stated defaults. Environment omissions are API-controlled, not promised preservation. Topics and explicitly declared inner lists have their authoritative collection semantics. These distinctions remain the same through CLI and public Action use.

## Exported values

Export completes every requested read and conversion before replacing the destination. Complete failures do not produce plausible partial/empty YAML. Incomplete data remains an acquisition failure. Correctly typed out-of-UI Security statuses follow their [Security policy](#secret-scanning-and-push-protection), and public-fork approval strings follow their [fork policy](#fork-pr-workflow-policies). Complete workflow permission strings, merge/squash message pairs and creation-policy strings follow their [workflow](#default-workflow-permissions), [menu](#commit-message-observation-normalization) and [creation-policy](#creation-policy-observation-normalization) fallback policies. Complete Pages modes and legacy folders follow their [raw-source policy](#existing-pages-configuration); Actions execution follows its [visible/latent policy](#execution-policy-observation-normalization). Ruleset enforcement follows its [ID-bound policy](#ruleset-enforcement-observation-normalization); Environment deployment modes follow their [raw-pair policy](#environment-deployment-mode-observation-normalization), and pattern ref types follow their [raw-identity policy](#environment-ref-type-observation-normalization). Other unknown API-only choices still fail pending the remaining #480 work. Ordinary export includes About, Labels and supported visible General controls; archive and additional settings require explicit options. Hidden conditional controls are omitted without replaying latent API values as selected intent. IDs, false/zero/empty values and nullable Pages domains retain their declared meanings.

An export followed by validate/plan roundtrips exact supported acquired values, subject to the write-only and API-controlled omission limits below. A normalized fallback does not establish raw equality. Export does not claim live entitlement, effective protection or transactional apply. It overwrites existing declaration content, including write-only intent it cannot export; review before adoption.

## Rulesets

`repository.settings.rulesets` selects whole repository-owned branch/tag Rulesets by optional positive numeric ID. No ID means create-only; a collision requires explicit ID adoption. Unknown/wrong-target/type IDs fail without create fallback. Unlisted objects are not deleted. Omission/`[]` selects nothing. Names compare exactly without trimming/case folding; duplicate names/IDs and colliding renames fail.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 123
        ruleset_name: release-tags
        enforcement_status: active
        target_tags: {include_by_pattern: ['v*']}
        rules: {restrict_deletions: true}
```

Every entry requires `ruleset_name`, `enforcement_status` (`active` or `disabled`) and exactly one of `target_branches`/`target_tags`. API-only evaluate is not a selectable UI choice. Branch targets support `include_default_branch`, `include_all_branches`, `include_by_pattern` and `exclude_by_pattern`; tag targets support `include_all_tags` and the pattern arrays. Patterns use the UI spelling without internal ref prefixes. An explicit empty target object selects empty include/exclude lists; omitting both target objects is invalid. No protected scope is guessed.

Omitted `bypass_list` and unselected `rules` become empty policy on an owned object. Basic rule booleans are `restrict_deletions`, `block_force_pushes`, `require_linear_history` and `require_signed_commits`; false/omission leaves that rule unselected. The other four groups are `require_a_pull_request_before_merging`, `require_status_checks_to_pass`, `require_code_scanning_results` and `require_code_quality_results`. Omitting a rule group removes it; it does not preserve an existing group.

A bypass entry selects exactly one actor: `role` (`repository_admin`, `maintain`, `write`), positive `team`, `app` or `user`, or `deploy_keys: true`. `bypass_mode` is required and must be `always`, `exempt` or `for_pull_requests_only`; existing branch/actor applicability constraints apply. No actor/grant/credential is created. Organization-only actors and unadopted rules are unsupported.

### Ruleset enforcement observation normalization

A complete, correctly typed enforcement string outside Active/Disabled, including API-only `evaluate`, unknown, empty or whitespace strings, projects to the named `disabled` observation fallback. Preserve the exact raw token and its positive Ruleset ID outside desired policy. Export warns with the UI field, fallback, ID and name, in exported name order. Missing/null/wrong types, incomplete nested policy, ownership/ID failures and incomplete pagination remain acquisition errors without partial export.

A selected normalized ID requires one whole-object update even when desired Disabled equals the projection. Plan/detailed Summary show the contextual warning, a UI enforcement decision marked normalized/write-required and the complete effective supported policy. The effective object is not a raw observation. Unlisted IDs remain unmanaged, undisclosed and unwritten; desired whole-object defaults, name/ID collision guards and write-only dismissal limits are unchanged. Same-version private reports carry closed validation/presentation evidence and never supply apply input. Existing Rulesets collection counts remain pending the #481 audit.

Apply uses fresh collection preflight and exact desired whole-object payloads. Normalized updates always require an independent post-write GET, including basic Rulesets without review/analysis rules. Both mutation response and GET must retain canonical desired policy, correct ID and repository ownership; neither verification normalizes unknown values into success. A lying canonical acknowledgment followed by retained `evaluate`/unknown GET fails with uncertain effects. Visible fresh diagnostics never supply verification values. Canonical equal objects without explicit write-only dismissal intent remain no-ops; earlier completed resource writes remain reported on later failure. Scanning Security alerts and complete #481 reporting remain pending.

### Ruleset Quality severity observation normalization

For a present, complete code-quality rule, a correctly typed severity outside the four mapped UI choices projects to `errors`, with exact raw provenance bound to the positive Ruleset ID. Unknown, empty, whitespace and UI-spelling API strings follow this policy; missing/null/wrong-type required parameters remain acquisition errors. An absent rule stays absent. Neither observation fallback invents desired intent or makes an incomplete policy usable.

Enforcement and Quality keep independent records on one whole-object change. Export warnings follow Ruleset name order, then enforcement, Required approvals and Quality within each object. Selected normalized equality still requires a write; unlisted IDs remain private. Plan and detailed Summary show normalized/write-required UI choices, or `inactive (rule removed; write required)` when desired policy removes the Quality group, followed by one effective policy. Same-version private reports validate each concrete record independently and retain status-only privacy; existing collection count stays one pending #481.

Apply sends only the complete desired policy and verifies the original mutation response plus an independent fresh GET. Unknown severity cannot prove an Errors match, and removing the rule requires actual absence: a retained canonical or unknown rule fails with uncertain effects. Warning projection never supplies verification values. Canonical equal policy without explicit write-only dismissal intent remains a no-op; acquisition failures preserve export destinations and make no Ruleset write.

### Ruleset Required approvals observation normalization

A complete present PR rule acquires `required_approving_review_count` as an integer. UI choices are 0–10. Other integers, including -1 and API-only 11, project to the named 0 fallback while retaining exact integer provenance per positive Ruleset ID. No string, fraction, overflow, null or missing member supplies a count. An absent PR rule stays absent. Native transport validation remains separate from the UI range.

A selected normalized count requires one whole-object update, including desired 0 matching the projection. Plan and detailed Summary show unquoted integers marked normalized/write-required, or inactive/rule-removed intent when the desired PR group is absent. Enforcement, Required approvals and Quality retain independent concrete records and warn in that order within name-sorted objects. Unlisted IDs stay unmanaged and private. Existing collection counts stay one pending #481; ordinary canonical equality without explicit write-only dismissal intent stays a no-op.

Private `required_approvals_normalization` metadata requires a string path and integer raw/fallback members, including explicit zero; present-zero, forged and ID-conflicting records fail closed. Report values never supply mutation input. Apply verifies original acknowledgment and independent fresh GET against exact desired whole policy: retained 11/-1 cannot confirm 0, and a retained canonical PR rule cannot confirm removal. Incomplete acquisition makes no Ruleset write and preserves export destinations; later failure retains earlier resource effects. Scanning Security alerts remains pending.

### Ruleset Allowed merge methods observation normalization

A complete present pull-request rule requires a nonempty, unique list of strings. If any acquired `allowed_merge_methods` member is outside `merge`, `squash` and `rebase`, project the entire list to the named default: Merge, Squash, Rebase. Retain the exact original list and its order, including empty or whitespace strings, as owned ID-bound provenance. Missing/null lists, null or non-string members, empty lists, duplicates, incomplete policies and failed pagination remain acquisition errors. An absent PR rule stays absent.

Export warns and saves only effective choices. Plan and detailed Summary show the original list, the normalized default and the selected desired choices in named UI order; fallback-equal selection still requires one whole-object update. Existing declared-object defaults apply when the methods field is omitted within a selected PR rule. Removing the PR rule shows an inactive/rule-removed decision without inventing an after-list. Unlisted IDs and omitted Ruleset collections remain undisclosed and unwritten.

Apply verifies the original mutation acknowledgment and an independent fresh GET against the desired whole policy. A retained unsupported list cannot confirm the full default, and a retained canonical PR rule cannot confirm removal. Fresh warnings are diagnostic evidence only. Incomplete acquisition preserves export destinations and makes no Ruleset write; later failures retain earlier or uncertain effects.

Private `merge_methods_normalization` metadata requires a string path and present typed raw/fallback lists, rejects empty, forged and ID-conflicting evidence even beside valid companion records, and never supplies apply input or proves acquisition completeness. Status-only Summary hides policy/provenance; the existing collection count stays one pending #481. Scanning Security alerts remains under #480.

### Ruleset Team reviewer approvals observation normalization

A complete Team reviewer acquires `minimum_approvals` as an integer. UI choices are 0–10; other integers project to the named 0 fallback while retaining the exact raw count bound to the positive Ruleset ID and positive Team ID. Records identify reviewers by ID, never list position. Canonical companions retain their own counts and file patterns. Only Team reviewers are supported, with unique IDs and at most 15 entries; normalization never invents a reviewer or replaces its identity.

Missing/null counts, fractions, overflow, wrong types, missing/null file-pattern lists, invalid identities, duplicate reviewers and incomplete policy or pagination remain acquisition errors. File patterns preserve order, repetitions, negations and explicit emptiness. An absent PR rule stays absent; an explicit empty reviewer list stays empty. Observation defaults never fill missing declared `approvals` or `file_patterns` members.

Requested export saves only effective UI policy after complete acquisition. Warnings include Ruleset ID/name and `reviewer=<Team ID>`, ordered by numeric reviewer ID after Allowed merge methods and before Quality severity. The diagnostic identity is context, not a new YAML key. Selected fallback-equal intent still requires one whole-object update. Plan and detailed Summary show each raw warning and normalized/write-required count, followed by one effective policy. Removing a reviewer shows `inactive (reviewer removed; write required)`; removing the PR rule shows `inactive (rule removed; write required)`. Replacing a Team ID removes the old reviewer and selects the new one rather than comparing unrelated counts. Unlisted Ruleset IDs remain unmanaged and undisclosed.

Apply verifies the original acknowledgment and independent fresh GET against the exact desired whole policy. Retained out-of-range counts cannot confirm 0, a retained canonical reviewer cannot confirm reviewer removal, and a retained PR rule cannot confirm rule removal. Fresh warnings require complete owned readback and do not supply verification values. Failed acquisition makes no Ruleset write and preserves export destinations; a later failure reports earlier or uncertain effects honestly.

Private reports retain typed, reviewer-bound `team_approvals_normalizations` with explicit integer zero and independently validated companion records. They cannot prove complete acquisition or supply apply input. Status-only Summary hides policy and provenance, and the existing collection count stays one pending #481. Scanning Security alerts remains under #480.

### Ruleset scanning Alerts observation normalization

Each tool in a complete code-scanning rule acquires `alerts_threshold` as a string. Exact `none`, `errors`, `errors_and_warnings` and `all` choices remain unchanged. Other strings, including empty and whitespace values, project to the named `errors` fallback while retaining the exact raw value under the positive Ruleset ID and exact tool name. Tool names are identities: no trimming, case folding, substitution or synthetic tool is introduced. Canonical tools and each tool's Security alerts choice remain intact.

Missing/null/non-string thresholds or names, invalid/duplicate names, malformed companion fields and incomplete acquisition remain errors. Unsupported Security alerts still fail pending their own normalization. An absent scanning rule stays absent; a present empty tool list stays empty. Each declared tool still requires its own Alerts and Security alerts choices; observed fallbacks do not fill omitted declarations.

Requested export saves effective policy and warns after complete acquisition. Scanning Alerts warnings follow Team reviewer Approvals and precede Quality severity, ordered by exact tool name. Each includes Ruleset ID/name and `tool="name"` context. Selected fallback-equal or different choices require one whole-object update. Removing a tool or replacing its name retains the old tool's warning and `inactive (tool removed; write required)` decision; removing the scanning rule shows `inactive (rule removed; write required)`. Plan and detailed Summary show one effective policy. Unlisted Ruleset IDs stay unmanaged and undisclosed.

Original acknowledgment and an independent fresh GET must confirm the exact desired whole policy. A retained unknown Alerts string cannot confirm Errors; a retained canonical tool or scanning rule cannot confirm removal. Contextual fresh warnings never supply successful verification values. Failed acquisition makes no Ruleset write and preserves export destinations; later failure retains honest reporting of earlier or uncertain effects. Private `scanning_alerts_normalizations` records are presentation evidence, not apply input or acquisition proof; status-only output stays private and collection count remains one pending #481. Scanning Security alerts remains under #480.

### Ruleset bypass-mode observation normalization

A complete acquired `bypass_mode` string outside the native `always`, `exempt` and `pull_request` choices projects to the UI default `always`. Keep its exact raw string, including empty/whitespace values, bound to the Ruleset ID and the actor's exact type and optional numeric ID. Canonical companions keep their own identities and choices. Missing/null modes, malformed identities, duplicate actors, incomplete acquisition and unsupported actor/target applicability remain errors; normalization never invents an actor, role, grant or ID.

Requested export warns after complete acquisition and saves only effective UI policy. Warnings follow Ruleset name order, then enforcement, Required approvals, Allowed merge methods, Team reviewer Approvals (numeric reviewer ID order), scanning Alerts (tool-name order), Quality and bypass actors ordered by type and numeric ID. Actor diagnostics include `actor_type` and include `actor_id` only when present. These are identity context, not public YAML aliases. Unlisted Ruleset IDs stay unmanaged and undisclosed in plan/apply.

Selected normalized equality still requires one whole-object update. Plan and detailed Summary show the raw warning and `always` marked as a normalized observation, followed by the desired UI mode marked write-required. Removing an actor, explicitly or through existing whole-object declaration defaults, shows `inactive (actor removed; write required)`. Each operation has one effective-policy line; canonical equal objects without explicit write-only dismissal intent remain no-ops. Observation fallback does not populate omitted desired resources or change whole-object ownership.

Apply verifies the original mutation response and an independent fresh GET against the desired observable policy. A canonical acknowledgment followed by the same unknown raw mode cannot confirm `always`; a retained canonical actor cannot confirm removal. Fresh warnings require complete owned readback and never supply verification values. Incomplete acquisition makes no Ruleset write and preserves export destinations; a later failure reports possible and earlier completed effects without implying rollback.

Private same-version reports retain actor-bound `bypass_mode_normalizations`, validate each record independently beside scalar evidence, and omit nil actor IDs rather than accepting JSON null. They preserve status-only privacy and the existing collection count one; records never enter public YAML or apply input and cannot prove acquisition completeness. Scanning Security alerts remains under #480; complete reporting remains #481.

### PR review and required check parameters

`rules.require_a_pull_request_before_merging` is a branch-only group:

| Field | Default / accepted value |
| --- | --- |
| `required_approvals` | 0; UI integers 0–10 |
| `allowed_merge_methods` | all three: `merge`, `squash`, `rebase`; nonempty unique selected choices |
| `dismiss_stale_pull_request_approvals_when_new_commits_are_pushed`, `require_review_from_code_owners`, `require_approval_of_the_most_recent_reviewable_push`, `require_conversation_resolution_before_merging`, `require_an_additional_approval_for_unattributed_copilot_pull_requests` | false |
| `require_review_from_specific_teams` | empty; up to 15 `{reviewer: positive-team-id, approvals: 0..10, file_patterns: [...]}` entries; approvals and patterns required |
| `restrict_who_can_dismiss_pull_request_reviews`, `people_teams_or_apps_that_can_dismiss_reviews` | temporary write-only exception below |

Reviewer file patterns preserve order/repetition, including negations. Other independent collections canonicalize by identity. A dismissal actor selects one `role`, `team`, `app` or `user`; nonempty actors require an explicitly selected restriction checkbox. Omitted restriction controls send no value. Explicit controls use existing false/empty defaults for omitted companions; explicit false differs from omission.

`rules.require_status_checks_to_pass` contains `required_status_checks` (`[{check: name, source: {any_source: true}}]` or `{source: {app: positive-id}}`), `require_branches_to_be_up_to_date_before_merging` and `do_not_require_status_checks_on_creation`. Defaults are empty checks and false flags. Omitted source means any source, not a retained App. Explicit source must select true Any source or App, not both. YAML null is invalid.

### Code scanning and code quality parameters

`rules.require_code_scanning_results` requires `code_scanning_tools`, each with `tool_name`, `alerts` (`none`, `errors`, `errors_and_warnings`, `all`) and `security_alerts` (`none`, `critical`, `high_or_higher`, `medium_or_higher`, `all`). Tool names are case-sensitive identities; duplicates fail and collection order canonicalizes by name. Explicit `[]` remains empty; no threshold default is inferred.

`rules.require_code_quality_results` requires `severity`: `errors`, `warnings_and_higher`, `notes_and_higher` or `all`. These UI choices translate internally; API tokens are not aliases. Complete acquisition is required. The rules constrain merges; they do not enable analysis, create workflows, change entitlement or expand credentials. Include every supported protection that must remain. See [Code Analysis](features/rulesets.feature.md).

### Plan, apply, and export

Complete supported observations, including bypass visibility, are required. Unknown writable fields/rules, missing values, invalid types, unreadable details and failed pages stop mutation/export. Inherited organization/enterprise and repository push Rulesets remain outside mutation ownership; push Rulesets are excluded from export with a diagnostic. Other protection can still affect effective behavior.

Plan shows only listed creates/ID updates and default-derived reductions; it never invents deletion of unlisted objects. A nonempty collection counts once as managed. Apply reads fresh state after earlier families, prepares selected operations before the first Rulesets write and writes in stable name order. Fresh comparable matches make no write unless explicit write-only intent exists.

POST/PUT responses must confirm identity, ownership and requested comparable policy. PR/check/analysis changes require independent GET verification, including rule removal. Mismatch/failed readback returns nonzero and stops later operations: writes may already have applied. No retry or rollback occurs. Confirmed created IDs follow [guarded writeback](#apply-id-writeback).

Review-dismissal restriction is temporarily write-only. Validate/apply accept explicit UI controls; plan rejects them before remote reads, export always omits both even when GET reports them, and successful apply reports API acceptance with the value unverified. An echoed response does not prove storage/enforcement; every repeated explicit apply sends it again. Omitted intent never invents false or borrows an observed value. Missing comparable state and malformed present objects still fail. Replacing a declaration with export loses this explicit write intent.

A prior bounded experiment saw identical omitted GET values after omitted/false/true PUTs on a temporary disabled Ruleset. It did not establish stored values, default elision or a general preservation rule. Existing Metadata read/Administration write, complete bypass visibility and platform eligibility are prerequisites; no workaround or permission expansion occurs. See the [Rulesets specification](features/rulesets.feature.md) and [UI mapping](contributing/ui-ruleset-mapping.md).

## Environments

`repository.settings.environments` selects whole supported objects by `name`, without numeric Environment/pattern ID writeback. Names are nonempty, have no leading/trailing whitespace/control characters and use Unicode simple folding for client identity. Spelling remains; slash/percent are URL-encoded as one segment. Omission/`[]` selects nothing; unlisted Environments are never deleted. Deletion/rename markers, secrets, variables and custom protection apps are unsupported.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: production
        deployment_protection_rules:
          wait_timer: 0
          required_reviewers: []
          prevent_self_review: false
          allow_administrators_to_bypass_configured_protection_rules: false
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules:
          - {ref_type: branch, name_pattern: 'release/*'}
          - {ref_type: tag, name_pattern: 'v*'}
```

Omitted built-in fields are omitted from PUT on create/update. GitHub controls their result; this promises neither preservation nor reset. Name-only intent sends an empty PUT, not an existence assertion. Plan marks omitted outcomes API-controlled instead of inventing defaults; a partial request can remain update intent even when current visible values look equal.

`deployment_protection_rules` contains optional `wait_timer` (0–43200 minutes), up to six unique `required_reviewers` (each `{person: positive-id}` or `{team: positive-id}`), `prevent_self_review` and `allow_administrators_to_bypass_configured_protection_rules`. True prevent-self-review cannot accompany an explicitly empty reviewer set. No null is accepted. Reviewer eligibility, grants and entitlement remain GitHub prerequisites; principals are not invited.

`deployment_branches_and_tags` is `no_restriction`, `protected_branches_only` or `selected_branches_and_tags`. There is no public raw boolean-pair/null alias. Protected mode may permit all branches when no branch protections exist; configured intent does not prove protection.

`deployment_branch_or_tag_rules` is an authoritative list of `{ref_type: branch|tag, name_pattern: pattern}`. Type+pattern is case-sensitive identity; branch and tag may share a pattern. Omission makes no pattern CRUD; `[]` removes all observed patterns. Omitted mode plus a list requires existing selected mode; creating with patterns needs explicit selected mode. Explicit incompatible modes fail. Patterns are a separate API collection, not PUT members. Mode changes make no promise about inactive patterns.

Reads require complete inventory, built-in/custom-protection details and typed paginated patterns when applicable; missing type is not assumed branch. Unknown/custom protection fails instead of being erased. Apply preflights, upserts in stable name order, then verifies custom mode, creates missing patterns and deletes obsolete patterns by verified IDs. Retained pairs keep IDs. Fresh readback verifies only requested values/list. Duplicate-create 303, unavailable reads and mismatches fail; earlier/uncertain effects remain without retry, rollback or an intermediate-enforcement guarantee.

Administrator bypass is a real writable setting. Export requires/retains its complete boolean, even false. A prior temporary public-Environment experiment confirmed explicit false/true and retained preceding values on omission with other members fixed; this is not a platform-wide omission or entitlement guarantee. No default true is inserted.

`export --environments` includes complete supported policy and active selected patterns without Environment/pattern IDs; ordinary export adds no Environment permissions. Existing Actions read, Administration write, custom protection visibility and account eligibility are required. See [Environment behavior](features/environments.feature.md) and [mapping](contributing/ui-environments-mapping.md).

### Environment deployment mode observation normalization

Complete typed equal policy booleans `(false,false)` or `(true,true)` are outside the three UI choices. Project the named `no_restriction` fallback while retaining both acquired booleans separately from effective policy. A valid API null remains exact No restriction without provenance. Missing policy, missing/null/wrong-type boolean members, unknown controls, failed reads and incomplete pages remain acquisition errors.

Raw `custom_branch_policies:true` still requires complete paginated patterns and custom-protection checks before publishing an observation, including a projected equal pair. Keep those latent pattern IDs/values in separate acquired state; effective No restriction never exports or replays them as selected patterns. Acquired empty patterns differ from an incomplete read. Pattern ref-type observations follow the policy below; unrepresentable principal identities are not invented.

Export warns in folded environment-name order and saves only effective policy. Plan/detailed Summary label every Environment before-object as **effective before policy**. An explicitly selected normalized mode adds a contextual raw-pair warning and normalized/write-required UI decision, including fallback-equal No restriction. Only selected names are compared; unlisted names stay undisclosed and unwritten. Private reports retain closed typed pair metadata, actual partial requests, status-only privacy and the existing collection count one pending #481; they never supply apply input.

Apply verifies an explicit mode against the original nullable pair in the acknowledgment and an independent complete fresh read. Retained equal booleans cannot confirm canonical No restriction. Pattern intent under an unknown mode needs explicit Selected; neither fallback nor the raw custom flag proves active applicability. A canonical mode acknowledgment alone does not permit pattern CRUD if the intermediate fresh read retains an unknown mode. Writes remain ordered create-before-delete with verified pattern identities and honest partial effects.

Omitted mode stays omitted from PUT. Its server-selected result is neither preserved nor certified, and no unowned mode warning is printed. Requested ordinary fields and selected patterns are still verified; name-only intent remains API-controlled. No observed pair or client default is copied into an omitted request. Malformed acquisition preserves export destinations and makes no Environment write; later failures retain prior resource and uncertain effects. Scanning Security alerts and #481 remain pending.

## Unsupported settings

Only the listed UI controls are accepted. Organization settings/custom properties/forking policy, collaborator grants/invitations/team access, repository renaming/visibility/transfer/deletion, legacy branch protection, inherited/push Rulesets, Pages lifecycle/visibility, Environment deletion/rename/custom apps/secrets/variables and unadopted Actions/security controls remain outside scope. A visible GitHub control or writable API field does not adopt it. See the [architecture](contributing/architecture.md) and proposed [access design](features/repository-access.feature.md) without treating proposals as configuration.

## Existing Pages configuration

`repository.settings.pages` manages only a positively observed existing site; creation/deletion/visibility changes remain unsupported.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    pages:
      build_and_deployment:
        source: deploy_from_a_branch
        branch: main
        select_folder: docs
      custom_domain: null
      enforce_https: true
```

`build_and_deployment.source` is `deploy_from_a_branch` or `github_actions`. Branch mode uses a complete literal `branch` plus `select_folder: root|docs`; partial tuples fail. Switching from workflow to branch requires that tuple; workflow mode rejects branch/folder intent. No branch/content/workflow is created. Omitted Pages/empty mapping manages nothing; other omitted members remain unmanaged.

`custom_domain` is a nullable scalar, never a `{value: ...}` mapping. Null (including a bare empty scalar) removes it. Omission leaves it unmanaged; empty string and quoted `null` are invalid. Strings must be ASCII DNS hostnames with at least two labels and normal length limits, without trailing dot, URL, wildcard, port or whitespace. Comparison/export lowercase without IDNA or DNS verification. `enforce_https` is boolean; null is invalid.

`export --pages` includes mode, active branch/folder, nullable domain and HTTPS only after complete acquisition. Visibility/readiness metadata are excluded. Missing/null/wrongly typed required fields, invalid branches or 404 fail rather than becoming absent-site state and preserve the destination.

A complete legacy-mode response with a folder string outside `/` and `/docs`, including an empty string, projects to `select_folder: root`. Its exact raw folder, UI path and named default remain observation provenance; the branch is preserved and never invented. Export warns and saves only the effective UI choice. A declared branch/folder pair retains a warning and a write-required folder decision even when both declared values equal the effective projection. The pair still counts as two managed native fields. Omitted source remains unmanaged and does not disclose its raw folder. Inactive workflow source is not acquired.

A correctly typed publishing mode outside `legacy` and `workflow` projects to `source: deploy_from_a_branch` only with a complete typed branch/folder tuple and valid literal branch. Missing/null/wrongly typed source members remain errors; no branch is inferred. Empty and whitespace mode strings retain exact provenance. Mode and folder records remain independent, and export warns in that order. A declared publishing mode counts one native field and retains a write-required decision even when it equals the projection. Explicit source intent with an unknown raw mode requires `source: deploy_from_a_branch`; the fallback alone cannot prove applicability. Omitted publishing mode remains unmanaged and is not disclosed.

Plan is read-only; an inactive source is not invented as an observed tuple. Apply reads fresh state, skips exact matches, sends declared-only PUT values and verifies requested values, omitted applicable companions, public visibility and repository identity. Verification checks exact raw mode and branch/folder strings: an unchanged unknown token cannot confirm a requested canonical value, and omitted active mode/source must retain their original raw values. Domain canonicalization remains unchanged. Apply warnings use only fresh acquired provenance, independently scoped to declared publishing mode and source intent. Explicit unarchive precedes it; final archive follows. Failure can leave earlier writes or uncertain effects, without retry/fallback/rollback.

Convergence does not prove DNS, certificate or deployment readiness. Pending metadata alone is not drift. GitHub can publish or create a CNAME-file commit as a side effect; the tool does not trigger builds/deployments or edit DNS/content. Existing Pages read/write, Administration write and actor authorization are prerequisites; public GET does not prove write grants. See [Pages behavior](features/pages.feature.md).

[^ui-inventory]: Read-only UI wording inventory inspected 2026-10-28; internal conversion details and the other family inventories are linked there. This reference changes the public YAML composition, not the recorded UI inspection history.
[^github-update-repository]: Repository update request fields; original inspection 2026-10-11. This does not expand the supported UI inventory.
[^github-repository-openapi]: Fixed GitHub OpenAPI evidence for nullable observed description/homepage; original inspection 2026-10-12.
[^github-labels]: Labels listing/mutation and existing permission boundary.

### Environment ref-type observation normalization

A complete unsupported string `type`, including empty or whitespace, projects to the named `branch` fallback. Retain the exact original type, positive API ID and case-sensitive pattern name separately from effective policy. Missing/null/wrong-type fields, invalid names/IDs, duplicate raw identities, unstable counts and incomplete pages remain acquisition failures; they do not produce defaults or partial warnings.

In canonical Selected mode, export warns with the Environment name, pattern ID, exact name and original/fallback type, then saves effective `ref_type`/`name_pattern` values without API IDs or provenance. Warnings follow folded Environment-name order, mode warning first, then ascending pattern ID. Canonical companions stay silent. Distinct raw identities that project to the same branch/name are retained separately: export warns and fails without replacing its destination, rather than collapsing them. Raw patterns under effective No restriction remain latent and produce neither ref-type warnings nor exported rules.

An explicit pattern list owns the observed collection, including `[]`; an omitted list owns none of its values or diagnostics. Plan and detailed Summary retain one actionable collection update at fallback equality, exact raw evidence and the complete effective before policy. A genuine collision is labeled unavailable effective pattern data, not an empty list. Only selected Environment names are observed. Same-version private reports retain closed per-ID evidence and canonical companions; status-only output hides these details. Reports never supply apply input, and the existing managed collection count remains one pending #481.

Apply sends declared fields in the Environment PUT and canonical types in separate pattern requests. It confirms fresh canonical Selected mode, creates required pairs before deleting obsolete raw identities, and retains the IDs of existing canonical companions. Before deletion, an individual GET must match the exact original type, case-sensitive name and ID. Creation acknowledgments remain strict; unsupported or malformed acknowledgments stop before deletion. Independent complete readback must confirm the requested canonical collection: a retained unsupported type cannot prove success merely because its fallback matches. Fresh owned warnings expose the acquired record, not a verification substitute.

Acquisition failures make no Environment write. Later failures retain earlier or uncertain effects, report failure and do not trigger automatic retry or rollback. Caller YAML is unchanged. A complete canonical match is a silent no-op on repeat; omitted fields remain API-controlled. These source behaviors do not apply live settings or publish a release. Scanning Security alerts and complete #481 reporting remain unfinished.
