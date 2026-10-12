---
type: ADR
title: Never overwrite an existing release tag
description: Keep a version tag bound to its original commit.
---

## Status

Accepted — implemented by the manual release workflow.

## Context

A release version can collide with an existing tag. Moving that tag would make the same version point to different source code.

## Decision

Do not overwrite an existing release tag; stop if it points to a different commit.

## Consequences

An interrupted run may reuse an existing tag only after verifying that its version and commit match the intended release. If that identity cannot be proven, stop rather than move the tag. Published asset immutability is selected separately in [ADR 0018](0018-preserve-published-assets.md). [ADR 0014](0014-draft-release-publication.md) controls when the completed release becomes public.
