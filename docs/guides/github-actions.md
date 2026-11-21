# Plan pull requests and apply settings with GitHub Actions

The [two example workflows](../../examples/github-actions/README.md) let you review the settings declared on a pull request and apply the merged declaration to the repository. Plan runs trusted workflow code against the proposed YAML. Apply runs on the exact commit that reached `main` and uses its own approval and credential.

## Public Action and supported checkout

Use a fresh GitHub-hosted `ubuntu-24.04` x64 runner with Bash, curl, GNU sha256sum and Git. Prepare the repository at `GITHUB_WORKSPACE` with a resolvable `HEAD` and a direct GitHub.com `origin`. When using `actions/checkout`, omit its `path` input and set `persist-credentials: false`. The Action uses the checkout commit and origin to identify the configuration revision and target, even for local validation. Export can create `.github/settings.yml`. The other commands need the file to exist. Installation downloads the CLI even for validate. [Action inputs and outputs](../references/action.md)

## Published package and consumer pins

The examples pin the [published v0.44.0 package](https://github.com/daiksud/gh-settings/releases/tag/v0.44.0) at `948873d260be87aecc3d0597f715a8c09e1347b9`. That pin runs the CLI from that release. If your declaration uses newer fields, select a reviewed published version that supports them and pin its full commit SHA in both workflows. A release tag selects the corresponding CLI version. A full SHA must resolve to one matching stable tag.

Installation fetches the release's `gh-settings-linux-amd64` binary and `SHA256SUMS` from fixed upstream HTTPS paths. It checks the digest and the version reported by the binary. Both assets come from the same release. The verified version and digest appear in the Action outputs.

## Authentication and App permissions

Install one GitHub App on the target repository. Grant access to the resources this declaration manages, then narrow each job's installation token to the repository and permissions that job needs. Metadata read is automatic. The PR example requests Contents read to retrieve the candidate file, plus Administration read for its settings reads. The main example requests Administration write. Keep default post-job token revocation enabled. [Permission matrix by YAML field and command](../references/settings/permissions.md) · [Token Action at the example revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510)

Complete reads sometimes require write access. Examples include Ruleset bypass actors and repository event policies. Protect any plan job with a write-capable token as carefully as apply. If the App cannot obtain the full state for a declared field, check its installation grants, token request and endpoint support. Validate needs no settings API token. [GitHub App installation authentication](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app-installation)

## Ordered owner setup

1. **Protect the credential paths.** Require review of changes to settings, workflows, dependencies and ownership rules on `main`. Set up the plan and apply Environments and verify which workflows may request each one. Follow [Protect the apply path](protect-apply.md) before supplying the key.
2. **Prepare the declaration and workflows.** Use a configuration the pinned package accepts. Copy the [PR and main workflows](../../examples/github-actions/README.md#copy-the-workflows) to the indicated paths. Retain their event guards, exact commit selection, full-SHA pins and concurrency settings.
3. **Bind the App.** Configure `SETTINGS_APP_CLIENT_ID` as a variable and `SETTINGS_APP_PRIVATE_KEY` as a secret in the protected `settings-plan` and `settings-apply` Environments. Both use the same App. Verify the [approval and branch restrictions](protect-apply.md#verify-the-protection-before-use) before enabling credential-bearing runs.
4. **Review a PR plan.** Open a PR changing `.github/settings.yml`. Check the target, proposed configuration SHA and plan Summary. Differences, no differences and no managed settings all count as successful plans.
5. **Review the pending apply.** After merge, inspect the declaration at the exact main commit, the workflow revision and the App permissions before approving the job. The main workflow applies the merged declaration directly. [Review a pending run](protect-apply.md#review-a-pending-run)

### Trusted PR data

The PR example uses `pull_request_target`. Its trusted base workflow verifies the event's PR number and head SHA, retrieves filtered commit/tree metadata and reads only the regular `.github/settings.yml` blob by its identity and size. It rejects moved heads, unsafe file modes, bad data and failed reads. The prepared checkout uses the candidate SHA and a fixed origin for the base repository.

Keep candidate scripts, dependencies, local Actions and artifacts out of the credential-bearing job. All steps share a runner. A fork whose configuration the App cannot read must fail the plan. Review the GitHub event policy and the workflow that will receive the token. [GitHub guidance for pull_request_target](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target)

## Export a local configuration

In a trusted checkout, add an export step after minting a target-scoped installation token. This example assumes that the token step has the ID `app`.

```yaml
- name: Export settings
  id: export-settings
  uses: daiksud/gh-settings@948873d260be87aecc3d0597f715a8c09e1347b9 # v0.44.0
  with:
    command: export
    token: ${{ steps.app.outputs.token }}
```

After success, `steps.export-settings.outputs.config-path` is `.github/settings.yml` in the current job. `config-sha` identifies the checked-out commit. The Action writes the file locally without committing or transferring it. Decide how to handle plaintext variables before sharing the result.

Read the export diagnostics on stderr. An unavailable setting can be omitted on first export or retained as unverified intent on re-export. The step fails if the CLI cannot write a required diagnostic or save the declaration. A local file can still exist after a failed initial write or an output error. Export suppresses inherited `GH_DEBUG` and command stdout. It produces no plan Summary. [CLI export behavior](../references/commands.md#export)

## Failure and recovery

| What happened | What to check |
| --- | --- |
| Unexpected job asks for an Environment | Reject the request. Inspect the workflow, event and credential path. If a secret was exposed, revoke or rotate it and inspect the target. |
| Download, checksum or version verification failed | Check the selected release, its assets and the workflow pin before rerunning. |
| Validation failed | Correct the declaration for the selected package and submit it through review. |
| Plan cannot read a declared resource | Check the target, App installation, token request and complete visibility required by that resource. |
| Token creation or cleanup failed | Check the protected Environment bindings and the token Action's post-job logs. Keep credential values out of diagnostics. |
| Apply failed or was cancelled | Inspect current settings and reported IDs. Some writes can already be effective. [Recovering from a failed apply](../references/commands.md#recovering-from-a-failed-apply) |

To stop further runs, disable the adopted workflow and cancel pending or running jobs as appropriate. Review any writes already sent.
