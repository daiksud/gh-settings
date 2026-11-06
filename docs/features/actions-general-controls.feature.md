---
type: Feature
title: Actions cache limits and private sharing access
---

The [configuration reference](../settings.md#actions-cache-policy) owns YAML semantics and billing cautions; the [command reference](../commands.md#export-behavior) owns explicit export selection. These are three independent controls, not additions to the meaning of `--actions` or `--retention`.

## Declaration and acquisition

- Cache retention and size are optional whole integers; their bounds are 1–365 days and 1–10000 GB. Public retention has an additional 90-day cap. Empty cache sections are unmanaged; zero, null, fractional and string values fail validation.
- Access accepts only the three UI choices. Observation and apply require private visibility and the appropriate User/Organization owner type; public/internal and unsupported owner metadata cannot become an empty policy. Raw REST enum spellings are not YAML aliases.
- Each export flag reads only its selected resource, in addition to ordinary export. Duplicate flags fail before remote work. Every selected read and conversion must succeed before saving; denied, missing, malformed or incomplete responses preserve the destination.
- Plan reads only declared controls and is nonmutating. Numeric comparisons render numbers; access comparisons render UI choices. Same-version Action reports and credential-free Summary retain the existing validated comparison boundary.

## Endpoint boundaries

| Control | Repository-relative GET/PUT path | Only PUT member | Read permission |
| --- | --- | --- | --- |
| Cache retention | `actions/cache/retention-limit` | `max_cache_retention_days` | Administration read |
| Cache size eviction limit | `actions/cache/storage-limit` | `max_cache_size_gb` | Actions read |
| Private sharing access | `actions/permissions/access` | `access_level` | Administration read |

All three PUTs require Administration write. Cache retention and sharing access applicability also require repository metadata; cache storage reads only its limit endpoint. The [cache API](https://docs.github.com/en/rest/actions/cache) and [access API](https://docs.github.com/en/rest/actions/permissions#set-the-level-of-access-for-workflows-outside-of-the-repository) are the external contracts; installation-token documentation is not proof of a particular installation's grants or live write behavior.

## Apply and failure

Fresh complete current reads precede PUT. Matching controls skip mutation. Each write sends one independently expected native member and requires a separate GET confirming the requested value. No API-required field is invented, and a PUT response alone is not verification. Upper-policy refusal, denied access or different readback fails rather than clamping or pretending success. No retry, rollback, parent-policy change, cache deletion or billing configuration occurs. A later failure preserves earlier successful writes and any uncertain effect of the failed operation.

## Executable evidence

[TestActionsGeneralControlsExportPlanApply](../../tests/acceptance/actions_general_controls_test.go) launches the built CLI against isolated local API fixtures. It covers public cache retention/storage and both private-owner access choices: independent export, old-flag isolation, validate, matching/changed plan, typed reports and credential-free Summary, exact literal PUT bodies, application preflight followed by fresh transport GET/PUT/GET verification, repeat apply, omission isolation, destination-preserving incomplete/denied reads, upper-policy refusal and unchanged readback. These fixtures use dummy credentials, do not change live settings, and do not establish charges, cache eviction timing, hosted CI, published-package behavior or effective platform permissions.
