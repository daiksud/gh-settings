# Feature: Manage repository Actions cache limits

A repository owner can manage the cache retention period and cache size eviction limit independently. The [cache reference](../../references/settings/actions/general.md#actions-cache-policy) defines their YAML paths, bounds and billing implications. Both controls participate in [automatic export](readable-settings-export.feature.md).

## Rule: Acquire each limit independently

### Scenario: Retain declared retention when its value cannot be read

- Given a valid file declares cache retention and the size limit
- And GitHub returns a complete size limit but denies the retention read
- When the owner runs `gh settings export`
- Then export writes the observed size limit and retains the prior retention declaration
- And it reports the unavailable observation

### Scenario: Validate numeric controls

- Given a declaration sets `cache_retention` to a string, a fraction, zero or a value above 365
- When the owner validates the declaration
- Then validation fails locally
- And the same type checks apply to `cache_size_eviction_limit`, whose range is 1–10000 GB

Public repositories have an additional retention limit of 90 days. A declared empty cache section selects neither control. Acquisition checks the applicable repository metadata for retention. Malformed or ambiguous endpoint responses cannot establish a matching value.

## Rule: Apply only the selected limit and verify its value

### Scenario: Change the size limit from 10 to 20 GB

- Given the repository has a 10 GB cache size limit
- And the declaration selects only a 20 GB limit
- When the owner runs plan, apply and then plan again
- Then the first plan shows the numeric difference
- And apply sends only `max_cache_size_gb` to the cache storage-limit endpoint
- And a fresh read confirms 20 before the second plan reports no difference
- And the retention value stays unmanaged

Cache retention uses `actions/cache/retention-limit` and the single PUT member `max_cache_retention_days`. Cache size uses `actions/cache/storage-limit` and `max_cache_size_gb`. Retention reads require Administration read, size reads require Actions read, and either write requires Administration write. See the [permission matrix](../../references/settings/permissions.md) and [GitHub cache API](https://docs.github.com/en/rest/actions/cache).

### Scenario: The changed value cannot be confirmed

- Given a requested limit differs from the acquired value
- When GitHub denies the PUT or the independent readback returns the old value
- Then apply fails and reports the operation
- And it neither retries the write nor rolls back an earlier successful change

## Test mapping

The [CLI process fixtures](../../../tests/acceptance/actions_general_controls_test.go) exercise independent export and retention, typed plans and Action summaries, exact PUT bodies, readback, repeated apply, omission and failure. The [cache adapters](../../../internal/github/actions_cache_test.go) reject ambiguous response fields. [Archived CLI](../../../tests/acceptance/archive_cache_test.go) and [Action fixtures](../../../tests/github-actions/composite_archived_cache_acceptance_test.go) exercise defensive read-only matching outside ordinary v1 support. Local fixtures cannot establish live charges, eviction timing or permission grants.
