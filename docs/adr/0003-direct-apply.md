---
type: ADR
title: Apply declared values directly
description: Record the agreed decision to apply declared values directly.
---

## Status

Accepted — implemented and available since v0.4.0.

## Context

An informational plan observes remote values at one point in time. The agreed apply contract uses the declaration itself as the update input.

## Decision

Apply declared supported fields directly to the update API without fetching current settings, computing differences, or comparing against a saved plan.

## Consequences

External changes to managed values are overwritten by the declared values when the update succeeds. The plan diff may differ from changes at apply time. Saved plan artifacts and drift protection are outside the MVP. Validation and rollback policies are separate decisions.
