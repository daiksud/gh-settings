# Local CI and release Actions

Each child directory defines a composite Action in `action.yml`. The [repository workflows](../workflows/README.md)
select the runner, event and permissions. These Actions group the steps a job runs.

- PR and main builds share [lint](lint/action.yml), [unit tests](unit-test/action.yml) and
  the [native CLI build](build/action.yml).
- [Release acceptance](release-acceptance/action.yml) verifies a transferred artifact and
  passes the binary to the [CLI integration tests](integration-test/action.yml) or
  [public Action end-to-end tests](e2e-test/action.yml).
- The manual [release workflow](../workflows/release.yml) uses the `release-*` Actions to
  resolve a version, reuse eligible artifacts, publish verified binaries and check the downloads.

When changing an external Action dependency, check the
[dependency inventory](lock-workspace-dependencies/action.yml) and the
[lockfile procedure](../../docs/development/repository-automation.md#workflow-dependency-lockfile).
