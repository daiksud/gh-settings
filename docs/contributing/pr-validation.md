# Independent PR message and code validation

The [PRLint workflow](../../.github/workflows/prlint.yml) validates the PR title and body on creation, push, reopening, and metadata edits. Its stable `pr-message` job uses the existing [commitlint policy](../../commitlint.config.mjs) and [policy tests](../../tests/github-actions/commitlint.sh). It has read-only permissions, no Draft/Ready changes, and no product build or acceptance work.

Types, scope, compatibility, and meaningful body content still require the [code-review policy](../../.agents/skills/code-review/SKILL.md); passing syntax checks cannot establish semantic correctness.

## Code-only validation

The [code workflow](../../.github/workflows/ci-pr.yml) receives PR edits but starts product work only on creation, push, reopening, or a base edit. A title/body-only edit skips the code path without writing a new `pass` result. A separate metadata concurrency group prevents that edit from cancelling running code validation.

For a code event, the status-only `code-start` job marks the tested pull-request merge candidate's `pass` commit status pending. The workflow then sets Draft, resolves target matrices, runs commit-stage, sets Ready, and runs process and public-Action acceptance matrix legs against the same manifested binary. The status-only `code-result` job publishes success on that same merge candidate only when every prerequisite succeeded. Failed, skipped, or cancelled prerequisites cannot yield success. An interrupted run can leave a pending status, which remains blocking until a valid code run completes.

The orchestration uses GitHub's event triggers and per-PR concurrency to cancel superseded code runs. It does not reimplement head/base freshness, execution-generation, or ruleset enforcement checks. GitHub protection settings own merge eligibility and required-check freshness. The artifact manifest separately verifies candidate source SHA, target, binary name, digest, and product version.

Only dedicated status and Draft/Ready jobs receive write permissions. They do not check out or execute candidate code or downloaded artifacts. Code and acceptance jobs use read-only permissions and checkouts without persisted credentials. Forks and Dependabot PRs without sufficient Draft/Ready permissions require human intervention; there is no privileged fallback.

## Required checks and rollout

The observed default-branch Ruleset requires `pass`, `pr-message`, and the trusted Environment guard `gh-settings` from GitHub Actions. All three must succeed for the intended revision. A metadata-only edit revalidates `pr-message` independently without replacing the previous code result.

A separate, explicitly authorized protection change is required before replacing or removing any required context. Preserve the existing required checks during the transition; do not fabricate a passing check or bypass a failure. Review approval, CODEOWNER review, and thread-resolution policy remain separate from the CI result.

[Repository automation](repository-automation.md#protected-main-and-required-checks) describes the operational settings and their verification limits. Repository source tests cannot establish that GitHub's current protection, Environment reviewer configuration, or credential boundaries are correctly enforced.
