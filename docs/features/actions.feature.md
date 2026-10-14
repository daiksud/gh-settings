---
type: Feature
title: Owner-approved settings automation
description: Specify inert PR and main workflow examples around the existing CLI.
---

## Feature: Review settings before approved direct apply

### Rule: Examples are inert until separately adopted

- Workflow examples and support files live under `examples/github-actions/`.
- Merging examples does not enable settings workflows, install an App, enter secrets, or apply settings.
- Every job uses a fresh GitHub-hosted Linux runner and reviewed full-SHA Action pins.
- The CLI is a fixed released binary verified against its committed SHA-256 before execution; prepare and apply use the same pin from the same configuration commit.

### Rule: PRs have only read access

- On a PR, check out the containing repository's PR revision without persisted credentials.
- Validate the checkout-root `.github/settings.yml` before read-only plan.
- Valid configuration and successful plan exit successfully whether differences exist or not.
- Missing/invalid configuration fails before API access; authentication/API failures fail visibly.
- Fork changes have no App credential, Environment, apply, or privileged fallback path.
- Per-PR concurrency cancels superseded executions.

### Rule: Main prepare establishes review context

- Only `push` on Branch `main` is eligible; PR, other branch, tag, manual, PR-target, and workflow-run contexts cannot unlock apply.
- Check out the triggering configuration SHA rather than a moving main branch.
- Validate, plan with the read token, then render/write the Summary before prepare succeeds.
- Invalid input, API failure, or detectable Summary render/write failure blocks apply.
- Expose only required non-secret handoff values, including the configuration SHA and CLI identity; do not save a plan artifact.
- A newer main commit cannot alter this run's configuration input.
- Repository-scoped concurrency does not cancel the running pipeline; GitHub may replace a pending run.

### Rule: The Summary separates identities from settings data

- First run shows `github.actor` and `github.triggering_actor`; reruns show the original actor separately from the rerun initiator and incremented attempt.
- Include event, run URL/attempt, workflow path/ref and exact workflow-source SHA/link.
- Include target, configuration SHA and exact configuration-file link, CLI version/checksum, and observed before → desired after output.
- Workflow-source SHA can differ from configuration SHA; each link uses the displayed identity.
- Distinguish changed settings, no differences, no managed fields, and errors.
- Render quotes, newlines, Markdown fences, HTML, and workflow-command-like values as data that cannot forge review context.
- State that direct apply can overwrite intervening changes to declared remote fields.
- The owner must not approve missing/incomplete review content. Successful prepare alone does not prove Summary publication; no publication polling is required.

### Rule: Owner approval gates credentials and apply

- Apply needs successful prepare and the same main guard, then waits at `settings-apply`.
- Required reviewer is the owner; self-approval is allowed, administrator bypass is disabled, and only Branch `main` is eligible with no Tag rule.
- While waiting, or after rejected approval, no App token is minted and no apply runs.
- The Environment key is available only inside the approved job; the official token Action narrows Administration write to the one target repository and retains normal revocation.
- The token is supplied only to the apply step, never checkout, bootstrap, Summary, global environment, artifact, or job/workflow output.
- Apply uses prepare's exact configuration commit and verified CLI identity.
- A successful apply reports success; missing credentials or CLI failure fails visibly. Output failure may follow a successful update; no rollback is promised.
- Ordinary reruns use GitHub's approval behavior without custom freshness, drift, or reapproval checks.

### Verification boundary

Each implementation PR adds its focused offline checks; composed coverage follows before adoption. Markdown/local-link checks validate this contract. Actual Environment waiting/rejection, branch/tag enforcement, and token revocation require separately authorized GitHub verification; offline fixtures cannot establish those platform behaviors.
