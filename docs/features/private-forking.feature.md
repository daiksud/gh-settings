# Private repository forking policy

Published v0.18.0 implements [ADR0032](../adr/0032-private-forking-applicability.md) and the [user contract](../settings.md#private-repository-forking). Current source adds the #415 explicit-unarchive correction; published v0.18.0 and v0.18.1 retain the earlier composition rejection. Published v0.17.0 and older packages reject the field and option. Tests below use local HTTP fixtures and dummy credentials, not live GitHub changes.

## Behavior and acceptance evidence

| Scenario | Required result | Owner |
| --- | --- | --- |
| Omitted, true, false and invalid types | Omitted means unmanaged; native booleans round-trip; null/string/number/collection fails before API | [Codec](../../internal/workspace/forking_test.go) |
| Archive composition | `archived: false` composes and unarchives before the fork update; `archived: true` and both export flags fail locally | [Codec](../../internal/workspace/forking_test.go), [CLI](../../internal/cli/forking_test.go), [application](../../internal/app/forking_test.go) |
| Applicability | Read-only plan may observe an archived target; mutation requires an active, non-fork, private organization repository with a complete positive organization policy | [Adapter](../../internal/github/forking_test.go) |
| Incomplete or denied observation | Missing/null/wrong-type/identity, public/internal/personal/fork targets and denied/404/redirected responses fail; no false/default fallback | [Adapter](../../internal/github/forking_test.go) |
| Read-only plan | Only declared policy adds reads and one native boolean comparison; unacquired state discards all results | [Application](../../internal/app/forking_test.go), [process](../../tests/acceptance/forking_test.go) |
| Fresh update | Re-observe target and organization policy; skip confirmed match; one changed scalar PATCH; no unrelated payload or organization mutation | [Adapter](../../internal/github/forking_test.go) |
| Concurrency and uncertainty | Identity/policy/state change before write stops; response/readback mismatch or denial after write fails with uncertain effect; no retries | [Adapter](../../internal/github/forking_test.go) |
| Partial failure | Earlier ordinary writes remain when later policy read/write fails; no rollback | [Application](../../internal/app/forking_test.go), [process](../../tests/acceptance/forking_test.go) |
| Export and presentation | Ordinary export performs no organization read; explicit export round-trips policy; failure preserves file; CLI/Action retain native booleans and private reports suppress details | [CLI](../../internal/cli/forking_test.go), [process](../../tests/acceptance/forking_test.go) |

## API contract

Use repository GET/PATCH and organization GET with the documented REST version `2026-03-10`. Repository Metadata read is needed for private observations; the mutation needs Administration write and an eligible repository administrator. Organization GET must expose the actual matching policy field; its baseline public accessibility is insufficient. See the official [repository endpoint](https://docs.github.com/en/rest/repos/repos#update-a-repository), [organization endpoint](https://docs.github.com/en/rest/orgs/orgs#get-an-organization) and [organization forking guide](https://docs.github.com/en/organizations/managing-organization-settings/managing-the-forking-policy-for-your-organization).

Repository observations before and after the organization GET must agree, including archive state. A changed apply performs another complete active-target observation before its PATCH and another after the validated response. These checks cannot atomically bind a name-addressed API. The observed boolean is repository configuration, not effective permission to create any particular fork. Internal repositories, fork networks, archive-after-policy composition and live policy/grant expansion are outside the adopted subset.

## Restriction reassessment matrix

| Declaration | Organization allow | Current repository | Plan | Apply |
| --- | --- | --- | --- | --- |
| `allow_forking: true` or `false` | true and completely observed | active | Compare | Skip a match or update narrowly |
| `allow_forking: true` or `false` | false, hidden, denied or unknown | any | Fail incomplete/inapplicable | Fail before a fork-setting write |
| `allow_forking` without lifecycle intent | true and completely observed | archived | Compare read-only | Fail before a fork-setting write |
| `allow_forking` with `archived: false` | true and completely observed | archived | Compare unarchive first, then fork policy | Unarchive first, then re-observe and update |
| `allow_forking` with `archived: true` | any | any | Reject locally | Reject locally |

GitHub's repository policy guide is explicit that organization allowance is required before either allowing or disallowing forks for a repository, so the positive organization-policy requirement is retained. GitHub documents archived repositories as read-only and explicit false as the unarchive operation; the established lifecycle phase therefore justifies the focused unarchive-first composition. Fixtures prove ordering and failure behavior, not live token grants or mutation acceptance.
