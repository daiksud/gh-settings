# Repository settings

## v0.18.0 package

Published [v0.18.0](https://github.com/daiksud/gh-settings/releases/tag/v0.18.0), built from immutable source `6d3146b30860692f84dc54e32d746e6fbbb55c66`, adds private plan reporting, Sponsor visibility, archive lifecycle, existing-site Pages configuration, bounded custom-property values and private repository forking. Its native binaries and published Action passed the [release workflow](https://github.com/daiksud/gh-settings/actions/runs/39979971859). It also includes the v0.17.0 Rulesets, PR/Issue access, Actions, security, retention and fork-workflow features. The maintained examples select v0.18.0; adopting its pin does not expand caller grants, activate this repository's protected observer, or change settings by itself.

The current source fixes Ruleset plan presentation so only declaration-owned values are shown while mutation still preserves complete remote state. That #412 correction is newer than the immutable v0.18.0 source and is not in the published package. v0.18.0 detailed Ruleset plans can display retained undeclared attributes; do not treat a newer source checkout as a published binary update.

## Scope and status

Configuration lives in `.github/settings.yml` at the Git checkout root and manages only that checkout's GitHub.com repository. v0.18.0 supports twenty-five repository fields, topics, labels, the adopted Rulesets/Actions/security subsets, existing-site Pages and bounded custom-property values, using the same file-format version. v0.17.0 supports twenty-two repository fields and the earlier resource set. See the [command reference](commands.md) for behavior and the [Action package boundary](github-actions.md#published-package-and-consumer-pins) before adopting a configuration.

## Supported fields

The keys inside `settings.repository` follow GitHub's repository update request body parameters.[^github-update-repository] Discussions additionally uses the parameter sent by GitHub's official CLI. The PR-access fields introduced in v0.17.0 use the explicit GraphQL-to-YAML mapping below.

| Field | Type | Meaning |
| --- | --- | --- |
| `description` | string | Repository description |
| `homepage` | string | Repository homepage URL |
| `has_issues` | boolean | Whether issues are enabled |
| `has_projects` | boolean | Whether repository projects are enabled |
| `has_wiki` | boolean | Whether the wiki is enabled |

The booleans accept `true` or `false`; the string fields accept strings, including empty strings. Explicit null and other value types are invalid. All other repository fields and read-only response metadata are excluded, even when GitHub exposes them as writable parameters.

## Private repository forking

Published v0.18.0 supports `settings.repository.allow_forking` for active, non-fork, private repositories owned by organizations. Published v0.17.0 and older packages reject this key and `export --forking`.

```yaml
schemaVersion: 1
settings:
  repository:
    allow_forking: false
```

Omission is unmanaged. `true` permits private forking at the repository setting; `false` prohibits it there. Null, quoted values and other types are invalid. Public, internal, personal-owned, fork and archived repositories are outside this adopted subset. The matching organization must explicitly report that private repository forks are allowed, even to manage false. Hidden, missing, denied or disabled organization policy is an error, never a default or observed false. This setting does not delete existing forks, change organization/enterprise policy, or configure fork PR workflows and credentials.

Ordinary export omits the field and adds no organization-policy read. `gh settings export --forking` includes the current value after complete applicability checks; a failed read preserves the existing file. It composes with other applicable export selections except `--archive`. Declaring `allow_forking` together with any `archived` value is rejected locally; manage lifecycle separately.

Plan performs only reads and displays a native boolean difference. Apply processes this setting after other declared resources, checks complete current target/policy state, skips a match, then rechecks before a PATCH containing only `allow_forking`. The response and an independent readback must confirm the same repository/organization identities and desired value. A pre-write failure makes no forking-policy write; a failed PATCH or readback leaves its effects uncertain. Earlier successful resource writes remain applied. There is no automatic retry, rollback, fork creation/deletion or atomic lock against concurrent changes. Success verifies the repository setting, not every actor's ability to fork under higher-level restrictions.

Private repository Metadata read and existing visibility of the organization policy are required for plan/export; apply additionally requires Administration write and an eligible repository administrator. [Organization GET](https://docs.github.com/en/rest/orgs/orgs#get-an-organization) exposes full details to organization owners; classic credentials need `admin:org` for those details. Endpoint success alone does not establish visibility. The tool never expands grants or issues credentials. The [repository policy guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/managing-the-forking-policy-for-your-repository) explains applicability; [fixture coverage](features/private-forking.feature.md) does not establish live token capabilities or policy mutation behavior.

## Repository archive lifecycle

Published v0.18.0 supports the optional `settings.repository.archived` boolean. Published v0.17.0 and older CLI/Action packages reject it. The field manages the selected checkout repository's archive state, not repository renaming, visibility, or individual issues/releases.

```yaml
schemaVersion: 1
settings:
  repository:
    archived: false
    description: Active maintenance
```

Omitting `archived` leaves the lifecycle unmanaged and adds no archive-specific API read. `false` means explicitly unarchive and `true` means explicitly archive. Null, quoted strings, numbers and other YAML types are invalid. Ordinary `gh settings export` does not add the field: use `gh settings export --archive` to include its currently observed value after all requested reads succeed. A failed read leaves any existing configuration file untouched.

Plan reads only the declared archive state and validates the repository's positive ID, canonical owner/name and boolean; it does not mutate the repository. With other settings, comparisons remain typed and read-only, with an explicit unarchive-before-other-settings or archive-after-other-settings note for a changed lifecycle field. An already archived target with `archived: true` and any other managed resource is rejected during plan, not represented as an approvable change. Apply validates a complete current identity before any write. An explicit unarchive runs first, then existing resource writes; a final archive runs only after every other update succeeds, with a fresh state/identity check. Archive PATCH requests contain **only** `archived`, and each response and readback must confirm the target ID, name, and requested state. Matching values cause no lifecycle write.

On an already archived repository, an `archived: true` declaration combined with other managed settings fails **before all writes**; use an explicit separate `archived: false` declaration to resume updates. A failed unarchive prevents subsequent writes. A confirmed unarchive remains applied if a later settings update fails; that fact appears in the error even when the CLI discards the detailed result. A failure during final archive may leave prior settings updates applied, and a failed archive PATCH, transport error, or readback can leave the remote lifecycle state uncertain. There is no rollback, automatic retry, transactional identity lock or implicit temporary unarchive. Running a repository's own Action against its archived source may not be a viable way to recover it; keep an external admin recovery route.

The [repository REST API](https://docs.github.com/en/rest/repos/repos#update-a-repository) requires appropriate repository Administration write permission for lifecycle changes, and read access for observation. These grants must already exist. Tests use local simulated HTTP responses; **no live archive/unarchive is performed or verified**. See the [lifecycle behavior specification](features/repository-lifecycle.feature.md) and [ADR 0027](adr/0027-repository-lifecycle.md).

## Secret scanning and push protection

Current source supports two optional booleans under `settings.security`. Published v0.16.0 and older CLI/Action packages reject this group and `export --security secret-scanning`; use v0.17.0 or newer. The Action still installs its selected published CLI.

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

## Private vulnerability reporting

Current source supports optional boolean `settings.security.private_vulnerability_reporting`. Published v0.16.0 and older CLI/Action packages reject the key and `export --security private-vulnerability-reporting`; use v0.17.0 or newer. The maintained Action pin selects v0.18.0.

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

Current source adds two independent optional fields under `settings.actions`. Published v0.16.0 and older CLI/Action packages reject these keys and `export --workflow-permissions`; use v0.17.0 or newer. Newer Action source still installs its selected published CLI.

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

Current source supports `settings.actions.fork_pr_approval_policy` and `settings.actions.private_fork_workflows`. Published v0.16.0 and older CLI/Action packages reject these keys and the `--fork-approval` / `--private-fork-workflows` export options. Use v0.17.0 or newer; caller grants are unchanged.

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

Current source supports optional integer `settings.actions.artifact_and_log_retention_days`. Published v0.16.0 and older CLI/Action packages reject this key and `export --retention`; use v0.17.0 or newer. The maintained Action pin selects v0.18.0.

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

Current source adds optional `settings.actions`. Published v0.16.0 and older CLI/Action packages reject this group and `export --actions`; use v0.17.0 or newer. The six execution-policy fields below manage repository execution policy, not workflow files, individual runs, token grants or organization policy. [Default workflow permissions](#default-workflow-permissions) use separate fields and transport.

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

Published v0.18.0 supports optional `settings.repository.has_sponsorships`, mapped to GraphQL `hasSponsorshipsEnabled`. Published v0.17.0 and older CLI/Action packages reject this key.

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

Current source adds optional `settings.repository.issue_creation_policy`, mapped to GraphQL `issueCreationPolicy`. Published v0.16.0 and older CLI/Action packages reject this key. Use v0.17.0 or newer; newer Action source still installs its selected published CLI.

```yaml
schemaVersion: 1
settings:
  repository:
    issue_creation_policy: COLLABORATORS_ONLY
```

When Issues are enabled, `ALL` permits creation by anyone and `COLLABORATORS_ONLY` limits it to collaborators, following [GitHub's enum contract](https://docs.github.com/en/graphql/reference/issues#issuecreationpolicy). Omission leaves the setting unmanaged. Null, empty strings, booleans/numbers, case variants and unknown policies are invalid. This controls creation eligibility independently of `has_issues`; it does not enable Issues, edit individual Issues, invite collaborators or change grants.

Plan uses a separate complete authenticated GraphQL query only when this policy is declared. PR-only and older declarations retain their reads. Export includes the observed policy after all reads succeed, including when Issues are disabled. Missing/null/wrong-type/unsupported values and partial GraphQL errors fail instead of becoming `ALL`; failed export preserves the old file. Policy reads and ID lookup disable rename following, so stale origins fail.

Direct apply reuses the validated opaque repository-ID lookup and `updateRepository` transport. Send only declared Issue/PR values; never synthesize `hasIssuesEnabled`, `has_issues` or PR companions. If PR and Issue controls are both declared, one lookup and one mutation carry them together. Explicitly declared REST fields such as `has_issues` run first, then GraphQL, topics, labels and Rulesets. Repeated scalar apply still sends declared values. Mutation response identity and requested values must match; failures stop later work and may leave earlier writes applied, with no rollback or retry.

Read credentials need complete policy visibility and write credentials must be authorized to update the target repository; caller grants are unchanged. Isolated fixtures and read-only observations do not prove live mutation compatibility, token permissions or published-package support. See the [feature specification](features/issue-creation-policy.feature.md).

## Pull request access

Current source adds two independent `settings.repository` values. Published v0.16.0 and older CLI/Action packages reject them; use v0.17.0 or newer. Checking out newer Action source still installs its selected published CLI.

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

Published v0.16.0 and current source support optional `settings.repository.default_branch`, a string selecting an existing branch as the repository default. Published v0.15.1 and older CLI/Action packages reject this key. It uses the existing `schemaVersion: 1` repository group. Ordinary export includes twenty-three repository fields, topics and labels; archive and private forking require their explicit options.

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

Published v0.16.0 and current source support optional `settings.repository.is_template`, a boolean controlling whether the repository can be used as a template. Published v0.15.1 and older CLI/Action packages reject this key. It uses the existing `schemaVersion: 1` repository group; no repository-generation command is added.

```yaml
schemaVersion: 1
settings:
  repository:
    is_template: true
```

Omission leaves the setting unmanaged; explicit `false` removes template status. Null, quoted booleans and other types are invalid. Plan compares a complete non-null boolean from the existing repository REST GET and remains read-only. Export includes the observed value after all reads succeed. Missing, null or malformed observations fail instead of becoming false.

Apply sends only declared fields through the existing Administration-write repository PATCH, with no preliminary read or GraphQL request. GitHub's [template prerequisites](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-template-repository), including the Git LFS restriction, still apply; API rejections remain errors rather than triggering workarounds. The tool does not create repositories, copy files, change template provenance, or manage template contents. See the [feature contract](features/template-repository.feature.md).

## Web commit sign-off

Published v0.16.0 and current source support optional `settings.repository.web_commit_signoff_required`, a boolean requiring sign-off on commits made through GitHub's web interface. It extends the repository group under `schemaVersion: 1`. Published v0.15.1 and older CLI/Action packages reject this key; adopt v0.16.0 or newer before using it. Ordinary export includes twenty-three repository fields, topics, and labels; published v0.15.x export retains sixteen repository fields.

```yaml
schemaVersion: 1
settings:
  repository:
    web_commit_signoff_required: true
```

Omission leaves the field unmanaged. Explicit `false` disables the requirement; null, quoted booleans and other types are invalid. Plan reads a complete non-null boolean from the existing repository REST GET, compares declared values only, and makes no GraphQL request for this field. Export includes the observed boolean after all reads succeed. Missing or wrongly typed values fail the read instead of becoming false.

Apply sends the field only when declared, in the existing repository PATCH with Administration write permission and no preliminary read. No new token grant is introduced. This is a sign-off requirement, not cryptographic commit-signature enforcement or a DCO check. It does not rewrite commits or change branch protections. The [feature contract](features/web-commit-signoff.feature.md) maps the request, failure and presentation tests. GitHub's [repository update API](https://docs.github.com/en/rest/repos/repos#update-a-repository) documents the exact field name.

## Discussions (v0.15.0)

v0.15.0 adds optional `settings.repository.has_discussions`, a boolean controlling GitHub Discussions. It extends the repository group; it does not manage discussion categories or individual discussions. Published v0.14.0 and older packages reject this key. The pinned v0.18.0 CLI/Action supports it; update older consumer pins before adopting the declaration.

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

Published v0.10.0 does not accept this key. The pinned v0.18.0 CLI/Action includes it; older consumer pins must be updated before adopting this declaration.

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

Published v0.11.0 rejects these keys. The pinned v0.18.0 CLI/Action supports them; update older consumer pins before adopting these declarations.

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

Published v0.12.0 rejects these keys. The pinned v0.18.0 CLI/Action supports them; update older consumer pins before adopting these declarations. The [feature contract](features/pull-request-options.feature.md) records exact API mapping, hosted proof and test coverage.

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

Published v0.13.0 rejects these keys. The pinned v0.18.0 CLI/Action supports them; update older consumer pins before adopting these declarations.

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

`settings.labels` manages repository label definitions, supported since v0.9.0 and included in the pinned v0.18.0 Action examples. Older v0.8.0 packages do not support Labels. It does not manage individual issue/PR assignments, although deleting a definition removes its assignments.

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

The document requires integer `schemaVersion: 1`, a file-format version independent of the CLI release. The `settings` mapping is optional and can contain optional `repository` and `topics` mappings plus an optional `labels` sequence. v0.17.0 additionally accepts `rulesets` and the optional `actions` and `security` mappings described above. Unknown root keys, unsupported versions and unsupported groups fail validation; there is no automatic migration or compatibility parsing.

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

Empty `settings`, `repository` or `topics` mappings are also valid. Current source also accepts empty `actions` and `actions.selected_actions` mappings as unmanaged. When no resources are managed, plan and apply succeed locally without an origin, authentication or API requests. An explicit `names: []` still manages topics; `labels: []` still manages all label membership. These declarations require a comparison or update rather than a local no-op.

Validation rejects unknown or duplicate keys, non-string keys, wrong types, malformed YAML and multiple documents. YAML aliases and merge keys are resolved before supported values are checked. Partial declarations are accepted; unsupported root-level field names are rejected. See [validate behavior](commands.md#validate-behavior) for output and failure handling.

## Exported values

Export declares every supported observed repository field and the complete topic set. Export additionally reads all label pages and saves the complete label collection sorted by identity, with names, lowercase colors and descriptions, including `labels: []` when none exist. This makes the observed membership authoritative. Review the saved file and remove resources or attributes you want to leave unmanaged before applying it.

GitHub can return null for an unset description or homepage. Export normalizes those response values to empty strings and writes quoted YAML strings (`""`); it preserves existing empty strings and false booleans. Missing response fields and null booleans are errors. Topic names must be present as a valid array; missing, null, invalid or unreadable names fail export instead of becoming an empty set.

For labels, response descriptions may be null and are normalized to empty strings. Missing required label fields, invalid names/colors/descriptions, ambiguous identities or a failed later page fail the whole read. All reads and serialization complete before export opens the destination.

This response normalization does not make explicit YAML null valid input. The published read schema permits null for description/homepage, while the update schema defines optional strings without `nullable`.[^github-repository-openapi] The [export command](commands.md#export-behavior) documents file replacement and write-failure behavior.

## Rulesets

Current source adds `settings.rulesets` for repository-owned branch/tag Rulesets with `deletion`, `non_fast_forward`, `required_linear_history`, `required_signatures`, `pull_request` and `required_status_checks` rules. Published v0.16.0 and older CLI/Action packages reject this group and the new export option. Use v0.17.0 or newer for these capabilities; checking out newer Action source still installs its selected published CLI.

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

Plan is read-only and shows operation names plus only declaration-owned values. A retained member's omitted attributes do not appear; declared inner collections expose their managed membership differences, and explicit false, zero and empty values remain visible. A Ruleset deletion shows only its name and protection-removal warning, not the deleted member's unmanaged attributes. The collection counts once as managed. Apply separately retains complete observations, runs after repository/topics/labels, reads fresh Rulesets with the apply credential, prepares all operations before the first Rulesets mutation, and preserves omitted siblings in complete PUT bodies. Creates/updates run in name order, then deletes in name order. An unchanged policy causes no Rulesets write; a repeat apply after successful reconciliation also causes no Rulesets write absent concurrent changes. Earlier resources retain their own direct-write behavior.

The first API failure stops later operations; earlier writes can remain applied, failed writes have uncertain effect, and there is no transaction, rollback, automatic retry or saved-plan execution. Review plan, including nested removals, before authorizing apply. Concurrent edits can change the fresh apply result.

[Rulesets REST](https://docs.github.com/en/rest/repos/rules) lists Metadata read for GET and Administration write for mutations, but complete detail GET also needs write visibility to return bypass actors. Missing actors are an error, not an empty list. No token grant or App permission is expanded automatically. Branch/tag Rulesets are available for public repositories on Free and applicable public/private repositories on Pro/Team/Enterprise; evaluate requires Enterprise. [GitHub availability and layering](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) and [available rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) apply. Other rule types remain unsupported; the [feature contract](features/rulesets.feature.md) records implementation scope and test evidence.

## Unsupported settings

Collaborator grants, repository invitations and team access remain unsupported in source and published packages. The [access design](features/repository-access.feature.md) describes proposed ownership and adoption gates; its examples are not accepted configuration.

Environment configuration, lifecycle, deployment protections and branch/tag policies remain unsupported in source and published packages. The contributor [Environment design](features/environments.feature.md) reserves possible syntax and records adoption gates; it does not extend the accepted YAML schema or export options.

Pages lifecycle and visibility changes, and unadopted Actions settings, are not supported. Rulesets support is limited to the current-source subset above; push and inherited Rulesets, and legacy branch protection are not managed. Organization settings and multi-repository management are outside the tool's scope. Contributor [architecture and future direction](contributing/architecture.md) explains design choices without extending the configuration contract.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the five body parameters and their request body placement. This reference does not expand the supported scope.

[^github-delete-labels]: [GitHub — Deleting a label](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/managing-labels#deleting-a-label), including removal from associated issues and pull requests.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including listing and mutation permissions.

[^github-repository-openapi]: [GitHub official OpenAPI schema](https://github.com/github/rest-api-description/blob/58b1e0c00b39b9e46c24dc4a0ba6a2669c90e9c1/descriptions/api.github.com/api.github.com.json), checked on 2026-10-12 for nullable full-repository description/homepage values and optional string parameters in `PATCH /repos/{owner}/{repo}`.

## Existing Pages configuration

Published v0.18.0 supports `settings.pages` for a positively observed existing site. Published v0.17.0 and older packages reject this group and `export --pages`. Site creation, deletion and visibility changes remain unsupported and tracked in #350.

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

## Custom property values

Published v0.18.0 accepts `settings.custom_properties` for the adopted organization-owned property subset. Published v0.17.0 and older packages reject this group and `export --custom-properties`. Required/default-backed/explicitly-required properties, enterprise sources, organization-only editing, URL properties and regex-constrained strings remain outside the adopted subset and are tracked in #354.

```yaml
schemaVersion: 1
settings:
  custom_properties:
    service: "payments"
    regions: ["east", "west"]
    retired: "false"
    previous_owner: null
```

Omission leaves a property unmanaged; `{}` manages nothing. Null explicitly unsets a named property, including an empty YAML scalar. Empty string and empty array retain those values. An array is a unique set of strings; order does not cause drift. Boolean property values must be quoted `"true"` or `"false"`. YAML booleans, numbers, objects, duplicates and unsupported value shapes fail local validation. Names are case-sensitive ASCII letters/digits or `_`, `-`, `$`, `#`, with length 1–75. Strings and array members are printable ASCII excluding double quotes, at most 75 characters. Property existence, schema type and allowed selections are validated during remote operations.

Only organization-sourced properties editable by organization and repository actors are adopted. Complete schema must establish `required: false`, `require_explicit_values: false`, a null default and, for strings, no regex constraint. Supported types are string, single-select, multi-select and true/false. Unknown names, unavailable constraints and invalid selections fail. Required/default-backed, explicitly required, enterprise-sourced, organization-only, URL and regex-constrained properties remain unsupported. Schema definitions are never created or changed.

Use `export --custom-properties` to read the full organization schema and repository values. It declares every supported observed schema name, including null for unset values; remove names you want unmanaged before applying. An unsupported property or incomplete read fails the whole opted-in export and preserves the old file. Ordinary export does not request properties. The complete schema/value arrays and authenticated GraphQL string-constraint reads must be accessible; unavailable data and 404 are errors, not empty state.

Plan remains read-only and displays JSON-native string/array/null differences. Apply reads fresh state, skips a match, rereads before mutation, PATCHes only declared names, and verifies selected schemas, values, retained unowned values and repository/organization identity. It follows other resource writes and precedes final archive; explicit unarchive comes first. Earlier writes remain applied after failure; an attempted property update can have uncertain outcome. There is no retry, rollback or atomic concurrency guarantee.

Value reads require repository Metadata read; schema reads require organization Custom properties read plus authenticated access to the GraphQL schema fields. Value updates require repository Custom properties write and the schema's actor authorization. A successful read does not prove write grants. No schema administration permission is requested. See the [behavior and evidence limits](features/custom-properties.feature.md).
