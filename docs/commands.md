# Settings commands

## Environment export

Ordinary `export` observes all supported named Environment policies and independently inventories their variables. Unavailable policy or variable observations emit info and retain prior intent; one Environment variable failure does not stop another.

`validate`, `plan` and `apply` accept the [Environment configuration contract](settings.md#environments); policy-selected names select ordinary create-or-update, not numeric IDs; [variable-only entries](#environment-variable-commands) do not upsert policy. No Environment writeback occurs, regardless of `--no-writeback` or GitHub Actions mode. Unlisted objects remain untouched. Name-only entries send an empty upsert body. Omitted built-in values remain API-controlled; the client promises neither preservation nor reset. Inspect plan before apply.

Environment deployment-mode observations follow the [raw-pair contract](settings.md#environment-deployment-mode-observation-normalization). Export warns only after complete selected acquisition, in folded-name order. Detailed plan/Summary label all Environment before-objects as effective policy and add a scoped normalized/write-required UI decision only for explicit mode intent. Status-only hides names and pair values. Omitted mode sends no field, retains API-controlled outcomes and emits no unowned warning; explicit mode requires canonical raw confirmation, while unknown-mode pattern intent requires explicit Selected.

## Scope and status

The four settings commands manage the GitHub.com repository containing the current Git checkout. Configuration lives at that checkout's root `.github/settings.yml`, including when commands run from a subdirectory. The [settings reference](settings.md) defines repository fields, topics, labels and configuration semantics.

```text
gh settings export
gh settings validate
gh settings plan
gh settings apply [--no-writeback]
```

| Command | Contract |
| --- | --- |
| `export` | Read all supported readable settings and save `.github/settings.yml`. Existing configuration is not required. |
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

The [topics contract](settings.md#topics) defines names, omission and explicit clearing. Export includes the observed topics; plan compares the managed set; apply reads fresh membership and replaces that set only when it differs, then verifies it independently. A configuration containing only `repository.about.topics` is valid managed intent. Omitted topics skip the resource; `[]` clears it and `{}` is invalid. The command is a local no-op only when no other resource is managed.

## Labels command behavior

The [labels contract](settings.md#labels) uses ID-based listed-object ownership: ordinary apply manages only listed objects; [explicit collection pruning](settings.md#collection-pruning) adds deletions for present lists. An ID selects an existing label; no ID means create-only and a name collision reports the existing ID.

Plan shows creates and ID-bound updates, including default-derived values; `--prune labels` also shows unlisted deletions and their Issue/PR assignment loss. Color is required; omitted description defaults to empty on create and update. A supported rename keeps the ID. Apply skips matching objects and independently reads each created/updated label at its resulting name to verify the same ID and complete supported settings before proceeding. A failed readback stops later writes, including prune deletions, with uncertain effects.

Apply records confirmed new IDs in the original YAML after creation verification;
`--no-writeback` or `GITHUB_ACTIONS=true` disables this. A positive ID from a create
response remains visible for recovery even if verification fails, but is only
automatically recorded after successful independent readback. Earlier confirmed
IDs remain available after later remote or local writeback failure. See
[Label ownership and ID writeback](settings.md#labels). There is no automatic
commit, push, create retry or rollback.

## Actions event policy commands

Ordinary `export` observes the complete [Actions event/actor policy collection](settings.md#actions-event-policies), including IDs and explicit workflow include/exclude selections, in stable order. Every page/detail must succeed; otherwise info reports the family and its valid existing declaration is retained.

`validate` checks the complete listed objects locally. Without prune, omission
and `[]` make no policy requests. `--prune actions-policies` adds unlisted
deletions for a present list; omission remains unmanaged and [] selects all.
Plan reports creates, ID-bound updates and requested deletions without writing.
Prune validates every supported candidate before any remote write, performs
creates/updates first, rechecks exact state before DELETE and confirms absence
in a complete fresh inventory. Deleting a policy removes its workflow restriction.
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

Ordinary `export` attempts this supported setting automatically. Complete readable values replace prior intent; unavailable observations emit bounded info and preserve prior declarations while independent reads continue. See [automatic export](#export-behavior) for scope, omission and file-safety rules.

Plan shows the native boolean comparison. Apply reads fresh state, skips a match and verifies narrow PATCH updates. The flag stores configuration without enabling or requesting reviews and does not read or edit instruction files. API permission and review-entitlement boundaries remain separate; this option does not expand either.

## Copilot MCP tools commands

Ordinary `export` attempts this supported setting automatically. Complete readable values replace prior intent; unavailable observations emit bounded info and preserve prior declarations while independent reads continue. See [automatic export](#export-behavior) for scope, omission and file-safety rules.

Plan shows native booleans at `repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests`. Apply reads fresh complete settings, skips matches, PATCHes only `mcp_enabled` and independently verifies the requested boolean and unchanged sibling settings. Failures retain honest partial or unknown-effect reporting. This control does not configure MCP servers or secrets, grant access, request reviews or establish MCP execution.

## Copilot review effort commands

Ordinary `export` attempts this supported setting automatically. Complete readable values replace prior intent; unavailable observations emit bounded info and preserve prior declarations while independent reads continue. See [automatic export](#export-behavior) for scope, omission and file-safety rules.

Plan and detailed Action reports show `default`, `lite`, `balanced` or `max` at `repository.settings.copilot.code_review.review_effort_level`. Default is an observable configured-null state, not proof that GitHub erased an unavailable saved override. Apply reads fresh availability, skips observable matches, sends only the declared effort when needed and independently verifies readback. Unsupported or unavailable observations are errors, never successful fallback. This command neither enables nor requests Copilot reviews.

## Pull request limit exemptions

Ordinary `export` attempts this supported setting automatically. Complete readable values replace prior intent; unavailable observations emit bounded info and preserve prior declarations while independent reads continue. See [automatic export](#export-behavior) for scope, omission and file-safety rules.

An explicitly declared `exemption_list` owns complete membership; `[]` clears it,
while omission makes no list request. Apply removes obsolete users before adding
new users and verifies each stage; later failures do not roll back confirmed
removals. It runs before separately selected cap controls. Detailed output
contains membership, while status-only Summary omits it. No temporary restriction,
automatic exemption rule or collaborator grant is managed.

## Environment variable commands

Ordinary `export` inventories all supported Actions, Agents and Environment variables through their complete paginated list APIs. Values are plaintext in the output file; secret APIs are never used. An incomplete or unsupported inventory emits info and retains its prior valid declaration. Environment names must be acquired completely, then variable inventories proceed independently for each parent. See [automatic export](#export-behavior).

Variable-only declarations do not upsert an Environment or touch its policy, patterns or secrets. A missing Environment fails. Combined policy/variable declarations apply selected policy first, then variables in stable Environment/name order, before repository Actions and Agents variables. Earlier successful writes remain applied on later failure. Values are plaintext in detailed output; status-only Summary hides details. No bulk selection, deletion, rename or ID writeback is supported.

## Repository variables commands

Ordinary `export` inventories all supported Actions, Agents and Environment variables through their complete paginated list APIs. Values are plaintext in the output file; secret APIs are never used. An incomplete or unsupported inventory emits info and retains its prior valid declaration. Environment names must be acquired completely, then variable inventories proceed independently for each parent. See [automatic export](#export-behavior).

Plan is read-only and identifies the selected family. Ordinary apply upserts only declared names, preserving unlisted variables and skipping equal values. Explicit [variable prune scopes](settings.md#variable-pruning) add unlisted deletions for present lists. Actions variables are processed before Agents variables, with stable name order within each family. Earlier successful writes remain applied after a later failure. Values are plaintext and appear in configuration, detailed plan/report and Summary; never use these features for secrets. Actions variables require the repository **Variables** read/write permission; Agents variables require the separate **Agent variables** read/write permission. Neither grants access to the other family.

## Export behavior

Run `gh settings export` without selection flags. It reads supported About/General,
Topics, Labels, Rulesets, Actions execution/workflow/cache/retention/contributor
approval settings, Actions event/actor policies, Environments and their variables,
Actions/Agents repository variables, Copilot review controls, pull request limits
and exemptions, Pages, archive state and supported Security controls.
Product-wide security switches, private-only excluded controls, organization
settings and secret values remain outside the target. Unimplemented controls remain
unavailable regardless of Preview status.

Complete readable values replace prior declarations, including false and valid
empty strings/lists. An independent unavailable, incomplete or unsupported family
emits `info: export` on stderr and acquisition continues. First export omits it;
re-export retains its valid prior declaration where compatible with acquired
settings. Retained values are unverified intent. Failed pagination never supplies
an empty or partial inventory. Variable inventories include plaintext values, never secret values; protect the output.
Existing variable schema restrictions still apply: an unrepresentable inventory
is reported and retained, not rewritten into plausible values.

Environment names are listed completely independently of policy details. A
failed variable inventory preserves that Environment's prior variables and does
not stop other Environments. Complete absence can remove stale observed entries;
unknown never implies absence. CodeQL state and details share one response:
confirmed `not-configured` exports false and removes incompatible old details
with info; unknown state retains intent. Existing API-specific 404 and explicit
normalization policies remain unchanged. Plan/apply remain strict.

The old export selection options are removed under the v0.x policy and report
that supported settings are observed automatically. This is a breaking CLI and
export-scope change. Export does not apply settings. In particular, the existing
Pages-absence declaration can delete a later site when subsequently applied;
complete exported collections also become explicit ownership input to prune.
Inspect the YAML before plan/apply.

Checkout/origin target resolution remains required. About/basic General, merge
controls, PR access, Issue permissions and Sponsorships are independent groups;
an unavailable group does not stop the others. Each group still requires complete
observations before adoption. Confirmed disabled Issues or Pull requests suppress
incompatible retained creation permissions; an unknown gate remains unknown.
Partial or absent About/General is valid output, even when only later families
are readable. Successful file production does not prove remote repository
existence or access when all remote observations are unavailable.

Invalid or unreadable existing files, family-info or normalization-warning output
failures and destination write failures remain fatal. Failed diagnostic delivery
stops export before saving: an initial destination stays absent and an existing
file keeps its exact bytes. Read-only acquisition may continue after the output
failure; earlier diagnostics may already have been printed. Existing files are
validated and snapshotted before
API reads; guarded atomic replacement rejects changed/removed/symlink destinations and a file that
appears during initial export. Initial file IO failure can leave a partial new
file; this is not a concurrent-directory lock.

Success exits0 with no stdout. Read `.github/settings.yml` after success; do not
redirect stdout to that path. Info gives only bounded family/HTTP/network reasons,
never upstream messages, response bodies, URLs, variable values or credentials.
See the [behavior specification](features/readable-settings-export.feature.md).

```sh
gh settings export
gh settings validate
```

## Validate behavior

Validate reads `.github/settings.yml` and checks the [configuration contract](settings.md#configuration-semantics). It requires one YAML root mapping with the exact string `version: daiksud/gh-settings@v0`, accepts partial UI declarations and empty repository/settings mappings, and rejects unsupported versions, unknown or duplicate keys, non-string keys, malformed or multiple documents, and wrong shapes or value types. Alias and merge values are checked after YAML resolves them. Superseded envelopes, scalar permission sections, flat companions and API-shaped aliases are rejected without compatibility or migration. Public page/section paths are defined by the current-source reference; published packages keep their own contracts.

Success is silent and exits 0. Failures exit 1 and report a diagnostic on stderr, with no stdout. Validation requires a Git checkout but no GitHub origin or authentication. It makes no API requests and changes neither the file nor GitHub settings. Passing local validation does not guarantee that GitHub will accept an update.

## Immutable release policy commands

Ordinary `export` automatically attempts the repository
[immutable release policy](settings.md#immutable-release-policy), requiring
Administration read. Unavailable export acquisition emits info and retains prior
intent if present. GET 404 is
interpreted as disabled under the required-permissions precondition, without
supplying owner-enforcement metadata or certifying actual credential grants.
Other read failures remain errors for plan/apply and unavailable observations for
export. Owner
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

Ruleset operations show only selected whole supported objects, with UI control names, numeric IDs and fixed-default changes. Environment objects show requested fields of the effective before policy and the actual request; omitted fields remain API-controlled and are not shown as managed values. Unlisted Rulesets appear as deletes only with explicit `--prune rulesets`. ID updates can rename; ID-less collisions fail instead of adopting existing names. Normalized enforcement, Required approvals, Allowed merge methods, scanning Alerts and Security alerts (each in tool-name order), bypass modes add independent contextual warnings and UI decisions marked normalized/write-required, followed by one complete effective policy. Allowed merge methods warnings retain original list order, while desired choices use named UI order. Both scanning threshold warnings retain exact tool names. Bypass warnings identify the selected UI actor and its ID when applicable. removing a scanning tool shows an inactive/tool-removed decision; removing a bypass actor shows an inactive/actor-removed decision. Each requires confirmed absence. Fallback-equal selected intent stays actionable; unlisted normalized IDs remain undisclosed. See the [raw enforcement](settings.md#ruleset-enforcement-observation-normalization), [Required approvals](settings.md#ruleset-required-approvals-observation-normalization), [Allowed merge methods](settings.md#ruleset-allowed-merge-methods-observation-normalization), [scanning Alerts](settings.md#ruleset-scanning-alerts-observation-normalization), [scanning Security alerts](settings.md#ruleset-scanning-security-alerts-observation-normalization), [bypass-mode](settings.md#ruleset-bypass-mode-observation-normalization) contracts for strict mutation response/fresh GET verification. A projected fallback never substitutes for exact raw confirmation.

Successful comparison exits 0 whether differences exist or not. File, validation, origin, authentication, API, decoding and output errors exit 1 with a diagnostic on stderr. Read and partial-apply errors identify the affected UI resource where known. Failures before output leave stdout empty. Plan changes neither configuration nor GitHub settings. Its output is an observation, not a saved execution plan or drift protection for apply.

The [public Action](github-actions.md#inputs-outputs-and-failures) separately requires target/commit metadata and explicit plan/apply tokens, including for empty declarations.

## Prune option

Use `gh settings plan --prune labels,rulesets` and
`gh settings apply --prune labels,rulesets`. This adds deletions without limiting
other declared work. `--no-writeback` can be combined with apply. One
comma-separated flag is allowed; repeats, duplicates, unknown names and empty
elements fail before target resolution. `all` must stand alone and is currently
rejected because Environment-object deletion is adopted but not yet
implemented. Actions Policies use `actions-policies`; variable pruning uses
`actions-variables`, `agents-variables` and
`environment-variables` independently. See the [complete scope/presence and recovery contract](settings.md#collection-pruning).

## Apply behavior

Workflow permissions are applied after execution policy as a separate operation. Partial declarations send only their own keys; fresh complete readback verifies desired and preserved values. Failure retains earlier execution/resource writes and does not broaden existing credentials. See the [workflow-permission contract](settings.md#default-workflow-permissions).

Actions policy is applied after existing repository/topics/labels/Rulesets operations. It uses fresh preflight and readback, preserves API-required and selected companion values, and enables a currently disabled repository only after any requested selected-list update. Endpoints are not transactional; see the [Actions policy ordering and failure contract](settings.md#actions-execution-policy).

Apply validates the configuration before resolving origin, configuring authentication or contacting GitHub. When no resources are managed, it prints `No managed repository settings.` and exits 0 locally. A topics-only declaration, including `repository.about.topics: []`, reads the current complete membership and writes only when it differs. A nonempty label selection reads current labels and preflights ID-selected object updates and ID-less creates. `labels: []` selects nothing.

Declared resources are sent in this order, skipping resources that are unmanaged:

| Resource | Request |
| --- | --- |
| `repository.about` and `repository.settings.general` scalars | REST-backed controls require fresh complete base/selected merge observations, skip exact matches and send only changed controls in one repository PATCH.[^github-update-repository] A changed UI menu sends its complete pair; normalized raw pairs remain write-required. Independent readback verifies all originally declared REST members, including initially matching members omitted from PATCH; failed verification can retain other members in that PATCH. PR access, Issue permissions and Sponsorships share identity-checked, declared-only GraphQL updates; declared creation policies also use selective preflight and fresh same-ID raw readback. Earlier REST success remains partial progress if the later access substep fails; GraphQL-only intent skips PATCH. |
| `repository.about.topics` | Fresh complete GET; skip matching membership, otherwise PUT the complete sorted set and verify through an independent GET. `[]` clears it. |
| `repository.labels` | Complete label reads, then required ID-selected PATCH or create-only POST, followed by independent same-ID/full-settings GET at the resulting name.[^github-labels] |
| `repository.settings.rulesets` | Complete supported reads, then selected-object PUT/create-only POST with independent ID/ownership/whole-policy GET verification. Failed verification retains reported creation IDs for recovery and stops later work; unlisted deletion requires `--prune rulesets`. |
| `repository.settings.environments` | Stable name-order upserts, then explicit typed pattern reconciliation; omitted built-ins are API-controlled. |
| `repository.settings.environments[].variables` | After selected Environment policies, fresh selected-name GET and conditional POST/PATCH with exact GET verification. Variable-only intent sends no policy PUT. |
| `repository.settings.actions.variables` | Stable name-order fresh individual GET, conditional POST or value-only PATCH, then exact GET verification. Unlisted deletion requires its explicit variable prune scope. |
| `repository.settings.agents.variables` | After Actions variables, the same selected-name upsert and verification using only Agents endpoints. Unlisted deletion requires its explicit variable prune scope. |
| `repository.settings.actions.general` | Execution policy, workflow permissions, retention, cache retention, cache storage, then public-fork approval. Each uses its adopted preflight/readback contract. |
| `repository.settings.copilot.code_review.use_custom_instructions` | Fresh GET, conditional single-boolean PATCH and independent GET verification; no review activation or instruction-file changes. |
| `repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests` | Fresh complete GET, conditional MCP-only PATCH and independent GET verification of the toggle and unchanged sibling settings. |
| `repository.settings.copilot.code_review.review_effort_level` | Fresh effort/availability GET, conditional single-field PATCH and independent GET verification; Default uses observable configured-null comparison, not proof of hidden-override erasure. |
| `repository.settings.advanced_security` | Secret protection/push protection in dependency order, then private vulnerability reporting. |
| `repository.settings.general.releases` | Immutable release policy after private reporting and before Pages. |
| `repository.settings.pages` | Existing-site declared-only update with identity/visibility/readback checks. |

An explicit archive lifecycle surrounds these operations: unarchive first, final archive last. The [archive contract](settings.md#repository-archive-lifecycle) owns preflight and failure behavior.

The version and outer YAML mappings are not API fields. False and empty strings are sent explicitly; omitted repository fields remain outside update requests. Selected labels send complete supported values, including description defaults. Apply rejects redirects, including on its label reads; update origin when a repository moves.

Repository REST controls reconcile against fresh state. Topics compare validated membership without regard to order; matching Topics perform no PUT. Missing, null, malformed, denied or redirected preflight responses stop before a Topics write. PUT errors and failed or mismatching readback stop later resources with uncertain write effects. Nonempty label selections use fresh observations and stable create/update order. A valid ID fixes update identity; a name collision without an ID fails before label writes. Updates may rename the same object. Current source performs no implicit label deletion.

For the nineteen members of the shared repository PATCH, base controls use the complete REST observation and merge controls use selected GraphQL booleans and owned menu pairs. An exact match needs no PATCH or second read. Changed scalars omit matching and undeclared members; changed menus send both declared components. Base-only intent does not query GraphQL. A post-write observation must confirm the original declared values and required raw menu companions. Default branch and Discussions also require matching mutation-response values when sent. An unavailable preflight fails before PATCH; mutation or readback failure may leave changes applied. Successful reads and no-ops are not reported as completed writes.

Apply never consumes a saved plan. Concurrent remote changes can affect or invalidate an update; fresh checks and response verification are not a lock. Earlier writes remain applied after a later failure, with no automatic rollback/retry.

Success prints `Applied repository settings.` for managed work, followed by created IDs when applicable. Normal CLI apply adds new IDs to the original settings file; `--no-writeback` or `GITHUB_ACTIONS=true` disables this. Existing IDs are not replaced, comments are preserved as practicable and formatting may normalize. A changed/unsafe file or persistence failure after remote creation is a nonzero partial failure that reports recoverable IDs. Retain those IDs before retrying; no Git commit or push is made. Output errors can occur after remote/local changes succeed; created label/Ruleset IDs remain in stderr recovery diagnostics. Rulesets use the same writeback/opt-out mechanism as labels.

## Failures and limits for mutation

Local validation cannot establish remote permissions or server-side acceptance. Permission and API failures return a nonzero status.[^github-update-repository]

Updates have no transaction, automatic retry or rollback. A failed command can leave repository settings or topics updated when a later label read or mutation fails, and earlier label changes can remain applied. After a failed/cancelled apply or output error, inspect current GitHub settings before deciding whether to retry; retrying reconciles REST-backed repository controls, Topics and Labels from fresh reads.

Contributors can find the [behavior specifications](features/README.md), [testing guide](contributing/testing.md) and [internal Action interfaces](contributing/actions-internals.md) separately.

[^github-labels]: [GitHub Labels REST API](https://docs.github.com/en/rest/issues/labels), including repository label listing, create/update/delete endpoints and fine-grained token permissions.

[^github-update-repository]: [GitHub REST API — Update a repository](https://docs.github.com/en/rest/repos/repos#update-a-repository), checked on 2026-10-11 for the update endpoint and failure responses. This reference does not expand the supported scope.

## Fork workflow export choices

Ordinary `export` attempts this supported setting automatically. Complete readable values replace prior intent; unavailable observations emit bounded info and preserve prior declarations while independent reads continue. See [automatic export](#export-behavior) for scope, omission and file-safety rules.

Plan and the credential-free Action Summary use the UI paths, booleans and approval choices. Matching apply skips these PUTs. A differing private partial declaration must explicitly include the required run flag; enabling run also requires the three companion choices. Incomplete/denied/inapplicable observations fail visibly, and direct-apply partial effects remain possible. See the [configuration contract](settings.md#fork-pr-workflow-policies) for exact values, applicability and failure behavior.

Ordinary `export` includes [Pages configuration](settings.md#pages-configuration). Under the existing required-permissions precondition, target GET 404 exports `source: deploy_from_a_branch` and `branch: none`. Applying that declaration later intentionally deletes any site then present. Other unavailable observations emit info and retain existing intent.

Actions workflow permissions, fork contributor approval, Security and Pages differences use their public YAML paths and UI choices on both sides. Observation-normalization warnings retain the raw value as provenance and do not establish raw-state equality.

## Pages commands

Ordinary `export` includes [Pages configuration](settings.md#pages-configuration). Under the existing required-permissions precondition, target GET 404 exports `source: deploy_from_a_branch` and `branch: none`. Applying that declaration later intentionally deletes any site then present. Other unavailable observations emit info and retain existing intent.

Plan remains read-only and displays creation, deletion or declared configuration differences. Apply creates absent sites only with complete explicit Source intent, directly deletes with the exact Branch none selector, and updates present sites without erasing omitted controls. Contradictory deletion and incomplete creation intent fail before Pages writes. A confirmed creation followed by optional domain/HTTPS failure remains applied and is reported as partial progress; no automatic rollback or write retry occurs. Site visibility and deployment readiness are not managed.

## Persistent pull request limits commands

Ordinary `export` attempts this supported setting automatically. Complete readable values replace prior intent; unavailable observations emit bounded info and preserve prior declarations while independent reads continue. See [automatic export](#export-behavior) for scope, omission and file-safety rules.
