# Settings model and application terms

These terms describe values passed between Go packages. The [architecture guide](architecture.md#implementation-ownership) identifies the package that owns each boundary. The [settings reference](../references/settings/README.md) defines the public YAML.

## Intent and observation

| Term | Meaning in gh-settings | Definition |
| --- | --- | --- |
| Declaration | The values selected for management. Optional fields distinguish omission from explicit false, an empty string or a selected empty collection. Each resource defines which values are valid. | [`settings.Declaration`](../../internal/settings/settings.go) |
| UI model | The typed representation of the YAML grouped by GitHub UI page and control. The command package converts this to native declarations for comparison and API operations. | [`uimodel.Repository`](../../internal/settings/uimodel), [checkout conversion](../../internal/command/checkout_config.go) |
| Observation | Validated state acquired for a resource or group of fields. For example, an empty label inventory requires successful acquisition of every page. An unread group supplies no values. | [Repository observations](../../internal/settings/requested_observation.go), [label observations](../../internal/settings/label_observation.go) |
| Normalization | A resource-specific projection from an observed raw value to a supported UI choice. When the projection is lossy, the owning model retains the evidence needed to warn, compare and verify the requested value. | [Documented observation policies](../references/settings/configuration.md), [repository observations](../../internal/settings/requested_observation.go) |
| Managed collection | The collection's declaration and the command's prune selection determine ownership. An explicit empty topics list clears all topics. A present empty labels list selects no ordinary label changes, but permits pruning every acquired label when label pruning is requested. | [Topics declaration](../../internal/settings/settings.go), [prune scope](../../internal/settings/prune.go) |
| Selected identity | A resource's stable selector. For labels, an existing ID selects the object even across a rename. A declaration with no ID requests creation, and its name is checked for collisions. | [Label model](../../internal/settings/labels.go), [label contract](../references/settings/labels.md) |

## Comparison and progress

| Term | Meaning in gh-settings | Definition |
| --- | --- | --- |
| Comparison | The managed count and ordered, typed changes for a plan. Collections and individual controls have their own counting rules. A managed comparison can contain zero changes. | [`settings.Comparison`](../../internal/settings/comparison.go) |
| Plan result and error | A complete comparison and available target, or a typed failure stage with its diagnostic. A local unmanaged result needs no target. | [`app.PlanResult` and `app.PlanError`](../../internal/app/plan.go) |
| Prune candidate | A remote object selected for deletion by an explicit scope and a present collection. Global preflight fixes candidate identities before writes, and resource-specific checks decide whether deletion can proceed. | [Prune preparation](../../internal/app/prune.go), [collection pruning](../references/settings/pruning.md) |
| Apply progress | The known operations during direct reconciliation. Records distinguish a completed operation, unchanged state, a failed read, uncertain write effects and later known work left unexecuted. Confirmed writes can survive a later failure. | [`app.ApplyResult` and `app.ApplyOperation`](../../internal/app/apply_result.go) |
| Created ID | An identity returned during creation. A resource can surface an ID for recovery even when verification fails. Only confirmed IDs qualify for automatic recording in the declaration. | [Apply results](../../internal/app/apply_result.go), [ID writeback](../references/settings/labels.md#apply-id-writeback) |

## Action presentation

| Term | Meaning in gh-settings | Definition |
| --- | --- | --- |
| Plan report | A temporary, same-version JSON representation for rendering the Action's job summary. It contains an outcome, managed count and the applicable typed changes or failure category. Apply always reads the declaration. | [`actions.PlanReport`](../../internal/actions/plan.go) |
| Run metadata | The nonsecret identities of the workflow run, consumer checkout, Action and executing CLI used to present the result. The binary's version and digest identify what actually ran. | [Internal Action interfaces](actions-internals.md#source-actions-metadata) |
