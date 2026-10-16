---
type: Glossary
title: Repository settings model
description: Distinguish partial repository declarations from complete observed state.
sources:
  - id: ownership
    resource: concept.md#implementation-ownership
---

These terms describe the five managed repository fields in the [settings model](../internal/settings/repository.go). YAML and GitHub request/response conversions belong at their respective boundaries, outside this I/O-free model.

| Term | Meaning | Go type |
| --- | --- | --- |
| Repository declaration | Only the fields the maintainer chose to manage. A nil field is omitted; explicit false and empty strings remain declared values. | `settings.RepositoryDeclaration` |
| Observed repository state | All five normalized values from a successful repository read. Ordinary string and boolean values represent the complete state; empty strings and false are observed values. | `settings.RepositoryState` |

The GitHub read boundary checks completeness and normalizes permitted null strings to empty strings. Export declares every observed field; PATCH includes only declared fields. The [configuration semantics](settings.md#configuration-semantics) retain the existing versioned/grouped YAML contract.
