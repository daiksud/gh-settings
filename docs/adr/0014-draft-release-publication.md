---
type: ADR
title: Publish only a complete draft release
description: Stage and verify all required release contents before publication.
---

## Status

Accepted — agreed, not implemented.

## Context

A release can be interrupted after tagging or during asset upload. Publishing as soon as the release record exists would expose an incomplete distribution.

## Decision

Stage the release as a draft and publish it only after all required artifacts and release notes are present and verified.

## Consequences

An incomplete draft remains unpublished while unfinished steps are assessed for safe resumption. Publication requires the two supported executables and their SHA-256 checksums, alongside release notes, as specified in the [release design](../releases.md). A draft does not permit moving an existing tag or replacing a conflicting asset under [ADR 0013](0013-immutable-release-identity.md) and [ADR 0018](0018-preserve-published-assets.md).
