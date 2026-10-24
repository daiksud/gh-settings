---
type: Feature
title: Repository Sponsor button visibility
description: Manage the Sponsorships boolean without managing funding destinations or payments.
---

## Feature: Manage repository Sponsor button visibility

Published v0.18.0 supports `settings.repository.has_sponsorships` under `schemaVersion: 1`. It maps to GraphQL `hasSponsorshipsEnabled`. Published v0.17.0 and older packages reject the key.

### Rule: Preserve omission and native booleans

#### Scenario: Disable the button setting explicitly

- Given the complete observed Sponsorships value is true
- And the declaration contains `has_sponsorships: false`
- When plan runs
- Then it reports one native true-to-false change
- And CLI and credential-free Action Summary retain the boolean values

#### Scenario: Omit or invalidate the setting

- Given `has_sponsorships` is omitted
- When another declared field is planned or applied
- Then no Sponsorships read or update is added to that operation
- And a locally declared null, string, number, sequence or mapping fails before remote work

### Rule: Require a complete independent observation

#### Scenario: Export and compare false

- Given a successful authenticated query returns false
- When full export completes
- Then the source file contains explicit false among twenty-three ordinarily exported repository fields
- And validation followed by plan reports no differences for unchanged values
- And an unread, null or malformed value is not substituted for false

#### Scenario: Reject incomplete or partial reads

- Given a required repository value is unavailable, malformed or returned with GraphQL errors
- When plan or export attempts to observe it
- Then the operation fails without exposing an earlier partial comparison
- And export preserves the existing file
- And no funding-file lookup or write is used as a fallback

### Rule: Update only declared repository fields

#### Scenario: Apply the display boolean

- Given a current checkout origin identifies the requested repository
- When direct apply runs with only Sponsorships declared
- Then one rename-disabled lookup validates its opaque repository ID and name
- And one `updateRepository` mutation sends only that ID and the declared boolean
- And no REST PATCH or unrelated setting update is sent
- And the mutation response must retain the same identity and requested value
- And repeated scalar apply still sends its declared value

#### Scenario: Share a mixed GraphQL mutation

- Given PR access, Issue creation policy and Sponsorships are explicitly declared
- When apply runs
- Then they share one validated lookup and one declared-only mutation
- And omitted companion values are not synthesized

#### Scenario: Fail after an earlier REST write

- Given a declared REST setting was updated before a rejected Sponsorships mutation
- When apply reports failure
- Then it retains the warning that earlier repository writes remain applied
- And later topics, labels and other resources are not updated
- And it performs no retry or rollback

### Rule: Do not manage financial destinations

#### Scenario: Preserve funding configuration independently

- Given funding configuration is absent or an existing `FUNDING.yml` has specific bytes
- When the Sponsorships setting is enabled or disabled
- Then the tool neither creates that file nor changes its existing bytes
- And it does not choose recipients, alter inherited links, create sponsorships or make payments
- And fixture success does not assert live button rendering without valid funding configuration

### Verification

[Model tests](../../internal/settings/sponsorships_test.go) own acquired-state, omission, native false and partial-export behavior. [Codec tests](../../internal/workspace/sponsorships_test.go) own strict scalar types and round trips. [Adapter tests](../../internal/github/sponsorships_test.go) inspect exact queries/variables, field selection, declared-only combined mutations, response identity/boolean checks, partial errors and read routing. Existing adapter tests retain authorization, rename, redirect and unrelated-field boundaries.

[Process acceptance](../../tests/acceptance/sponsorships_test.go) executes the supplied binary through export, validate, plan, Action report/Summary and direct apply with an isolated Unix-socket API. It verifies false export, native output, repeat updates, absent/unchanged funding files, incomplete reads/exports, wrong targets, ignored or partial mutations, redirects, earlier REST writes and prevention of later writes. [Report tests](../../internal/actions/sponsorships_test.go) reject non-boolean presentation values. Existing full-export fixtures and report-count boundaries include the new scalar.

These fixtures use dummy credentials and modeled responses. They do not toggle a real repository, prove live token grants or missing-funding UI behavior, publish a package, or initiate any financial transaction. GitHub's [GraphQL contract](https://docs.github.com/en/graphql/reference/repos#updaterepositoryinput) defines the boolean, and its [Sponsor button guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/displaying-a-sponsor-button-in-your-repository) owns administrator access and funding prerequisites.
