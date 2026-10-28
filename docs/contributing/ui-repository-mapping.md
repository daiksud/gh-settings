---
type: Reference
title: Prepared About, General and Labels UI mappings
description: Internal UI model, API translation, defaults and activation handoff for the repository family.
sources:
  - id: repository-ui
    resource: https://github.com/daiksud/gh-settings/settings
  - id: about-ui
    resource: https://github.com/daiksud/gh-settings
  - id: labels-ui
    resource: https://github.com/daiksud/gh-settings/labels
  - id: merge-ui
    resource: https://github.com/daiksud/agents/settings
---

## Preparation boundary

This is the internal mapping prepared for [Issue #482](https://github.com/daiksud/gh-settings/issues/482). It does not describe accepted YAML yet. The current codec, export, plan/apply and reports continue using their current contract. [Issue #479](https://github.com/daiksud/gh-settings/issues/479) activates the prepared mappings together; [#480](https://github.com/daiksud/gh-settings/issues/480) owns out-of-UI normalization and warnings, and [#481](https://github.com/daiksud/gh-settings/issues/481) owns reporting integration.

The [UI model](../../internal/settings/uimodel/) contains display terminology without transport enums or serialization tags. The `ui_*.go` files in [the GitHub adapter](../../internal/github/) translate it to and from the existing validated adapter inputs. These are temporary internal preparation boundaries, not another public configuration format or compatibility API.

## UI inventory and paths

The English Web UI headings, control labels and menu choices below were inspected read-only on 2026-10-28. About and Labels were inspected on gh-settings. General was inspected on gh-settings; the already-enabled merge method on agents exposed its otherwise conditional menu. No values were saved. These observations verify wording and visible choices, not live API mutation behavior or permission coverage.[^repository-ui][^about-ui][^labels-ui][^merge-ui]

Prepared paths use the exact UI wording converted to snake_case. In this table `A` means `repository.about` and `G` means `repository.settings.general`. Each listed control is supported internally. About and Labels are siblings of Settings.

| UI location / label | Prepared path | Internal API conversion |
| --- | --- | --- |
| About / Description | `A.description` | Repository REST `description`; GET null becomes empty through the existing reader |
| About / Website | `A.website` | Repository REST `homepage`; GET null becomes empty |
| About / Topics | `A.topics` | Topics GET/PUT `names`; canonical sorted set |
| General / Template repository | `G.template_repository` | REST `is_template` |
| General / Default branch | `G.default_branch` | REST `default_branch`; retain existing branch validation/readback |
| Features / Wikis | `G.features.wikis` | REST `has_wiki` |
| Features / Issues | `G.features.issues` | REST `has_issues` |
| Features / Projects | `G.features.projects` | REST `has_projects` |
| Features / Discussions | `G.features.discussions` | REST `has_discussions` |
| Features / Sponsorships | `G.features.sponsorships` | GraphQL `hasSponsorshipsEnabled` read / `hasSponsorshipsEnabled` update input |
| Features / Pull requests | `G.features.pull_requests` | GraphQL `hasPullRequestsEnabled` read / update input |
| Issue permissions / Creation allowed by | `G.features.issue_permissions.creation_allowed_by` | GraphQL `issueCreationPolicy` read / update input |
| Pull request permissions / Creation allowed by | `G.features.pull_request_permissions.creation_allowed_by` | GraphQL `pullRequestCreationPolicy` read / update input |
| Pull Requests / Allow merge commits | `G.pull_requests.allow_merge_commits` | GraphQL `mergeCommitAllowed` / REST `allow_merge_commit` |
| Pull Requests / Allow squash merging | `G.pull_requests.allow_squash_merging` | GraphQL `squashMergeAllowed` / REST `allow_squash_merge` |
| Pull Requests / Allow rebase merging | `G.pull_requests.allow_rebase_merging` | GraphQL `rebaseMergeAllowed` / REST `allow_rebase_merge` |
| Pull Requests / Always suggest updating pull request branches | `G.pull_requests.always_suggest_updating_pull_request_branches` | GraphQL `allowUpdateBranch` / REST `allow_update_branch` |
| Pull Requests / Allow auto-merge | `G.pull_requests.allow_auto_merge` | GraphQL `autoMergeAllowed` / REST `allow_auto_merge` |
| Pull Requests / Automatically delete head branches | `G.pull_requests.automatically_delete_head_branches` | GraphQL `deleteBranchOnMerge` / REST `delete_branch_on_merge` |
| Merge commits / Default commit message | `G.pull_requests.merge_commit.default_commit_message` | GraphQL `mergeCommitTitle` + `mergeCommitMessage` / REST `merge_commit_title` + `merge_commit_message` |
| Squash merging / Default commit message | `G.pull_requests.squash_merge.default_commit_message` | GraphQL `squashMergeCommitTitle` + `squashMergeCommitMessage` / REST `squash_merge_commit_title` + `squash_merge_commit_message` |
| Commits / Require contributors to sign off on web-based commits | `G.commits.require_contributors_to_sign_off_on_web_based_commits` | REST `web_commit_signoff_required` |
| Danger Zone / Archive this repository | `G.danger_zone.archive_this_repository` | REST `archived`; retain existing identity/preflight/readback lifecycle sequencing |
| Labels / Name | `repository.labels[].name` | API `name` on read/create, `new_name` on update |
| Labels / Description | `repository.labels[].description` | API `description`; null GET becomes empty; omitted desired description retains the existing whole-object empty default |
| Labels / Color | `repository.labels[].color` | UI `#rrggbb` to/from API six hex digits; canonical lowercase |
| Selected label identity | `repository.labels[].id` | Existing numeric ID selects a label; absent ID remains create-only |

The two Default commit message menus are distinguished by their merge-method context. A declaration sends both API members together. It cannot request a menu choice while explicitly disabling that method. An omitted enable checkbox remains unspecified. Existing server-state dependency validation for merge-commit defaults still applies; conversion does not invent `true`.

### Menu choices and observation defaults

The following literals are independent of API field names. The API pairs are owned by [the menu adapter](../../internal/github/ui_commit_message.go).

| Menu | UI choice | Prepared choice | API title / message |
| --- | --- | --- | --- |
| Merge | Default message | `default_message` | `MERGE_MESSAGE` / `PR_TITLE` |
| Merge | Pull request title | `pull_request_title` | `PR_TITLE` / `BLANK` |
| Merge | Pull request title and description | `pull_request_title_and_description` | `PR_TITLE` / `PR_BODY` |
| Squash | Default message | `default_message` | `COMMIT_OR_PR_TITLE` / `COMMIT_MESSAGES` |
| Squash | Pull request title | `pull_request_title` | `PR_TITLE` / `BLANK` |
| Squash | Pull request title and commit details | `pull_request_title_and_commit_details` | `PR_TITLE` / `COMMIT_MESSAGES` |
| Squash | Pull request title and description | `pull_request_title_and_description` | `PR_TITLE` / `PR_BODY` |
| Either creation policy | All users | `all_users` | `ALL` |
| Either creation policy | Collaborators only | `collaborators_only` | `COLLABORATORS_ONLY` |

The [model constants](../../internal/settings/uimodel/general.go) provide explicit, separate observation fallbacks for #480:

| Prepared field | Fallback |
| --- | --- |
| Merge Default commit message | `default_message` (`DefaultMergeCommitMessage`) |
| Squash Default commit message | `default_message` (`DefaultSquashMergeCommitMessage`) |
| Issue permissions / Creation allowed by | `all_users` (`DefaultIssueCreationAllowedBy`) |
| Pull request permissions / Creation allowed by | `all_users` (`DefaultPullRequestCreationAllowedBy`) |

These fallbacks are planned normalization policy, not the currently selected values of the inspected repositories. They never populate an omitted desired field. At this preparation stage, observation conversion rejects incomplete pairs and out-of-UI pairs/tokens; #480 must add warning/provenance handling before the overall new contract is complete. Known API-only combinations such as `COMMIT_OR_PR_TITLE` / `BLANK` must not become new UI choices.

Correctly typed booleans have no out-of-UI values. Free text, branch names, topics, label names/IDs/colors and descriptions do not have finite selectable enums requiring an unknown-choice fallback. Their existing validation, nullable-string acquisition semantics and identity rules remain authoritative. Invalid types, missing required fields, failed requests and incomplete pagination remain errors; they are not candidates for these defaults.

### Unsupported controls and other families

The observed General page also contains Repository name, social preview, Restrict editing to collaborators only, Preserve this repository, Allow comments on individual commits, Include Git LFS objects in archives, Limit how many branches and tags can be updated in a single push, Auto-close issues with merged linked pull requests, visibility, transfer, deletion and disabling branch protection. These controls are not implemented by this preparation. About's Include in the home page controls are also unsupported. A visible control does not authorize adopting it.

Releases / Enable release immutability is an already supported setting prepared by the Security family in #485, despite its General-page location. Other family controls belong to #483–#487. Do not infer their final UI paths from the current API-shaped configuration.

## Activation and validation handoff

Issue #479 should consume `AboutDeclaration` / `AboutObservation`, `GeneralDeclaration` / `GeneralObservation`, and `LabelsDeclaration` / `LabelsObservation`, preserving which independently requested values were acquired. Existing readers remain responsible for complete raw data; a projection is not proof of acquisition. Existing application sequencing, resource selection, lifecycle checks, declared-only writes and honest partial failure remain required.

Label lists keep ID-selected, listed-object ownership. An empty list does not delete unlisted labels; no-ID entries are create-only. Topics retain authoritative set membership. Each converter owns returned values so later caller changes cannot alter its intent.

The [test specification](testing.md#internal-ui-repository-preparation) records independent fixtures and reused owner checks. These local fixtures prove internal conversion and current-interface regression behavior. They do not prove a new YAML parser, normalized reporting, effective repository protections or live mutations.

[^repository-ui]: General settings UI inspected read-only on gh-settings, 2026-10-28.
[^about-ui]: Edit repository details dialog: Description, Website, Topics and unsupported home-page controls, 2026-10-28.
[^labels-ui]: New label dialog: Name, Description and Color with the displayed `#` prefix, 2026-10-28.
[^merge-ui]: Existing enabled merge method on agents exposed its three choices; gh-settings exposed the four squash choices, 2026-10-28.
