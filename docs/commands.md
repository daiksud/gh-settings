# Settings commands

## Environment export

`gh settings export --environments` includes all supported named Environment policies, never variables, and composes with other export options. It requires complete Environment/custom-protection reads and preserves the existing output-file safety contract. Ordinary export does not read Environments.

`validate`, `plan` and `apply` accept the [Environment configuration contract](settings.md#environments); policy-selected names select ordinary create-or-update, not numeric IDs; [variable-only entries](#environment-variable-commands) do not upsert policy. No Environment writeback occurs, regardless of `--no-writeback` or GitHub Actions mode. Unlisted objects remain untouched. Name-only entries send an empty upsert body. Omitted built-in values remain API-controlled; the client promises neither preservation nor reset. Inspect plan before apply.

Environment deployment-mode observations follow the [raw-pair contract](settings.md#environment-deployment-mode-observation-normalization). Export warns only after complete selected acquisition, in folded-name order. Detailed plan/Summary label all Environment before-objects as effective policy and add a scoped normalized/write-required UI decision only for explicit mode intent. Status-only hides names and pair values. Omitted mode sends no field, retains API-controlled outcomes and emits no unowned warning; explicit mode requires canonical raw confirmation, while unknown-mode pattern intent requires explicit Selected.

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

```text
gh settings export [--pull-request-limits] [--pull-request-limit-exemptions] [--copilot-mcp] [--copilot-review-effort] [--copilot-custom-instructions] [--variable NAME]... [--agent-variable NAME]... [--environment-variable ENVIRONMENT NAME]... [--environments] [--pages] [--archive] [--rulesets] [--actions-policies] [--actions] [--workflow-permissions] [--retention] [--cache-retention] [--cache-storage] [--fork-approval] [--security FAMILY]...
gh settings validate
gh settings plan
gh settings apply [--no-writeback]
```

| Command | Contract |
| --- | --- |
| `export` | Read ordinary and explicitly selected settings and save `.github/settings.yml`. Existing configuration is not required. |
| `validate` | Check the configuration locally against the supported schema. |
| `plan` | Validate, then read managed resources and show comparisons/selected-object operations. |
| `apply` | Validate, then execute the declared families in order with their existing reconciliation/preflight rules. |

There are no target override, configuration-path or output-format flags. Use `gh settings --help` for usage and `gh settings --version` for the installed version. Unsupported commands or arguments fail with exit status 1.

## Repository and authentication

Export and plan/apply operations with managed resources resolve the checkout-scoped `origin` remote. Supported origins directly identify GitHub.com through HTTPS, `git@github.com:owner/repo.git` or a full SSH URL. Rewrite-only aliases and username-less `github.com:owner/repo` are rejected; configure a supported direct URL instead.

Only local/worktree origin values, including repository-specific included configuration, can select the target. System, global and command Git configuration cannot supply it. A direct GitHub origin retains its repository identity when Git `insteadOf` changes the transport destination. `GH_REPO` is ignored; missing or unsupported origins fail before API access.

The extension uses the authentication selected by GitHub CLI, normally the user's own authority. Check access with `gh auth status`; authenticate with `gh auth login` when needed. gh-settings has no separate credential store, does not mint credentials, and does not retry a denied API operation as another identity. Standard `gh` authentication selection (including explicitly configured environment credentials) remains GitHub CLI behavior, not an independent fallback in this extension. `validate` performs no API work and needs no authentication. For automation, use the separate [single-App Actions policy](github-actions.md#authentication-and-app-permissions), not a required App/PAT combination. Credentials need permissions for every resource the command reads or writes:

| Resource | Read permission | Write permission |
| --- | --- | --- |
| Repository fields and topics | Metadata read | Administration write |
| Labels | Issues or Pull requests read | Issues or Pull requests write |
| Merge methods; automatic branch deletion; squash commit defaults; auto-merge and branch updates; normal merge commit defaults | GraphQL query; ordinary Actions Contents read verified | Administration write through REST PATCH |

Administration write alone does not grant label write permission.[^github-labels] Source export always reads labels as well as repository fields/topics; source apply uses its apply credential for the fresh label read and any required mutations. Public REST reads may be unauthenticated; GraphQL reads use an authenticated read token. Local validation and valid declarations with no managed resources do not require an origin or authentication.

Plan does not mutate settings, but complete Ruleset bypass observation requires write access to the Ruleset. A read-only token can therefore be insufficient. Feature-specific API permissions and unverified App-integration boundaries are listed in the [permission table](github-actions.md#authentication-and-app-permissions); denied/missing reads remain errors, never empty/default/success.

## Topics command behavior

The [topics contract](settings.md#topics) defines names, omission and explicit clearing. Export includes the observed topics; plan compares the managed set; apply replaces that set. A configuration containing only `repository.about.topics` is valid managed intent. Omitted topics skip the resource; `[]` clears it and `{}` is invalid. The command is a local no-op only when no other resource is managed.

## Labels command behavior

The [labels contract](settings.md#labels) uses ID-based listed-object ownership: ordinary apply manages only listed objects; [explicit collection pruning](settings.md#collection-pruning) adds deletions for present lists. An ID selects an existing label; no ID means create-only and a name collision reports the existing ID.

Plan shows creates and ID-bound updates, including default-derived values; `--prune labels` also shows unlisted deletions and their Issue/PR assignment loss. Color is required; omitted description defaults to empty on create and update. A supported rename keeps the ID. Apply records confirmed new IDs in the original YAML after each creation; `--no-writeback` or `GITHUB_ACTIONS=true` disables this. Created IDs remain visible for recovery, including when later remote operations or local writeback fail. There is no automatic commit, push, create retry or rollback.

## Actions event policy commands

`export --actions-policies` independently selects the complete
[Actions event/actor policy collection](settings.md#actions-event-policies), including
IDs and explicit workflow include/exclude selections, sorted by policy name,
event name, actor identity and pattern. It composes with other export options;
duplicates fail before repository/API/file work. Ordinary export and `--actions`
do not read it. All pages/details must be complete and supported before saving;
unsupported policies or failed reads preserve the existing destination.

`validate` checks the complete listed objects locally. Omission and `[]` make no
policy requests. Plan reports creates and ID-bound updates without writing.
Apply preflights the full collection, skips matches, guards updates against fresh
detail drift, and independently verifies acknowledgment and final GET. No ID
means create-only; exact name collisions fail. IDs select repository-owned
objects, including rename, without create fallback. Unlisted policies remain
unchanged; there is no deletion or atomic rollback. Omitted `target_workflows`
uses All workflows for creation and preserves existing targeting for updates;
declaring both arrays replaces the selection.

Confirmed new IDs use guarded source-file writeback unless `--no-writeback` or
`GITHUB_ACTIONS=true` suppresses it. Known IDs remain in recovery diagnostics even
when a later step fails. An uncertain create is never retried automatically;
inspect remote state and record the ID before retrying. Fine-grained/App tokens
need **Administration write for observation as well as mutation**, including
export and plan. Current-source support does not establish support in the
selected published package or live active-policy enforcement.

## Copilot custom instructions commands

Current source accepts the [Use custom instructions boolean](settings.md#copilot-code-review-custom-instructions). `export --copilot-custom-instructions` explicitly selects this setting and composes with other options; duplicate flags fail before target/API/file work. Without an explicit Copilot selection, ordinary export and `--actions` acquire no Copilot code-review settings. A complete true/false read is required before saving.

Plan shows the native boolean comparison. Apply reads fresh state, skips a match and verifies narrow PATCH updates. The flag stores configuration without enabling or requesting reviews and does not read or edit instruction files. API permission and review-entitlement boundaries remain separate; this option does not expand either.

## Copilot MCP tools commands

`export --copilot-mcp` selects only the [MCP tools boolean](settings.md#copilot-code-review-mcp-tools), including false. It composes with the other Copilot flags and export options; duplicates fail before target/API/file work. Ordinary export and `--actions` do not select it. Complete selected acquisition is required before saving; a failure preserves the existing file.

Plan shows native booleans at `repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests`. Apply reads fresh complete settings, skips matches, PATCHes only `mcp_enabled` and independently verifies the requested boolean and unchanged sibling settings. Failures retain honest partial or unknown-effect reporting. This control does not configure MCP servers or secrets, grant access, request reviews or establish MCP execution.

## Copilot review effort commands

`export --copilot-review-effort` selects only the [Review effort level](settings.md#copilot-code-review-effort-level). It composes with `--copilot-custom-instructions`, `--copilot-mcp` and other export options; duplicate flags fail before target/API/file work. Complete selected acquisition is required before saving, and a failure preserves the existing file.

Plan and detailed Action reports show `default`, `lite` or `balanced` at `repository.settings.copilot.code_review.review_effort_level`. Default is an observable configured-null state, not proof that GitHub erased an unavailable saved override. Apply reads fresh availability, skips observable matches, sends only the declared effort when needed and independently verifies readback. Unsupported or unavailable observations are errors, never successful fallback. This command neither enables nor requests Copilot reviews.

## Pull request limit exemptions

`export --pull-request-limit-exemptions` independently reads the complete
[Exemption list](settings.md#pull-request-limit-exemption-list), requiring
Administration write. It does not read or enable the cap. The existing
`--pull-request-limits` selection continues to export only its three controls.

An explicitly declared `exemption_list` owns complete membership; `[]` clears it,
while omission makes no list request. Apply removes obsolete users before adding
new users and verifies each stage; later failures do not roll back confirmed
removals. It runs before separately selected cap controls. Detailed output
contains membership, while status-only Summary omits it. No temporary restriction,
automatic exemption rule or collaborator grant is managed.

## Environment variable commands

`export --environment-variable ENVIRONMENT NAME` selects one known non-sensitive [Environment variable](settings.md#environment-variables). Repeat the flag for additional Environment/name pairs; quote Environment names containing spaces. Names validate before target/API work, with duplicate pairs rejected using folded Environment identity and uppercase variable identity. Ordinary export and `--environments` make no variable requests. Every selected value must be acquired before the output file is saved.

Variable-only declarations do not upsert an Environment or touch its policy, patterns or secrets. A missing Environment fails. Combined policy/variable declarations apply selected policy first, then variables in stable Environment/name order, before repository Actions and Agents variables. Earlier successful writes remain applied on later failure. Values are plaintext in detailed output; status-only Summary hides details. No bulk selection, deletion, rename or ID writeback is supported.

## Repository variables commands

Current source accepts separate [Actions](settings.md#actions-repository-variables) and [Agents](settings.md#agents-repository-variables) repository variable declarations. `export --variable NAME` selects one known non-sensitive Actions name; `export --agent-variable NAME` selects one Agents name. Repeat either flag for other names. Names validate before target/API work, including case-insensitive duplicate rejection within each family. The same canonical name can be selected in both families independently. Ordinary export and `--actions` do not read variables. All selected individual GETs must succeed and find the requested names before saving; a failure preserves the existing output file. There is no bulk/all selection.

Plan is read-only and identifies the selected family. Apply upserts only declared names, preserving unlisted variables and skipping equal values. Actions variables are processed before Agents variables, with stable name order within each family. Earlier successful writes remain applied after a later failure. Values are plaintext and appear in configuration, detailed plan/report and Summary; never use these features for secrets. Actions variables require the repository **Variables** read/write permission; Agents variables require the separate **Agent variables** read/write permission. Neither grants access to the other family.

## Export behavior

`export --cache-retention` and `export --cache-storage` independently select [cache limits](settings.md#actions-cache-policy). They compose with other export flags; duplicates fail before API/file work. Ordinary export, `--actions` and `--retention` do not acquire them. Complete selected reads are required before saving. Cache storage needs Actions read; cache retention needs repository metadata and Administration read. Both updates need Administration write and their preflight/readback permissions.

`export --retention` for the [repository retention policy](settings.md#check-run-status-artifact-and-log-retention). It composes with the other selections; duplicates fail before reads or file changes. Both configured days and the read-only maximum must be acquired completely; only days are exported. Retention reads occur after workflow permissions and before security families. Other export selections compose with it.

Current source supports `export --security codeql-configuration` for [existing CodeQL configuration](settings.md#codeql-default-configuration), `export --security ai-scan-pull-requests` for [AI Scan enablement](settings.md#ai-scan-for-pull-requests), `export --security codeql-default-setup` for [CodeQL enablement](settings.md#codeql-default-setup), `export --security dependabot` for [alerts and security updates](settings.md#dependabot-alerts-and-security-updates), `--security secret-scanning` for [Secret scanning](settings.md#secret-scanning-and-push-protection), `--security private-vulnerability-reporting` for [public-repository reporting](settings.md#private-vulnerability-reporting), and `--security immutable-releases` for release policy. Repeat distinct families to compose them with other selections; duplicate/missing/unknown families, `all` and excluded product-wide enablement selections fail before reads or destination changes. Selected security reads must complete before saving. Ordinary export does not acquire security policy. Dependabot requires Administration read, interprets GET 404 as disabled under required permissions, and never exports paused metadata. Its apply path requires a 204 acknowledgment and matching fresh family readback; the configuration reference owns the endpoint-specific verification and ordering contract.

`export --workflow-permissions` independently includes complete default workflow-token/PR-approval policy. It composes with `--actions` and `--rulesets` in any order; duplicate/unknown options fail before export. All reads precede saving.

`export --actions` additionally reads complete repository Actions execution policy; active selected policy includes its complete selection. It can be combined with `--rulesets` in either order. Extra resources are read after labels (Rulesets before Actions when both are requested), and all reads finish before saving. Inactive selected values are omitted, never guessed. See the [Actions policy contract](settings.md#actions-execution-policy) for required credentials and transition limits.

`export --rulesets` additionally reads complete repository-owned branch and tag Rulesets. Ordinary export retains its existing requests/permissions and does not include Rulesets. Complete bypass visibility is required; unsupported/hidden data fails before saving. Complete out-of-UI enforcement, Required approvals, Allowed merge methods, scanning Alerts, scanning Security alerts, bypass modes emit contextual normalization warnings in exported name order: enforcement, Required approvals, Allowed merge methods, scanning Alerts and Security alerts (each in tool-name order), then bypass actors ordered by type and numeric ID; other successful exact observations stay silent. See the [Rulesets contract](settings.md#rulesets) before adoption.

Export uses the [repository and authentication rules](#repository-and-authentication) above. It reads repository settings, then topics, then every labels page, and serializes the complete supported configuration before opening the destination. Label reads use pages of 100 until a short page; a failed later page fails the operation before any file replacement. Unsupported fields and read-only metadata are excluded. Export preserves false and empty strings; the [settings reference](settings.md#exported-values) explains response normalization and invalid-response handling.

Export creates `.github` when missing and always overwrites an existing regular `.github/settings.yml`, without a confirmation or overwrite flag. Symlinks at `.github` or `settings.yml`, including dangling links, and other non-regular destinations are rejected. A read or serialization failure leaves the destination untouched. A write failure can leave it truncated or partially written; use Git to restore tracked configuration.

Success writes no stdout and exits 0. Explicit observation-normalization warnings can appear on stderr; see the [workflow](settings.md#default-workflow-permissions), [commit-message](settings.md#commit-message-observation-normalization), [creation-policy](settings.md#creation-policy-observation-normalization), [public-fork](settings.md#fork-pr-workflow-policies) [Security](settings.md#secret-scanning-and-push-protection) and [Pages](settings.md#pages-configuration) field semantics. Export changes no GitHub settings. Failures exit 1 and report diagnostics on stderr, with no stdout. Read the saved file after success; do not redirect export stdout, because the shell can truncate the destination before the command runs:

```sh
gh settings export
cat .github/settings.yml
```

## Validate behavior

Validate reads `.github/settings.yml` and checks the [configuration contract](settings.md#configuration-semantics). It requires one YAML root mapping with the exact string `version: daiksud/gh-settings@v0`, accepts partial UI declarations and empty repository/settings mappings, and rejects unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple documents, and wrong shapes or value types. Alias and merge values are checked after YAML resolves them. Superseded envelopes, scalar permission sections, flat companions and API-shaped aliases are rejected without compatibility or migration. Public page/section paths are defined by the current-source reference; published packages keep their own contracts.

Success is silent and exits 0. Failures exit 1 and report a diagnostic on stderr, with no stdout. Validation requires a Git checkout but no GitHub origin or authentication. It makes no API requests and changes neither the file nor GitHub settings. Passing local validation does not guarantee that GitHub will accept an update.

## Immutable release policy commands

`export --security immutable-releases` independently includes the repository
[immutable release policy](settings.md#immutable-release-policy), requiring
Administration read. Omitted selection adds no policy request. GET 404 is
interpreted as disabled under the required-permissions precondition, without
supplying owner-enforcement metadata or certifying actual credential grants.
Other failures remain errors and preserve the export destination. Owner
enforcement is never exported as intent.

Plan compares only declared values without writes and rejects an owner-enforced
disable. Apply follows private vulnerability reporting, uses fresh preflight,
body-free PUT/DELETE, exact 204 and fresh readback; matching state is a no-op.
Administration write is required. GET 404 readback uses the same endpoint
contract, with owner enforcement still unknown.
Write/readback failures can leave earlier changes applied; no retry or rollback
occurs. Individual releases, tags and assets are outside this command's scope.

## Plan behavior

Plan validates the configuration before resolving origin, configuring authentication or reading GitHub. A valid declaration with no managed resources prints `No managed repository settings.` and exits 0 locally without an origin, credentials or API calls. `repository.about.topics: []` remains a managed clearing declaration. In current source, `labels: []` selects no objects and requires no label comparison.

Plan reads only declared resources, in repository-then-topics-then-labels order, completing every page of a declared label collection. It uses the same target resolution and response handling as export. Declared values and collection membership are compared; false, empty strings and empty topic/label lists remain explicit values. The GitHub client may sanitize some API control characters to caret notation before comparison.

About and General comparisons retain the existing stable field order, followed by Topics and selected-object operations. Settings differences use the current UI declaration paths and choices:

```text
repository.settings.general.features.issues: true → false
repository.about.topics: ["old"] → ["github-cli","settings-as-code"]
```

One exact UI menu choice can produce two internal component comparisons. An out-of-UI merge/squash pair instead retains one warning and menu-level write-required decision with both raw strings, even if the declared choice equals its projected default. Both exact and normalized compound choices render as a single UI menu decision. Normalized Actions execution uses one UI policy decision at `repository.settings.actions.general.actions_permissions.policy` with visible-only projection warnings; an observed disabled repository remains known Disable and keeps latent raw policy private. Workflow defaults use `repository.settings.actions.general.workflow_permissions.default_permissions`; Secret Protection uses its `enabled` and `push_protection` children. Native comparison identifiers remain internal and are not YAML aliases. Actions execution/workflow permissions, public-fork approval, Security status strings, creation policies, merge/squash menu pairs, complete Pages modes and legacy folders, Ruleset enforcement/Required approvals/Allowed merge methods/scanning Alerts/scanning Security alerts/bypass modes and Environment deployment modes have explicit normalization policies; other acquired values require their documented complete typed representations.

Strings are double-quoted with Go string escaping, including quotes, backslashes, newlines and control characters. Booleans use `true` or `false`; topics use sorted JSON arrays. Equal exact managed values are omitted; normalized projections retain their explicit write-required decisions. When all managed values match, output is `No differences for managed repository settings.`

Label creates and updates appear in stable identity order, with no implicit deletes. Names and attribute values are quoted and escaped. For example:

```text
repository.labels: create "bug" color="#d73a4a" description=""
repository.labels: update "help wanted" (id 123) description: "Old description" → ""
```

A nonempty label selection counts as one managed item; an empty selection manages nothing. The effective whole supported object is compared, including default-derived description clearing.

Ruleset operations show only selected whole supported objects, with UI control names, numeric IDs and fixed-default changes. Environment objects show requested fields of the effective before policy and the actual request; omitted fields remain API-controlled and are not shown as managed values. Unlisted Rulesets never appear as deletes. ID updates can rename; ID-less collisions fail instead of adopting existing names. Normalized enforcement, Required approvals, Allowed merge methods, scanning Alerts and Security alerts (each in tool-name order), bypass modes add independent contextual warnings and UI decisions marked normalized/write-required, followed by one complete effective policy. Allowed merge methods warnings retain original list order, while desired choices use named UI order. Both scanning threshold warnings retain exact tool names. Bypass warnings identify the selected UI actor and its ID when applicable. removing a scanning tool shows an inactive/tool-removed decision; removing a bypass actor shows an inactive/actor-removed decision. Each requires confirmed absence. Fallback-equal selected intent stays actionable; unlisted normalized IDs remain undisclosed. See the [raw enforcement](settings.md#ruleset-enforcement-observation-normalization), [Required approvals](settings.md#ruleset-required-approvals-observation-normalization), [Allowed merge methods](settings.md#ruleset-allowed-merge-methods-observation-normalization), [scanning Alerts](settings.md#ruleset-scanning-alerts-observation-normalization), [scanning Security alerts](settings.md#ruleset-scanning-security-alerts-observation-normalization), [bypass-mode](settings.md#ruleset-bypass-mode-observation-normalization) contracts for strict mutation response/fresh GET verification. A projected fallback never substitutes for exact raw confirmation.

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding and output errors exit 1 with a diagnostic on stderr. Read and partial-apply errors identify the affected UI resource where known. Failures before output leave stdout empty. Plan changes neither configuration nor GitHub settings. Its output is an observation, not a saved execution plan or drift protection for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) separately requires target/commit metadata and explicit plan/apply tokens, including for empty declarations.

## Prune option

Use `gh settings plan --prune labels,rulesets` and
`gh settings apply --prune labels,rulesets`. This adds deletions without limiting
other declared work. `--no-writeback` can be combined with apply. One
comma-separated flag is allowed; repeats, duplicates, unknown names and empty
elements fail before target resolution. `all` must stand alone and is currently
rejected because later scopes, including undecided Environment deletion, are not
implemented. See the [complete scope/presence and recovery contract](settings.md#collection-pruning).

## Apply behavior

Workflow permissions are applied after execution policy as a separate operation. Partial declarations send only their own keys; fresh complete readback verifies desired and preserved values. Failure retains earlier execution/resource writes and does not broaden existing credentials. See the [workflow-permission contract](settings.md#default-workflow-permissions).

Actions policy is applied after existing repository/topics/labels/Rulesets operations. It uses fresh preflight and readback, preserves API-required and selected companion values, and enables a currently disabled repository only after any requested selected-list update. Endpoints are not transactional; see the [Actions policy ordering and failure contract](settings.md#actions-execution-policy).

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no resources are managed, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `repository.about.topics: []`, performs an update. A nonempty label selection reads current labels and preflights ID-selected object updates and ID-less creates. `labels: []` selects nothing.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `repository.about` and `repository.settings.general` scalars | Declared REST members use repository PATCH.[^github-update-repository] Declared merge/squash message menus additionally require selective complete GraphQL preflight and fresh raw readback; failed verification can retain other members in that PATCH. PR access, Issue permissions and Sponsorships share identity-checked, declared-only GraphQL updates; declared creation policies also use selective preflight and fresh same-ID raw readback. Earlier REST success remains partial progress if the later access substep fails; GraphQL-only intent skips PATCH. |
| `repository.about.topics` | Topics PUT with the complete sorted set; `[]` clears it. |
| `repository.labels` | Complete label reads, then required ID-selected PATCH or create-only POST.[^github-labels] |
| `repository.settings.rulesets` | Complete supported reads, then selected-object PUT/create-only POST; unlisted deletion requires `--prune rulesets`. |
| `repository.settings.environments` | Stable name-order upserts, then explicit typed pattern reconciliation; omitted built-ins are API-controlled. |
| `repository.settings.environments[].variables` | After selected Environment policies, fresh selected-name GET and conditional POST/PATCH with exact GET verification. Variable-only intent sends no policy PUT. |
| `repository.settings.actions.variables` | Stable name-order fresh individual GET, conditional POST or value-only PATCH, then exact GET verification. No unlisted deletion. |
| `repository.settings.agents.variables` | After Actions variables, the same selected-name upsert and verification using only Agents endpoints. No unlisted deletion. |
| `repository.settings.actions.general` | Execution policy, workflow permissions, retention, cache retention, cache storage, then public-fork approval. Each uses its adopted preflight/readback contract. |
| `repository.settings.copilot.code_review.use_custom_instructions` | Fresh GET, conditional single-boolean PATCH and independent GET verification; no review activation or instruction-file changes. |
| `repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests` | Fresh complete GET, conditional MCP-only PATCH and independent GET verification of the toggle and unchanged sibling settings. |
| `repository.settings.copilot.code_review.review_effort_level` | Fresh effort/availability GET, conditional single-field PATCH and independent GET verification; Default uses observable configured-null comparison, not proof of hidden-override erasure. |
| `repository.settings.advanced_security` | Secret protection/push protection in dependency order, then private vulnerability reporting. |
| `repository.settings.general.releases` | Immutable release policy after private reporting and before Pages. |
| `repository.settings.pages` | Existing-site declared-only update with identity/visibility/readback checks. |

An explicit archive lifecycle surrounds these operations: unarchive first, final archive last. The [archive contract](settings.md#repository-archive-lifecycle) owns preflight and failure behavior.

The version and outer YAML mappings are not API fields. False and empty strings are sent explicitly; omitted repository fields remain outside update requests. Selected labels send complete supported values, including description defaults. Apply rejects redirects, including on its label reads; update origin when a repository moves.

Repository fields/topics retain their direct-write behavior. Nonempty label selections use fresh observations and stable create/update order. A valid ID fixes update identity; a name collision without an ID fails before label writes. Updates may rename the same object. Current source performs no implicit label deletion.

Apply never consumes a saved plan. Concurrent remote changes can affect or invalidate an update; fresh checks and response verification are not a lock. Earlier writes remain applied after a later failure, with no automatic rollback/retry.

Success prints `Applied repository settings.` for managed work, followed by created IDs when applicable. Normal CLI apply adds new IDs to the original settings file; `--no-writeback` or `GITHUB_ACTIONS=true` disables this. Existing IDs are not replaced, comments are preserved as practicable and formatting may normalize. A changed/unsafe file or persistence failure after remote creation is a nonzero partial failure that reports recoverable IDs. Retain those IDs before retrying; no Git commit or push is made. Output errors can occur after remote/local changes succeed; created label/Ruleset IDs remain in stderr recovery diagnostics. Rulesets use the same writeback/opt-out mechanism as labels.

## Failures and limits for mutation

Local validation cannot establish remote permissions or server-side acceptance. Permission and API failures return a nonzero status.[^github-update-repository]

Updates have no transaction, automatic retry or rollback. A failed command can leave repository settings or topics updated when a later label read or mutation fails, and earlier label changes can remain applied. After a failed/cancelled apply or output error, inspect current GitHub settings before deciding whether to retry; retrying resends declared repository/topic values and computes label operations from another fresh read.

Contributors can find the [behavior specifications](features/README.md), [testing guide](contributing/testing.md) and [internal Action interfaces](contributing/actions-internals.md) separately.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including repository label listing, create/update/delete endpoints and fine-grained token permissions.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.

## Fork workflow export choices

`gh settings export --fork-approval` adds the public contributor approval choice and requires Administration read. Ordinary export does not acquire it. It composes with the other supported export selections; duplicates fail before remote work. Private-only export flags are [unsupported](settings.md#excluded-private-actions-controls).

Plan and the credential-free Action Summary use the UI paths, booleans and approval choices. Matching apply skips these PUTs. A differing private partial declaration must explicitly include the required run flag; enabling run also requires the three companion choices. Incomplete/denied/inapplicable observations fail visibly, and direct-apply partial effects remain possible. See the [configuration contract](settings.md#fork-pr-workflow-policies) for exact values, applicability and failure behavior.

`export --pages` includes [Pages configuration](settings.md#pages-configuration) or explicit Branch none deletion intent for absence. It composes with other selections and rejects duplicates; ordinary export adds no Pages request. Other failed/incomplete reads preserve the destination. See [Pages commands](#pages-commands) for lifecycle behavior.

Actions workflow permissions, fork contributor approval, Security and Pages differences use their public YAML paths and UI choices on both sides. Observation-normalization warnings retain the raw value as provenance and do not establish raw-state equality.

## Pages commands

`export --pages` acquires [Pages configuration](settings.md#pages-configuration). Target GET 404 exports explicit `source: deploy_from_a_branch` and `branch: none`, without folder/domain/HTTPS. Applying that export intentionally deletes any site present at apply time. Other read failures preserve the destination; ordinary export makes no Pages request.

Plan remains read-only and displays creation, deletion or declared configuration differences. Apply creates absent sites only with complete explicit Source intent, directly deletes with the exact Branch none selector, and updates present sites without erasing omitted controls. Contradictory deletion and incomplete creation intent fail before Pages writes. A confirmed creation followed by optional domain/HTTPS failure remains applied and is reported as partial progress; no automatic rollback or write retry occurs. Site visibility and deployment readiness are not managed.

## Persistent pull request limits commands

`export --pull-request-limits` selects all three [persistent cap controls](settings.md#persistent-pull-request-limits),
including latent values while disabled. The flag composes with other export
options; duplicates fail before target/API/file work. Ordinary export makes no
cap request. Administrator access and complete typed acquisition are required;
failed export preserves the existing file. Plan remains read-only. Apply uses
fresh preflight, narrow PATCH and bounded independent confirmation, with honest
uncertain effects on readback failure. Temporary restrictions and automatic
exemption rules are not selected. The explicit Exemption list has its separate
selection described above.
