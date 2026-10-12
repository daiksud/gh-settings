---
type: ADR
title: Use the YAML organization's Go YAML v4 library
description: Select go.yaml.in/yaml/v4 for configuration reading and writing.
sources:
  - id: yaml-upstream-policy
    resource: https://github.com/yaml/go-yaml/blob/823c494d8493a1c6533fd86120256f26e0ac5e57/README.md#version-intentions
    title: YAML organization go-yaml maintenance and version policy
  - id: yaml-v4-rc6
    resource: https://github.com/yaml/go-yaml/tree/v4.0.0-rc.6
    title: YAML v4.0.0-rc.6 tag
---

## Status

Accepted — agreed, not implemented.

## Context

The YAML dependency was previously undecided. The upstream project is maintained by the official YAML organization, not the Go team. Its version policy reserves v1, v2, and v3 for security fixes and directs ongoing development, including routine bug fixes, to v4.[^yaml-upstream-policy]

On 2026-10-12, the latest v4 tag observed was `v4.0.0-rc.6`, a release candidate.[^yaml-v4-rc6] The maintainer accepts prerelease risk because the intended use is limited to reading and writing YAML configuration. Starting with the upstream's development line avoids a planned major migration from v3 later; it does not guarantee that migration or adaptation will never be needed.

## Decision

Use `go.yaml.in/yaml/v4` for YAML configuration reading and writing.

## Consequences

The library choice follows the YAML organization's upstream direction. This selects the v4 series, not a fixed release-candidate version; a concrete dependency version will be pinned during implementation.

Release-candidate APIs and behavior may change, and defects may require adaptation. The limited reading/writing scope is a reason to accept that risk, not evidence that serious bugs are absent. No stability or absence-of-critical-bugs guarantee is made.

The choice does not change the [five-field settings schema](../settings.md), [declared-field ownership](0002-managed-fields.md), or [strict validation policy](0004-strict-validation.md). Null handling and future composition across update endpoints remain unresolved. Implementation must preserve omitted fields rather than inject defaults; choosing a parser does not settle additional configuration semantics.

This ADR adds no implementation, module files, or installed dependencies. Go and GitHub API library choices remain separate decisions in [ADR 0007](0007-go-implementation.md) and [ADR 0008](0008-official-go-gh-library.md).

[^yaml-upstream-policy]: [YAML organization go-yaml README](https://github.com/yaml/go-yaml/blob/823c494d8493a1c6533fd86120256f26e0ac5e57/README.md#version-intentions), checked on 2026-10-12 for official YAML organization maintenance, version policy, and v4 import path. The source SHA records evidence, not a dependency pin.
[^yaml-v4-rc6]: [YAML v4.0.0-rc.6 tag](https://github.com/yaml/go-yaml/tree/v4.0.0-rc.6), checked on 2026-10-12 against upstream v4 tags. This is an observed release candidate, not a guarantee or a selected exact version.
