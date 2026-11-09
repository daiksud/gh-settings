---
type: Feature
title: Manage selected Actions repository variables
description: Upsert explicitly named non-sensitive repository variables while preserving unlisted variables.
---

## Feature: Manage Actions repository variables

### Rule: Select named non-sensitive values

- `repository.settings.actions.variables` is an optional array of required `name` and nonempty string `value` pairs.
- Omission and `[]` select nothing and make no variable requests. Unlisted variables remain unchanged.
- Names contain ASCII letters, digits and underscores, start with a letter or underscore, and cannot start with `GITHUB_` (case insensitive). Identity is case insensitive; canonical names are uppercase.
- Missing/null/empty values fail locally. GitHub rejected an empty value with HTTP 422 in the live probe; no whitespace-trimming restriction is inferred. Values are plaintext and must contain only non-sensitive information. No secret detection or masking is promised.
- Only repository Actions variables are supported. Secrets, Environment, organization and Agents variables are excluded. There is no delete or rename declaration.

### Rule: Read selected names and verify writes

- Plan reads only declared names. Only HTTP 404 means absent; denied, malformed or incomplete reads fail.
- Apply reads fresh selected names, creates absent variables, updates changed values and skips equal values. It verifies each successful write with an individual GET before continuing.
- Failures stop later writes; earlier changes remain applied. A failed mutation or verification can have an unknown remote effect. There is no rollback, retry or concurrency lock.
- Complete acquired GET string values remain observations, including an empty string if returned. Export rejects an observed empty value before saving because it cannot produce a supported nonempty declaration.
- Ordinary export makes no variable requests. Repeatable `--variable NAME` opts into individual reads of known non-sensitive variables. A missing selected export name fails without replacing output.
- Configuration, export and plan reports can contain the plaintext values; never use these interfaces for credentials or other secrets.

### API and permissions

The [Variables REST API](https://docs.github.com/en/rest/actions/variables) provides individual GET, POST creation and PATCH update. Fine-grained tokens and GitHub App tokens require repository Variables read for reads and write for mutations. Credentials are supplied by the caller; implementation does not grant access.

The control was observed on GitHub Free, personally owned public `daiksud/test-gh-settings-public` on 2026-11-09. UI creation was read through the API, an API update appeared in the Variables tab, and the disposable variable was removed. This is adoption evidence, not a runtime entitlement check or proof of every credential's permissions.

### Test mapping

| Boundary | Evidence |
| --- | --- |
| Required value, strict YAML, omission, empty-value rejection, ASCII identity and round trip | [Codec tests](../../internal/workspace/ui_variables_test.go) |
| Selected GET, complete values, absence, exact write payloads and verification failures | [Transport tests](../../internal/github/variables_test.go) |
| Preserve Actions General during export projection | [Projection test](../../internal/github/ui_variables_export_test.go) |
| Read-only plan, selected export, sorted upsert progress and partial failure | [Application tests](../../internal/app/variables_test.go) |
| Explicit named export options and invalid selections | [CLI tests](../../internal/cli/variables_test.go) |
| Typed plaintext reports, escaped Summary and status-only privacy | [Report tests](../../internal/actions/variables_test.go) |
| Actual binary, exact API version, no redirects, create/update/skip, unlisted preservation and failed-export file preservation | [Process acceptance](../../tests/acceptance/variables_test.go) |

Automated tests use local fixtures and dummy credentials, without live repository changes. Live adoption evidence does not establish GitHub App credential permissions or workflow consumption of a variable.
