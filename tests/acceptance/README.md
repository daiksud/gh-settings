# CLI acceptance tests

These tests run an already-built `gh-settings` executable in temporary Git checkouts. The [API fixture](fixture.go) serves GitHub responses over a local Unix socket. The [process helper](process.go) captures stdout, stderr and the exit result from each command.

Add a case alongside the setting or command it exercises. The [apply tests](apply_test.go) follow an edited declaration through validation and plan, then check that apply skips a write when the remote value already matches. The [local process acceptance guide](../../docs/development/testing.md#local-process-acceptance) explains how to supply the binary and run the suite.
