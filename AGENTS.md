# Repository agent instructions

Follow [CONTRIBUTING](CONTRIBUTING.md) for development, verification, and documentation placement.

## Branches, classification, and releases

- Treat branch names only as work identifiers. Do not derive PR types, labels, or release versions from them, or rename a branch solely to correct classification.
- Check PR titles and labels against the actual changes and their impact on CLI or public Action users. Correct the metadata during review; syntax checks do not establish semantic correctness.
- Keep each PR focused and preserve its reviewed title, body, and compatibility footers in the squash commit. Follow the code-review skill for the detailed policy.
- Merging and releasing are separate actions. Without an explicit user request to release, do not create or push release tags, dispatch release workflows, or publish GitHub Releases. When requested, follow the documented manual release procedure and all its validation gates.

## Code review

Read and apply [.agents/skills/code-review/SKILL.md](.agents/skills/code-review/SKILL.md) for every code or PR review, including self-review and reassessment after fixes. It owns this repository's review policy, including PR scope, squash messages, breaking changes, and implementation review criteria.

When both Copilot and Codex are unavailable because their code-review quotas are exhausted, use the owner's authorized administrator-bypass squash merge after self-review, resolving actionable findings, and passing required checks for the intended head. Record the quota evidence or owner's report and the bypass in the PR or tracking Issue. This exception waives unavailable review approval, not correctness checks, unresolved findings, or release authorization. Use existing bypass permission; do not weaken protection rules or fabricate reviews or check results. When either service is available, use the normal review process.

## Communication language

Write GitHub Issue and pull request titles, bodies, and comments in English. Communicate with the user in Japanese.
