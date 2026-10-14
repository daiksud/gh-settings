---
type: Guide
title: Adopt owner-approved settings workflows
description: Set up, review, and recover the inert GitHub Actions examples.
---

## Delivery and adoption boundary

The [examples](../examples/github-actions/README.md) compose the released CLI; they are not a root reusable Action. Merging them installs no settings workflow and changes no repository setting. The [contract](adr/0022-owner-approved-actions.md) and [scenarios](features/actions.feature.md) define the supported behavior.

Every job runs on a fresh GitHub-hosted `ubuntu-24.04` runner. Bash, curl, GNU sha256sum, Git, and Python 3 are runner prerequisites. Shared persistent self-hosted runners are outside this setup. Workflow and configuration commits must remain available for ordinary reruns.

Implementation is complete independently of operational setup. [#92](https://github.com/daiksud/gh-settings/issues/92), [#93](https://github.com/daiksud/gh-settings/issues/93), and [#94](https://github.com/daiksud/gh-settings/issues/94) remain manual checks until real evidence is recorded. No first target has been assumed. Prefer an owner-approved disposable/test repository for verification.

## Files and checks

Use one reviewed repository revision for both workflow examples and all six support files:

| Inert source | Adoption destination / purpose |
| --- | --- |
| `examples/github-actions/settings-pr.yml` | `.github/workflows/settings-pr.yml` — PR validate and read-only plan |
| `examples/github-actions/settings-main.yml` | `.github/workflows/settings-main.yml` — main prepare and approved apply |
| `examples/github-actions/install-cli.sh` | Same path — fixed released binary, checksum/version verification |
| `examples/github-actions/plan.sh` | Same path — validate before plan |
| `examples/github-actions/render-summary.py` | Same path — quoted review Summary |
| `examples/github-actions/prepare.sh` | Same path — exact configuration/CLI/target handoff |
| `examples/github-actions/verify-apply.sh` | Same path — verify handoff before token minting |
| `examples/github-actions/apply.sh` | Same path — direct CLI apply with step-scoped token |

Keep the helpers at those paths in the target repository. Prepare and apply check out the same triggering configuration commit. Neither job builds source, imports a PR artifact, nor replays a saved plan. Declare supported values in `.github/settings.yml` using [schemaVersion 1](settings.md#configuration-semantics); omitted fields stay unmanaged. Do not use export redirection with v0.5.0.

The [development checks](development.md#development-checks) run Actionlint, ShellCheck, bootstrap/plan/prepare/apply fixtures, Summary tests, and the composed Go tests alongside the existing product checks. The composed fixture runs the real YAML shell commands with a mock CLI and local approval/token stubs. Existing built-product acceptance separately exercises the real CLI against a local API fixture. It does not establish actual GitHub protection or revocation.

The CLI pin is `v0.5.0`, asset `gh-settings-linux-amd64`, SHA-256 `b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4`. Review version, asset, and committed digest together against published assets before updating. Review the checkout and official token Action full-SHA pins and their exact supported inputs; the token Action currently uses v3.3.0's `client-id` and explicit `permission-administration: write`. Do not resolve `latest` or a moving Action tag at runtime, cache PR-supplied executables, or disable token revocation.

## Ordered owner setup

These are future owner operations, not actions performed by merging this guide. Record non-secret evidence in the matching manual issues. Never put private keys, token values, or credential screenshots in repository files, issues, logs, artifacts, or chat.

1. **Choose the target and scope (#92).** Name the exact repository, dedicated App, and approved persistent permissions. Approve a selected-repositories installation limited to that target set. Grant repository **Administration: read and write**, automatic **Metadata: read**, and no unrelated repository/organization access. Do not substitute a broad App or personal access token. Record only non-secret App/client/installation identifiers and selected repository names. The workflow separately requests a token for only its containing repository.
2. **Configure the target Environment (#93).** Verify the target plan/visibility supports the controls; stop if unsupported. Create `settings-apply`, require the verified owner as reviewer (`daiksud` only when that is the target owner), leave prevention of self-review disabled, and disable administrator bypass. Select deployment branches/tags and add only a **Branch** rule named `main`. Add no Tag rule, wildcard, PR ref, or protected-branches-only fallback. Read the rules back before storing credentials.
3. **Enter credentials securely (#92/#93).** The owner generates/handles the dedicated App key through GitHub's secure flow. After protection is verified, the owner submits Environment secret `SETTINGS_APP_PRIVATE_KEY` and Environment variable `SETTINGS_APP_CLIENT_ID`. Record names/existence only. There must be no repository/organization-wide key fallback. Review every eligible workflow, helper, Action, and CLI pin: an Environment name is not a workflow allowlist, and anyone who can change approved code can change privileged behavior.
4. **Authorize adoption and exact test settings (#94).** Name the target, reviewed example revision, concrete temporary before/desired values, activation, apply, and cleanup scope. Copy the two workflow examples to the destinations above only through a separately scoped adoption PR. Preserve the support files from that same reviewed revision. For repositories already containing the helpers, the later copy commands are:

   ```sh
   mkdir -p .github/workflows
   cp examples/github-actions/settings-pr.yml .github/workflows/settings-pr.yml
   cp examples/github-actions/settings-main.yml .github/workflows/settings-main.yml
   ```

   Merge this adoption only after its ordinary review/CI gates and explicit activation scope are satisfied. This project does not assume self-management of `daiksud/gh-settings`.
5. **Verify real gates and results (#94).** Exercise valid/invalid PR plan; inspect a known main prepare Summary. Observe apply waiting before any key/token use. Reject one run and confirm no update; approve an eligible run as the owner, including intentional self-approval for an owner-triggered run. Test actual Environment branch/tag enforcement with the temporary probe below, separately from main workflow filtering, without loosening rules. Read applied values and omitted fields back, inspect token cleanup evidence, and safely verify revocation where possible. Record rerun/attempt actors and run URLs. Restore only explicitly approved temporary values and read them back. Unverified platform checks remain open.

## Distinguish workflow filtering from Environment enforcement

A non-main branch or tag cannot reach the main example's guarded apply job. That observation proves workflow filtering, not Environment branch/tag enforcement. Include a separately reviewed temporary probe, its exact source/refs, and removal in #94's owner authorization. On the approved test target only, use `.github/workflows/settings-environment-probe.yml` with this minimal shape:

```yaml
name: Temporary settings Environment probe
on:
  push:
    branches: [main, settings-gate-probe]
    tags: [main]
permissions: {}
jobs:
  probe:
    environment: settings-apply
    runs-on: ubuntu-24.04
    timeout-minutes: 5
    steps:
      - run: echo 'Environment admitted this ref.'
```

The probe has no checkout, secret reference, App token, or settings mutation. Ensure the reviewed probe file exists at each tested commit before pushing its approved branch/tag. Expect the main deployment to wait for owner review, while `settings-gate-probe` and Tag `main` are denied by the unchanged Environment rules. Inspect the actual deployment/job denial evidence; absence of a workflow run proves nothing. Do not overwrite an existing branch/tag to obtain evidence. Remove only the authorized temporary probe and refs afterward. Never broaden the privileged main workflow's triggers for this test.

## Review a pending run

Open the Actions run Summary before approving its waiting `settings-apply` deployment. Check the original actor and initiating/rerun actor, event, run/attempt, workflow path/ref and exact workflow-source revision, target, configuration revision/file link, pinned CLI identity, and observed before → desired after values. The workflow-source SHA and configuration SHA are separate identities; each link uses its displayed revision.

Do not approve a missing or incomplete review Summary. A successful prepare does not prove that GitHub published it. Detectable rendering/write failure fails prepare; no publication polling or additional approval mechanism is provided. Quoted settings are data, including apparent HTML, Markdown headings/fences, and workflow commands.

No differences and no managed fields are informational successful outcomes, not special diff exit codes. Direct apply sends declared values without reading current settings; intervening remote changes to managed fields can be overwritten. Omitted fields are outside the request. Approval does not save the observed plan or guarantee that it stays current.

PR concurrency cancels superseded PR runs. Main uses one repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. Ordinary reruns keep GitHub's approval behavior and show the rerun actor/attempt. There is no custom queue, drift check, freshness timer, stale-commit check, or reapproval controller.

## Failure and recovery

| Situation | Maintainer action |
| --- | --- |
| Missing/invalid config | Correct the declaration and rerun through normal review; validation fails before API access. |
| CLI download/checksum/version error | Inspect the fixed release identity and reviewed pin. Do not execute an unchecked binary or switch to source/latest. |
| Plan read/API permission error | Inspect ordinary read-token access and the target origin. Do not provide administration credentials to PR/prepare. |
| Failed prepare or rejected approval | Apply cannot proceed. Fix the cause or initiate a new eligible run; no rollback is implied. |
| Missing App key/client ID or token error | Verify the protected Environment and dedicated installation scope. Have the owner use secure entry; no privileged token fallback. |
| Failed/cancelled apply or output failure | Read target settings before deciding what to do: an update may already have succeeded. The CLI does not promise atomicity or automatic rollback. |
| Token cleanup failure | Inspect the official Action's post-job logs and dedicated installation. Treat revocation as unverified; do not expose the token while investigating. |
| Disable further executions | Disable the adopted settings workflow in Actions and cancel pending/running executions as appropriate; review any update already sent. Disabling does not undo settings. Credential revocation/protection changes remain owner security operations. |

## References

- [GitHub contexts](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts)
- [Environment setup and supported controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
- [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510)
