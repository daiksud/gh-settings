# Deploy keys

The [configuration contract](../settings.md#deploy-keys) owns the UI mapping,
permissions, public-key syntax, immutable lifecycle and pruning boundaries.

```gherkin
Feature: Manage repository deploy keys without implicit replacement
  Scenario: Export only a complete public-key inventory
    Given a personally owned public target and complete paginated keys
    When export acquires the family
    Then canonical public material and exact read/write grants include numeric IDs
    And unknown or incomplete acquisition retains prior intent or omits first export

  Scenario: Verify and bind same-title keys independently
    Given two new public keys with the same title and different grants
    When apply creates each key
    Then each exact canonical payload has its own independent GET and inventory verification
    And each confirmed ID is inserted against its original canonical key
    And source comments remain intact
    And repeating the bound declaration sends no write

  Scenario: Keep immutable replacement explicit
    Given a bound key is missing or its title, key or grant differs
    When plan or apply compares it
    Then the operation fails without adopting or recreating a key
    And ordinary empty declarations preserve all remote keys

  Scenario: Guard every explicit deletion
    Given a present deploy_keys list and deploy-keys or standalone all prune
    When complete preflight freezes the target, selected keys and unlisted candidates
    Then creation precedes deletion and verified own creations remain retained
    And each deletion rechecks complete inventory and exact candidate identity
    And unknown, false and true enabled observations remain distinct
    And only a 204 followed by complete absence confirms deletion
    And drift, unavailable reads or uncertain effects stop later operations

  Scenario: Separate recovery evidence from verified identity
    Given a successful create response contains a new positive ID
    But the acknowledgment or independent readback is incomplete
    Then apply reports recovery identity without local binding or success outputs
    And later work stops without rollback or retry
```

## Verification

| Owner | Evidence |
| --- | --- |
| [Model](../../internal/settings/deploy_keys_test.go) | Maintained parser, single-line/private/options guards, comments and canonical identity, immutable selection, enabled tri-state and dynamic metadata |
| [Transport](../../internal/github/deploy_keys_test.go) | Complete target and pages, required native fields, exact canonical URL, payload polarity, recovery-ID limits, independent detail/full inventory, target/state drift and fresh absence |
| [Application](../../internal/app/deploy_keys_test.go) | Independent export retention, global preflight, own-unarchive, fixed create-before-delete membership, no-op, binding failure and read/uncertain/partial outcomes |
| [Workspace](../../internal/workspace/created_deploy_key_ids_test.go) | Same-title key binding with comments, existing ID, alias/merge, duplicate material and concurrent edit guards |
| [Action report](../../internal/actions/deploy_keys_test.go) | Safe typed projection, strict shape, exact integer IDs, duplicate detection and managed count |
| [Built CLI](../../tests/acceptance/deploy_keys_test.go) | Literal export/create/bind/repeat/prune lifecycle, invalid private input before remote work, debug-enabled unverified recovery without writeback |
| [Actual supplied Action](../../tests/github-actions/composite_deploy_keys_acceptance_test.go) | Explicit credential/target, detailed/status-only output, unchanged YAML, create-only collision, bound no-op and output withholding on failed prune readback |

These are local API fixtures using static public material. They establish neither
live GitHub key acceptance, installed credentials, SSH usability nor published
Action behavior. No private key parsing or live key operations are part of testing.
