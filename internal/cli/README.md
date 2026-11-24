# Command-line interface

This package translates command-line arguments into operations and their results into terminal output and exit status. Start with [run.go](run.go) for argument parsing, help, command dispatch and diagnostics. The [command reference](../../docs/references/commands.md) describes the public behavior.

To change how a plan prints differences, start with [comparison.go](comparison.go). It arranges the terminal output and uses the [presentation package](../presentation/README.md) for text shared with the public Action.
