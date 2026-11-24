# Product and workflow tests

These suites exercise the built CLI, its public GitHub Action and the automation used to release them.

| Area | What to test here |
| --- | --- |
| [acceptance](acceptance/README.md) | Run a user command in a temporary checkout and check its output, settings file or GitHub API requests. |
| [github-actions](github-actions/README.md) | Check what a workflow runs, how the public Action installs and invokes the CLI, and which outputs a caller receives. |
| [fixtures](fixtures) | Reuse the Ruleset JSON examples shared by adapter and CLI tests. |
| [release.sh](release.sh) and [release-fixture.py](release-fixture.py) | Exercise publication and recovery with a local stand-in for the GitHub CLI. |

The [testing guide](../docs/development/testing.md#test-ownership) also identifies the package-level tests for settings parsing, comparison, API transport and presentation.
