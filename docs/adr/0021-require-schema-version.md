---
type: ADR
title: Require a configuration schema version
description: Identify the YAML file format independently of the CLI release.
---

## Status

The public version marker decision is superseded by [Issue #479](https://github.com/daiksud/gh-settings/issues/479). The [current configuration reference](../settings.md) defines the required marker. Identifying the file format independently of the CLI release remains applicable. The original adoption status follows for historical context.

Accepted — implemented by export and validate.

## Context

The configuration now groups settings under `settings.repository`. As its shape evolves, a reader needs to know which file format it is validating without guessing from the keys or the installed CLI version.

## Decision

Require root `schemaVersion: 1`, an integer identifying the configuration file format independently of the CLI release. `export` always emits it. `validate` rejects missing, mistyped, and unsupported versions; it does not infer a version from the rest of the document.

## Consequences

Earlier flat files require both a `schemaVersion: 1` root key and relocation of their five fields under `settings.repository`. The version is tool metadata, not a GitHub API field. Omitted and empty mappings inside `settings` retain their unmanaged meaning. Future format versions need an explicit decision and implementation; this ADR provides no automatic migration.
