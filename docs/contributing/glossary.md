# Settings model and application terms

These terms describe the shared Go model. [Architecture](architecture.md#implementation-ownership) explains the code boundaries; the [settings reference](../settings.md) defines user-facing YAML.

| Term | Meaning | Go type |
| --- | --- | --- |
| Settings declaration | The repository, topics, and labels values selected for management; all omitted resources are unmanaged | `settings.Declaration` |
| Repository declaration | Optional fields chosen by the maintainer; nil means omitted, while false and empty strings remain explicit values | `settings.RepositoryDeclaration` |
| Topics declaration | A nil names pointer is unmanaged; an explicitly empty list clears the topic set | `settings.TopicsDeclaration` |
| Labels declaration | A nil pointer is unmanaged; a present list owns the entire membership, with an empty list deleting all observed labels | `Declaration.Labels` as `*[]settings.LabelDeclaration` |
| Label declaration | A required identifying name plus optional color/description; omitted attributes of retained labels preserve remote values | `settings.LabelDeclaration` |
| Observed repository state | All supported values from a complete successful repository read, with permitted null strings normalized to empty strings | `settings.RepositoryState` |
| Observed topics | Topic names read from GitHub and compared as a set with stable sorted output | `[]string` |
| Observed labels | Complete name/color/description values from every successfully read page; nullable response descriptions become empty strings | `[]settings.LabelState` |
| Label identity | Unicode simple case folding without trimming or rename inference; ambiguous identities fail and case-only changes retain remote spelling | `settings.LabelIdentity` |
| Empty declaration | No repository fields, topics names, or labels collection are managed; explicitly empty topics/labels are still managed | `Declaration.Empty` |
| Comparison | Managed-field/collection count, native field changes, and separate label operations; each declared collection counts once, even when no differences exist | `settings.Comparison` |
| Change | A resource-qualified field path and native before/after values: string, boolean, or topic list | `settings.Change` |
| Label change | A typed create/update/delete operation with a name and applicable observed-before/declared-after values, never saved apply input | `settings.LabelChange` |
| Plan result/error | A comparison and available target identity; typed failure stages preserve the underlying diagnostic | `app.PlanResult`, `app.PlanError` |
| Apply result | Internal outcome, resolved target, and ordered known operation progress; distinguishes unmanaged, completed, matching labels, failure, and partial failure without claiming rollback or readback | `app.ApplyResult` |
| Apply operation progress | Completed, failed read, failed write with unknown remote effect, or known but unexecuted operation; individual label operations become known after complete fresh reads | `app.ApplyOperation` |
| Plan report | Same-version ephemeral JSON for Actions presentation, with field changes and optional typed label operations, never input to apply | `actions.PlanReport`, `actions.PlanChange`, `actions.LabelPlanChange` |
| Label report attributes | Optional color/description fields preserve the operation's attribute presence; complete before values are required where applicable | `actions.LabelAttributes` |
| Run metadata | Explicit nonsecret run, workflow, consumer, Action, and CLI identity context kept separate from plan data | `actions.RunMetadata` |

The [model](../../internal/settings/) has no YAML/JSON mappings, I/O, or presentation responsibility. [Application operations](../../internal/app/) compose those model values with workspace and GitHub boundaries. The [Actions internals reference](actions-internals.md) defines the report and metadata interfaces.
