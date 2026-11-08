# Repository agent instructions

Follow [CONTRIBUTING](CONTRIBUTING.md) for development, verification, and documentation placement.

## Project-wide feature adoption scope

As of 2026-11-08, adopt settings only when **all** of these conditions hold:

- The repository owner uses GitHub Free.
- The repository is owned by a personal account.
- The repository is public.
- The control is visible and configurable in GitHub's Web UI under those conditions.

Non-Free plans, organization ownership, private/internal repositories and controls hidden or unavailable in that UI are outside the adoption scope, even when a public API field exists. Verify this eligibility separately from the complete official public API read/update path needed for implementation. Reassess API-unconfirmed candidates under these same conditions. This replaces the earlier personal-public, price-independent Security scope and applies across gh-settings, not only to Security.

Preview features, UI paths that create/edit feature configuration files, and custom Dependabot auto-triage remain excluded. GitHub preset ON/OFF is distinct from custom rules and must satisfy the scope above. Historical decisions and existing API/entitlement descriptions do not override this policy. See the [feature adoption policy](docs/settings.md#feature-adoption-scope).

Existing functionality has not been fully audited against this narrower policy. Do not claim that all implemented features comply. This policy documentation does not remove features, change runtime behavior, authorize live settings changes or broaden previous exclusions; any implementation reconciliation requires separately scoped work.

## v0.x compatibility policy

During v0.x, do not preserve superseded CLI or public Action interfaces, YAML
formats, aliases, or behavior solely for backward compatibility. Do not add
automatic migrations, importers, fallback paths, or parallel implementations
for obsolete contracts. When a behavior changes, update its authoritative
public documentation and behavior specifications and remove obsolete code and
tests instead of retaining historical contracts alongside the current one.

This does not waive validation, checkout/target isolation, credential security,
data protection, honest failure and partial-failure reporting, or the
immutability of published releases. Incompatible public changes may use either a `!` title marker or a
`BREAKING CHANGE:` / `BREAKING-CHANGE:` footer, following Conventional Commits. PRLint only runs
commitlint on the proposed squash message; reviewers assess the actual impact
under the [code-review policy](.agents/skills/code-review/SKILL.md).

## Repository settings changes

Use the reviewed `.github/settings.yml` and manual `settings-apply` workflow
for this repository's normal settings changes. Read the
[settings change procedure](docs/contributing/repository-automation.md#repository-settings-changes)
before choosing an execution path or recovering created resource IDs. After
merge, dispatch it from `main`; settings are not applied automatically. The
hosted workflow uses its own App authentication; local CLI authentication is
not a prerequisite. Verify the relevant apply run and remote state, then
record confirmed creation IDs through the documented follow-up PR procedure.

## Branches, classification, and releases

- Treat branch names only as work identifiers. Do not derive PR types or release versions from them, or rename a branch solely to correct classification.
- Check the proposed squash title, body, and compatibility footers against the actual changes and their impact on CLI or public Action users. Correct the message during review; syntax checks do not establish semantic correctness.
- Keep each PR focused and preserve its reviewed title, body, and compatibility footers in the squash commit. Follow the code-review skill for the detailed policy.
- Merging and releasing are separate actions. Without an explicit user request to release, do not create or push release tags, dispatch release workflows, or publish GitHub Releases. When requested, follow the documented manual release procedure and all its validation gates.

## Code review

Read and apply [.agents/skills/code-review/SKILL.md](.agents/skills/code-review/SKILL.md) for every code or PR review, including self-review and reassessment after fixes. It owns this repository's review policy, including PR scope, squash messages, breaking changes, and implementation review criteria.

Protected `main` requires one approving review, dismissal of stale approvals after a push, and resolved review threads. Require CODEOWNER approval. When the sole CODEOWNER authors a PR, another eligible reviewer, including a permitted GitHub Actions bot approval, can satisfy the review requirement; the author cannot approve their own PR. The `approve-codex-review` workflow creates a GitHub Actions approval when Codex reacts with 👍 and its latest Review Summary marks both Code Review and Security Review as Completed, with both full reviewed SHAs matching the current PR HEAD. It withholds repeat approval when this PR already has a `github-actions[bot]` review for the same HEAD in `APPROVED` or `DISMISSED` state, including reviews created before this guard and same-HEAD re-reviews. It adds no base, time, receipt, finding-flag or prior-review-request conditions; GitHub Rulesets determine merge eligibility. Comments, reactions, or a successful workflow run alone are not approving reviews; Copilot reviews do not count toward required approvals.

Use only a normal squash merge after obtaining a valid approving review, resolving actionable findings and threads, and passing all required checks on the final HEAD. Preserve the existing repository-administrator pull-request-only bypass declared in `.github/settings.yml`; do not add or broaden bypass actors, or use any bypass to merge. If Codex and Copilot are unavailable, leave the PR unmerged until an eligible reviewer can approve it. Never bypass CI, approval, resolved-thread requirements, or release authorization; do not push directly to `main`. Verify the live GitHub Ruleset and Actions review permissions before claiming protection is enforced.

## Communication language

Write GitHub Issue and pull request titles, bodies, and comments in English. Communicate with the user in Japanese.
