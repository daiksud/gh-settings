# Public Action CLI installer

The root [action.yml](../action.yml) calls [install-cli.sh](install-cli.sh) from the Action
checkout before it runs a settings command. The installer resolves the Action tag or pinned
commit to a CLI release, downloads the Linux x64 binary into the runner's temporary directory
and checks both its SHA-256 digest and its reported version. It prints the verified executable
path for the next step.

Local composite calls have no published ref. The [candidate fixture](../tests/github-actions/prepare-candidate-download.sh)
sets `SETTINGS_CANDIDATE_VERSION` after verifying the supplied binary's version.

When changing release selection or verification, run `bash tests/github-actions/action-install.sh`
from the repository root. The [installer fixtures](../tests/github-actions/action-install.sh)
exercise the download and integrity checks offline. The [Action guide](../docs/guides/github-actions.md#published-package-and-consumer-pins)
describes the release references that callers can use.
