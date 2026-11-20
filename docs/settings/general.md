---
type: Reference
title: "General settings"
description: "General settings declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Template repository

`repository.settings.general.template_repository` is an optional boolean. It controls template availability without adding repository generation or copying. See [template repository](../features/template-repository.feature.md).

## Default branch

`repository.settings.general.default_branch` selects an existing branch using a valid literal name; it does not create, rename or move a branch. Empty, whitespace/ref-syntax-invalid or wrong-type values fail local validation. Existing existence/readback and target checks remain authoritative. See [default branch](../features/default-branch.feature.md).

## Immutable release policy

Use `repository.settings.general.releases.enable_release_immutability`, a boolean. This General-page control is independent from Advanced Security. Ordinary `export` automatically attempts this policy, with info and omission/prior-declaration retention when unavailable. Owner enforcement is read-only metadata, never YAML intent.

The key follows GitHub's [Enable release immutability UI label](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/prevent-release-changes).
The former `immutable_release` key is rejected, including declarations containing
both names. There is no input alias or automatic rewrite. The published v0.45.0
package accepts the former name and rejects the new name; adopt a compatible
published package before changing operational files. This repository preserves
its v0.45.0 declaration and pins until that separate adoption; see the
[operational format boundary](../contributing/repository-automation.md#release-immutability-format-and-operational-package).

A complete successful 200 observation requires both enabled and owner-enforced booleans. Under required permissions, HTTP 404 means disabled without an authorization-discrimination request, but supplies no owner-enforcement metadata. This does not certify actual grants or change other endpoints' 404 semantics. Explicit 401/403, timeouts and malformed/incomplete successful reads remain errors.

An owner-enforced policy cannot be planned/applied as false. Apply runs after private vulnerability reporting and before Pages, reads fresh state, skips a match, sends body-free PUT/DELETE, requires 204 and verifies fresh readback. A post-DELETE 404 confirms disabled intent under the same precondition; unknown owner enforcement remains unknown. Failed writes/readback can already have changed the policy; no retry or rollback occurs. Individual releases, tags and assets are never mutated by this feature, and disabling the policy does not make existing immutable releases editable. Administration read/write and applicable role must already exist. See [the full contract and evidence limits](../features/immutable-releases.feature.md).

## Sponsor button visibility

Use `repository.settings.general.features.sponsorships`. This boolean controls the Sponsor button, not funding destinations. Omission is unmanaged; false disables. `.github/FUNDING.yml` is never created, read, changed or deleted by this control. Ordinary export includes the observed value through the existing shared GraphQL path. Apply compares the declared boolean with fresh state, skips a match and independently verifies a changed value after mutation acknowledgment. See [Sponsorships](../features/sponsorships.feature.md).

### Execution-policy observation normalization

A complete, correctly typed unknown `allowed_actions` string (including empty or whitespace) retains its exact raw value and projects to the named `allow_all_actions_and_reusable_workflows` fallback. Both acquired booleans remain unchanged. With execution enabled, export warns and emits that fallback; a declared policy keeps one UI decision actionable even at fallback equality. Disable owns one native field; enabling owns two, or five with all three selected companions. Managed counts reflect native fields internally.

With execution disabled, the visible choice is the known `disable_actions`, so the latent unknown policy produces no projection warning. Export omits latent choices, and a matching Disable plan has no differences. Enabling requires an explicit canonical policy; selected intent still needs all three companions and configures them before enabling. Private radio reports retain raw provenance for closed report validation/presentation but detailed output does not disclose hidden policy; status-only output suppresses all policy values.

A needed base update sends only changed declared policy/SHA members with the endpoint-required enabled value. Disable and SHA-only updates preserve the exact raw policy; SHA-only updates do not warn about omitted policy intent. Fresh raw readback must confirm declared canonical policy or preserved raw policy, both booleans and any managed selected companions. An unchanged unknown string cannot confirm a declared fallback, and a different unknown token cannot confirm preservation. Failed verification retains uncertainty about writes and earlier completed resource work. Direct repeated apply skips raw matches, including an already disabled target with unknown latent policy. Missing/null/wrong types and failed acquisition never become fallbacks.

## Issue creation policy

Use `repository.settings.general.features.issue_permissions.creation_allowed_by`: `all_users` or `collaborators_only`. It can be declared independently with the Issues checkbox omitted; it never implicitly enables Issues. Explicit Issues=false plus this choice is invalid. Export omits it behind disabled Issues. Existing declared-only GraphQL writes and independent observation remain; it does not manage existing issues or access grants. See [Issue permissions](../features/issue-creation-policy.feature.md).

## Discussions

`repository.settings.general.features.discussions` is an optional boolean, independent from discussion categories and individual discussions. Complete typed observation is required; absent values cannot become false. See [Discussions](../features/discussions.feature.md).

## Pull request access

Use boolean `repository.settings.general.features.pull_requests` and independent choice `features.pull_request_permissions.creation_allowed_by` under the same General root. Choices are `all_users` and `collaborators_only`. Explicit false plus a permission choice is invalid; omitted enabling checkbox is independent. Export hides its policy when Pull requests is disabled. Shared identity-checked GraphQL reconciliation skips fresh matches and sends only changed declared values. Mutation responses and independent same-ID readback confirm all original declared members, including the checkbox. No collaborator invitation/grant is implied. See [PR access](../features/pull-request-access.feature.md).

## Creation-policy observation normalization

Complete successfully acquired Issue/PR creation-policy strings outside their two UI choices project to the named `all_users` fallback with native value `ALL`. Empty and whitespace strings remain exact acquired evidence. One scalar record retains its YAML path, original token and fallback. Missing/null members, wrong types, failed/partial reads and incomplete PR availability remain errors. Repository IDs/names retain strict identity validation.

Export warns only for visible normalized policies and saves UI choices; disabled feature controls hide their latent policy without warnings or replay. Plan retains one write-required field decision even when desired intent equals the projected default. Omitted desired policies remain unmanaged. Apply acquires identity and only the originally declared PR availability, PR/Issue creation policies and Sponsorships members. Exact native matches require no mutation or second read; otherwise one shared mutation sends only changed declared values. Its response and an independent readback must confirm every original declared member against the original repository ID, including matching members omitted from the payload. Unknown raw policy strings remain write-required even at fallback equality. Checkbox-only intent does not acquire or warn about omitted policies; policy-only intent does not acquire or enable an omitted feature checkbox. Missing/null/wrong-type selected values and failed or partial acquisition stop before mutation. An unchanged unknown token, changed ID or declared-member drift after mutation fails with uncertain write effects. Only actual successful writes count toward later partial progress; earlier completed REST writes remain applied if access fails. No rollback occurs.

## Merge methods

Use General's `pull_requests.allow_merge_commits`, `allow_squash_merging` and `allow_rebase_merging` booleans. These independent optional checkboxes do not create individual merges. Explicit false remains intent. See [merge methods](../features/merge-methods.feature.md).

## Merge commit defaults

General's `pull_requests.merge_commits.default_commit_message` accepts `default_message`, `pull_request_title` or `pull_request_title_and_description`. It translates to a complete API pair. Explicitly disabling merge commits with this menu fails locally. Existing runtime dependency validation also rejects menu intent on an observed disabled merge method unless the declaration enables it. It never enables the method implicitly. See [merge defaults](../features/merge-commit-defaults.feature.md).

## Squash commit defaults

General's `pull_requests.squash_merging.default_commit_message` accepts `default_message`, `pull_request_title`, `pull_request_title_and_commit_details` or `pull_request_title_and_description`. One UI choice translates to both internal API members; partial API pairs are not public YAML. Explicitly disabling squash while declaring the menu fails. An omitted enabling checkbox remains omitted. See [squash defaults](../features/squash-commit-defaults.feature.md).

## Auto-merge and branch updates

General's `pull_requests.allow_auto_merge` and `always_suggest_updating_pull_request_branches` are independent optional booleans. Existing platform prerequisites still apply. See [PR options](../features/pull-request-options.feature.md).

## Automatic branch deletion

General's `pull_requests.automatically_delete_head_branches` controls GitHub's automatic deletion after merge; it does not manage or delete individual branches directly. See [branch deletion](../features/delete-branch-on-merge.feature.md).

## Commit-message observation normalization

A complete successful GraphQL read with two strings outside a merge/squash menu's UI choices projects to that menu's `default_message`. One provenance record retains both exact raw strings, its YAML menu path and fallback. Missing/null members, wrong types and failed or partial acquisition remain errors. Hidden disabled menus are omitted from export; visible normalized menus warn and save only the UI choice. Omitted desired menus remain unmanaged without warnings or replay.

Plan retains one warning and one menu-level write-required decision even when the declaration equals the projected default. That projection does not prove remote equality. Apply selectively reads declared menus, sends only declared native members in PATCH and verifies both raw members through a fresh GraphQL read. Unchanged out-of-UI readback fails with possible write effects retained. A failed preflight makes no repository PATCH; a later verification failure does not roll back other members in that PATCH.

## Web commit sign-off

`repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits` is an optional boolean. It controls the web-based commit sign-off requirement, not signature verification or existing commits. See [sign-off](../features/web-commit-signoff.feature.md).

## Repository archive lifecycle

The [v1 support boundary](scope.md#feature-adoption-scope) excludes ordinary settings
management on already-archived repositories. The explicit archive/unarchive
transition described here remains separate. The read-only observations and
matching cases below describe existing defensive implementation behavior, not
supported `export` → `plan` → `apply` coverage for an already-archived repository.

Use `repository.settings.general.danger_zone.archive_this_repository`. Omission makes no archive-specific request; false explicitly unarchives and true explicitly archives. Ordinary export includes the observed boolean after complete reads; unavailable state retains prior intent with info.

```yaml
version: daiksud/gh-settings@v0
repository:
  about: {description: Active maintenance}
  settings:
    general:
      danger_zone: {archive_this_repository: false}
```

Plan validates positive repository ID, canonical owner/name and archive state, then completely observes other managed resources without writing. Apply validates current identity before writes. Explicit unarchive runs first; final archive runs only after every other update succeeds and a fresh identity/state check. Archive PATCH contains only the internal archive member, and response/readback must confirm ID, name and state. Matching archive-only intent makes no lifecycle write.

On an already archived target, archive=true with only General/About, Topics, Labels, Rulesets, existing ID-bound Webhooks, Autolinks and deploy keys, selected runner custom labels, existing Environments, OIDC subject settings, pull request limits and exemptions, Copilot review effort and selected custom-instructions and MCP tools toggles, Actions retention, cache retention and storage, workflow permissions and fork approval, selected Actions and Agents repository variables, selected Environment variables, Security, and Pages can succeed as a read-only apply when every selected value is independently observed and matches. Label declarations own their complete supported objects, including the empty description default; an ID-less creation request is not a match. Ruleset declarations select complete objects by ID, including owned defaults; a changed or normalized selected policy is not a match. Webhooks require matching selected public controls and stable repository and webhook identities across fresh reads; ID-less webhook creation is not a match. Webhook secrets are opaque and unmanaged. Autolinks require stable repository and owner identities and exact matching selected ID-bound controls across two complete reads; ID-less creation is not a match. Deploy keys require a stable personal-public repository and owner identity and two complete reads matching each selected existing ID, title, canonical public key and grant. Creation requests and prune cannot match. Selected identity, creation time, API URL and enabled state must remain stable; unavailable enabled state remains unknown. Public key material is omitted from diagnostics. Separate reads do not establish key usability. For each selected runner, its complete custom-label set must match by Unicode simple-fold identity across independently acquired detail and label reads. The repository, owner and runner identities, label classes and each endpoint’s known or unknown optional label IDs must remain stable across two observations. Read-only label collisions, unavailable or ambiguous observations and prune are rejected. An empty selected custom-label list owns the empty set; an omitted or empty outer runner list selects nothing without prune. Workflow permissions compare only selected controls; an unknown default normalized for display cannot establish a selected match. OIDC matching verifies selected subject template and immutable settings across fresh identity-bound reads, preserving claim key order and treating omitted companions as unmanaged. Selected Actions and Agents variables require complete exact-name/value observations; a 404 is absent under the usual required-permissions precondition and is not a match. The read-only path does not include variable values in its diagnostics. Environments must exist with stable numeric identities across two reads, and their selected protection controls, deployment branch mode and patterns, and custom App IDs must match. Omitted controls remain unmanaged; unknown selected values cannot establish a match. Selected Environment variables require complete exact-name/value observations, with each existing parent Environment checked for stable numeric identity before and after its variable reads. A missing, changed or unavailable selected variable or parent cannot establish a match. Omitted and explicitly empty variable arrays select no values without prune; the read-only path does not expose variable values in diagnostics. Selected Copilot custom-instructions and MCP tools toggles must match on two independent reads; omitted toggles remain unmanaged. A null configured Copilot review effort cannot prove that an unavailable saved override is absent, so `default` does not qualify for this read-only path. The repository identity and archive state are checked before and after these reads, but separate reads do not lock the repository or form an atomic snapshot. Without pruning, explicitly empty collections that ordinary apply leaves unmanaged do not block this match; empty Topics and pull request exemption lists remain authoritative and must be observed. When other managed settings are present, any difference, unavailable observation, other unsupported managed setting or prune selector retains the fail-closed guard. With archive-only intent, a prune selector for an omitted collection remains unmanaged and is a no-op. Ordinary export can include other settings and unverified values and therefore may still require explicit unarchive. Explicitly unarchive before making changes. Failed unarchive stops later work. A confirmed unarchive survives a later failure; final archive failure can leave earlier updates applied. A write/readback failure can leave lifecycle state uncertain. There is no rollback, automatic retry, temporary unarchive, transaction or identity lock. Keep an external recovery route if the archived repository's own Actions cannot run.

Existing Administration read/write grants and actor authorization are prerequisites. Local fixtures do not establish live permissions or lifecycle behavior. See the [lifecycle specification](../features/repository-lifecycle.feature.md) and [ADR 0027](../adr/0027-repository-lifecycle.md).

## Plural commit-message groups

General / Pull Requests uses `merge_commits.default_commit_message` for the
Merge commits menu and `squash_merging.default_commit_message` for Squash merging.
Only these intermediate keys changed; `default_commit_message`, its scalar
choices, gate booleans, native title/message pairs and API fields are unchanged.
The former `merge_commit` and `squash_merge` groups are rejected, including old/new
names together. There are no aliases or automatic migrations. See the
[operational package boundary](../contributing/repository-automation.md#plural-ui-keys-and-operational-package)
before changing a declaration consumed by a published package.
