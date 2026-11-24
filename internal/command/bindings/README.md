# GitHub client bindings

This package supplies GitHub implementations for the application's [reader and writer functions](../../app/resources.go). Start with [Reader and Writer](bindings.go) to find the client method for a resource. The functions they return construct the client when invoked, so commands that only validate a local file can run without GitHub credentials.

Look in the resource-specific files when a call needs more than a client method. For example, the [Actions bindings](actions.go) attach the normalization warning handler before applying a workflow permissions change. The [GitHub package](../../github/README.md) owns the request and readback.
