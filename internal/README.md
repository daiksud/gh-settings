# Go implementation

The CLI and the public composite Action use the same Go executable. To trace a command from its arguments to the settings it changes, start with the [executable entry point](../main.go) and the [shared operation flow](../docs/development/architecture.md#shared-operation-flow).

| Package | Where to make a change |
| --- | --- |
| [command](command/README.md) | Connect the executable to a checkout and supply operations to the CLI and Action. |
| [cli](cli/README.md) | Parse public commands and flags, print terminal output and choose the exit status. |
| [actions](actions/README.md) | Produce Action metadata and plan reports, or render the GitHub job summary. |
| [app](app/README.md) | Decide which settings to read or write and in what order. |
| [workspace](workspace/README.md) | Resolve the target from Git, decode settings YAML and save changes to the checkout. |
| [settings](settings/README.md) | Define what a declaration manages and compare it with observed state. |
| [github](github/README.md) | Map UI settings to GitHub operations, send requests and verify the result. |
| [presentation](presentation/README.md) | Format setting paths, choices and changes for both the terminal and the Action. |
