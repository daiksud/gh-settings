# GitHub Actions consumer examples

Copy these workflows into the repository whose settings you want to manage. The main example has one job and three steps: checkout, create an App token and call `daiksud/gh-settings`. Consumers need no external scripts.

Both workflows use a reviewed full Action SHA; the [published package and pins](../../docs/github-actions.md#published-package-and-consumer-pins) identify the selected release. The [public Action guide](../../docs/github-actions.md) owns inputs, outputs, runner requirements and setup instructions.

The selected immutable v0.19.0 package includes the declared-only Ruleset report correction required by this privileged observer pattern. It removes organization-only custom properties/private repository forking; remove those declarations before adopting this pin. It does not include unreleased label/Ruleset ID ownership or named Environment management. Published label/Ruleset collections remain authoritative; review deletion consequences and resource-specific permissions in the [package guide](../../docs/github-actions.md#published-package-and-consumer-pins).

## Copy the workflows

| Source | Destination |
| --- | --- |
| [settings-pr.yml](settings-pr.yml) | `.github/workflows/settings-pr.yml` |
| [settings-main.yml](settings-main.yml) | `.github/workflows/settings-main.yml` |

Commit `.github/settings.yml` with [schemaVersion 1](../../docs/settings.md#configuration-semantics) at the repository root. These three files are the consumer workflow/configuration files; CLI installation, validation and PR Summary rendering belong to the Action. Copying them does not configure GitHub branch or Environment protections and does not install a workflow-scope guard.

Use a fresh GitHub-hosted `ubuntu-24.04` x64 job. Main checks out its exact `github.sha` without persisted credentials. PR plan does **not** check out PR code: its trusted inline preparation creates only Git metadata and the pinned candidate `.github/settings.yml` at `GITHUB_WORKSPACE`. The fixed containing-repository GitHub.com `origin` selects the target. No external caller script is needed.

## Environment and credentials

Complete the [ordered owner setup](../../docs/github-actions.md#ordered-owner-setup) before enabling main apply, including [required reviews/checks on main](../../docs/github-actions.md#protect-main-with-required-checks), the Environment gate, and [protection verification](../../docs/github-actions.md#verify-the-protection-before-use). The example uses these names:

| Resource | Name |
| --- | --- |
| Protected plan Environment | `settings-plan` |
| Protected apply Environment | `settings-apply` |
| Environment variable | `SETTINGS_APP_CLIENT_ID` |
| Environment secret | `SETTINGS_APP_PRIVATE_KEY` |

Use the **same App** in both Environments, installed on the target repository. PR plan mints a short-lived token with Contents read for metadata/blob retrieval and Administration read for the example's base policy reads. Main mints a separate target-scoped Administration-write token. Keep default post-job revocation. There is one explicit App-token input per settings call, no PAT/personal-login combination and no ambient fallback.

Those grants are not universal: labels need Issues or Pull requests read/write, complete Ruleset bypass observation needs write access even for plan, and Pages/Environments/security have their own boundaries. Use the [feature permission and App-evidence table](../../docs/github-actions.md#authentication-and-app-permissions) before adding declarations; adjust only the separately approved App/token scope. The samples do not change real grants. Strongly recommend exact Environment Branch `main` restrictions **plus** enforced review/protection of workflow and credential-consuming paths before entering keys. Branch restrictions alone do not identify the trusted workflow.

## PR review and main apply

`settings-pr.yml` runs the trusted base workflow on `pull_request_target` for PRs targeting main. After its protected `settings-plan` job starts, it mints the App token and prepares the event's exact PR head as **data only**. Preparation fetches filtered commit/tree metadata, refuses downloaded blobs from that fetch, and separately reads the one regular `.github/settings.yml` blob with size/SHA verification. It never materializes PR scripts, Actions, dependencies or other executable files. Unsafe/missing files, inaccessible refs/blobs, moved heads and API errors fail with a fixed diagnostic; no fork credential or PAT fallback is attempted.

The trusted pinned Action validates before settings observation and renders Summary. Differences, no differences and no managed fields succeed; invalid configuration or failed reads fail. The configuration output SHA identifies the candidate head, while workflow identity remains the trusted base revision. Do not copy a PR file into a base checkout and mislabel it as base configuration. Fork data follows the same restrictions; this example's local fixtures do not establish every private-fork/App visibility combination. Check applicable GitHub Actions event policy before enabling `pull_request_target`; do not bypass it automatically.

`settings-main.yml` runs for a push to Branch `main` that changes `.github/settings.yml` or the adopted main workflow. Its single `apply` job waits for the `settings-apply` Environment approval before any step runs. After approval it:

1. Checks out the triggering configuration commit.
2. Creates the target-scoped App installation token.
3. Calls `command: apply` with that token.

Review the workflow path, event, actors, pinned steps and credential scope as well as the PR plan and exact main configuration before approval; use the [approval checklist](../../docs/github-actions.md#review-a-pending-run). Main does not produce another plan, depend on the PR run or consume its plan. The Action step's outcome and logs report apply results. The [run review guide](../../docs/github-actions.md#review-a-pending-run) explains overwrite semantics and concurrency.

A required check name alone does not identify the workflow that produced it. If PR review is to be the only manual approval, first meet the [independent enforcement conditions](../../docs/github-actions.md#when-pr-review-is-the-only-manual-approval).

The caller owns the Environment, credentials, token lifecycle, event/ref guards and concurrency. Composite steps share the caller job: privileged plan is safe only with trusted executing code and candidate YAML treated strictly as data. Never build, test, install dependencies or execute local Actions from a PR head in either credential-bearing job. App/API permission evidence, runtime revocation and live protection remain adoption checks, not claims proved by copying YAML. If apply fails or is cancelled, inspect current settings before retrying; the [recovery guide](../../docs/github-actions.md#failure-and-recovery) covers partial success and token cleanup.

Contributors changing these examples should use the [testing guide](../../docs/contributing/testing.md); this project's own workflows and Environment policy are documented under [repository automation](../../docs/contributing/repository-automation.md).
