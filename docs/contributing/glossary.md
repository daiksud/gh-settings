# Settings model and application terms

These terms describe the shared Go model. [Architecture](architecture.md#implementation-ownership) explains the code boundaries; the [settings reference](../settings.md) defines user-facing YAML.

| Term | Meaning | Go type |
| --- | --- | --- |
| Settings declaration | The repository and topics values selected for management; all omitted resources are unmanaged | `settings.Declaration` |
| Repository declaration | Optional fields chosen by the maintainer; nil means omitted, while false and empty strings remain explicit values | `settings.RepositoryDeclaration` |
| Topics declaration | A nil names pointer is unmanaged; an explicitly empty list clears the topic set | `settings.TopicsDeclaration` |
| Observed repository state | All supported values from a complete successful repository read, with permitted null strings normalized to empty strings | `settings.RepositoryState` |
| Observed topics | Topic names read from GitHub and compared as a set with stable sorted output | `[]string` |
| Empty declaration | No repository fields and no topics names are managed; this differs from explicitly declaring an empty topic set | `Declaration.Empty` |
| Comparison | Managed-field count and ordered changes; a positive count with no changes differs from no managed fields | `settings.Comparison` |
| Change | A resource-qualified field path and native before/after values: string, boolean, or topic list | `settings.Change` |
| Plan result/error | A comparison and available target identity; typed failure stages preserve the underlying diagnostic | `app.PlanResult`, `app.PlanError` |
| Apply result | `Applied: true` means all required writes completed; false without error is a local no-op, while false with an error does not establish the remote outcome | `app.ApplyResult` |
| Plan report | Same-version ephemeral JSON for Actions presentation, never input to apply | `actions.PlanReport`, `actions.PlanChange` |
| Run metadata | Explicit nonsecret run, workflow, consumer, Action, and CLI identity context kept separate from plan data | `actions.RunMetadata` |

The [model](../../internal/settings/) has no YAML/JSON mappings, I/O, or presentation responsibility. [Application operations](../../internal/app/) compose those model values with workspace and GitHub boundaries. The [Actions internals reference](actions-internals.md) defines the report and metadata interfaces.
