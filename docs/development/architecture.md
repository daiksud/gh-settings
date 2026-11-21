# Architecture and design

One Go application serves the CLI and the public composite Action. It reads a declaration from the current Git checkout, identifies that checkout's GitHub.com repository and reconciles only the settings the declaration owns. The [settings reference](../references/settings/README.md) describes ownership for each resource. The [development policies](policies.md) explain how to assess a proposed addition.

## Implementation ownership

| Boundary | What it owns | Entry point |
| --- | --- | --- |
| Workspace | Git root, checkout-scoped origin, configuration path, strict YAML decoding, guarded file replacement and created-ID recording | [Checkout](../../internal/workspace/checkout.go), [codec](../../internal/workspace/config.go), [files](../../internal/workspace/config_file.go) |
| UI model | Configuration names and choices as they appear in YAML | [UI types](../../internal/settings/uimodel) |
| Settings model | Declarations, acquired observations, resource identity, comparisons, and the rules for values omitted from an update | [Declaration](../../internal/settings/settings.go), [observations](../../internal/settings/requested_observation.go), [comparison](../../internal/settings/comparison.go) |
| GitHub adapter | Mapping between UI and native values, REST/GraphQL transport, pagination, fresh preflight, exact request payloads and readback | [UI-to-native conversion](../../internal/github/ui_declaration.go), [GitHub adapters](../../internal/github) |
| Application | Local validation, resource selection, operation ordering, prune preflight and recording confirmed or uncertain effects | [Application operations](../../internal/app), [readers and writers](../../internal/app/resources.go) |
| Presentation | Shared text and UI choices, terminal diagnostics and exit status, typed Action reports and credential-free Summary rendering | [Shared presentation](../../internal/presentation), [CLI](../../internal/cli/run.go), [Actions](../../internal/actions/run.go) |
| Composite Action | Verified CLI installation, input and credential scope, step status, report cleanup and outputs | [Action metadata and steps](../../action.yml), [installer](../../action/install-cli.sh) |

[Root composition](../../main.go) passes a checkout and concrete reader/writer functions into the application. Constructing a checkout does not read a file, resolve an origin or initialize GitHub authentication. The workspace layer decodes UI-oriented data. [Checkout adapters](../../checkout_config.go) call `github.DeclarationFromUI` to form native intent, and `github.UIFromExportDeclaration` to project an export before saving. This keeps GitHub API conventions outside the file codec. The [glossary](glossary.md) explains the types that travel between these layers.

## Shared operation flow

| Operation | Sequence |
| --- | --- |
| [Validate](../../internal/app/validate.go) | Load and check the declaration locally. Validation needs no target repository or credentials. |
| [Plan](../../internal/app/plan.go) | Validate the prune selection and declaration. If there is ordinary managed intent or a present collection selected for prune, resolve the target and acquire the required observations. Return the complete typed comparison or an error identifying the failed stage. An unmanaged declaration succeeds locally. |
| [Apply](../../internal/app/apply.go) | Validate and determine managed intent as for plan. Resolve the target, preflight all selected prune candidates before any write, then reconcile against fresh observations. An explicit unarchive precedes ordinary writes. Each resource performs its updates and any permitted deletions in its own sequence. A requested archive is last, after a fresh identity and state check. |
| [Export](../../internal/app/export.go) | Resolve the target and snapshot the destination, validating any prior declaration before API reads. Attempt supported families independently. A completely acquired family replaces its prior intent. For an unavailable family, retain the prior declaration if present and emit a bounded diagnostic. Save the completed projection only after required diagnostics have been delivered and the destination check succeeds. |

