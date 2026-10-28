# Repository settings

## Scope and status

Configuration lives in `.github/settings.yml` at the Git checkout root and manages only that checkout's GitHub.com repository. The supported resources and fields are described below. See the [command reference](commands.md) for behavior and the [Action package boundary](github-actions.md#published-package-and-consumer-pins) before adopting a configuration.

## Supported fields

The keys inside `settings.repository` follow GitHub's repository update request body parameters.[^github-update-repository] Discussions additionally uses the parameter sent by GitHub's official CLI. PR-access fields use the explicit GraphQL-to-YAML mapping below.

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The booleans accept `true` or `false`; the string fields accept strings, including empty strings. Explicit null and other value types are invalid. All other repository fields and read-only response metadata are excluded, even when GitHub exposes them as writable parameters.

## Repository archive lifecycle

The tool supports the optional `settings.repository.archived` boolean. The field manages the selected checkout repository's archive state, not repository renaming, visibility, or individual issues/releases.

```yaml
schemaVersion: 1
settings:
  repository:
    archived: false
    description: Active maintenance
```

Omitting `archived` leaves the lifecycle unmanaged and adds no archive-specific API read. `false` means explicitly unarchive and `true` means explicitly archive. Null, quoted strings, numbers and other YAML types are invalid. Ordinary `gh settings export` does not add the field: use `gh settings export --archive` to include its currently observed value after all requested reads succeed. A failed read leaves any existing configuration file untouched.

Plan reads the declared archive state, validates the repository's positive ID, canonical owner/name and boolean, then completely observes each other declared resource; it does not mutate the repository. With other settings, comparisons remain typed and read-only, with an explicit unarchive-before-other-settings or archive-after-other-settings note for a changed lifecycle field. Current source can report matching or changed declarations when an already archived target declares `archived: true`; a missing, denied, partial or invalid observation still fails instead of becoming a match. Apply in current source still rejects it before any write: explicitly declare `archived: false` before changing other settings. Otherwise, apply validates a complete current identity before any write. An explicit unarchive runs first, then existing resource writes; a final archive runs only after every other update succeeds, with a fresh state/identity check. Archive PATCH requests contain **only** `archived`, and each response and readback must confirm the target ID, name, and requested state. Matching archive-only values cause no lifecycle write.

On an already archived repository, applying an `archived: true` declaration combined with other managed settings fails **before all writes** even after a plan reports that those settings match or differ; use an explicit separate `archived: false` declaration to resume updates. A failed unarchive prevents subsequent writes. A confirmed unarchive remains applied if a later settings update fails; that fact appears in the error even when the CLI discards the detailed result. A failure during final archive may leave prior settings updates applied, and a failed archive PATCH, transport error, or readback can leave the remote lifecycle state uncertain. There is no rollback, automatic retry, transactional identity lock or implicit temporary unarchive. Running a repository's own Action against its archived source may not be a viable way to recover it; keep an external admin recovery route.

