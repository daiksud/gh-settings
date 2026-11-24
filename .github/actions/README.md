# Local CI and release Actions

Each child directory defines a composite Action in `action.yml`. The [repository workflows](../workflows/README.md)
select the runner, event and permissions. These Actions group the steps a job runs.

- The [screening Action](screening/action.yml) runs lint, unit tests and offline fixtures
  before the [native CLI build](build/action.yml) on Linux. The macOS candidate uses the same native build.
  [Candidate packaging](package-candidate/action.yml) uploads each binary with its identity manifest.
- [Release acceptance](release-acceptance/action.yml) verifies a transferred artifact and
  passes the binary to the [CLI integration tests](integration-test/action.yml) or
  [public Action end-to-end tests](e2e-test/action.yml).
- The [version workflow](../workflows/version.yml) classifies the release interval and compares
  native binaries when the public Action package matches and a PATCH remains possible.
- The manual [release workflow](../workflows/release.yml) selects eligible artifacts with
  [release-ci.py](../scripts/release-ci.py), then [publishes the verified binaries](release-publish/action.yml)
  and [checks the public downloads](release-verify/action.yml).

When changing an external Action dependency, check the
[dependency inventory](lock-workspace-dependencies/action.yml) and the
[lockfile procedure](../../docs/development/repository-automation.md#workflow-dependency-lockfile).
