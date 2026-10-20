---
type: Feature
title: Pull request access
description: Declarative pull request availability and creation policy.
---

## Contract

Current source adds unreleased `settings.repository.has_pull_requests` (boolean) and `pull_request_creation_policy` (`ALL` or `COLLABORATORS_ONLY`). Published v0.16.0 and older packages reject these keys. Omission leaves each setting unmanaged; false disables PR access without synthesizing a creation policy. Null, wrong types and unknown enum values are invalid.

The [GraphQL repository contract](https://docs.github.com/en/graphql/reference/repos) exposes both fields and `updateRepository` inputs. Read both policy values completely when either is needed, rejecting missing/null/unknown values and partial-data errors. Export includes both values only after the complete observation succeeds; failure preserves the old file. PR-disabled state does not imply an absent or default policy.

Plan uses query operations only. Direct apply resolves the repository's opaque ID from the checkout owner/name, verifies target identity, and sends only declared PR values through `updateRepository`. It never uses undocumented REST keys or changes the omitted companion setting. Verify the declared mutation response values and target; errors after a mutation have uncertain effects. Existing repository REST writes run first, then PR access, topics, labels and Rulesets. Stop on first error, retaining earlier writes without rollback.

No live settings, token grants, individual PR operations or release changes are part of implementation verification. Local dummy-credential fixtures prove requests and behavior, not actual write compatibility or permissions. Reads need complete policy visibility; writes need a credential authorized to update the repository. No permission is expanded automatically.

## Acceptance

Cover true/false, both policies, independent omission, invalid YAML/enum/null, disabled observation, complete and partial-error reads, exact lookup/mutation identity and selected values, response mismatches and redirects. Exercise export/validate/plan/apply and native CLI/Action reports with the built CLI against isolated local fixtures. Mixed REST/GraphQL failures stop later resources and report earlier writes honestly.

## Existing test mapping

| Boundary | Tests |
| --- | --- |
| Presence, observation and comparison | [PR access model](../../internal/settings/pr_access_test.go) |
| YAML types, omission and round trip | [PR access codec](../../internal/workspace/pr_access_test.go) |
| Selective complete reads, exact mutations, identity and errors | [PR access API](../../internal/github/pr_access_test.go) |
| Native report values and invalid enum/types | [PR access Action presentation](../../internal/actions/pr_access_test.go) |
| Built CLI lifecycle, native Action report/Summary and failure sequencing | [PR access process acceptance](../../tests/acceptance/pr_access_test.go) |

The process suite also retains the existing repository export/read/write fixtures, now including the two additional fields and narrow query. Verification uses local files/Git and dummy credentials; the read-only public-repository example of disabled PRs with policy `ALL` is observation evidence, not a write grant or preservation guarantee.
