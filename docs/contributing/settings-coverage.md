---
type: Reference
title: Repository settings coverage evidence
description: Separate target UI eligibility, official API evidence and current implementation coverage.
sources:
  - id: v1-goal
    resource: https://github.com/daiksud/gh-settings/issues/668
  - id: settings-contract
    resource: ../settings.md
  - id: official-openapi
    resource: https://github.com/github/rest-api-description/blob/bd6df26d2a0104874fe5025a25ebcde356413364/descriptions/api.github.com/api.github.com.json
---

## Reading this inventory

This contributor reference records coverage evidence for the [v1 goal][v1-goal],
under the [feature adoption policy](../settings.md#feature-adoption-scope).
The target is a personally owned public repository. Account-plan names and
Preview status do not exclude a configurable target UI control. Organization
settings and API-only fields do not enlarge the target UI surface.

The inventory was initially checked against source commit
`3f7fb20acef7a911708cc5effff69e4d5d1a5e41` and official documentation on
**2026-11-13**. The [fixed official OpenAPI description][openapi] was updated on
2026-11-12. UI dates below belong to the linked observation records: reading an
old record or an official guide is not a new inspection of an authenticated UI.
An unconfirmed row is an evidence gap, not proof that a control or API is absent.
Source coverage rows follow the contracts in this revision; the initial inventory
is not certification of a later release. This reference alone adds no accepted
YAML, command, Action operation or live setting.

Each row keeps four different questions separate:

- **Policy:** `Eligible` is within the adopted product boundary when configurable
  in the target UI; `Excluded` is an explicit product decision; `Non-target`
  identifies organization/private-only or API-only concepts.
- **UI evidence:** `Observed` refers to the [2026-11-13 read-only pass](#current-ui-observation-boundary); `Recorded` links an earlier dated observation; `Documented`
  means an official guide describes the control; `Unconfirmed` means target UI
  availability, choices or prerequisites still need evidence. None means that
  the current credential can execute the feature.
- **API evidence:** `Read/write` identifies an official settings read/update
  contract, including separately named lifecycle operations. `Documented limitation`
  identifies an explicit official API restriction. `Partial` identifies
  a known endpoint without complete settings-management evidence. `Unconfirmed`
  means that no complete applicable contract is established here.
- **Source coverage:** `Implemented` refers only to the linked current-source
  configuration and behavior contract. `Not implemented` is distinct from an API
  limitation. The [cross-cutting gaps](#cross-cutting-product-gaps) prevent these
  rows from being a claim of complete CLI/Action v1 support.

API definitions establish modeled request/response support. They do not establish
live persistence, complete response fields, permissions for a particular token,
feature execution, billing or enforcement. The [testing guide](testing.md)
distinguishes fixture evidence, executable checks and live observations. Keep
implementation work and delivery order in Issues rather than this reference.

### Current UI observation boundary

A read-only browser pass on the personally owned public
[gh-settings repository](https://github.com/daiksud/gh-settings/settings) on
2026-11-13 inspected General, the expanded sidebar and the pages named below.
It saved no form, toggled no setting and made no API mutation. An unsaved classic
branch-protection form and an existing Ruleset editor were inspected. Visibility
of a checkbox does not prove its dependent form is usable or its value writable.
The Collaborators page required sudo-mode authentication, which was not performed.

| Inspected surface | Observation boundary |
| --- | --- |
| About dialog on the [repository home page](https://github.com/daiksud/gh-settings) | Description, Website, Topics and Include in home page: Releases, Deployments, Packages; dialog canceled |
| [General](https://github.com/daiksud/gh-settings/settings) | Visible controls, including the additional General controls below; global Disable branch protection rules action was not invoked |
| [Branches](https://github.com/daiksud/gh-settings/settings/branches) and [Rulesets](https://github.com/daiksud/gh-settings/settings/rules) | Classic-rule top-level controls and Ruleset PR options; coverage checkbox visible/enabled, threshold subform expanded 2026-11-14; team-review checkbox visible, usable team selection unconfirmed |
| [Tags](https://github.com/daiksud/gh-settings/settings/tag_protection) | Protected tags described as deprecated, with navigation to tag Rulesets; no active legacy setter observed |
| [Interaction limits](https://github.com/daiksud/gh-settings/settings/interaction_limits) and [Code review limits](https://github.com/daiksud/gh-settings/settings/code_review_limits) | Temporary interaction controls, persistent PR controls and restriction on approving/request-changes reviews |
| [Actions OIDC](https://github.com/daiksud/gh-settings/settings/actions/oidc-configuration) | Default-template control; immutable-subject control disabled/checked on the current repository and editable/unchecked on legacy personal-public dotfiles, observed unsaved 2026-11-14 |
| [Copilot Code review](https://github.com/daiksud/gh-settings/settings/copilot/code_review) | Default/maximum effort, custom instructions, MCP, approvals and external-license restriction; a No access banner did not make these forms non-configurable |
| [Copilot Cloud](https://github.com/daiksud/gh-settings/settings/copilot/coding_agent), [Internet access](https://github.com/daiksud/gh-settings/settings/copilot/internet_access) and [MCP servers](https://github.com/daiksud/gh-settings/settings/copilot/mcp) | Cloud configuration, two firewall sections and a shared JSON MCP configuration editor; no content submitted |
| [Dreaming](https://github.com/daiksud/gh-settings/settings/copilot/dreaming) and [Agent suggestions](https://github.com/daiksud/gh-settings/settings/suggestions) | Dreaming toggle unavailable for this account; four automation-level choices visible for suggestions |
| [Advanced Security](https://github.com/daiksud/gh-settings/settings/security_analysis) | Individual security controls, private-reporting details, two Dependabot presets and New rule entry; configuration not changed |
| [Environments](https://github.com/daiksud/gh-settings/settings/environments) and [Pages](https://github.com/daiksud/gh-settings/settings/pages) | Existing Environment built-ins, branch/tag selection and custom-App help; no available custom App selection demonstrated. Absent Pages site showed source/Branch None; visibility was promotional text, not an editable control |
| Webhooks, Deploy keys, GitHub Apps, Actions Runners and Codespaces | Add/create entry points or inventories observed; creation/prebuild forms and App-specific settings not fully inspected; no installed-App names or private configuration retained |

Actions Policies were visible; Policy insights was observational rather than a
settings form. The sidebar also exposed Secrets and variables entries for Actions,
Agents, Codespaces and Dependabot. Actions/Agents exposed Variables separately;
Codespaces/Dependabot showed secrets only. Autolinks was not observed in this sidebar;
its row retains official-document evidence only. This pass is broader than the
older mappings but is not an exhaustive survey of every conditional form.

## About, Labels and General

The [2026-10-28 UI inventory](ui-repository-mapping.md#ui-inventory-and-paths)
owns detailed control names, conditional menus and their conversions. Supported
groups below link the public contract instead of copying every field.

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| About: description, website and topics | Eligible | Observed, 2026-11-13 | Read/write: [repository REST][repos-api], including Topics GET/PUT | Implemented: [supported fields](../settings.md#supported-fields), [topics](../settings.md#topics) |
| About: Include in the home page Releases, Deployments and Packages | Eligible | Three controls observed, 2026-11-13 | Unconfirmed complete settings contract | Not implemented |
| Labels: name, description, color and collection | Eligible | Recorded, 2026-10-28 | Read/create/update/delete: [Labels REST][labels-api] | Implemented: [labels](../settings.md#labels), with ID ownership, matching-object no-op, independent create/update readback and explicit prune |
| Template repository and default branch | Eligible | Recorded, 2026-10-28 | Read/write: [repository REST][repos-api] | Implemented: [template](../settings.md#template-repository), [default branch](../settings.md#default-branch) |
| Features: Wikis, Issues, Projects, Discussions, Sponsorships and Pull requests | Eligible | Recorded, 2026-10-28 | Read/write: repository REST and [Repository / updateRepository GraphQL][repositories-graphql], as mapped in the UI inventory | Implemented: [supported fields](../settings.md#supported-fields), [sponsor visibility](../settings.md#sponsor-button-visibility), [PR access](../settings.md#pull-request-access) |
| Issue and PR creation permissions | Eligible | Recorded conditional controls, 2026-10-28 | Read/write: [Repository / updateRepository GraphQL][repositories-graphql] | Implemented: [issue policy](../settings.md#issue-creation-policy), [PR policy](../settings.md#pull-request-access) |
| Merge methods, commit-message choices, auto-merge, branch updates and automatic branch deletion | Eligible | Recorded conditional controls, 2026-10-28 | Read: Repository GraphQL; update: [repository REST][repos-api] | Implemented: [merge methods](../settings.md#merge-methods), [message choices](../settings.md#commit-message-observation-normalization), [auto-merge](../settings.md#auto-merge-and-branch-updates), [branch deletion](../settings.md#automatic-branch-deletion) |
| Web commit sign-off | Eligible | Recorded, 2026-10-28 | Read/write: [repository REST][repos-api] | Implemented: [sign-off](../settings.md#web-commit-sign-off) |
| Immutable releases | Eligible | [Recorded General / Releases control](ui-security-mapping.md#verified-controls-and-placement), 2026-10-28 | Read/enable/disable: [repository immutable-release endpoints][repos-api] | Implemented: [immutable release policy](../settings.md#immutable-release-policy) |
| Archive/unarchive | Eligible | Recorded, 2026-10-28 | Read/write: [repository REST][repos-api] | Implemented: [archive lifecycle](../settings.md#repository-archive-lifecycle) |
| Social preview image | Eligible | Observed, 2026-11-13 | Partial: [Repository GraphQL][repositories-graphql] exposes `openGraphImageUrl` and `usesCustomOpenGraphImage`; image update contract unconfirmed | Not implemented |
| Restrict wiki editing to collaborators | Eligible | Recorded, 2026-10-28 | Unconfirmed complete settings contract | Not implemented |
| Preserve this repository | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings contract | Not implemented |
| Allow comments on individual commits | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings contract | Not implemented |
| Include Git LFS objects in archives | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings contract | Not implemented |
| Limit branches/tags updated in one push | Eligible, including Preview | Observed, 2026-11-13 | Unconfirmed complete settings contract | Not implemented |
| Auto-close issues with merged linked pull requests | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings contract | Not implemented |
| Disable branch protection rules globally | Eligible | Observed Danger Zone action, 2026-11-13; not invoked | Unconfirmed complete setting/state contract; individual rule deletion does not establish equivalence | Not implemented |

Earlier observations of the additional controls appear in the mapping's
[additional General controls](ui-repository-mapping.md#unsupported-controls-and-other-families).
Their previous omission from an implementation slice is not a technical API
limitation. Repository name and Danger Zone identity operations are recorded
separately under [explicit boundaries](#explicit-product-and-target-boundaries).

## Access, moderation and branch protection

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| Collaborators: existing personal-repository membership | Eligible | [Documented personal-repository control][personal-access] | Read/add/remove: [Collaborators REST][collaborators-api]; arbitrary role selection is organization-only | Implemented: [user-ID binding, active/pending reconciliation](../settings.md#personal-collaborators); [explicit guarded removal](../settings.md#collaborator-pruning) implemented |
| Pending collaborator invitations and cancellation | Eligible | [Documented personal invitations][personal-invitations] | Read/cancel: [Invitations REST][invitations-api]; creation uses Collaborators REST and may return a pending invitation | Implemented: complete export, initial invitation and independent readback; pending remains distinct from accepted access. Explicit cancellation with independent readback implemented; email-only roundtrip remains a gap |
| Email-only collaborator invitation | Eligible | Documented in the personal invitation guide | Partial: the confirmed collaborator setter addresses a username; complete email-only observation/redeclaration is unconfirmed | Not implemented; do not silently discard unresolved identities |
| Persistent PR creation cap, draft counting and Exemption list | Eligible | [Recorded personal-public controls](../features/pull-request-limits.feature.md), 2026-11-11 | Read/write: [repository interaction-limit creation-cap and bypass-list endpoints][interactions-api] | Implemented: [cap](../settings.md#persistent-pull-request-limits), [Exemption list](../settings.md#pull-request-limit-exemption-list) |
| Temporary interaction restrictions | Eligible | Observed, 2026-11-13: existing users, prior contributors and collaborators | Read/set/remove: [repository interaction restrictions REST][interactions-api] | Not implemented; expiry is separate from persistent PR limits |
| PR limit automatic exemption rules | Eligible | Referenced by the [current cap contract](../settings.md#persistent-pull-request-limits); full target UI inventory unconfirmed | Unconfirmed complete rule read/update contract | Not implemented |
| Code review limits: limit approving/request-changes reviews to users explicitly granted read or higher access | Eligible | Observed checkbox, 2026-11-13; [documented public control][review-limits-ui] | Unconfirmed complete settings read/update contract | Not implemented |
| Classic branch protection: pattern rules and controls | Eligible | Observed unsaved form, 2026-11-13; [documented public availability][protected-branches] | Read/create/update/delete: [BranchProtectionRule GraphQL][branches-graphql]; [protected-branch REST][branch-protection-api] addresses concrete branches | Partial: [classic protection](../settings.md#classic-branch-protection) implements exact patterns, conversation resolution, signed commits, linear history, branch locking, administrator enforcement, deletions and complete active PR-review, required status-check and deployment groups; remaining controls are unowned companions; explicit [classic pruning](../settings.md#classic-branch-protection-pruning) removes whole unlisted rules after complete private snapshot/absence checks. Lock branch and administrator enforcement were observed in the unsaved form on 2026-11-14; provider source corroborates administrator polarity and PR-required/count-zero mapping; review checkboxes/counts were observed unsaved on 2026-11-14. Fork-sync remains unowned. The status-check source menu was observed unsaved on 2026-11-14; explicit any/App-node mapping follows the official input contract, with active alias coherence as a product observation boundary. Required deployments and empty selection were observed unsaved on 2026-11-14, with read-only Environment inventory corroboration. No live protection update was tested |

Personal repositories have an owner and collaborators, not the organization role
model. The API's `permission` parameter must not manufacture personal-repository
role choices. Invitation state must remain distinct from a confirmed grant.
For classic protection, configurable wildcard patterns require a rule-level
model; querying protection for existing branch names alone is not a complete
inventory. Rulesets and classic protection remain separate resources, with no
implicit migration, replacement or removal.

## Rulesets

The [Rulesets UI mapping](ui-ruleset-mapping.md) records the supported branch/tag
model. [Repository Rulesets REST][rules-api] provides the collection, individual
read and create/update/delete operations; complete observation and preservation
of protections remain separate obligations from endpoint existence.

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| Branch/tag identity, enforcement, targets, bypass actors and supported rules | Eligible | Recorded in the Rulesets mapping, 2026-10-28 | Read/write/lifecycle: repository Rulesets REST | Implemented within [Rulesets](../settings.md#rulesets), including creation/update/deletion restrictions, review/check rules, deployments, scanning/quality thresholds and automatic Copilot review |
| Additional approval for unattributed Copilot PRs | Eligible, including Preview | Observed checked checkbox, 2026-11-13; [documented behavior][rules-ui] | Read/write: PR rule `require_extra_approval_for_unattributed_changes` in repository Rulesets REST | Implemented: explicit true/false, selected PR-group default true, complete export, fresh drift checks, conditional whole-object update and independent readback; [ownership contract](../settings.md#pr-review-and-required-check-parameters) |
| Restrict code coverage | Eligible, including Preview | Checkbox observed 2026-11-13; threshold form observed 2026-11-14: optional blank inputs, 0–100, arbitrary step; [UI guide][coverage-ui] | Read/write: `code_coverage`, `minimum_coverage`, `max_coverage_drop`; optional response members have no documented default; [API announcement][coverage-announcement] also lists Team/Enterprise prerequisites | Implemented: [branch thresholds](../settings.md#code-coverage-thresholds), declaration-only zero defaults, complete acquired values, fresh drift checks and independent confirmation; live acceptance/enforcement unverified |
| Require review from specific teams | Eligibility unresolved | Top-level checkbox observed, 2026-11-13, but [official guide][rules-ui] says unavailable on user-owned repositories; usable team selection unconfirmed | Partial: team-review parameters exist; target applicability unresolved | Not implemented; do not equate a visible checkbox with selectable organization teams |
| Require secret scanning alerts to be resolved | Eligible if target UI is configurable | [Documented rule][rules-ui]; target UI availability unconfirmed | Unconfirmed complete settings read/write mapping in this audit | Not implemented |
| Merge queue | Eligible if target UI is configurable | Target UI availability/choices unconfirmed | Partial: rules schema exists; complete applicable control mapping unconfirmed | Not implemented; current source rejects acquired policy |
| Evaluate enforcement | Eligible if target UI is configurable | Absent from the recorded personal-repository menu; current eligibility unconfirmed | Partial: API enforcement concept exists; target UI correspondence unconfirmed | Not implemented; current source rejects it rather than treating it as Disabled |
| Commit/branch/tag metadata restrictions | Eligible if target UI is configurable | Target UI availability/choices unconfirmed | Partial: repository rule schemas exist; complete applicable control mapping unconfirmed | Not implemented |
| License compliance | Eligible if target UI is configurable | Target UI availability/choices unconfirmed | Unconfirmed complete applicable read/write contract | Not implemented |
| Review-dismissal restrictions | Eligible if target UI is configurable | Target UI applicability and actor choices unconfirmed | Partial: PR rule schema exists; absence does not prove disabled protection | Not implemented; acquired restrictions remain a protection boundary |

The personal-public coverage checkbox is fresh UI evidence despite the
[coverage announcement][coverage-announcement]'s plan wording. Neither the
checkbox nor the API definition proves that the unexpanded threshold form or a
live mutation succeeds. Preserve those evidence limits rather than excluding
the control for cost. The team-review discrepancy likewise remains explicit. The
[current Rulesets contract](../settings.md#plan-apply-and-export) owns rejection
of unsupported protections; this inventory does not relax it.

## Actions, Copilot and Agents

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| Actions execution, selected allow/block patterns and full-SHA policy | Eligible | [Recorded Actions UI](ui-actions-mapping.md#inventory-and-paths), 2026-10-28 | Read/write: [Actions permissions and selected-actions REST][actions-policies-api] | Implemented: [execution policy](../settings.md#actions-execution-policy) |
| Workflow token permissions and PR creation/approval | Eligible | Recorded Actions UI, 2026-10-28 | Read/write: Actions workflow-permissions REST | Implemented: [workflow permissions](../settings.md#default-workflow-permissions) |
| Public-fork workflow approval | Eligible | Recorded Actions UI, 2026-10-28 | Read/write: Actions fork-PR contributor-approval REST | Implemented: [fork approval](../settings.md#fork-pr-workflow-policies) |
| Check/run/status/artifact/log retention | Eligible | Recorded Actions UI, 2026-10-28 | Read/write: Actions artifact-and-log-retention REST | Implemented: [retention](../settings.md#check-run-status-artifact-and-log-retention) |
| Cache retention and size eviction limit | Eligible | [Recorded cache controls](ui-actions-mapping.md#inventory-and-paths) | Read/write: [Actions cache retention/storage-limit REST][actions-cache-api] | Implemented: [cache policy](../settings.md#actions-cache-policy) |
| Actions Policies: targeting, event and actor restrictions | Eligible | [Recorded Policies UI](../features/actions-event-policies.feature.md); page/creation entry observed, 2026-11-13 | Read/create/update/delete: repository `actions/policies` in [Actions Policies REST][actions-policies-api] | Implemented: [event policies](../settings.md#actions-event-policies), including explicit prune |
| Actions repository variables | Eligible | [Recorded variables UI](../features/repository-variables.feature.md) | Read/create/update/delete: [Actions Variables REST][variables-api] | Implemented: [Actions variables](../settings.md#actions-repository-variables) |
| Agents repository variables | Eligible | Recorded variables UI | Read/create/update/delete: [Agents Variables REST][agents-variables-api] | Implemented: [Agents variables](../settings.md#agents-repository-variables) |
| Copilot custom-instructions usage, MCP usage and default/Lite/Balanced effort | Eligible | [Recorded effort and companion controls](../features/copilot-review-effort.feature.md) | Read/write: [repository Copilot settings GET/PATCH][copilot-api] | Implemented: [instructions](../settings.md#copilot-code-review-custom-instructions), [MCP](../settings.md#copilot-code-review-mcp-tools), [effort](../settings.md#copilot-code-review-effort-level) |
| Copilot review effort Max | Eligible, including Preview | [Documented Max choice][copilot-max]; availability is repository-specific | Read/write: repository settings accepts efforts in acquired `available_review_efforts`; [API announcement][copilot-api-announcement] confirms UI/API choices | Implemented: [effort declaration](../settings.md#copilot-code-review-effort-level), automatic export, conditional apply and independent readback; fresh availability is required |
| Copilot review severity filtering | Eligible if target UI is configurable | Current target UI choice/availability unconfirmed | Read/write: repository Copilot settings `severity_filter`; availability conditions apply | Not implemented; existing adapters preserve the acquired companion |
| Copilot approval controls and optional path restrictions | Eligible | Observed toggle and up-to-15 path patterns, 2026-11-13 | Documented limitation: [configuration guide][copilot-config] says approvals are not available through the API | Not implemented |
| Copilot maximum review effort policy | Eligible | Observed selector, 2026-11-13 | Documented limitation: configuration guide says maximum effort is not available through the API | Not implemented |
| Block reviews requested with an external Copilot license | Eligible | Observed checkbox, 2026-11-13 | Documented limitation: configuration guide excludes billing settings from API support; no separate applicable contract confirmed | Not implemented |
| OIDC Subject claim: default template/custom claim keys | Eligible | Observed default checkbox and ordered custom textarea, 2026-11-14; no client-side minimum, no Save | Read/write: [OIDC customization/sub GET/PUT][oidc-api]; complete active union with exact immutable companion replay | [Implemented active template](../settings.md#actions-oidc-subject-template): export, validation, plan, no-op/update and independent readback; fixtures do not prove cloud authentication or server acceptance of empty keys |
| OIDC immutable subject claim | Eligible when configurable | Observed disabled/checked control with creation/rename-date condition, 2026-11-13; editable legacy control observed 2026-11-14 | Read/write: OIDC GET/PUT includes `use_immutable_subject`; optional missing read field remains unknown | [Implemented configuration](../settings.md#actions-oidc-subject-template): independently selected boolean, leaf export/retention, known no-op and fresh active-template replay/readback. Editable legacy dotfiles checkbox observed unsaved 2026-11-14; persisted legacy opt-out and effective token format unverified; no date heuristic or reversibility promise |
| Repository self-hosted runner configuration | Eligible if target UI is configurable | Observed inventory and New self-hosted runner, 2026-11-13; registration form not inspected | Partial: [runner REST][runners-api] covers inventory/labels/registration/removal; it does not establish every UI control or host installation | Not implemented |
| Agent suggestions for issues: automation level | Eligible | Observed Full control, Cautious, Balanced and Full automation, 2026-11-13; [documented configuration][suggestions-ui] | Unconfirmed settings read/update contract; issue-suggestion payloads do not establish the setting | Not implemented |
| Copilot Cloud: sandbox size | Eligible | Observed No repository default / S / M / L, 2026-11-13 | Unconfirmed complete settings read/update contract | Not implemented |
| Copilot Cloud: require approval for workflow runs | Eligible | Observed, 2026-11-13 | Partial: [Cloud configuration GET][openapi] exposes `require_actions_workflow_approval`; setter unconfirmed | Not implemented |
| Copilot Cloud: allow automations | Eligible | Observed, 2026-11-13 | Partial: Cloud GET exposes `is_automations_enabled`; setter unconfirmed | Not implemented |
| Copilot Cloud: allow shared automations | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings read/update contract | Not implemented |
| Copilot Cloud: require write access to trigger automations | Eligible | Observed, 2026-11-13 | Partial: Cloud GET exposes `require_write_access_for_automation_triggers`; setter unconfirmed | Not implemented |
| Copilot Cloud: only update the triggering resource | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings read/update contract | Not implemented |
| Copilot Cloud: require workflow approval for automation PRs | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings read/update contract distinct from ordinary workflow approval | Not implemented |
| Copilot Cloud: CodeQL, review, secret and dependency validation toggles | Eligible | Four independent controls observed, 2026-11-13 | Partial: Cloud GET exposes `enabled_tools` booleans; setters unconfirmed | Not implemented; these are configuration, not requests to execute checks |
| Copilot Cloud internet access: firewall, recommended and custom allowlists | Eligible | Separate controls observed, 2026-11-13 | Partial: Cloud GET exposes firewall/allowlist fields; setters unconfirmed | Not implemented |
| Copilot code-review internet access: firewall, recommended and custom allowlists | Eligible | Separate controls observed, 2026-11-13 | Unconfirmed complete review-specific settings read/update contract | Not implemented; Cloud endpoint is not assumed equivalent |
| Shared non-secret MCP server configuration | Eligible | Observed JSON configuration editor, 2026-11-13; not a repository-file editor | Partial: Cloud GET exposes `mcp_configuration`; complete shared update contract unconfirmed | Not implemented; secret values remain excluded |
| Copilot Dreaming | Eligible if configurable, including Preview | Observed unavailable toggle requiring a different Copilot entitlement, 2026-11-13 | Unconfirmed settings read/update contract | Not implemented; current account unavailability is not a personal-public policy exclusion |

## Security and analysis

The [security mapping](ui-security-mapping.md) and
[current security contract](../settings.md#security-ui-settings-not-managed-by-gh-settings)
separate product headings from individual controls. A Code Security heading is
not evidence of a product-wide enable switch. Configuration is also distinct
from submitting dependency data, executing scans or generating fixes.

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| Dependabot alerts and security updates | Eligible | Recorded security controls | Read/enable/disable: [repository vulnerability-alerts and automated-security-fixes REST][repos-api] | Implemented: [Dependabot](../settings.md#dependabot-alerts-and-security-updates); intrinsic Dependency graph side effect is documented |
| Secret protection and push protection | Eligible | [Recorded UI](ui-security-mapping.md#verified-controls-and-placement), 2026-10-28 | Read/write: repository `security_and_analysis` in [repository REST][repos-api] | Implemented: [secret scanning](../settings.md#secret-scanning-and-push-protection) |
| Private vulnerability reporting enablement | Eligible | Recorded UI, 2026-10-28 | Read/enable/disable: repository private-vulnerability-reporting REST | Implemented: [reporting](../settings.md#private-vulnerability-reporting) |
| CodeQL default setup, languages, query suite and runner configuration | Eligible | [Recorded configuration evidence](../features/security.feature.md#codeql-default-configuration-controls) | Read/write: [code-scanning default-setup GET/PATCH][code-scanning-api] | Implemented: [setup](../settings.md#codeql-default-setup), [configuration](../settings.md#codeql-default-configuration) |
| AI Scan for pull requests | Eligible | Observed, 2026-11-13; [recorded contract](../features/security.feature.md#ai-scan-for-pull-requests) | Read/write: repository code-scanning AI Scan `pr_scan` | Implemented: [AI Scan](../settings.md#ai-scan-for-pull-requests) |
| CodeQL threat model | Eligible, including Preview | [Documented default-setup choice][codeql-ui] | Read/write: default-setup `threat_model` supports `remote` and `remote_and_local` | Implemented: [threat-model declaration](../settings.md#codeql-default-configuration), automatic export, conditional apply and independent companion-preserving readback |
| Repository-wide AI Scan | Eligible, including Preview | Observed control, 2026-11-13 | Read/write: AI Scan repository_scan; read-only target GET confirmed both statuses, 2026-11-14 | Implemented: [independent repository toggle](../settings.md#ai-scan-for-repositories), complete observation, no-op and companion-preserving verification; no scan execution |
| AI application scans | Eligible, including Preview | Observed control, 2026-11-13 | Unconfirmed settings read/update contract; scan execution is not configuration | Not implemented |
| Independent Dependency graph enablement | Eligible | Observed, 2026-11-13; [earlier investigation][security-inventory] | Unconfirmed independent personal-repository setter/observation; dependency data APIs do not establish it | Not implemented independently |
| Automatic dependency submission | Eligible | Observed, 2026-11-13 | Unconfirmed settings read/update contract; snapshot submission is different | Not implemented |
| Dependabot malware alerts | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings read/update contract | Not implemented |
| Dependabot grouped security updates | Eligible | Observed, 2026-11-13 | Unconfirmed complete settings read/update contract | Not implemented |
| Dependabot GitHub auto-triage presets | Eligible | Low-impact development-dependency and malware-dismissal presets observed, 2026-11-13 | Unconfirmed complete preset identity/read/update contract | Not implemented |
| Custom Dependabot auto-triage rules | Eligible | New rule entry observed, 2026-11-13; [documented rule concept][dependabot-triage]; full form not inventoried | Unconfirmed complete custom-rule read/update contract | Not implemented; former policy exclusion does not prove an API limitation |
| Copilot Autofix / Agentic Autofix settings | Eligible | Both controls observed, 2026-11-13 | Unconfirmed settings contract; alert-level generation/commit APIs execute remediation | Not implemented |
| Global code-scanning check-run failure thresholds | Eligible | Severity controls observed, 2026-11-13 | Unconfirmed settings contract; Ruleset thresholds are a different control | Not implemented |
| Private reporting: Require a CWE assignment | Eligible | Observed, 2026-11-13 | Unconfirmed detail read/update contract; enablement endpoint is insufficient | Not implemented |
| Private reporting: Daily advisory limit | Eligible | Observed, 2026-11-13 | Unconfirmed detail read/update contract | Not implemented |
| Private reporting: Trusted reporters | Eligible | Observed, 2026-11-13 | Unconfirmed membership read/update contract | Not implemented |
| Other secret-scanning patterns, notifications and bypass controls | Eligible if configurable at repository scope | Complete target control inventory unconfirmed | Unconfirmed complete settings contracts; alert bypass or user-level settings are not substitutes | Not implemented |

The [2026-11-08 investigation][security-inventory] and its
[API recheck][security-api-recheck] are dated evidence. Their old Free-only,
Preview and custom-rule exclusions are superseded by the adopted scope; their
unconfirmed API findings remain unconfirmed until a suitable contract is found.
Do not turn the age of an investigation into a claim that no API exists today.

## Pages and Environments

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| Pages publishing source, branch/folder, custom domain, HTTPS and site lifecycle | Eligible | [Recorded Pages UI](ui-pages-mapping.md#verified-ui-inventory), 2026-10-28 | Read/create/update/delete: [Pages REST][pages-api] | Implemented: [Pages](../settings.md#pages-configuration); DNS, certificate/deployment readiness and file-authoring are distinct |
| Pages visibility/access | Eligible only if configurable in the target public-repository UI | Observed promotional text without editable control, 2026-11-13; broader target applicability unconfirmed | Partial: Pages API visibility metadata does not establish an eligible configurable control | Not implemented; do not infer an editable public-repository setting from an API field |
| Environment policy, reviewers, timer, administrator bypass and branch/tag patterns | Eligible | Observed built-ins, 2026-11-13; [detailed earlier mapping](ui-environments-mapping.md#verified-ui-controls) | Read/create/update: [Environments REST][environments-api]; pattern CRUD via [branch policy REST][deployment-patterns-api] | Implemented: [Environments](../settings.md#environments); existing omitted built-ins are preserved from fresh state and matching partial intent skips; creation omissions remain API-controlled; disabling custom mode requires complete empty raw patterns, preventing implicit loss of an omitted collection |
| Environment variables | Eligible | [Recorded UI/API evidence](../features/environments.feature.md#selected-environment-variables), 2026-11-11 | Read/create/update/delete: name-addressed Environment [Variables REST][variables-api] | Implemented: [Environment variables](../settings.md#environment-variables), independent of policy intent |
| Environment deletion through pruning | Eligible | Environment lifecycle is documented; deletion consequences need explicit coverage | Read/delete: [Environments REST][environments-api] | Implemented: explicit `environments` and standalone `all`; complete fixed snapshots, known-empty custom protection gate, child metadata drift checks and fresh absence confirmation |
| Existing Environment protection App selection | Eligible | [Official checkbox UI](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/configure-custom-protection-rules); known-empty enabled/available object reads checked 2026-11-14; no live nonempty selection/write verified | Complete enabled/available reads, POST, independent GET and DELETE: [protection rules REST][protection-rules-api] | Implemented: [custom_rules App-ID sets](../settings.md#custom-protection-apps), omission preservation, independent readback and unchanged known-empty parent-prune gate; installation/grants excluded; combined capacity counting remains unconfirmed |
| Environment rename | Eligible if target UI is configurable | Target UI rename control unconfirmed | Unconfirmed identity-preserving rename contract | Not implemented; do not treat create/delete as a proven rename |

## Integrations and remaining settings pages

This section makes uninspected pages visible rather than treating the existing
implementation's families as an exhaustive UI inventory. An available endpoint
does not establish every control on its corresponding page. Existing settings
must not be erased merely because part of the resource cannot be managed.

| Control or group | Policy | UI evidence | API evidence | Source coverage |
| --- | --- | --- | --- | --- |
| Repository webhooks: non-secret configuration, events and activation | Eligible if target UI is configurable | Add webhook observed, 2026-11-13; full form unconfirmed; [documented controls][webhooks-api] | Read/create/update/delete: repository hooks REST; secret values remain excluded | Not implemented; secret preservation and delivery side effects need a complete contract |
| Deploy keys: title, public key and read/write grant | Eligible | Add deploy key observed, 2026-11-13; [documented UI controls](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/managing-deploy-keys) | Read/create/delete: [deploy-key REST][deploy-keys-api]; no update endpoint | Implemented: [canonical public-key identity, guarded creation and explicit pruning](../settings.md#deploy-keys); immutable replacements are separate applies; private keys excluded |
| Autolink references | Eligible | [Documented repository UI](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/configuring-autolinks-to-reference-external-resources); no fresh authenticated form observed | Read/create/delete: [Autolinks REST][autolinks-api]; no documented in-place update | Implemented: [immutable ID binding, verified creation and explicit prune](../settings.md#autolink-references); replacement requires separate deletion and creation; no live write or rendered-link claim |
| Existing GitHub App repository integration configuration | Eligible only for configurable repository controls | Installed-App inventory observed, 2026-11-13; no additional repository setting established | Partial: installation inventory is not proof of complete configurable repository state or App-credential mutation support | Not implemented; App installation/authorization excluded |
| Email notifications | Eligible if target UI is configurable | Target control inventory unconfirmed | Unconfirmed repository settings read/update contract | Not implemented |
| Codespaces prebuild configuration | Eligible if target UI is configurable | Prebuild configurations and Set up prebuild observed, 2026-11-13; setup form not inspected | Unconfirmed complete applicable settings contract; workspace execution and file editing are distinct | Not implemented |
| Additional sidebar, conditional and feature-dependent controls | Eligible if target UI is configurable | Unconfirmed until explicitly inventoried | Unconfirmed until matched to applicable official settings APIs | No completeness claim; record each newly observed control separately |

## Explicit product and target boundaries

These are policy or target decisions, not claims that an API is unavailable.
The feature adoption policy owns their meaning; this table prevents them from
being confused with the missing implementations above.

| Surface | Policy | UI/API distinction | Source coverage |
| --- | --- | --- | --- |
| Repository rename, visibility change, transfer and deletion | Excluded | UI/API feasibility does not override this lifecycle/identity decision | Not implemented; archive/unarchive remains separately supported |
| Secret values and private keys | Excluded | Secret names/metadata or write endpoints do not supply exportable secret values | No secret-value management; plaintext non-secret variables remain supported |
| Feature configuration-file authoring | Excluded | UI editors and Contents APIs can create files, but that is a separate product boundary | No Dependabot/CodeQL/Actions workflow, funding, issue-template or instruction-file authoring; gh-settings' own YAML is not excluded |
| Organization settings, membership, teams and inherited policies | Non-target | Repository-visible inherited effects do not become owned organization settings | Preserve ownership boundaries; do not adopt organization APIs as personal-repository substitutes |
| Private-repository Actions sharing and private-fork workflow controls | Non-target | [Current applicability record](../settings.md#excluded-private-actions-controls) describes private-only controls | Not accepted configuration |
| Push Rulesets and their file restrictions | Non-target where restricted to private/internal repositories | [Official rules guide][rules-ui] identifies that applicability; forks require separate target evidence | Not accepted configuration; acquired unsupported protection is not discarded |
| Organization-only custom properties/forking controls | Non-target | API availability does not create a configurable personal-public UI control | Not accepted configuration |
| Legacy tag protection | No active target control observed | Tags page on 2026-11-13 identifies protected tags as deprecated and links to tag Rulesets | Not implemented; tag Rulesets remain supported separately |
| Installing or authorizing GitHub Apps | Excluded | Installing an integration or broadening its grants is distinct from selecting an already available Environment protection App | No installation/grant management |
| Product-wide Code Security / Advanced Security switch | Non-target UI concept on available evidence | [Recorded UI-scope decision](../features/security.feature.md#product-switch-exclusion) found headings, not a target product switch | No product-wide setter; individual CodeQL/AI/secret controls are separate |
| Account-level settings, subscriptions, grants and feature execution | Non-target | A repository setting can depend on these without managing them | No subscription purchase, grant expansion, scan/review execution or protection bypass is implied |

## Cross-cutting product gaps

| Destination | Current source and evidence limit |
| --- | --- |
| Same supported operations through CLI and public Action | Current-source CLI and [public Action](../github-actions.md#export-a-local-configuration) support export/plan/apply. Action export produces a local file and success-only path output without publication. The selected immutable package still determines availability. |
| Automatic acquired-state export | [Automatic export](../settings.md#exported-values) and [retained Security intent](../settings.md#automatic-security-export-and-retained-intent) are implemented. About/General groups and other independent unavailable families retain compatible valid prior intent with info; checkout/origin and local file failures remain fatal. Failed info or normalization-warning delivery prevents saving. No successful export certifies remote repository existence when all observations are unavailable. This does not add missing setting families. |
| Complete meaningful false/zero/empty values | Existing family contracts retain observed values. Variable inventories containing values outside the current nonempty-value schema remain unavailable with info, not fabricated empties. Conditional and normalized observations retain their documented limits. |
| Unowned companions and repeated matching apply | Shared Repository REST controls skip fresh exact matches, write only changed declared controls and independently verify original intent. Topics also skip matching membership and independently verify changed sets. The four shared GraphQL access controls also skip raw native matches, send changed members only and independently verify every originally declared member against the original repository ID. Existing Environment partial policies skip fresh matches and preserve omitted built-ins/raw active patterns. Creation omissions stay API-controlled; disabling custom mode with omitted nonempty raw patterns is rejected, with explicit clear-then-switch semantics. Existing Ruleset selected objects have whole-policy semantics. These contracts require reconciliation with the destination before claiming uniform preservation/idempotence. |
| Consistent optional pruning | [Collection pruning](../settings.md#collection-pruning) covers labels, Rulesets, Actions/Agents/Environment variables and Actions Policies. Environment parent pruning is implemented with child-state/custom-protection gates; standalone `all` selects every supported scope, with omitted collections unmanaged. New collection adoption must also resolve its own deletion effects. |
| Complete observation and convergence | Readbacks and partial-failure evidence are described per family. API acknowledgment, normalization equality, pending invitation or asynchronous acceptance alone cannot establish the destination state. Unknown observations must remain unknown. |
| One released CLI/Action contract | [Package identity and consumer pins](../github-actions.md#published-package-and-consumer-pins) determine actual availability. Current source documentation does not update an older immutable package. |
| Exhaustive target UI/API evidence | The dated browser pass, earlier mappings and official contracts cover the named groups; unconfirmed rows, the sudo-blocked Access page and unexpanded forms remain evidence gaps. No exhaustive conditional-control survey or live write/readback certification is claimed. |

[v1-goal]: https://github.com/daiksud/gh-settings/issues/668
[openapi]: https://github.com/github/rest-api-description/blob/bd6df26d2a0104874fe5025a25ebcde356413364/descriptions/api.github.com/api.github.com.json
[repos-api]: https://docs.github.com/en/rest/repos/repos
[repositories-graphql]: https://docs.github.com/en/graphql/reference/repos
[labels-api]: https://docs.github.com/en/rest/issues/labels
[personal-access]: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/repository-access-and-collaboration/permission-levels-for-a-personal-account-repository
[personal-invitations]: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/repository-access-and-collaboration/inviting-collaborators-to-a-personal-repository
[collaborators-api]: https://docs.github.com/en/rest/collaborators/collaborators
[invitations-api]: https://docs.github.com/en/rest/collaborators/invitations
[interactions-api]: https://docs.github.com/en/rest/interactions/repos
[review-limits-ui]: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/managing-pull-request-reviews-in-your-repository
[protected-branches]: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
[branches-graphql]: https://docs.github.com/en/graphql/reference/branches
[branch-protection-api]: https://docs.github.com/en/rest/branches/branch-protection
[rules-api]: https://docs.github.com/en/rest/repos/rules
[rules-ui]: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets
[coverage-ui]: https://docs.github.com/en/code-security/how-tos/maintain-quality-code/restrict-code-coverage
[coverage-announcement]: https://github.blog/changelog/2026-09-18-manage-the-code-coverage-ruleset-condition-with-the-rest-api/
[actions-policies-api]: https://docs.github.com/en/rest/actions/policies
[actions-cache-api]: https://docs.github.com/en/rest/actions/cache
[oidc-api]: https://docs.github.com/en/rest/actions/oidc
[variables-api]: https://docs.github.com/en/rest/actions/variables
[agents-variables-api]: https://docs.github.com/en/rest/agents/variables
[copilot-api]: https://docs.github.com/en/rest/copilot/copilot-code-review
[copilot-max]: https://github.blog/changelog/2026-10-28-copilot-code-review-merge-with-confidence-with-max-and-approvals/
[copilot-api-announcement]: https://github.blog/changelog/2026-11-06-copilot-code-review-api-management-and-review-overview-updates/
[copilot-config]: https://docs.github.com/en/copilot/how-tos/copilot-on-github/set-up-copilot/configure-code-review
[suggestions-ui]: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/manage-rationale-confidence-approvals
[runners-api]: https://docs.github.com/en/rest/actions/self-hosted-runners
[code-scanning-api]: https://docs.github.com/en/rest/code-scanning/code-scanning
[codeql-ui]: https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/manage-your-configuration/edit-default-setup
[security-inventory]: https://github.com/daiksud/gh-settings/issues/323#issuecomment-6719839333
[security-api-recheck]: https://github.com/daiksud/gh-settings/issues/323#issuecomment-6728386507
[dependabot-triage]: https://docs.github.com/en/code-security/concepts/supply-chain-security/dependabot-auto-triage-rules
[pages-api]: https://docs.github.com/en/rest/pages/pages
[environments-api]: https://docs.github.com/en/rest/deployments/environments
[deployment-patterns-api]: https://docs.github.com/en/rest/deployments/branch-policies
[protection-rules-api]: https://docs.github.com/en/rest/deployments/protection-rules
[webhooks-api]: https://docs.github.com/en/rest/repos/webhooks
[deploy-keys-api]: https://docs.github.com/en/rest/deploy-keys/deploy-keys
[autolinks-api]: https://docs.github.com/en/rest/repos/autolinks
