---
type: ADR
title: Require a configuration schema version
description: Identify the YAML file format independently of the CLI release.
---

## Status

Accepted — implemented by export; validation is future work.

## Context

The configuration now groups settings under `settings.repository`. As its shape evolves, a reader needs to know which file format it is validating without guessing from the keys or the installed CLI version.

## Decision

Require root `schemaVersion: 1`, an integer identifying the configuration file format independently of the CLI release. `export` always emits it. The planned `validate` command will reject missing, mistyped, and unsupported versions; it will not infer a version from the rest of the document.

## Consequences

Earlier flat files require both a `schemaVersion: 1` root key and relocation of their five fields under `settings.repository`. The version is tool metadata, not a GitHub API field. Omitted and empty mappings inside `settings` retain their unmanaged meaning. Future format versions need an explicit decision and implementation; this ADR provides no automatic migration.
