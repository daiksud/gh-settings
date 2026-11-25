---
type: Reference
title: "Actions repository variables"
---

# Actions repository variables

`repository.settings.secrets_and_variables.actions.variables` corresponds to **Secrets and variables → Actions → Variables → Repository variables**. Use it for non-sensitive values shared across the repository.

The same Variables tab also lists **Environment variables**. Its **Manage environment variables** link opens Settings → Environments. Declare those values once under `repository.settings.environments[].variables` for the named [Environment](../environments.md#environment-variables).

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    secrets_and_variables:
      actions:
        variables:
          - name: BUILD_CHANNEL
            value: stable
          - name: OPTIONAL_LABEL
            value: nightly
```

Both `name` and nonempty string `value` are required. Names contain only ASCII letters, digits and underscores, cannot start with a digit or `GITHUB_` (case insensitive), and are unique ignoring case. Internal identity and export names are uppercase. Missing/null/empty values are invalid; quote values that YAML would interpret as numbers or booleans. Without prune, omission and `[]` select nothing and unlisted variables remain unchanged. `--prune actions-variables` makes a present list authoritative for retention; [] deletes all acquired variables, while omission always remains unmanaged. There is no YAML delete or rename member.

Plan reads each selected name individually and shows creates or value updates. Apply reads fresh state, creates absent names with POST, updates changed values with value-only PATCH, skips equal values, and requires exact individual GET readback after a mutation. Only GET 404 means absent under the caller-permissions precondition. Other failed/incomplete reads fail. Mutations and verification failures can have remote effects; earlier writes remain applied, without rollback, retry or concurrency protection.

Ordinary `export` lists all supported Actions repository variables through complete paginated inventory reads. Unavailable/incomplete/unsupported inventories retain prior valid declarations with info. Actions, Agents and Environment inventory pages reject duplicate members, ambiguous aliases of selected fields, malformed Unicode, trailing data and oversized responses. An unavailable preflight prevents deletion; an invalid readback after deletion leaves its effects uncertain. Values are **plaintext** in YAML, exports, detailed plans and Summary. Actions, Agents and Environment variable API requests suppress inherited `GH_DEBUG` transport logging to keep raw request and response bodies out of debug logs. Secrets and organization variables remain outside scope. Agents and Environment inventories are separate independent families; output does not classify or mask variable values.

GitHub's [Variables REST API](https://docs.github.com/en/rest/actions/variables) requires repository Variables read for reads and write for mutations with fine-grained/App tokens. An empty-value create was rejected with HTTP 422 (`Variable value cannot be empty.`); validation rejects empty desired values before requests. No whitespace-trimming rule is inferred. GitHub enforces its [variable limits](https://docs.github.com/en/actions/reference/workflows-and-actions/variables#limits-for-configuration-variables); server rejections are errors. No permission is granted automatically. Free/personal/public UI eligibility and API persistence were observed separately; this is not a runtime entitlement gate. See the [behavior specification](../../../development/features/repository-variables.feature.md).

## Variable pruning

`--prune actions-variables`, `--prune agents-variables` and
`--prune environment-variables` select independent collections. Their
comma-separated combination uses the common [prune contract](../pruning.md#collection-pruning).
Pruning requires complete name/value inventories, including every page (API
maximum 30 entries per page) and a stable, exact `total_count`. Duplicate names,
missing/null values, count drift and incomplete/denied pages fail before writes.
Prune acquires all current values in a selected collection, not just declared
names; values remain non-sensitive plaintext and can appear in detailed output.
Named export continues to read only its explicit selections.

All candidates are prepared before any remote write. Each family fresh-reads its
inventory, rejects changed candidates, upserts declared values before deleting
its fixed candidates, rechecks the exact acquired name/value before each DELETE,
and requires HTTP 204 plus fresh complete inventory absence. A concurrent change
can still race these checks; there is no lock, retry or rollback. Earlier writes
remain applied after later failure and uncertain DELETE/readback effects are
reported honestly. Plan and detailed Summary display deletion to `absent`, not
an empty value. Status-only output still hides names/values.

Environment-variable pruning touches only listed Environments with an explicit
`variables` array. `variables: []` prunes that Environment; an omitted array does
not. Existing-parent inventory and deletion are bound to its numeric identity,
so detected same-name parent replacement fails. A missing parent is an error
unless the same declaration explicitly creates its Environment policy first;
then no earlier variables are inferred and unexpected deletion candidates are
rejected. Variable-only intent never creates or deletes an Environment, changes
its policy/patterns or accesses secrets. [Environment-object deletion](../environments.md#environment-pruning)
uses the separate `environments` scope; standalone `all` selects both scopes,
while retaining every listed parent, including variable-only entries.

API evidence: [Actions variable lists/deletion](https://docs.github.com/rest/actions/variables)
and [Agents variable lists/deletion](https://docs.github.com/rest/agents/variables).

[Settings index](../README.md) · [GitHub App permissions](../permissions.md#secrets-and-variables)
