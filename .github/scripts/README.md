# Workflow helper scripts

CI and release jobs call these scripts to select a version, verify artifacts and check a
candidate Action against GitHub.

- [release-version.py](release-version.py) selects a version from the source commit,
  Git history and published releases. Its [tests](test_release_version.py) cover the selection rules.
- [release-artifact.sh](release-artifact.sh) writes and verifies the manifest for each binary.
  After inspection, [deployment-manifest.py](deployment-manifest.py) records the push run,
  version, source SHA and both artifact IDs and digests. [release-ci.py](release-ci.py)
  verifies that record and its artifacts before selecting them for release or the Topics check. Its [tests](test_release_ci.py) exercise reuse and integrity checks.
- [verify-public-topics.py](verify-public-topics.py) prepares settings and checks Topics
  before and after the candidate Action runs against a test repository. See the
  [verification workflow](../workflows/verify-public-action.yml) for the apply and restore sequence.

The [unit-test Action](../actions/unit-test/action.yml) contains the commands CI uses to run both Python test suites.
