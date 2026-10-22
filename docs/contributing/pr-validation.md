# Separate PR message and code validation

The [message workflow](../../.github/workflows/pr-message.yml) validates the current PR title and body together using the existing commitlint configuration. Its `pr-message` check runs for opened, synchronized, reopened, and edited PRs. It has a separate per-PR concurrency group, read-only permissions, and no Draft/Ready transitions, product builds, acceptance tests, or composite smoke tests.

The existing [PR CI](../../.github/workflows/ci-pr.yml) still validates messages and code in this preparatory stage. Duplicating the inexpensive message check temporarily preserves every existing merge requirement while the new required check is introduced. This is preparation for [#386](https://github.com/daiksud/gh-settings/issues/386), not evidence that metadata-only edits already avoid product tests.

## Safe activation order

1. Merge the independent message workflow through normal review and required CI. Observe a successful `pr-message` result from the intended workflow on a current PR revision.
2. Add `pr-message`, provided by GitHub Actions, to the existing default-branch required checks. Retain `pass`, `gh-settings`, strict branch freshness, required approval/CODEOWNER review, resolved threads, and all other protection. Inspect live settings to confirm the addition; a YAML file or a successful optional check does not make a check required.
3. Only after that verification, merge the dependent code-validation routing change. Title/body-only edits must leave the required code result untouched; changed heads and base retargets still run code validation. Keep the new message check independently required.

The repository settings declaration currently leaves Rulesets unmanaged. Do not introduce an incomplete authoritative Rulesets list just to add this check: doing so could delete unrelated policies or require a different observation credential. Apply the check addition through the repository's existing maintainer-controlled Ruleset configuration, and verify the complete policy afterward. This guide does not authorize bypassing approval, changing credentials, or weakening protections.

## Distinct check identities

A conditionally skipped job is treated as a successful check by GitHub. Merely skipping all product jobs and reporting a successful `pass` on a metadata edit could therefore conceal failed code validation.

The dependent routing change uses exactly two gate names: `pass` for opened, synchronized, reopened, and base-edited PRs; `metadata-only` for other edited events. The trusted Environment policy permits only the corresponding fixed expression on `ci-pr.yml`'s `pass` job. Neither outcome can become the reserved `gh-settings` name. Other dynamic job names, dynamic Environments, reusable workflow jobs, and additional privileged Environment locations remain prohibited.

A metadata-only run must not emit a skipped or successful `pass`. Its successful `metadata-only` result is not a substitute for successful code validation or the independently required `pr-message` check. Metadata edits must not cancel a running code-validation run. Base retargeting is a code-validation event even when the head SHA has not changed.

## Evidence and boundaries

Local workflow tests establish configured event, permission, naming, and gate behavior. Hosted runs establish actual check names and execution results. Neither proves that a live Ruleset requires the new check; that must be read back separately before activation. Retain failed-code and invalid-message cases when testing the transition, and do not change commitlint restrictions as part of it.

Automatic version selection and GitHub-generated release notes remain deferred to #385. This change does not dispatch a release or change product binaries, consumer pins, or release identity.
