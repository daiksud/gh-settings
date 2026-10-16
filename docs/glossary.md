---
type: Glossary
title: Repository settings model
description: Distinguish repository declarations, observed state and comparison results.
sources:
  - id: ownership
    resource: concept.md#implementation-ownership
---

These terms describe the five managed repository fields in the [settings model](../internal/settings/repository.go). YAML and GitHub request/response conversions belong at their respective boundaries, outside this I/O-free model.

| Term | Meaning | Go type |
| --- | --- | --- |
| Repository declaration | Only the fields the maintainer chose to manage. A nil field is omitted; explicit false and empty strings remain declared values. | `settings.RepositoryDeclaration` |
| Observed repository state | All five normalized values from a successful repository read. Ordinary string and boolean values represent the complete state; empty strings and false are observed values. | `settings.RepositoryState` |
| Repository comparison | Managed-field count plus ordered changes between observed state and a declaration. A positive count with no changes differs from no managed fields. | `settings.RepositoryComparison` with `settings.RepositoryChange` entries |

The GitHub read boundary checks completeness and normalizes permitted null strings to empty strings. Export declares every observed field; PATCH includes only declared fields. The [configuration semantics](settings.md#configuration-semantics) retain the existing versioned/grouped YAML contract.

The pure [comparison](../internal/settings/comparison.go) retains native string/string or bool/bool value pairs. Terminal messages, quoting and arrows belong to the [CLI renderer](../internal/cli/comparison.go); apply consumes the declaration directly.
