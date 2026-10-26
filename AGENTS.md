# Repository agent instructions

Follow [CONTRIBUTING](CONTRIBUTING.md) for development, verification, and documentation placement.

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
`BREAKING CHANGE:` footer, following Conventional Commits. PRLint only runs
commitlint on the proposed squash message; reviewers assess the actual impact
under the [code-review policy](.agents/skills/code-review/SKILL.md).

## Branches, classification, and releases

- Treat branch names only as work identifiers. Do not derive PR types, labels, or release versions from them, or rename a branch solely to correct classification.
- Check PR titles and labels against the actual changes and their impact on CLI or public Action users. Correct the metadata during review; syntax checks do not establish semantic correctness.
- Keep each PR focused and preserve its reviewed title, body, and compatibility footers in the squash commit. Follow the code-review skill for the detailed policy.
- Merging and releasing are separate actions. Without an explicit user request to release, do not create or push release tags, dispatch release workflows, or publish GitHub Releases. When requested, follow the documented manual release procedure and all its validation gates.

## Code review

Read and apply [.agents/skills/code-review/SKILL.md](.agents/skills/code-review/SKILL.md) for every code or PR review, including self-review and reassessment after fixes. It owns this repository's review policy, including PR scope, squash messages, breaking changes, and implementation review criteria.

Protected `main` requires one approving review, dismissal of stale approvals after a push, and resolved review threads. Require CODEOWNER approval. When the sole CODEOWNER authors a PR, another eligible reviewer, including a permitted GitHub Actions bot approval, can satisfy the review requirement; the author cannot approve their own PR. The `approve-codex-review` workflow creates a GitHub Actions approval only after Codex's code and security reviews both complete for the current HEAD without unresolved findings and Codex reacts with 👍. Comments, reactions, or a successful workflow run alone are not approving reviews; Copilot reviews do not count toward required approvals.

Use only a normal squash merge after obtaining a valid approving review, resolving actionable findings and threads, and passing all required checks on the final HEAD. Do not configure or use Ruleset bypass actors, including administrators. If Codex and Copilot are unavailable, leave the PR unmerged until an eligible reviewer can approve it. Never bypass CI, approval, signatures, resolved-thread requirements, or release authorization; do not push directly to `main`. Verify the live GitHub Ruleset and Actions review permissions before claiming protection is enforced.

## Communication language

Write GitHub Issue and pull request titles, bodies, and comments in English. Communicate with the user in Japanese.
