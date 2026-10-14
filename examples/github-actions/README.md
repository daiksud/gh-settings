# GitHub Actions examples

These support files are inert. No settings workflow is installed in `.github/workflows` by merging them. The [approved contract](../../docs/adr/0022-owner-approved-actions.md) separates code delivery from later owner-authorized setup and adoption.

## Fixed CLI bootstrap

On a fresh GitHub-hosted Linux x86-64 runner with Bash, curl, GNU sha256sum, and Git, run from the checkout root:

```sh
bash examples/github-actions/install-cli.sh "$RUNNER_TEMP/settings-cli"
```

The script prints the verified executable path. It downloads `gh-settings-linux-amd64` from the fixed upstream `v0.5.0` release, verifies the committed SHA-256 before making it executable, and checks `gh-settings v0.5.0`. Failures remove the destination. Review the version, asset, and checksum together when updating the script. The initial digest was checked against the release's `SHA256SUMS` and GitHub asset digest.

Use the same script from the same configuration commit in prepare and apply. Do not cache PR-supplied executable content or build repository source for privileged use. Shared persistent self-hosted runners are outside the supported setup.

Offline bootstrap fixtures run with `bash tests/github-actions/bootstrap.sh`; they stub downloading and replace only the expected digest for local test binaries. No token or GitHub update is used.