The [repository REST API](https://docs.github.com/en/rest/repos/repos#update-a-repository) requires appropriate repository Administration write permission for lifecycle changes, and read access for observation. These grants must already exist. Tests use local simulated HTTP responses; **no live archive/unarchive is performed or verified**. See the [lifecycle behavior specification](features/repository-lifecycle.feature.md) and [ADR 0027](adr/0027-repository-lifecycle.md).

## Secret scanning and push protection

Current source supports two optional booleans under `settings.security`. The Action still installs its selected published CLI.

```yaml
schemaVersion: 1
settings:
  security:
    secret_scanning: true
    secret_scanning_push_protection: true
```

`secret_scanning` controls the repository feature; `secret_scanning_push_protection` controls repository push protection. User-level push protection, partner notifications, patterns, bypasses, alerts, scanning workflows, paid-product enrollment and organization configurations are outside this resource. Dependabot, Code Security and Advanced Security keys remain unsupported. [Private vulnerability reporting](#private-vulnerability-reporting) is an independent public-repository control.

Each key is optional: omission is unmanaged, false explicitly disables, and `security: {}` manages nothing. Null, non-booleans, duplicate/unknown keys and an explicit scanning=false/push-protection=true pair fail local validation. No parent setting is implicitly enabled. Declaring push protection true with scanning omitted requires freshly observed scanning=true. Disabling scanning while push protection is active requires declaring push protection false too; unknown companion state fails before this family's writes. Enable scanning before protection; disable protection before scanning.

Explicit `export --security secret-scanning` reads both statuses and adds only these keys; combine it with the existing export selections in any order. Ordinary export retains its existing requests and omits security. Missing/null/unknown requested data, permission failure or an observation that cannot form a consistent declaration fails before replacing the destination. Complete false values are retained. An unchanged supported export produces zero differences for the exported values.

Plan reads both statuses only when security is managed, checks prerequisites and displays native boolean differences without writing. Apply runs after existing resources, including workflow permissions. It completes family preflight, then sends one narrow `security_and_analysis` PATCH per needed member in dependency order. Each update rereads fresh prerequisites, skips an already matching member, and verifies both the desired value and preserved companion afterward. It never replays unrelated security fields. A failed first read means no security write was attempted; later write/readback failures retain uncertain effects and any earlier confirmed operations. There is no transaction, drift lock, automatic retry or rollback. A successful rerun reconciles fresh state.

The [repository endpoint](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents Metadata read for GET, with admin/organization-owner/security-manager visibility for the security block, and Administration write plus applicable role for security updates. A normal read token or repository role alone does not establish the required token grants or visibility. Public availability differs from private/internal Secret Protection entitlement; existing applicable entitlement is required. This feature neither buys a product nor expands credentials. Upper configurations can return apparent PATCH success without changing settings, so readback mismatch fails visibly. API schema presence and a zero plan do not prove licensing or writable access. See the [design and acceptance specification](features/security.feature.md).

## Immutable release policy

Current source supports optional boolean `settings.security.immutable_releases`.

```yaml
schemaVersion: 1
settings:
  security:
    immutable_releases: true
```

Omission is unmanaged; false explicitly disables the repository policy. Null,
strings, collections and an `enabled` submapping are invalid. Ordinary export
does not read this endpoint. Explicit `export --security immutable-releases`
exports only the enabled boolean after all selected reads complete; it composes
with the other distinct security selections. `enforced_by_owner` is read-only
metadata, never YAML intent or a toggle payload.

The [policy GET](https://docs.github.com/en/rest/repos/repos#check-if-immutable-releases-are-enabled-for-a-repository)
requires Administration read. A 200 response must contain complete boolean
`enabled` and `enforced_by_owner` values. **HTTP 404 means disabled**, without
additional permission-discrimination requests. Owner enforcement is unknown on
that response; no false enforcement value is invented. Other failures, including
401, 403, redirects, malformed or incomplete 200 responses, remain errors.

This 404 rule is an explicit product tradeoff: GitHub's general authentication
behavior can return 403 or [404 for inaccessible private resources](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api#404-not-found-for-an-existing-resource).
An authorization-related 404 may therefore be misreported as disabled in export,
plan, preflight or readback. No additional authorization discriminator is used.
This is not evidence that this endpoint has actually returned an
insufficient-permission 404 in a live test, nor an exception for other endpoints.

Plan is read-only and shows native boolean differences. An owner-enforced policy
cannot be planned or applied as false. Apply runs after private vulnerability
reporting and before Pages, reads fresh policy, skips matching state, and sends
one body-free PUT (enable) or DELETE (disable) to the policy endpoint with
Administration write. Only 204 is accepted, followed by fresh readback. A 404
readback after DELETE is accepted as disabled under the same tradeoff; it is not
independent proof of authorization. Owner enforcement is never changed.

Preflight failures attempt no policy write. A write, unexpected write status or
readback failure may already have changed the policy; earlier successful writes
remain applied. No retry, rollback, saved-plan execution or transaction exists.
Individual releases, tags and assets are never read or mutated by this feature;
changing this policy does not promise to make existing immutable releases
editable. Local [fixtures and process tests](features/immutable-releases.feature.md)
establish client behavior, not live App permissions or enforcement. Published
packages support only their own versioned contracts.

## Private vulnerability reporting

Current source supports optional boolean `settings.security.private_vulnerability_reporting`.

```yaml
schemaVersion: 1
settings:
  security:
    private_vulnerability_reporting: false
```

Omission is unmanaged; false explicitly disables reporting, true enables it, and an empty security mapping manages nothing. Null, wrong types, duplicate/unknown keys and an `enabled` submapping fail local validation. The control applies only to public repositories: private/internal targets are inapplicable even for desired false. Unknown visibility or enabled state fails; it never becomes disabled. Reports, advisories, notifications, forms, organization configurations and paid-product enrollment are outside this feature.

Explicit `export --security private-vulnerability-reporting` includes the completely observed boolean. It does not require Secret scanning visibility or entitlement. Repeat `--security` with the distinct `secret-scanning` family to include both, in either argument order; duplicate/unimplemented families fail before reads or destination changes. Ordinary export retains its existing requests and omits security. Selected reads and serialization complete before saving; an unavailable/inapplicable request preserves the destination. An unchanged supported export produces zero differences for its declared values.

Plan checks public visibility and the dedicated enabled boolean with GETs only. Apply runs after other resources, including Secret scanning, and rereads applicability and state before its single PUT (true) or DELETE (false). Matching state makes no mutation. Complete fresh readback, including public applicability, must retain the requested boolean; apparent success with unchanged or missing state fails verification. Pre-write read/applicability failures are confirmed not to have attempted this toggle. Write or verification failures retain uncertain effects and any earlier completed resource writes. There is no transaction, automatic retry or rollback.

The [dedicated repository endpoint](https://docs.github.com/en/rest/repos/repos#check-if-private-vulnerability-reporting-is-enabled-for-a-repository) documents Metadata read for observation and Administration write plus applicable admin access for updates. A declaration containing only this control uses public metadata and reporting-state reads, without requiring Secret scanning visibility, Actions permissions or a paid entitlement. These grants must already exist; the CLI does not provision them. The [public-repository feature guide](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository) defines applicability. Local fixtures and a read-only enabled observation do not establish real update permissions, upper-policy behavior or actual reports. See the [security specification](features/security.feature.md#implemented-private-vulnerability-reporting-test-mappings).

## Default workflow permissions

Current source adds two independent optional fields under `settings.actions`. Newer Action source still installs its selected published CLI.

```yaml
schemaVersion: 1
settings:
  actions:
    default_workflow_permissions: read
    can_approve_pull_request_reviews: false
```

`default_workflow_permissions` accepts exactly `read` or `write`, selecting the default permissions of workflow `GITHUB_TOKEN`s. `can_approve_pull_request_reviews` is a boolean controlling whether Actions can submit approving PR reviews. They do not change individual workflow/job `permissions`, caller credentials, App grants, Rulesets bypass or the execution policy described below. A write default or approval permission can broaden what workflows can do; plan displays both transitions explicitly. Organization policy can prevent the requested setting, and repository configuration does not override that policy.

Each field is optional; omission is unmanaged and explicit false remains declared. Either field can be declared alone, including while repository Actions execution is disabled. Null, wrong types, case variants and unknown enum values fail local validation before origin/authentication/API access. Workflow-only declarations do not read or write the separate execution-policy endpoints and never infer `enabled`, `allowed_actions` or selected companions.

Ordinary export and `export --actions` keep their existing scope. Explicit `export --workflow-permissions` reads both values completely and adds them to the actions mapping; combine it with `--actions` and/or `--rulesets` in any order to include those independent resources. Missing/null/invalid values and permission failures stop before saving; they never become read/false defaults. An unchanged export produces zero differences for the exported values.

Plan reads the complete workflow policy only when at least one of its fields is declared. Apply runs after existing resources, including execution policy, reads fresh complete workflow state, and sends only declared optional PUT keys. An initial read failure is recorded as a confirmed read failure with no PUT attempt; PUT/readback failures retain an uncertain remote effect. It reads back both values to verify requested values and preserved companions; failure or unexpected state is reported with a warning that writes may already have applied. A concurrent change can therefore cause verification failure; this is not locking or rollback. Direct repeated apply still writes the declared values. An error stops later work and retains the known progress of completed earlier resources.

The [dedicated workflow-permission endpoint](https://docs.github.com/en/rest/actions/permissions#set-default-workflow-permissions-for-a-repository) requires target Administration read for observation and Administration write for update. These grants must already exist; the feature neither creates nor broadens them. Organization conflicts, unsupported access and redirects fail visibly. Local fixtures verify the transport and failure contract; read-only live observations do not establish actual mutation compatibility, organization policy or token permissions. No workflow, token, grant or live setting is changed by building this feature. See the [behavior specification](features/workflow-permissions.feature.md).

## Fork PR workflow policies

Current source supports `settings.actions.fork_pr_approval_policy` and `settings.actions.private_fork_workflows`. Caller grants are unchanged.

For **public repositories**, `fork_pr_approval_policy` accepts exactly `first_time_contributors_new_to_github`, `first_time_contributors`, or `all_external_contributors`. These select the external-contributor workflow approval requirement. GitHub considers both the PR author and event actor. Workflows triggered by `pull_request_target` run in the base context outside this contributor approval gate; this setting does not make those workflows safe to execute untrusted code.

```yaml
schemaVersion: 1
settings:
  actions:
    fork_pr_approval_policy: all_external_contributors
```

For **private repositories**, `private_fork_workflows` has four optional booleans:

| Key | Meaning |
| --- | --- |
| `run_workflows_from_fork_pull_requests` | Allow fork PR workflows to run. |
| `send_write_tokens_to_workflows` | Allow write `GITHUB_TOKEN` permissions in fork PR workflows. |
| `send_secrets_and_variables` | Make secrets and variables available to fork PR workflows. |
| `require_approval_for_fork_pr_workflows` | Require approval before fork PR workflows run. |

```yaml
schemaVersion: 1
settings:
  actions:
    private_fork_workflows:
      run_workflows_from_fork_pull_requests: true
      send_write_tokens_to_workflows: false
      send_secrets_and_variables: false
      require_approval_for_fork_pr_workflows: true
```

Sending write tokens or secrets can expose repository credentials to code from forks. Review those native boolean differences explicitly. The default read-only/no-secrets fork behavior, this fork-specific token control, default workflow permissions, approving PR reviews, and approvals to run a workflow are distinct controls. This feature never implicitly changes any of them, enables repository Actions, or edits workflows. An organization may prevent repository enabling; repository configuration cannot override it. Internal repositories are outside the adopted applicability contract.

Every field is optional: omission is unmanaged, an empty mapping manages nothing, and explicit false remains desired state. Null, wrong types, unknown keys/enums and duplicate keys fail locally. For private policies, a matching partial declaration requires no PUT. If any declared value differs, the API requires `run_workflows_from_fork_pull_requests`; declare it explicitly, even if unchanged. The tool will not resend an undeclared observed value to satisfy that requirement. A disabled-to-enabled transition additionally requires all four values explicitly, so latent token/secrets/approval choices cannot be activated implicitly. Disabling execution does not automatically reset stored companions.

Ordinary export, `--actions`, `--workflow-permissions` and `--retention` retain their scopes. Explicit `export --fork-approval` reads only the public contributor family; `export --private-fork-workflows` reads all four private booleans. Other independent export choices compose in any order. Each selected family first verifies exact repository visibility and a complete typed response before saving. Both fork families cannot be applicable to one fixed repository: selecting both fails and preserves the destination. Missing/null/denied/not-found/redirected responses are errors, never false/default state. A successful unchanged export produces zero differences.

Plan is read-only and requests only declared resources. Apply runs after retention and before security, with contributor approval before private fork policy if both are declared. Fresh applicability and complete reads precede each changed resource. PUT contains only declared keys (the contributor enum maps to API `approval_policy`); a matching fresh state skips PUT. Private readback verifies all desired and unowned companions without replaying them in the payload. Upper-policy refusal, ignored writes, unexpected companion changes, visibility drift or incomplete readback fail with uncertain effects after a PUT. Concurrent drift may therefore fail verification; this is not locking or rollback. Pre-write read/preflight errors confirm no PUT for that operation; earlier completed resource writes remain applied. A later inapplicable family can fail after an earlier valid update during direct apply. There is no transaction, retry, rollback, workflow dispatch, organization update or credential grant expansion.

The [contributor API](https://docs.github.com/en/rest/actions/permissions#set-fork-pr-contributor-approval-permissions-for-a-repository) and [private fork API](https://docs.github.com/en/rest/actions/permissions#set-private-repo-fork-pr-workflow-settings-for-a-repository) document existing Administration read/write credentials (classic tokens need `repo`). The [repository guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository) explains visibility and credential consequences. Local fixtures verify modeled optional-key preservation and failures, not actual live mutation compatibility or policy enforcement. No live policy update, token/secrets access, grant change or release was performed. See the [behavior specification](features/fork-workflows.feature.md).

## Check, run, status, artifact and log retention

Current source supports optional integer `settings.actions.artifact_and_log_retention_days`.

```yaml
schemaVersion: 1
settings:
  actions:
    artifact_and_log_retention_days: 30
```

Omission is unmanaged; an empty actions mapping manages nothing. The supported integer range is 1–400, further limited by the freshly observed repository `maximum_allowed_days`. Public repositories allow at most 90 days; private repositories allow at most 400, and a managing organization/enterprise may set a lower maximum. Zero, negatives, null, floats, strings and other types are invalid locally. The API's `maximum_allowed_days` is read-only metadata, not a YAML field or desired value. No value is defaulted or clamped.

This repository policy covers checks (suites, runs and annotations), workflow runs, commit statuses including third-party integrations, artifacts and logs. GitHub expanded the policy to checks/runs/statuses on October 1, 2026. Changes apply to newly created objects and do not retroactively change existing objects. Shorter retention can remove future diagnostic/history data sooner. This feature does not call individual artifact/run/status deletion APIs or edit workflow YAML `retention-days`. See [GitHub's repository retention guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#configuring-the-retention-period-for-checks-workflow-runs-commit-statuses-artifacts-and-logs-in-your-repository).

Explicit `export --retention` includes only the configured days in the actions mapping. It composes with the existing export choices in any argument order; ordinary export, `--actions` and `--workflow-permissions` retain their existing scope. Both integers must be acquired completely before saving, and an observed current setting above the current maximum cannot be exported as a valid declaration. Missing/null/invalid fields and access failures preserve the destination. An unchanged supported export produces zero differences for its values.

Plan reads the dedicated policy only when this field is declared and shows native integers without mutation. Apply runs after workflow permissions and before security families, rereads the complete policy, checks the current maximum, and PUTs only `days` when different. Matching state skips the write. A known current policy above its maximum can be corrected by declaring an allowed lower value; it is never silently normalized. Readback must retain desired days within the then-current maximum. The unowned maximum may change independently without causing failure if the desired value remains allowed. Initial read/preflight failure confirms no retention PUT; PUT/readback failures retain uncertain effects and earlier confirmed operations. There is no transaction, lock, automatic retry or rollback.

Retention is independent of Actions execution, selected Actions and workflow-token permissions, including while execution is disabled. No enabling or unrelated policy write is inferred. The [repository endpoint](https://docs.github.com/en/rest/actions/permissions#get-artifact-and-log-retention-settings-for-a-repository) documents Administration read/write for existing credentials (classic tokens need `repo`); the feature does not expand grants or read organization/enterprise policy endpoints. Local fixtures and a harmless GET establish modeled behavior and response shape, not live PUT compatibility, actual grants or cleanup timing. See the [behavior specification](features/retention.feature.md).

## Actions execution policy

Current source adds optional `settings.actions`. The six execution-policy fields below manage repository execution policy, not workflow files, individual runs, token grants or organization policy. [Default workflow permissions](#default-workflow-permissions) use separate fields and transport.

```yaml
schemaVersion: 1
settings:
  actions:
    enabled: false
    allowed_actions: selected
    sha_pinning_required: true
    selected_actions:
      github_owned_allowed: true
      verified_allowed: false
      patterns_allowed:
        - 'my-org/*'
        - '!my-org/blocked@*'
```

| Field | Type and meaning |
| --- | --- |
| `enabled` | Boolean; whether repository Actions execution is enabled. |
| `allowed_actions` | Exact string `all`, `local_only` or `selected`. |
| `sha_pinning_required` | Boolean requiring actions to use full commit SHAs; reusable workflows can still use tags. |
| `selected_actions.github_owned_allowed` | Boolean allowing GitHub-owned actions under selected policy. |
| `selected_actions.verified_allowed` | Boolean allowing Marketplace actions from verified creators under selected policy. |
| `selected_actions.patterns_allowed` | Ordered array of at most 1000 nonempty strings, passed to GitHub unchanged. |

Every field is optional and omission is unmanaged. Empty mappings manage nothing. Explicit false and `patterns_allowed: []` are retained; null, unknown keys, wrong types and unknown policy enums fail validation before remote work. A declared patterns list replaces the entire sequence; omitted entries are removed. Removing a negative block pattern can permit additional actions. The tool preserves case, order, duplicates, wildcards and negative `!` entries; it does not sort, expand, deduplicate or emulate GitHub matching. GitHub's [selected-policy API](https://docs.github.com/en/rest/actions/permissions#set-allowed-actions-and-reusable-workflows-for-a-repository) documents `patterns_allowed` as applying only to public repositories; configuring it does not establish enforcement on private repositories. See [GitHub's matching and policy guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#allowing-select-actions-and-reusable-workflows-to-run).

Selected fields can accompany `allowed_actions: selected` or an omitted policy. They cannot accompany `all` or `local_only`. With the policy omitted, current policy must already be selected. Switching from a nonselected policy to selected requires all three selected fields explicitly, because GitHub does not expose inactive selected values. Plan shows each such before value as unavailable, not false or empty. An incomplete transition fails before any Actions write. Existing selected policies permit partial declarations; omitted companions retain values acquired in a complete fresh read.

Normal export does not read this resource or require Administration read. Explicit `export --actions` includes the complete base policy and, only when selected, the complete selected policy. Inactive selected settings remain omitted, even if GitHub remembers them. An unchanged export has zero differences for exported values. Missing/null/invalid policy fields, hidden selected data and API errors fail before file replacement; disabled Actions do not justify inventing absent values. Options compose as `export --rulesets --actions` in either order.

Plan is read-only and reads this resource only when managed fields exist. Apply reads a fresh complete base policy and any needed active selected policy, then preflights the whole resource. The policy PUT requires `enabled`; if omitted, its freshly observed value is echoed without implicitly enabling Actions. Other base payload values are declared only. Selected PUT sends declared values merged with freshly read companions; no inactive companion is initialized implicitly. Policy changes precede selected writes because that endpoint requires selected policy. When enabling a currently disabled repository together with selected values, enabling runs last. Apply verifies requested values and the selected companions it sent with a fresh read. Repeated direct apply still writes declared values.

These endpoints are not transactional. An enabled repository can temporarily use remembered/default selected settings between the policy and selected-list writes. Failure can leave an earlier policy change applied, and a failed write or readback has an uncertain remote effect. There is no automatic rollback, saved-plan apply, concurrency lock or guarantee that organization/enterprise restrictions permit the desired change. Repository/organization restrictions and GitHub-controlled disabled states remain authoritative. Policy changes can prevent later workflows from running.

Read/export/plan require target Administration read; apply requires Administration write, including its preflight/readback. The default `GITHUB_TOKEN` generally does not supply this permission. Callers must provide an appropriately scoped existing credential; this feature neither creates nor broadens grants. Official [repository policy endpoints](https://docs.github.com/en/rest/actions/permissions#get-github-actions-permissions-for-a-repository) and the [behavior specification](features/actions-execution-policy.feature.md) describe the API and evidence boundaries. Local fixture success does not prove live write compatibility or actual token permissions.

## Sponsor button visibility

The tool supports optional `settings.repository.has_sponsorships`, mapped to GraphQL `hasSponsorshipsEnabled`.

```yaml
schemaVersion: 1
settings:
  repository:
    has_sponsorships: false
```

`true` enables the repository's Sponsorships setting and `false` disables it. Omission is unmanaged; null, strings, numbers, sequences and mappings are rejected before remote work. The [GraphQL repository and update-input contract](https://docs.github.com/en/graphql/reference/repos#updaterepositoryinput) defines this boolean separately from funding links. GitHub's [Sponsor button setup](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/displaying-a-sponsor-button-in-your-repository) requires admin access and describes funding configuration, including inherited defaults. This tool does not read, create, replace or remove `FUNDING.yml`, select recipients, change inherited funding links, subscribe to sponsorships, or make payments. An accepted boolean alone is not evidence that a button with usable links is rendered when funding configuration is missing.

Plan reads only this complete authenticated GraphQL boolean when declared; unrelated field-only plans retain their existing requests. Ordinary export includes twenty-three repository fields and requires this observation as well; archive and private forking are separate explicit selections. Missing/null/wrong-type values, unavailable repositories and partial GraphQL errors fail rather than becoming false; a failed export preserves the existing file. Read queries and the update's identity lookup disable rename following.

Apply uses the existing validated repository-ID lookup and shared `updateRepository` mutation, sending only declared GraphQL fields and verifying returned identity and values. A sponsorship-only declaration sends no REST PATCH. Combined PR, Issue-policy and Sponsorships declarations share one lookup and mutation; explicitly declared REST fields run first. Repeated direct scalar apply still writes. Failures stop later resources and can leave earlier writes applied; no saved-plan apply, rollback or automatic retry is introduced.

Use a credential with complete read visibility and authorization to update this repository setting; no credential grants are broadened automatically. Source and isolated process tests do not prove actual token permissions, missing-funding UI behavior or live mutation success. The [behavior specification](features/sponsorships.feature.md) owns verification cases and limits.

## Issue creation policy

Current source adds optional `settings.repository.issue_creation_policy`, mapped to GraphQL `issueCreationPolicy`.

```yaml
schemaVersion: 1
settings:
  repository:
    issue_creation_policy: COLLABORATORS_ONLY
```

When Issues are enabled, `ALL` permits creation by anyone and `COLLABORATORS_ONLY` limits it to collaborators, following [GitHub's enum contract](https://docs.github.com/en/graphql/reference/issues#issuecreationpolicy). Omission leaves the setting unmanaged. Null, empty strings, booleans/numbers, case variants and unknown policies are invalid. This controls creation eligibility independently of `has_issues`; it does not enable Issues, edit individual Issues, invite collaborators or change grants.

Plan uses a separate complete authenticated GraphQL query only when this policy is declared. PR-only declarations retain their separate reads. Export includes the observed policy after all reads succeed, including when Issues are disabled. Missing/null/wrong-type/unsupported values and partial GraphQL errors fail instead of becoming `ALL`; failed export preserves the old file. Policy reads and ID lookup disable rename following, so stale origins fail.

Direct apply reuses the validated opaque repository-ID lookup and `updateRepository` transport. Send only declared Issue/PR values; never synthesize `hasIssuesEnabled`, `has_issues` or PR companions. If PR and Issue controls are both declared, one lookup and one mutation carry them together. Explicitly declared REST fields such as `has_issues` run first, then GraphQL, topics, labels and Rulesets. Repeated scalar apply still sends declared values. Mutation response identity and requested values must match; failures stop later work and may leave earlier writes applied, with no rollback or retry.

Read credentials need complete policy visibility and write credentials must be authorized to update the target repository; caller grants are unchanged. Isolated fixtures and read-only observations do not prove live mutation compatibility, token permissions or published-package support. See the [feature specification](features/issue-creation-policy.feature.md).

## Pull request access

Current source adds two independent `settings.repository` values. Checking out newer Action source still installs its selected published CLI.

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

Omission leaves each value unmanaged. Explicit false disables PR access; GitHub hides the PR tab and existing PRs in the web interface and prevents new PR creation. GitHub's announcement notes a Mobile exception: the tab may remain visible while new PR creation is blocked. `COLLABORATORS_ONLY` restricts creation to collaborators with write access, while existing PRs remain visible and commentable when PR access is enabled. These effects follow [GitHub's access controls](https://github.blog/changelog/2026-02-13-new-repository-settings-for-configuring-pull-request-access/). No individual PR is edited. Null, quoted booleans, empty strings, case variants and unknown policies are invalid. Declaring one setting never adds the other to a request, including when disabling PRs.

Plan reads a narrow authenticated GraphQL query only when either value is declared. Policy reads and ID lookup explicitly disable rename following; a stale checkout origin fails instead of adopting the renamed repository. Export includes both values after complete reads. Missing/null/wrong-type values, unsupported policies and any GraphQL errors reject the entire observation rather than inventing false or `ALL`. A disabled PR feature can still have an observed policy; it is preserved as an independent value. Hidden or unavailable data remains an error. Read failures preserve the prior export file.

Direct apply performs existing repository REST updates first, then resolves the checkout repository's opaque GraphQL ID and sends only declared PR values through `updateRepository`, before topics/labels/Rulesets. The ID is transport identity, never YAML configuration. A PR-only declaration performs no REST PATCH and does not read current policy for comparison. Repeated scalar apply still sends declared values even when already matching. The mutation response must retain the requested values and identify the same repository; failure can occur after writes have applied. Earlier successful REST writes remain applied if a later lookup or mutation fails; later resources stop without retry or rollback.

Reads require complete policy visibility; writes require a credential authorized to update the target repository. Authentication and caller grants are unchanged, and no permission is expanded automatically. Local fixtures and read-only observations do not prove live write compatibility, token permissions or published-package support. See the [feature specification](features/pull-request-access.feature.md) for test mappings and evidence limits.

## Default branch

The tool supports optional `settings.repository.default_branch`, a string selecting an existing branch as the repository default. It uses the existing `schemaVersion: 1` repository group. Ordinary export includes twenty-three repository fields, topics and labels; archive and private forking require their explicit options.

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

## Template repository

The tool supports optional `settings.repository.is_template`, a boolean controlling whether the repository can be used as a template. It uses the existing `schemaVersion: 1` repository group; no repository-generation command is added.

```yaml
schemaVersion: 1
settings:
  repository:
    is_template: true
```

Omission leaves the setting unmanaged; explicit `false` removes template status. Null, quoted booleans and other types are invalid. Plan compares a complete non-null boolean from the existing repository REST GET and remains read-only. Export includes the observed value after all reads succeed. Missing, null or malformed observations fail instead of becoming false.

Apply sends only declared fields through the existing Administration-write repository PATCH, with no preliminary read or GraphQL request. GitHub's [template prerequisites](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-template-repository), including the Git LFS restriction, still apply; API rejections remain errors rather than triggering workarounds. The tool does not create repositories, copy files, change template provenance, or manage template contents. See the [feature contract](features/template-repository.feature.md).

## Web commit sign-off

The tool supports optional `settings.repository.web_commit_signoff_required`, a boolean requiring sign-off on commits made through GitHub's web interface. It extends the repository group under `schemaVersion: 1`. Ordinary export includes twenty-three repository fields, topics, and labels; published v0.15.x export retains sixteen repository fields.

```yaml
schemaVersion: 1
settings:
  repository:
    web_commit_signoff_required: true
```

Omission leaves the field unmanaged. Explicit `false` disables the requirement; null, quoted booleans and other types are invalid. Plan reads a complete non-null boolean from the existing repository REST GET, compares declared values only, and makes no GraphQL request for this field. Export includes the observed boolean after all reads succeed. Missing or wrongly typed values fail the read instead of becoming false.

Apply sends the field only when declared, in the existing repository PATCH with Administration write permission and no preliminary read. No new token grant is introduced. This is a sign-off requirement, not cryptographic commit-signature enforcement or a DCO check. It does not rewrite commits or change branch protections. The [feature contract](features/web-commit-signoff.feature.md) maps the request, failure and presentation tests. GitHub's [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents the exact field name.

## Discussions

The tool supports optional `settings.repository.has_discussions`, a boolean controlling GitHub Discussions. It extends the repository group; it does not manage discussion categories or individual discussions.

```yaml
schemaVersion: 1
settings:
  repository:
    has_discussions: false
```

Omission leaves the value unmanaged; explicit `false` remains a declared value. Null, quoted booleans, and other types are invalid. Export includes the observed value only after complete reads.

Plan reads the value from the existing repository REST GET without an additional GraphQL query. Apply sends it only when declared, through the existing Administration-write PATCH. When declared, a successful PATCH must return the same boolean; a missing, null, wrongly typed, or unequal value is an error. This check occurs after the request and does not roll back repository writes or prove transactional application. No prerequisite read, implicit enabling, or additional permission is introduced.

GitHub's [official CLI sends this field through repository PATCH](https://github.com/cli/cli/blob/6fcf2ea52718ad1ef8b12130f07430269e871352/pkg/cmd/repo/edit/edit.go#L76-L94), although the public REST update schema does not currently list it. The [feature contract](features/discussions.feature.md) describes strict reads, response verification, and test boundaries.

## Merge methods

The tool supports optional `allow_merge_commit`, `allow_squash_merge` and `allow_rebase_merge` booleans under `settings.repository`. Omission remains unmanaged, explicit false is retained, and null/wrong types fail validation. These fields extend the repository group; they are not a new resource family.

The [merge-method contract](features/merge-methods.feature.md) defines their narrow GraphQL read path, strict complete-read handling, unchanged REST PATCH semantics and read-only Actions evidence. Plan reads them only when declared; Export includes all three after a complete read.

## Automatic branch deletion

The tool supports optional `settings.repository.delete_branch_on_merge`, a boolean controlling GitHub's automatic deletion of head branches after pull requests merge. Omission leaves it unmanaged, explicit false is preserved, and null/non-boolean input is invalid. It extends the existing repository group; it does not manage or delete individual branches.

Export includes the value after a complete read. Plan uses the same fixed GraphQL read as merge methods; apply sends only declared fields through the existing Administration-write REST PATCH, with no preliminary read. Missing/null/invalid data or any GraphQL error fails the read instead of becoming false. See the [feature contract and read-only Actions evidence](features/delete-branch-on-merge.feature.md).

## Squash commit defaults

The tool supports two optional string enums to `settings.repository`, using `schemaVersion: 1`:

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

## Auto-merge and branch updates

The tool supports optional `allow_auto_merge` and `allow_update_branch` booleans to `settings.repository`, using the existing `schemaVersion: 1` format:

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

The [feature contract](features/pull-request-options.feature.md) records exact API mapping, hosted proof and test coverage.

## Merge commit defaults

The tool supports optional `merge_commit_title` and `merge_commit_message` string enums to `settings.repository` under `schemaVersion: 1`:

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

These select default title/body formatting for normal merge commits. Title uses the PR title or GitHub's classic merge message; body uses the PR body, PR title, or a blank body. `BLANK` is an enum, not an empty YAML string. These keys do not enable `allow_merge_commit`, change squash defaults, or merge PRs.

Omission remains unmanaged; title-only is valid when normal merge commits are enabled. Message requires an explicit `merge_commit_title` in the same mapping, matching GitHub's REST contract; a squash title cannot satisfy that dependency. GitHub rejects normal-merge title/message fields while normal merges are disabled. Local validation therefore rejects an explicit `allow_merge_commit: false` combined with either field. When `allow_merge_commit` is omitted, `validate` leaves it unmanaged and does not guess; `plan` uses the observed effective value and rejects the declaration if normal merges are disabled. Declaring `allow_merge_commit: true` with the defaults is valid, including when the plan enables the method. Null, empty strings, wrong types, case variants and unknown enums fail local validation before API access.

Export reads both values in the fixed complete GraphQL observation, but omits retained inactive normal-merge defaults when `allow_merge_commit` is false so the exported declaration remains valid. Plan compares declared fields only after checking the effective merge-method prerequisite; missing/null/invalid data or partial GraphQL errors fail the whole read. Apply sends declared values only through the existing Administration-write repository PATCH, with no preliminary read or implicit companion values. If the allow flag is omitted, direct apply cannot resolve the remote prerequisite first; declare `allow_merge_commit: true` with the defaults when enabling normal merges. See the [feature contract](features/merge-commit-defaults.feature.md) for exact API mapping, hosted evidence and tests.

## Topics

`settings.topics.names` manages the complete repository topic set. The topic declaration uses `schemaVersion: 1`.

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

Labels use ID-based, listed-object ownership.

```yaml
schemaVersion: 1
settings:
  labels:
    - id: 123
      name: bug
      color: d73a4a
      description: Something is not working
    - name: new-label
      color: abcdef
```

Only listed label objects are managed. Omitted `labels` and `labels: []` select nothing, require no label API calls and delete nothing. Unlisted remote labels and their issue/PR assignments remain untouched. Removing an entry stops managing it; there is no label-deletion declaration.

| Attribute | Contract |
| --- | --- |
| `id` | Optional positive REST integer ID. Present means update that existing label in the selected repository; an unknown ID is an error, never a create fallback. |
| `name` | Required nonempty, non-whitespace string without control characters. Spaces, punctuation and emoji are allowed. |
| `color` | Required six-character hexadecimal string without `#`, normalized to lowercase. There is no guessed color default. Quote numeric-only colors. |
| `description` | Optional string of at most 100 Unicode characters; omission defaults to `""` on both create and update, clearing an existing description. |

An ID-less entry is create-only. If its name already exists, plan/apply fail and show the existing ID to add for an intentional update. Names retain Unicode simple case folding for duplicate/collision checks, without trimming. Duplicate IDs or ambiguous names fail. With a valid ID, a different noncolliding name renames that same label and preserves its assignments; case-only differences retain the remote spelling. Renames must not collide with an existing label, even if another entry also plans to rename that label. Apply is not a rename transaction.

Plan compares complete supported object values, including default-derived description clearing, and displays IDs on updates. Export includes each observed ID and supported settings. Missing/invalid IDs, incomplete fields, duplicate identities and failed pages fail instead of becoming empty or successful state. Archive fields and read-only metadata remain unmanaged and are never sent.

### Apply ID writeback

Normal CLI apply appends each confirmed new ID to the original `.github/settings.yml` immediately after creation, before proceeding to another write. Existing IDs and unrelated values are retained. YAML comments are preserved where supported, but formatting may be normalized. The file must remain a regular file under a regular `.github` directory and match the initial snapshot plus earlier ID additions. Aliased/merged collection entries are not rewritten; use explicit mappings or disable writeback.

Use `gh settings apply --no-writeback` to disable this local change. With `GITHUB_ACTIONS=true`, writeback is always disabled, including through the public Action. Created IDs are printed for manual recovery/adoption; no Git commit or push is made. Plan never writes.

Remote creation and local writeback are not a transaction. If creation succeeded but writeback fails, apply stops, reports the created ID and leaves the remote object in place. Add that ID before retrying. Earlier confirmed IDs remain recorded if a later operation fails. A failed response may leave an unknown remote effect; inspect/export current state before attempting another create. No automatic create retry, delete or rollback occurs. File replacement is guarded and atomic, not a concurrent-editor lock.

Export labels to inspect their IDs. Give each existing object its ID, required color, and intended description, then review plan before apply.

Labels read and mutation credentials still need Issues or Pull requests read/write respectively; Administration write alone does not suffice.[^github-labels] No permissions are created or expanded. See the [command reference](commands.md#labels-command-behavior) and [behavior specification](features/labels.feature.md).

## Configuration semantics

The document requires integer `schemaVersion: 1`, a file-format version independent of the CLI release. The `settings` mapping is optional and can contain optional `repository` and `topics` mappings plus an optional `labels` sequence. The configuration also accepts `rulesets` and the optional `actions` and `security` mappings described above. Unknown root keys, unsupported versions and unsupported groups fail validation; there is no automatic import.

Declared values are managed, and YAML is authoritative for them. Omitted repository fields remain unmanaged. Topics retain authoritative membership. Labels and Rulesets select whole ID-owned objects, leaving unlisted objects untouched. Labels require color and default omitted description to empty; Rulesets have their own [required intent/default table](#rulesets). See the [labels](#apply-id-writeback) and [Rulesets](#plan-apply-and-export) contracts. For example:

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

The same type rules apply to `homepage`, `has_projects`, `has_wiki` and `has_discussions`. Quoted `'null'` is a string value, not YAML null. Null is invalid for the version, mappings, topic list and label declarations too.

A version-only document manages nothing:

```yaml
schemaVersion: 1
```

Empty `settings`, `repository` or `topics` mappings are also valid. Current source also accepts empty `actions` and `actions.selected_actions` mappings as unmanaged. When no resources are managed, plan and apply succeed locally without an origin, authentication or API requests. An explicit `names: []` still manages topics; current-source `labels: []` selects no objects. A managed topics clearing declaration requires comparison/update, unlike an empty label selection.

Validation rejects unknown or duplicate keys, non-string keys, wrong types, malformed YAML and multiple documents. YAML aliases and merge keys are resolved before supported values are checked. Partial declarations are accepted; unsupported root-level field names are rejected. See [validate behavior](commands.md#validate-behavior) for output and failure handling.

## Exported values

Export declares every supported observed repository field and the complete topic set. It reads all label pages and writes each label's ID, name, lowercase color and description in stable order; no labels produces `labels: []`. Listed label objects are managed as a whole, not authoritative remote membership. Remove objects you do not want managed, and review default-derived changes before applying.

GitHub can return null for an unset description or homepage. Export normalizes those response values to empty strings and writes quoted YAML strings (`""`); it preserves existing empty strings and false booleans. Missing response fields and null booleans are errors. Topic names must be present as a valid array; missing, null, invalid or unreadable names fail export instead of becoming an empty set.

For labels, response descriptions may be null and are normalized to empty strings. Missing required label fields, invalid names/colors/descriptions, ambiguous identities or a failed later page fail the whole read. All reads and serialization complete before export opens the destination.

This response normalization does not make explicit YAML null valid input. The published read schema permits null for description/homepage, while the update schema defines optional strings without `nullable`.[^github-repository-openapi] The [export command](commands.md#export-behavior) documents file replacement and write-failure behavior.

## Rulesets

Rulesets select whole supported repository-owned branch/tag objects by numeric REST ID.

```yaml
schemaVersion: 1
settings:
  rulesets:
    - id: 123
      name: release-tags
      target: tag
      enforcement: active
      conditions:
        ref_name:
          include: ["refs/tags/v*"]
      rules:
        - type: deletion
```

Only listed objects are managed. Omitted `rulesets` and `rulesets: []` select nothing and make no Rulesets requests. Unlisted repository Rulesets remain untouched; removing an entry never deletes the remote object. There is no Ruleset deletion declaration.

| Setting | Ownership / default on both create and update |
| --- | --- |
| `id` | Optional positive REST integer. Present selects an existing supported object in this repository; missing/wrong repository/type ID errors, never create fallback. |
| `name` | Required nonempty name without control characters. No ID means create-only: an existing name errors with its ID and guidance to add it. |
| `target` | Fixed default `branch`; `branch` and `tag` supported. |
| `enforcement` | Required: `active`, `disabled` or `evaluate`. Protection intent is not guessed. |
| `conditions.ref_name.include` | Required explicit pattern list; no guessed protected scope. |
| `conditions.ref_name.exclude` | Default `[]`. |
| `bypass_actors`, `rules` | Default `[]`, clearing these supported settings on an ID-selected object. |
| `null`, unknown keys, wrong types | Invalid before API work; null is not omission or clearing. |

Names use exact comparison without trimming or case folding, as in the existing Rulesets contract. Duplicate declared names or IDs fail. A valid ID and a new noncolliding name rename the same object. An ID-less name collision is never auto-adopted, and a missing ID is never replaced with a new object. A rename cannot collide even with another existing object scheduled to rename; apply is not a rename transaction.

Inner rules use unique `type`; bypass actors use `(actor_type, actor_id)`. Supported rule types are `deletion`, `non_fast_forward`, `required_linear_history`, `required_signatures`, `pull_request` and `required_status_checks`. Basic rules have no nonempty parameters.

Every declared bypass actor requires explicit `bypass_mode`: `always`, `pull_request` (branch/non-DeployKey only) or `exempt`. Types `Integration`, `RepositoryRole`, `Team` and `User` require positive `actor_id`; `OrganizationAdmin` and `DeployKey` omit it. OrganizationAdmin is unavailable for personal repositories. The adapter maps DeployKey omission to API null. No actor, access grant or credential is created.

### PR review and required check parameters

These are **fixed declaration defaults**, not claims about GitHub's omission behavior. Dismissal restriction has the temporary write-only exception below. Other supported members are not borrowed from remote state when YAML omits them; plan shows their default-derived protection changes.

| Rule / member | Default or requirement |
| --- | --- |
| `pull_request` | Requires branch target. Omitted parameters become the defaults below. |
| `allowed_merge_methods` | All three methods: `merge`, `rebase`, `squash`. Explicit value must be nonempty, unique and supported. |
| `dismiss_stale_reviews_on_push`, `require_code_owner_review`, `require_extra_approval_for_unattributed_changes`, `require_last_push_approval`, `required_review_thread_resolution` | `false`. |
| `required_approving_review_count` | `0`; explicit integer must be nonnegative. |
| `dismissal_restriction` | Temporary write-only support. Omission sends no value on create or update. An explicitly declared object uses `enabled: false`, `allowed_actors: []` for omitted members. Explicit declarations are accepted by validate/apply, but not plan; export always omits this field. |
| `required_reviewers` | `[]`. |
| `required_status_checks` rule | Check list `[]`, `strict_required_status_checks_policy: false`, `do_not_enforce_on_create: false`. |

Dismissal actors have positive `id` and type `User`, `Team`, `IntegrationInstallation` or `RepositoryRole`. Each required reviewer is `reviewer: {type: Team, id: 123}` with **required** `minimum_approvals` (0–10) and `file_patterns`; these are never copied from a retained reviewer. At most 15 teams are accepted. File patterns preserve order and repetitions, including negations. Other independent collections are canonicalized by identity.

A required check has nonempty `context` and optional positive `integration_id`. Omission means any source, not retention of a previous App binding. API null IDs normalize to omitted YAML IDs; YAML null remains invalid.

The extra-approval flag follows the [documented GitHub preview control](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#additional-approval-for-unattributed-copilot-pull-requests) and previously adopted observed REST shape. Unknown/unreadable supported values are not defaulted on read. Required parameter fields must be observable; missing protection cannot become false or empty. The optional dismissal-restriction wire contract is described below. Fixtures do not establish live write compatibility or entitlement.

### Plan, apply, and export

Nonempty selections acquire supported repository branch/tag observations, including complete bypass visibility. Unsupported rules, unknown writable fields, missing required values, null values, failed pages or unreadable details fail before Rulesets mutation. The documented optional absence of `pull_request.parameters.dismissal_restriction` is retained as unobserved state, not inferred to mean disabled. A present object still requires explicit `enabled` and `allowed_actors`; empty, incomplete and null objects fail. Only explicitly classified server-generated metadata is discarded. Inherited organization/enterprise and repository push Rulesets remain outside mutation ownership; push Rulesets are excluded from export with a diagnostic. Other protection can still affect effective behavior.

Plan is read-only. It shows creates or ID updates and supported comparable before/after objects, including fixed-default reductions and rename. A declaration containing `dismissal_restriction` fails plan explicitly before remote reads: that field cannot currently be compared. Omitted declarations do not manage or display its remote value. Plan never displays unlisted objects as changes or plans implicit deletion. A nonempty collection counts once as managed. Apply reads fresh state after earlier repository/topics/labels work, prepares every selected operation before the first Rulesets write, and writes in stable operation-name order. A fresh comparable match makes no Rulesets write unless a write-only value is explicitly declared; that value is sent on every apply. Successful ID writeback makes repeat apply select the created object instead of duplicate creation.

Every POST/PUT response must confirm positive ID, repository ownership and requested comparable settings. An update response must retain the selected ID. A write involving a pull-request rule also requires a fresh GET to match the comparable policy and identity, even if the mutation response echoed the request. The write-only dismissal restriction is excluded from value comparison, not from response type validation. Missing/mismatching comparable settings or failed readback return nonzero: writes may already have applied, later operations stop, and there is no retry or rollback. Successful creation is distinct from failed local ID persistence. Confirmed created IDs use the same [guarded writeback and opt-out contract](#apply-id-writeback) as labels; `--no-writeback` and `GITHUB_ACTIONS=true` disable it. IDs remain in output/recovery diagnostics, including output-write failure. No Git commit/push occurs.

Use `export --rulesets` to inspect supported IDs and policy before editing. Updates require explicit IDs, enforcement, and reference include rules. Removing an object from the declaration stops managing it without deleting its remote policy. Review plan before apply, especially reviewer, check, bypass, and merge-method defaults.

`dismissal_restriction` is temporarily **write-only** in current source. YAML and validate support explicit values; apply sends them and reports API acceptance with a notice that the value was not verified. API acceptance is not proof that GitHub stored or enforces that value, even if a response includes it. Omitted YAML sends no dismissal restriction and never invents false or borrows a remote value. Plan rejects an explicit declaration rather than inventing a no-op or drift. Export always omits this field, even when GET returns it: an exported file is incomplete for this option, and replacing an existing declaration with export output loses its explicit write intent. Other fields remain exportable and comparable. Required fields, malformed present objects and HTTP failures remain fail-closed. This narrow exception does not change other whole-object defaults. These semantics require a release containing this change.

GitHub documents dismissal restrictions as [generally available through REST, GraphQL and the UI](https://github.blog/changelog/2026-07-07-restrict-who-can-dismiss-reviews-in-rulesets/). One authorized experiment on a new disabled repository Ruleset found identical observable GETs after omitted, explicit-false and explicit-true PUTs; all omitted this field. It did **not** establish default-value elision, the hidden stored value, or why that test configuration returned no value. Matching GET absence never proves hidden-state preservation. The temporary object was deleted; the experiment does not establish behavior for other repositories or account plans.

[Rulesets REST](https://docs.github.com/en/rest/repos/rules) documents Metadata read for GET and Administration write for mutations; complete bypass visibility additionally requires Ruleset write access. Missing bypass actors are an error, never an empty set. GitHub validates account/plan prerequisites, including Enterprise-only evaluate; no permission expansion or workaround is performed. See [availability and layering](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) and the [behavior/test contract](features/rulesets.feature.md). This source change neither applies live policy nor publishes a release.

## Environments

Environments use name-selected whole-object policy management.

```yaml
schemaVersion: 1
settings:
  environments:
    - name: production
      wait_timer: 30
      prevent_self_review: true
      reviewers: [{type: User, id: 123}]
      deployment_branch_policy:
        protected_branches: true
        custom_branch_policies: false
```

Names are identity: a listed name updates that Environment or creates it when absent, using GitHub's ordinary name-addressed upsert. Omitted/empty collections select nothing. Unlisted names are never deleted or changed. Changing a name selects another object and leaves the old one unmanaged; there is no rename/delete operation. Unlike labels/Rulesets, Environment declarations have **no numeric `id`**, and apply never writes an Environment ID into YAML.

Names contain 1–255 Unicode characters without surrounding whitespace or control characters. Client identity and duplicate detection use Unicode simple folding; spelling is retained, and names including slash/percent are URL-encoded as one path segment. This is the client's case-insensitive comparison policy, not a claim that every undocumented GitHub Unicode equivalence is reproduced.

Each selected Environment is managed as one object. Optional fields omitted from YAML are omitted from its PUT request, on create and update; the client does not fill in default values or copy old values into the request. GitHub decides the effect of an omitted request field. This is not a promise that omitted values are preserved or reset. Use explicit values when a particular resulting policy is required.

| Field | Omission | Accepted explicit values |
| --- | --- | --- |
| `wait_timer` | Not sent; API-controlled | Integer 0–43200 minutes |
| `prevent_self_review` | Not sent; API-controlled | Boolean; true cannot accompany an explicitly empty reviewer set |
| `can_admins_bypass` | Not sent; API-controlled | Boolean; false prevents administrator bypass of deployment protection |
| `reviewers` | Not sent; API-controlled | At most six unique `{type: User\|Team, id: positive-integer}` references |
| `deployment_branch_policy` | Not sent; API-controlled | Null (all refs), or both opposite `protected_branches` and `custom_branch_policies` booleans |
| `deployment_branch_patterns` | No pattern CRUD requests | Array of `{type: branch\|tag, name: pattern}`; `[]` removes all observed patterns |

Only `deployment_branch_policy` accepts null. Other nulls, unknown fields, Environment IDs, deletion/rename markers and human-reference-only reviewers are rejected locally. GitHub validates reviewer access, account/plan eligibility and write permission; this tool does not invite principals or expand permissions. Protected-branches mode can permit all branches when no branch protections exist; successful configuration does not prove a deployment is protected.

Custom mode and a complete authoritative pattern list can be declared together:

```yaml
schemaVersion: 1
settings:
  environments:
    - name: production
      deployment_branch_policy:
        protected_branches: false
        custom_branch_policies: true
      deployment_branch_patterns:
        - {type: branch, name: 'release/*'}
        - {type: tag, name: 'v*'}
```

Pattern identity is the case-sensitive pair `(type, name)`, not a YAML/API ID. A branch and tag may use the same name. Duplicate pairs are invalid. A changed pair is added and the obsolete pair removed; retained pairs keep their observed IDs. GitHub owns pattern matching semantics. An omitted mode with an explicit list requires an existing custom-mode Environment; creation requires explicit custom mode. A list combined with explicit null/protected-branches mode is invalid.

Patterns belong to a separate REST collection, not the Environment PUT body. Omitting the list means the client sends no create/delete operations to that endpoint; it does not synthesize an empty list. It does not promise how a mode change affects inactive server-side patterns. An explicit list is authoritative and removes unlisted observed patterns. No Environment or unlisted Environment's patterns are deleted.

Plan shows the complete observed policy and the requested values/pattern list, without changing GitHub. Omitted built-in fields are marked API-controlled: their resulting values cannot be predicted, so even an apparently matching partial request remains an update intent rather than a verified no-op. Fully explicit matching policies can produce no differences. Nonempty selections count once as managed. Reads require complete inventory and selected detail/protection responses; missing/null containers, malformed values, unknown rules, custom deployment protection rules and unreadable responses fail rather than becoming empty/default state. Pattern reads require a complete typed paginated list: a missing `type` is not assumed to mean branch. Server-generated Environment/pattern IDs are not writable YAML. Other unsupported root settings are rejected, not silently dropped or guessed.

Administrator bypass is a supported writable protection setting, not disposable metadata. Export requires an explicit boolean observation and includes it even when false. Missing, null or mistyped responses fail rather than implying that bypass is enabled or disabled. For unchanged adoption, declare the observed `can_admins_bypass` along with every other built-in policy field: omission deliberately delegates to the API and is not a general preservation guarantee. No client default of true is inserted. The request field is implemented by the [go-github Environment API](https://github.com/google/go-github/blob/6eccc42c1ea8cd175b190f8e785d1a55656e3c49/github/repos_environments.go); GitHub's REST documentation does not currently describe it in the parameter table. A bounded live characterization confirmed explicit false/true PUT and GET on one temporary public-repository Environment using API version `2026-03-10`; omitting the flag while keeping the four other built-in request fields fixed retained the preceding false or true value in that test. The temporary object was deleted, with existing Environments unchanged. This establishes tested endpoint behavior, not a platform-wide omission, entitlement or effective deployment-enforcement guarantee. An API rejection or ignored explicit value remains a reported failure; unknown companion settings still block a write.

Apply preflights selected policies, then performs stable name-order upserts with fresh prerequisite reads and only the specified built-in fields. If a pattern list is supplied, it reads custom mode after PUT, creates missing typed patterns, then removes obsolete patterns by their verified API IDs. It verifies the requested values/list against complete fresh readback; omitted built-in outcomes are not claimed to match an invented default. A duplicate-create 303 or unavailable/malformed read is an error, not an adopted pattern. Earlier writes remain applied on later failure; a PUT/POST/DELETE/response/readback failure may have applied and is reported as uncertain. There is no transaction, automatic rollback, concurrency lock, intermediate deployment-enforcement guarantee or guarantee that the numeric Environment incarnation remains unchanged. Concurrent name-addressed creation/update follows normal GitHub upsert behavior.

Explicit `export --environments` includes every observed supported Environment policy, including the complete typed pattern list in custom mode, without numeric Environment/pattern IDs. It does not invent patterns for inactive modes or unknown responses. Unsupported/unreadable settings stop export before file replacement. Ordinary export does not acquire Environment permissions. A name-only declaration sends an empty PUT body: it is not merely an existence assertion, and the client does not predict the server's omission behavior. No secrets or Environment variables are read/exported/managed.

[REST Environments](https://docs.github.com/en/rest/deployments/environments) and [deployment branch policies](https://docs.github.com/en/rest/deployments/branch-policies) document Actions read and Administration write; [custom deployment protection reads](https://docs.github.com/en/rest/deployments/protection-rules#get-all-deployment-protection-rules-for-an-environment) are additionally required for selected existing objects. The API documentation defines separate pattern CRUD, but does not establish a general preservation/reset guarantee for omitted Environment fields or inactive patterns. Local fixtures establish client requests/failure handling, not live API preservation, entitlement or effective deployment security. See [behavior/test coverage](features/environments.feature.md) and the [published Action package boundary](github-actions.md#published-package-and-consumer-pins) before adopting these fields.

## Unsupported settings

Collaborator grants, repository invitations and team access remain unsupported in source and published packages. The [access design](features/repository-access.feature.md) describes proposed ownership and adoption gates; its examples are not accepted configuration.

Environment deletion/rename, custom branch/tag patterns, administrator-bypass management, secrets and variables remain unsupported. Published packages do not gain the new source capability merely by checking out newer configuration.

Pages lifecycle and visibility changes, and unadopted Actions settings, are not supported. Rulesets support is limited to the current-source subset above; push and inherited Rulesets, and legacy branch protection are not managed. Organization settings and multi-repository management are outside the tool's scope. Contributor [architecture and future direction](contributing/architecture.md) explains design choices without extending the configuration contract.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including listing and mutation permissions.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.

## Existing Pages configuration

The tool supports `settings.pages` for a positively observed existing site. Site creation, deletion and visibility changes remain unsupported and tracked in #350.

```yaml
schemaVersion: 1
settings:
  pages:
    build_type: legacy
    source: {branch: main, path: /docs}
    cname: null
    https_enforced: true
```

Omitting `pages` or using `{}` manages nothing and makes no Pages request. Omitted members stay unmanaged. `build_type` accepts `legacy` or `workflow`. A declared `source` requires both a literal branch and path (`/` or `/docs`); partial/empty/null tuples are invalid. Switching from workflow to legacy requires an explicit complete source, while workflow mode rejects a source declaration. No branch, content or workflow is created.

Only `cname` accepts YAML null (including a bare empty scalar), explicitly removing the custom domain. Omission preserves it; an empty string and quoted `"null"` are invalid. Domain strings must be ASCII DNS hostnames with at least two nonempty labels, normal hostname/label length limits, and no trailing dot, URL, wildcard, port or whitespace. Comparison/export use lowercase; no IDNA conversion or DNS verification occurs. `https_enforced: false` explicitly disables enforcement; null is invalid for booleans and other Pages fields.

Use `export --pages` to opt in without changing ordinary export permissions. Export writes mode, active legacy source, nullable domain and HTTPS enforcement only after a complete read. It excludes site visibility and readiness metadata. Any missing/invalid configured field or 404 fails instead of becoming an absent/disabled site; a failed export preserves the destination.

Plan is read-only and shows native string/boolean/null differences. Workflow source activation is shown as `inactive` before the declared tuple. Apply reads fresh state, skips a match, sends only declared PUT fields, then verifies the requested values, omitted applicable companions, site visibility and repository identity. Explicit unarchive precedes Pages; final archive follows it. Failure can leave prior writes or an uncertain Pages update applied; no retry, fallback write or rollback occurs.

Configuration convergence does not establish DNS, certificate or deployment readiness. Pending operational metadata does not create a configuration difference; a server refusal (including HTTPS not yet ready) fails the operation. GitHub may publish or create a CNAME-file commit as a consequence of settings changes; gh-settings neither triggers a build/deployment nor edits DNS/content itself.

Pages GET requires Pages read (public resources may be readable without authentication); mutations require Pages write plus Administration write and the applicable actor authorization. Public GET success does not prove write grants. A private repository can have a public site; observed `public` is preserved, never used as a write key. Site creation/deletion, visibility/access changes, unknown lifecycle keys and permission expansion remain unsupported. See the [Pages contract and evidence limits](features/pages.feature.md).
