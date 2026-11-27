---
type: Reference
title: GitHub App permission matrix
description: Match supported YAML controls to acquisition, mutation and credential boundaries.
---

# GitHub App permissions by setting

## Read this matrix

Rows name actual current-source YAML paths. `[]` denotes each collection entry;
a parent path groups its descendants only under the stated common grant and
conditional boundaries. These are documented API requirements checked against
the client, not a blanket claim of live App verification or v1 completeness.
The [support policy](scope.md#feature-adoption-scope) and selected immutable
package still determine applicability. Legacy v0 families have a separate table.

`validate` is local and needs no settings API token. Export and plan do not
mutate settings, but their GET/query permissions can be write-equivalent. Apply
needs the union of acquisition, write and independent verification grants;
explicit prune can require additional complete inventories. Omitted or empty
unmanaged resources can skip requests; the [ownership contract](pruning.md) fixes
that behavior. Metadata read is automatic for repository App installations.

Read/write below refers to GitHub App repository permissions. Write includes the
same permission's read level, not another permission family. Public unauthenticated
reads do not establish an authenticated token's complete visibility or mutation
authority. A 404 means absent/disabled only where the individual family contract
allows it under its required-permissions precondition. Missing data is not false.

## General, About and Labels

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.about.description`<br>`repository.about.website`<br>`repository.settings.general.template_repository`<br>`repository.settings.general.default_branch`<br>`repository.settings.general.features.wikis`<br>`repository.settings.general.features.issues`<br>`repository.settings.general.features.projects`<br>`repository.settings.general.features.discussions`<br>`repository.settings.general.commits`<br>`repository.settings.general.danger_zone` | Metadata read; complete selected repository fields. | Administration write, with fresh identity/readback. | Archive/unarchive has its separate lifecycle contract; already-archived targets remain outside v1. [API](https://docs.github.com/en/rest/repos/repos#update-a-repository). |
| `repository.settings.general.features.sponsorships`<br>`repository.settings.general.features.pull_requests`<br>`repository.settings.general.features.issue_permissions`<br>`repository.settings.general.features.pull_request_permissions` | Authorized GraphQL repository queries and complete selected values; exact installation-token minimum is not independently established. | Authorized GraphQL updateRepository mutation and actor access; a REST Administration grant alone is not proof. | These controls use hasSponsorshipsEnabled, hasPullRequestsEnabled, issueCreationPolicy and pullRequestCreationPolicy. Verify the exact App query/mutation; missing/partial fields fail. [API](https://docs.github.com/en/graphql/reference/mutations#updaterepository). |
| `repository.settings.general.pull_requests` | Authorized GraphQL merge-method/default queries; exact App field visibility must be verified. | Administration write for repository REST updates, plus complete GraphQL preflight/readback. | Includes merge methods, `merge_commits.default_commit_message` and `squash_merging.default_commit_message`, auto-merge, branch updates and automatic branch deletion. REST repository visibility does not prove GraphQL visibility. [API](https://docs.github.com/en/rest/repos/repos#update-a-repository). |
| `repository.settings.general.releases.enable_release_immutability` | Administration read. | Administration write; owner-enforced disable is rejected. | Target 404 is interpreted only under the required-permissions precondition; owner enforcement then remains unknown. [API](https://docs.github.com/en/rest/repos/repos#check-if-immutable-releases-are-enabled-for-a-repository). |
| `repository.about.topics` | Metadata read; public listing also permits unauthenticated access. | Administration write. | Topics own complete selected membership; no additional Contents grant is inferred. [API](https://docs.github.com/en/rest/repos/repos#replace-all-repository-topics). |
| `repository.labels` | Issues read OR Pull requests read; public listing also permits unauthenticated access. | Issues write OR Pull requests write, including explicit label prune. | Administration alone does not authorize label mutation. IDs and local writeback do not require a GitHub Contents write grant. [API](https://docs.github.com/en/rest/issues/labels). |

## Access and branch rules

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.collaborators` | Metadata read for collaborator inventory PLUS Administration read for repository invitations. | Administration write for invitations, membership and explicit collaborator prune. | Personal-repository actor/identity and pending-invitation guards still apply. No organization role selection is added. [API](https://docs.github.com/en/rest/collaborators/collaborators). |
| `repository.settings.moderation.interaction_limits` | Administration WRITE for both cap GET and Exemption list GET, including nonmutating export/plan. | Administration write, including list replacement; no permission grant is changed. | These are persistent controls. Temporary interaction restrictions are excluded and have a different API contract. [API](https://docs.github.com/en/rest/interactions/repos). |
| `repository.settings.rulesets` | Metadata read is the GET endpoint minimum, PLUS ruleset write access for complete bypass_actors visibility. | Administration write plus full policy observation and applicable actor/account authorization; explicit Ruleset prune uses the same complete-read boundary. | A metadata-only successful GET cannot establish an empty bypass list. The client requires that member; ensure the actual App has the required write access. [API](https://docs.github.com/en/rest/repos/rules#get-a-repository-ruleset). |

## Actions

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.actions.general.actions_permissions`<br>`repository.settings.actions.general.workflow_permissions`<br>`repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention`<br>`repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors` | Administration read. | Administration write, with the acquisition permissions retained. | Execution selection, SHA pinning, workflow token defaults/approvals, retention and public contributor approval retain independent omission and normalization contracts. [API](https://docs.github.com/en/rest/actions/permissions). |
| `repository.settings.actions.general.cache.cache_retention` | Administration read. | Administration write. | Cache retention is a separate API from cache storage size. [API](https://docs.github.com/en/rest/actions/cache#get-github-actions-cache-retention-limit-for-a-repository). |
| `repository.settings.actions.general.cache.cache_size_eviction_limit` | Actions read. | Actions read PLUS Administration write. | Actions read is still needed for preflight and independent readback; Administration write is not a replacement for it. [API](https://docs.github.com/en/rest/actions/cache#set-github-actions-cache-storage-limit-for-a-repository). |
| `repository.settings.actions.runners` | Administration read for complete runner inventory, detail and labels. | Administration write for selected custom labels and explicit runner prune, plus complete readback. | Prune deregisters unlisted runner IDs. This does not create runners or issue registration tokens. [API](https://docs.github.com/en/rest/actions/self-hosted-runners). |
| `repository.settings.actions.policies` | Administration WRITE for policy list/detail GET, including event/actor policies and export/plan. | Administration write, with complete preflight/readback and explicit policy-prune guards. | A read-only plan operation still uses a credential capable of writing policies. [API](https://docs.github.com/en/rest/actions/policies#list-repository-actions-policies). |
| `repository.settings.actions.oidc` | Actions read. | Actions write, including complete selected/fresh companion observation. | Subject template and immutable-claim settings do not configure the external cloud trust policy. [API](https://docs.github.com/en/rest/actions/oidc). |

## Webhooks, Copilot, Pages and deploy keys

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.webhooks` | Webhooks read plus complete target identity. | Webhooks write for supported creation/configuration updates and explicit whole-webhook prune. | Existing Active/events updates remain outside v1; creation and the supported config PATCH are separate contracts. Secrets are opaque and unmanaged. [API](https://docs.github.com/en/rest/repos/webhooks). |
| `repository.settings.copilot.code_review.general` | Administration read; the repository GET officially accepts installation tokens. | Administration write, plus fresh selected-setting observation. | Covers configured review effort, MCP tools, public repository search when available, and custom instructions. The production answer26 installation has not been independently tested here; see evidence limits below. [API](https://docs.github.com/en/rest/copilot/copilot-code-review#get-copilot-code-review-settings-for-a-repository). |
| `repository.settings.pages` | Pages read. | Pages write PLUS Administration write, and Pages read for acquisition; applicable actor authorization is also required. | Create, update and explicit branch:none deletion have the same compound grant boundary. A public GET is not proof of write access or site readiness. [API](https://docs.github.com/en/rest/pages/pages). |
| `repository.settings.deploy_keys` | Administration read and complete public-key inventory/target identity. | Administration write for creation and explicit deploy-key prune. | Public keys only; no private-key or credential provisioning. Immutable replacements remain separate operations. [API](https://docs.github.com/en/rest/deploy-keys/deploy-keys). |

## Advanced Security

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.advanced_security.dependabot` | Administration read for alerts status and security-update configuration. | Administration write. | These configuration APIs use Administration, not Dependabot alerts permission. Paused metadata and enforcement are separate from configured enablement. [API](https://docs.github.com/en/rest/repos/repos#enable-vulnerability-alerts). |
| `repository.settings.advanced_security.code_security.codeql_analysis` | Administration read for default setup and its complete configuration. | Administration write. | Languages, query suite, runner and threat model retain their configured-only contract; success does not certify analysis execution. `scan_settings.query_suite` selects one suite. [API](https://docs.github.com/en/rest/code-scanning/code-scanning#get-a-code-scanning-default-setup-configuration). |
| `repository.settings.advanced_security.code_security.ai_scan` | Code scanning alerts read (API permission security_events). | Code scanning alerts read PLUS Administration write. | Applies to the two supported AI Scan enablement controls, not general scan execution or unsupported product switches. [API](https://docs.github.com/en/rest/code-scanning/code-scanning#get-ai-scan-enablement-for-a-repository). |
| `repository.settings.advanced_security.private_vulnerability_reporting` | Metadata read and public-repository applicability. | Administration write and applicable actor authorization. | This is independent of the repository security_and_analysis block. [API](https://docs.github.com/en/rest/repos/repos#check-if-private-vulnerability-reporting-is-enabled-for-a-repository). |
| `repository.settings.advanced_security.secret_protection` | Complete security_and_analysis visibility. GET repository lists Metadata read, but selected security fields additionally require applicable owner/admin/security-manager visibility. | Administration write PLUS applicable security authority/entitlement and complete selected-state readback. | A successful ordinary repository GET is insufficient evidence. Missing, null or denied security data never means false. [API](https://docs.github.com/en/rest/repos/repos#update-a-repository). |

## Environments

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.environments[].name`<br>`repository.settings.environments[].deployment_branches_and_tags`<br>`repository.settings.environments[].deployment_branch_or_tag_rules`<br>`repository.settings.environments[].deployment_protection_rules.required_reviewers.reviewers`<br>`repository.settings.environments[].deployment_protection_rules.required_reviewers.prevent_self_review`<br>`repository.settings.environments[].deployment_protection_rules.wait_timer`<br>`repository.settings.environments[].deployment_protection_rules.allow_administrators_to_bypass_configured_protection_rules` | Actions read for environment identity, policy, patterns and current protection state. | Actions read PLUS Administration write. Parent environment prune additionally needs Environments read for child variable/secret metadata. | No secret values are acquired. Parent prune requires complete known protection/child inventories and independent absence confirmation. [API](https://docs.github.com/en/rest/deployments/environments). |
| `repository.settings.environments[].deployment_protection_rules.enable_custom_rules_with_github_apps` | Actions read for configured protection bindings; available-integration lookup needs Administration read when required for App additions. | Actions read PLUS Administration write; that write grant includes the Administration read lookup boundary. | Selects existing available Apps only; installation/authorization and permission expansion are excluded. Parent prune also requires Environments read. [API](https://docs.github.com/en/rest/deployments/protection-rules). |
| `repository.settings.environments[].variables` | Environments read PLUS Actions read for parent identity. | Environments write PLUS Actions read; environment-variables prune uses the same child boundary. | If parent policy is also created/changed/pruned, union these grants with the parent row. Variable and secret metadata APIs use Environments; no secret values are managed. [API](https://docs.github.com/en/rest/actions/variables#list-environment-variables). |

## Secrets and variables

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.secrets_and_variables.actions.variables` | Variables read. | Variables write, including `--prune actions-variables`. | Applies to Actions repository variables. [API](https://docs.github.com/en/rest/actions/variables). |
| `repository.settings.secrets_and_variables.agents.variables` | Agent variables read. | Agent variables write, including `--prune agents-variables`. | Applies to Agents repository variables. [API](https://docs.github.com/en/rest/agents/variables). |

## Existing v0 families outside v1

These rows describe retained v0 implementation, not supported v1 coverage.

| YAML paths | Export / plan acquisition | Apply / explicit prune | Conditions and official contract |
| --- | --- | --- | --- |
| `repository.settings.autolinks` | Administration read plus complete personal-public identity and inventory. | Administration write for immutable creation or explicit autolinks prune. | Outside v1 under the confirmed Free-plan scope. The documented APIs and retained implementation do not establish Free UI entitlement. [API](https://docs.github.com/en/rest/repos/autolinks). |
| `repository.settings.branch_protection_rules` | Complete GraphQL BranchProtectionRule and actor visibility, including unowned raw companions. | Authorized GraphQL create/update/delete and independent full readback. Exact App grant sufficiency has not been live-tested; do not derive it from a REST Administration table. | Outside v1 by product choice: use Rulesets. Preserve existing protection; no automatic migration or removal. Organization-only actor/control restrictions remain. [API](https://docs.github.com/en/graphql/reference/mutations#createbranchprotectionrule). |

## Registration, installation and each issued token

[App registration](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app)
declares requested capabilities. An installation must separately approve those
grants and repository access. Each
[installation token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)
is then restricted to the selected repositories and requested permission levels;
it cannot exceed the approved installation. Inspect the actual issued-token
permissions, not only the App's public registration. Pending grant approval and
repository selection can differ from registration. No row authorizes changing any
of these layers.

Use one installed App with separate short-lived, repository-scoped tokens for
export/plan and apply as needed. Request only the union required by that step's
actual declaration. In particular, cap/exemption and Actions-policy GETs need
Administration write, AI Scan apply needs both permission families, and cache-size
apply retains Actions read. Protect a write-capable planning credential even
though the plan command is read-only. Do not add a PAT/personal-login fallback.

A workflow's `permissions:` block controls its GITHUB_TOKEN, not the independently
minted App token. App-token action inputs must request the needed App grants as
well as repository scope; an App registration checkbox alone does not change an
existing workflow request. Token lifetime/revocation and trusted executing code
remain separate responsibilities. See the [workflow trust model](../../guides/github-actions.md#authentication-and-app-permissions).

The PR sample's exact candidate Git blob retrieval additionally needs Contents
read. That is a preparation requirement, not a settings mutation grant. A
configuration file's local ID writeback does not itself need GitHub Contents
write. Checkout/download/metadata prerequisites are independent from `validate`.

## Credential and evidence limits

GraphQL-backed General controls and classic branch protection require the exact
query/object visibility and mutation actor authorization described by their
contracts. The repository REST permission list does not prove GraphQL field
access. Classic organization-only restrictions, unsupported actors and unknown
preservation companions remain blockers rather than being discarded. The
[General](general.md), [Rulesets](rulesets.md) and
[classic v0](legacy-classic-protection.md) contracts retain those boundaries.

Repository security-block visibility can require owner/admin/security-manager
authority even after an ordinary repository GET succeeds. The client rejects
missing/partial observations or retains prior export intent; it does not convert
invisibility into disabled settings. Endpoint permission, actor authorization,
plan entitlement, feature execution and product support are separate facts.

The official repository Copilot code-review GET/PATCH contract accepts
installation tokens with Administration read/write. The production `answer26`
installation's access to that endpoint has not been independently live-tested.
Owner OAuth reads and public App-registration metadata are not evidence of an
issued operating-App token's access or of equality with the owner-observed
candidate. Configured review effort, inherited effective values and unavailable
overrides remain distinct; no projected fallback proves an exact match.

Environment policy uses Actions read for observation and Administration write
for mutation. Environment variable access is a separate Environments grant, and
parent prune also reads [child secret metadata](https://docs.github.com/en/rest/actions/secrets#list-environment-secrets)
with Environments read. No secret-value management is implied. Available custom
protection App lookup uses Administration read and never installs an App.

The matrix does not certify App grants, enforcement, scan/review execution,
subscription eligibility or safe live mutation. Use the family contracts and
verify the actual credential in the intended trusted runtime without publishing
keys, tokens, private variable values or sensitive endpoints.

<!-- Reviewed path groups; documentation coverage test rejects missing/stale rows. -->
<!-- permission-path v1: repository.about.description -->
<!-- permission-path v1: repository.about.website -->
<!-- permission-path v1: repository.settings.general.template_repository -->
<!-- permission-path v1: repository.settings.general.default_branch -->
<!-- permission-path v1: repository.settings.general.features.wikis -->
<!-- permission-path v1: repository.settings.general.features.issues -->
<!-- permission-path v1: repository.settings.general.features.projects -->
<!-- permission-path v1: repository.settings.general.features.discussions -->
<!-- permission-path v1: repository.settings.general.commits -->
<!-- permission-path v1: repository.settings.general.danger_zone -->
<!-- permission-path v1: repository.settings.general.features.sponsorships -->
<!-- permission-path v1: repository.settings.general.features.pull_requests -->
<!-- permission-path v1: repository.settings.general.features.issue_permissions -->
<!-- permission-path v1: repository.settings.general.features.pull_request_permissions -->
<!-- permission-path v1: repository.settings.general.pull_requests -->
<!-- permission-path v1: repository.settings.general.releases.enable_release_immutability -->
<!-- permission-path v1: repository.about.topics -->
<!-- permission-path v1: repository.labels -->
<!-- permission-path v1: repository.settings.collaborators -->
<!-- permission-path v1: repository.settings.moderation.interaction_limits -->
<!-- permission-path v1: repository.settings.rulesets -->
<!-- permission-path v1: repository.settings.actions.general.actions_permissions -->
<!-- permission-path v1: repository.settings.actions.general.workflow_permissions -->
<!-- permission-path v1: repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention -->
<!-- permission-path v1: repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors -->
<!-- permission-path v1: repository.settings.actions.general.cache.cache_retention -->
<!-- permission-path v1: repository.settings.actions.general.cache.cache_size_eviction_limit -->
<!-- permission-path v1: repository.settings.actions.runners -->
<!-- permission-path v1: repository.settings.actions.policies -->
<!-- permission-path v1: repository.settings.actions.oidc -->
<!-- permission-path v1: repository.settings.webhooks -->
<!-- permission-path v1: repository.settings.copilot.code_review.general -->
<!-- permission-path v1: repository.settings.pages -->
<!-- permission-path v1: repository.settings.deploy_keys -->
<!-- permission-path v1: repository.settings.advanced_security.dependabot -->
<!-- permission-path v1: repository.settings.advanced_security.code_security.codeql_analysis -->
<!-- permission-path v1: repository.settings.advanced_security.code_security.ai_scan -->
<!-- permission-path v1: repository.settings.advanced_security.private_vulnerability_reporting -->
<!-- permission-path v1: repository.settings.advanced_security.secret_protection -->
<!-- permission-path v1: repository.settings.environments[].name -->
<!-- permission-path v1: repository.settings.environments[].deployment_branches_and_tags -->
<!-- permission-path v1: repository.settings.environments[].deployment_branch_or_tag_rules -->
<!-- permission-path v1: repository.settings.environments[].deployment_protection_rules.required_reviewers.reviewers -->
<!-- permission-path v1: repository.settings.environments[].deployment_protection_rules.required_reviewers.prevent_self_review -->
<!-- permission-path v1: repository.settings.environments[].deployment_protection_rules.wait_timer -->
<!-- permission-path v1: repository.settings.environments[].deployment_protection_rules.allow_administrators_to_bypass_configured_protection_rules -->
<!-- permission-path v1: repository.settings.environments[].deployment_protection_rules.enable_custom_rules_with_github_apps -->
<!-- permission-path v1: repository.settings.environments[].variables -->
<!-- permission-path v1: repository.settings.secrets_and_variables.actions.variables -->
<!-- permission-path v1: repository.settings.secrets_and_variables.agents.variables -->
<!-- permission-path v0: repository.settings.autolinks -->
<!-- permission-path v0: repository.settings.branch_protection_rules -->
