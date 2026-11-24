# Settings operations

This package runs the four settings operations shared by the CLI and Action. Start in [validate.go](validate.go), [export.go](export.go), [plan.go](plan.go) or [apply.go](apply.go) when changing which resources an operation visits or how a failure affects the remaining work.

Files named for a resource handle its part of the operation. The [reader and writer types](resources.go) define the calls each operation can make. For deletion across resources, follow [prune preflight](prune.go) before the writes. For a failure after a write, inspect the [apply result](apply_result.go) to see how progress reaches the caller. The [architecture guide](../../docs/development/architecture.md#shared-operation-flow) summarizes the sequence for each command.

The [archived-repository tests](archive_matching_test.go) use two small packages in [testdata](testdata/) to check how matching handles distinct Go types with identical fields.
