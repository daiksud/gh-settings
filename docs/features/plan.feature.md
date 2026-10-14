---
type: Feature
title: Plan repository settings
description: Observe differences for declared repository fields without mutation.
---

## Feature: Plan declared settings

### Rule: Validate before reading GitHub

- Read `.github/settings.yml` from the Git root, including from a subdirectory.
- Reject missing files and invalid configuration before API access.
- Resolve the same GitHub.com origin as export and ignore `GH_REPO`.
- Read the current five fields using the existing GET normalization and validation.
- Never send PATCH or change the configuration file.

### Rule: Compare declared fields only

- Omitted fields are unmanaged and produce no difference.
- Explicit false and empty strings are managed values.
- Show changed fields in description, homepage, has_issues, has_projects, has_wiki order.
- Each line uses `settings.repository.FIELD: CURRENT → DESIRED`.
- Strings are double-quoted with Go string escaping; booleans use `true` or `false`.
- Equal managed values produce `No differences for managed repository settings.`
- A valid document without managed fields produces `No managed repository settings.`

### Rule: Report success and failures

- Successful comparison exits 0 whether differences exist or not.
- File, validation, origin, authentication, API, decoding, and output failures exit 1.
- Errors appear on standard error; failures before output leave standard output empty.

### Test mapping

Focused comparison and CLI tests cover declaration semantics, exact output, and failures. Process acceptance uses the already-built executable with an isolated local API fixture, checks exact exit codes and output, observes GET-only requests, and verifies configuration preservation.

| Behavior | Test |
| --- | --- |
| Five-field order, omission, false, empty strings, escaping, equality, no managed fields | `TestRepositoryPlan` |
| CLI operation selection and output failure | `TestRunSelectsOperation`, `TestPlanOutputFailure` |
| API, decoding, transport failures and GET-only access | `TestPlanSettingsReadFailures` |
| Validation before origin/authentication | `TestPlanCurrentValidatesFirst` |
| File/subdirectory, partial/empty declarations, GET nulls, API/output failures, no PATCH | `TestPlanAcceptance` |
| Export → edit → validate → plan | `TestExportEditValidatePlan` |