Plan and apply treat a present empty collection according to its ownership. For example, `topics: []` selects an empty set. An empty labels list manages nothing in an ordinary apply and selects the whole acquired inventory for explicit label pruning. [Prune scope selection](../../internal/settings/prune.go) is part of the no-op decision. Apply consumes the declaration directly and can encounter different remote state from an earlier plan. See [collection pruning](../references/settings/pruning.md) and the [apply contract](../references/commands.md#apply).

## Following a setting through the code

### Load and observe

A repository setting may map to several native fields. A merge-message menu, for example, supplies both a title and a message option. [Requested observation routing](../../internal/settings/requested_observation.go) groups scalar fields by the complete state needed for comparison. The GitHub adapter chooses the corresponding REST or GraphQL reads, and export can retain independently unavailable groups. A missing group cannot supply zero values for comparison.

The [merge settings reader](../../internal/github/merge_methods.go) uses a fixed GraphQL query with the checkout's owner and repository name. Missing or null required members and GraphQL errors fail the group. A [hosted read-only experiment](https://github.com/daiksud/gh-settings/actions/runs/39086857913) verified the original three merge-method booleans using a Contents-read token. For the permissions of today's fields and commands, use the [permission matrix](../references/settings/permissions.md#general-about-and-labels).

### Reconcile and verify

Put ownership and comparisons in the settings model, transport and endpoint-specific checks in the GitHub adapter, and multi-resource ordering in the application. Resource identity determines which observations can justify a write.

- **ID-selected objects.** [Label declarations and comparisons](../../internal/settings/labels.go) distinguish a create-only name from a selected numeric ID. A complete paged [observation](../../internal/settings/label_observation.go) detects collisions and distinguishes an empty inventory from a missing read. An ID-selected update owns the supported label object, including the empty default for omitted description. The [GitHub writer](../../internal/github/label_operations.go) verifies identity and attributes. [Label scenarios](features/labels.feature.md) map rename, collisions, readback and writeback to tests.
- **Partially owned objects.** A named Environment can declare policy, non-sensitive variables or both. Existing-object updates combine original intent with freshly observed omitted companions. Creation leaves omitted values to the API. Deployment patterns and protection Apps have their own ownership and verification. Variables follow their parent policy and retain their Environment identity. See the [Environment scenarios](features/environments.feature.md) and [Environment prune preflight](../../internal/app/environment_prune.go).
- **Conditional state.** Rulesets, classic branch protection, OIDC templates and Actions execution policy have controls whose active representation depends on another setting. Adapters can need raw values for preservation, identity or verification even when a UI control is inactive. Keep those values separate from the declared change passed to presentation. The [Ruleset](features/rulesets.feature.md), [classic protection](features/classic-protection.feature.md), [OIDC](features/oidc-subject.feature.md) and [execution-policy](features/actions-execution-policy.feature.md) scenarios describe their different preconditions.
- **Lifecycle and pruning.** [Global prune preparation](../../internal/app/prune.go) fixes the candidates from complete inventories. Resources perform their own identity and drift checks before deletion and confirm absence afterward. A deletion in one resource may precede an update in a later resource. Webhook snapshots, collaborator invitation dependencies and Environment child inventories contain private evidence that must stay outside reports. Consult the [pruning scenarios](features/prune.feature.md), [webhook scenarios](features/webhook-prune.feature.md) and [collaborator scenarios](features/repository-access.feature.md) for their exact limits.

The [built-product tests](testing.md#local-process-acceptance) check the complete path through the actual CLI. Use the [feature index](features/README.md) to find the more detailed tests that own the contract for a specific resource.

### Report progress and persist confirmed IDs

[Apply results](../../internal/app/apply_result.go) record the target, ordered operations and any created identities. A successful no-op, a pre-write read failure, a confirmed write and an attempted write with an unknown effect require different reporting. If a later operation fails, earlier confirmed writes remain applied. [Progress tests](../../internal/app/apply_progress_test.go) exercise these distinctions.

When a resource creates an object with a verified ID, the CLI can record it through the workspace's snapshot-bound recorder. A local writeback failure can follow a successful creation. Report the confirmed identity for recovery before another apply of the ID-less entry. The Action disables local ID writeback. See [ID writeback](../references/settings/labels.md#apply-id-writeback) for the file guards and [this repository's recovery procedure](repository-automation.md#created-resource-ids) for its workflow.

### Render the same result for the CLI and Action

The CLI owns terminal output and exit status. The composite Action executes the verified CLI and, for plan, passes a temporary typed report to a separate credential-free Summary step. Summary validates the report and metadata before opening the output destination. Shared presentation converts the native values into UI paths and choices. An unavailable observation, a hidden status-only value or a projected fallback never becomes apply input. The [internal Action interfaces](actions-internals.md) define the report, validation and output boundaries.

## Design decisions

The [architecture decision records](adr/README.md) retain the original reasoning and identify superseded parts. The [Go module](../../go.mod) records the selected Go, go-gh and YAML library versions. Design influences include [gh-infra](https://github.com/babarot/gh-infra), [noirbizarre/gh-settings](https://github.com/noirbizarre/gh-settings) and the [GitHub Terraform provider](https://github.com/integrations/terraform-provider-github).
