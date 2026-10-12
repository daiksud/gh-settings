---
type: ADR
title: Never overwrite published release assets
description: Keep a published asset name bound to its original bytes.
---

## Status

Accepted — implemented by the manual release workflow.

## Context

Replacing an executable or checksum file under an existing published asset name would change what users download without changing the release identity.

## Decision

Do not overwrite an existing published release asset; stop on a conflicting asset name or bytes.

## Consequences

An interrupted run may resume uploading only after verifying that already present assets have the same tested bytes and checksums. Matching source commit and version is insufficient proof of identical bytes. The recovery mechanism remains to be designed in the [release specification](../releases.md). Tag immutability and draft publication are separate decisions in [ADR 0013](0013-immutable-release-identity.md) and [ADR 0014](0014-draft-release-publication.md).
