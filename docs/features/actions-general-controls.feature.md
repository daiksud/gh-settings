---
type: Feature
title: Actions cache limits
---

The [configuration reference](../settings/actions.md#actions-cache-policy) owns YAML semantics and billing cautions; the [command reference](../commands.md#export-behavior) owns explicit export selection. These are three independent controls, not additions to the meaning of `--actions` or `--retention`.

## Declaration and acquisition

- Cache retention and size are optional whole integers; their bounds are 1–365 days and 1–10000 GB. Public retention has an additional 90-day cap. Empty cache sections are unmanaged; zero, null, fractional and string values fail validation.
- Ordinary export attempts each supported control. Complete values are adopted independently; denied, missing, malformed or incomplete observations emit info and retain prior declarations without stopping other reads.
- Plan reads only declared controls and is nonmutating. Numeric comparisons render numbers. Same-version Action reports and credential-free Summary retain the existing validated comparison boundary.
- Cache and repository applicability responses must be bounded, valid UTF-8 JSON objects with an HTTP 200 status. Duplicate members and case aliases of selected fields cannot establish a match; malformed or unavailable observations fail rather than becoming plausible values. The shared private Actions access reader likewise rejects ambiguous owner identity metadata.

## Endpoint boundaries

| Control | Repository-relative GET/PUT path | Only PUT member | Read permission |
| --- | --- | --- | --- |
| Cache retention | `actions/cache/retention-limit` | `max_cache_retention_days` | Administration read |
| Cache size eviction limit | `actions/cache/storage-limit` | `max_cache_size_gb` | Actions read |

Both PUTs require Administration write. Cache retention applicability also require repository metadata; cache storage reads only its limit endpoint. The [cache API](https://docs.github.com/en/rest/actions/cache) is the external contracts; installation-token documentation is not proof of a particular installation's grants or live write behavior.

## Apply and failure

Fresh complete current reads precede PUT. Matching controls skip mutation. Each write sends one independently expected native member and requires a separate GET confirming the requested value. No API-required field is invented, and a PUT response alone is not verification. Upper-policy refusal, denied access or different readback fails rather than clamping or pretending success. No retry, rollback, parent-policy change, cache deletion or billing configuration occurs. A later failure preserves earlier successful writes and any uncertain effect of the failed operation.

## Executable evidence

[TestActionsGeneralControlsExportPlanApply](../../tests/acceptance/actions_general_controls_test.go) launches the built CLI against isolated local API fixtures. It covers public cache retention/storage: independent export, old-flag isolation, validate, matching/changed plan, typed reports and credential-free Summary, exact literal PUT bodies, application preflight followed by fresh transport GET/PUT/GET verification, repeat apply, omission isolation, destination-preserving incomplete/denied reads, upper-policy refusal and unchanged readback. [Cache adapter tests](../../internal/github/actions_cache_test.go) also cover ambiguous selected fields and shared access metadata. [Archived CLI](../../tests/acceptance/archive_cache_test.go) and [composite Action](../../tests/github-actions/composite_archived_cache_acceptance_test.go) fixtures verify matching and rejection without writes. These fixtures use dummy credentials, do not change live settings, and do not establish charges, cache eviction timing, hosted CI, published-package behavior or effective platform permissions.
