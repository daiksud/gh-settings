# GitHub Actions tests

This directory tests the public composite Action, example caller workflows and this repository's CI and release workflows.

- Check example triggers and permissions in the [caller workflow tests](workflow_test.go), and PR or main CI wiring in the [candidate workflow tests](candidate_workflow_test.go). The [composite harness](composite_test.go) executes the Action's shell steps in a separate consumer checkout and captures its outputs.
- When a result depends on the real CLI, add a `TestBuilt` case that uses the supplied release-version binary. The [export cases](composite_export_acceptance_test.go) show how to run the installer and assert what the caller sees.
- The [candidate download fixture](prepare-candidate-download.sh) serves CI's verified binary through a [curl stand-in](candidate-curl.sh). The [plan fixture](plan-fixture/main.go) provides the local API for the hosted Action scenarios.

See [CI and public Action verification](../../docs/development/testing/ci.md) for the local test command and the scenarios GitHub runs with each candidate.
