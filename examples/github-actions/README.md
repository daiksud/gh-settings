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

## Read-only PR example

`settings-pr.yml` uses `pull_request` and the containing repository's synthetic PR merge revision (`github.sha`), including for forks. Checkout binds the CLI's origin target to that repository and does not persist credentials. The verified released CLI runs validate before GET-only plan, with the ordinary read-scoped token only on that step. Missing/invalid configuration and API failures fail the job; differences still exit successfully. Plan output is visible in the run log. No protected Environment, App key, apply command, or privileged fallback exists in this example.

The workflow remains outside `.github/workflows`. Later adoption must copy it deliberately, together with its support files; merging this example alone activates nothing. Lint with actionlint and run `bash tests/github-actions/plan.sh`. Existing built-product acceptance separately proves the CLI's GET-only plan and configuration preservation.

## Auditable review Summary

`render-summary.py OUTPUT_FILE EXIT_STATUS` uses Python 3's standard library to render explicit `SETTINGS_*` run metadata and the CLI output. Its tests list the required context fields. Workflow-source and configuration SHAs have separate exact-revision links. Original and initiating/rerun actors, run/attempt, event, target, and verified CLI identity appear separately from quoted settings data. No full GitHub context or credentials are dumped.

The renderer distinguishes changes, no differences, no managed fields, and errors. It quotes output inside an escaped HTML data block, so settings text cannot close the block and forge approval context. Test it with `python3 tests/github-actions/summary.py`. The later main prepare example supplies these fields and writes the result to GitHub's Summary.
