---
type: Feature
title: Declarative repository Rulesets design
description: Planned ownership, round-trip, reconciliation, and acceptance contract for existing Rulesets.
---

## Status and goal

Accepted design under [ADR 0025](../adr/0025-declarative-repository-rulesets.md), **not implemented or supported by any CLI/Action version**. Examples and command options below describe future implementation, not commands users can run today. Existing [settings support](../settings.md) is unchanged.

The adoption path is: export existing repository configuration, validate it, plan without changes and obtain zero Rulesets differences, edit the desired state, then reconcile creates/updates/deletes. The [labels contract](labels.feature.md) supplies collection membership and retained-attribute semantics; it does not establish Rulesets API identity or PUT behavior.

## Ownership and identity

| Object | Planned ownership and matching |
| --- | --- |
| Repository-owned branch/tag Ruleset | Member of `settings.rulesets`, identified by exact `name`; ID is observed transport identity, not YAML configuration. |
| Organization/enterprise/inherited Ruleset | Outside the managed collection; never exported as repository-owned, matched by name, updated, or deleted. |
| Repository push Ruleset | Outside the initial branch/tag collection; explicitly reported as excluded from Rulesets export coverage and never mutated. |
| Legacy branch protection | Separate cumulative policy; deferred under #338, with no automatic migration or modification. |
| Inner `rules` member | Identified by exact `type`; duplicate types are rejected. |
| Bypass actor | Identified by `(actor_type, actor_id)`; actor ID is absent for `OrganizationAdmin` and `DeployKey`. |
| Required status check | Identified by `(context, integration_id)`; absent integration means any source and differs from a specified integration. |
| Required reviewer / dismissal actor | Identified by `(reviewer.type, reviewer.id)` / `(type, id)` respectively; their scalar pattern lists are complete values when declared. |

Names are nonempty, non-whitespace-only UTF-8 strings without control characters. Do not trim, normalize, case-fold, or infer renames. Reject duplicate names in declarations or the complete owned branch/tag observation, including duplicates across different targets. Renaming a declared identity means create the new object and delete the old one; its ID/history is not retained. Compare the repository owner/name in `source` with the resolved checkout repository case-insensitively, independently of exact Ruleset-name identity: `Owner/Repo` and `owner/repo` identify the same GitHub repository. Require `source_type: Repository`; missing or contradictory source ownership/target/ID is an error, not permission to ignore an entry.

## Planned schema and presence

Use the existing `schemaVersion: 1` and resource grouping:

```yaml
schemaVersion: 1
settings:
  rulesets:
    - name: default
      target: branch
      enforcement: active
      bypass_actors: []
      conditions:
        ref_name:
          include: ["~DEFAULT_BRANCH"]
          exclude: []
      rules:
        - type: deletion
        - type: non_fast_forward
        - type: required_linear_history
```

This illustrative basic-only declaration is not an export of this repository's active protection. Removing PR/check rules from an authoritative `rules` list would remove those rules after that type is supported; do not use this example to replace existing policy.

| Declaration | Meaning |
| --- | --- |
| Omitted `settings.rulesets` | Unmanaged; plan/apply make no Rulesets requests. |
| `rulesets: null`, wrong shapes/types, duplicate keys, unknown YAML fields | Invalid before target/API access. Null is not accepted at any Rulesets YAML depth. |
| `rulesets: []` | Delete all owned branch/tag Rulesets after a complete supported read; no inherited, push, or legacy changes. |
| Nonempty Rulesets list | Authoritative owned branch/tag membership; create missing names, update retained members, delete undeclared owned members. |
| Omitted attribute on retained object | Preserve its observed value, including `target`, `enforcement`, conditions, bypass actors, and rules. |
| Declared `rules` or `bypass_actors` list | Authoritative inner membership; remove omitted members; `[]` removes all members. |
| Omitted `rules` or `bypass_actors` | Preserve the entire existing collection. |
| Omitted attribute on retained rule/actor/check | Preserve the observed attribute. Explicit `false` and `0` remain managed values where the API accepts them. |
| Declared scalar list, such as ref include/exclude | Own its entire value; removal removes that pattern and `[]` clears it if valid for that endpoint. Never infer a wildcard. |

Mappings are partial attribute declarations, not membership lists. An empty mapping manages no children and clears nothing. `parameters: {}` retains all parameters of an existing rule; it cannot satisfy required parameters of a new rule. An object collection member still requires its identity. Unknown keys, aliases/merges, duplicates and node types follow the existing strict YAML boundary.

For a newly created Ruleset require explicit `name`, `target`, `enforcement`, both ref condition lists, `bypass_actors`, and `rules`; no implicit active/default-branch/bypass policy is synthesized. A new rule or actor requires its documented request parameters, except genuinely optional API defaults may be delegated by omission. Validate all declaration shapes locally; existence-dependent creation requirements are checked after the fresh read and before any Rulesets mutation. Reject invalid branch/tag rule combinations and empty values the endpoint does not allow.

Bypass declarations use API actor types and IDs, not name lookup. A new actor must declare `bypass_mode`; a retained actor may omit it. Require positive numeric IDs for ID-bearing actor types; for `OrganizationAdmin` and `DeployKey`, omit YAML `actor_id` and map the API's ignored/null representation explicitly. Do not broaden bypass modes, resolve extra actors, or add grants automatically. Supported modes and account/target restrictions must match the selected API contract.

## Complete observation and export

- Planned `gh settings export --rulesets` adds Rulesets to the existing export. Ordinary export retains its current requests and permission requirements. This is a CLI export extension only: the public Action still accepts validate/plan/apply and does not gain an export command or input. Omitted opt-in excludes Rulesets, with no capability fallback.
- Inventory all pages of `GET /repos/{owner}/{repo}/rulesets?includes_parents=false&per_page=100&page=N`, validate repository ownership and IDs, then GET every repository-owned Ruleset by ID with `includes_parents=false`. Classify target from each detail response before validating branch/tag policy or excluding push policy; do not depend on an undocumented list-summary `target` field. Do not use effective branch-rule results: they omit disabled/evaluate policy and lose the editable Ruleset structure.
- Validate every required value, complete supported writable rule parameters and nested collections, unique identity, and ownership. Do not follow arbitrary response links for authenticated requests. An exact full page requires another page; later failure invalidates the observation.
- Missing `bypass_actors` means an incomplete observation, even if other fields are present. It never means no actors. Reject before saving or issuing Rulesets writes, including a declared empty outer list.
- Explicitly classify response-only metadata (`id`, `node_id`, source ownership, links, timestamps and other documented read-only fields); do not serialize it as desired state. Unknown content that might affect writable policy, unknown rules or parameters, and unrecognized enforcement/actor values fail the complete owned observation. Do not drop or blindly replay them.
- Export every supported owned branch/tag member and all its writable attributes, including explicit empty lists, booleans and numbers. Canonical ordering is by the identity keys above (numeric IDs numerically, absent optional IDs first); ref include/exclude, allowed merge methods and reviewer file patterns use exact string sorting for serialization and set equality, with duplicates rejected. Do not treat an arbitrary new API array as a set without verifying its contract. Mapping order has no semantic effect.
- Rules and actor/check lists compare by identity, not array position. Preserve meaningful parameter values and API defaults observed in complete reads; never normalize distinct policies into equality. API nullable identifiers, such as an any-source check integration, map to an absent optional YAML ID only after that equivalence is verified; missing required response fields remain errors.
- Finish all existing resource reads, Rulesets reads and serialization before opening the destination. Failure preserves the old configuration, with no partial export. Excluded push/inherited policy is outside the zero-difference guarantee, which concerns owned branch/tag desired state, not effective protection.

## Plan, apply and idempotence

Plan is read-only and reads Rulesets only when declared, completing observation before returning any comparison. Count the collection as one managed resource even when empty; render explicit create/update/delete identities, nested attribute/membership differences and protection-removal effects in the CLI and same-version Action report/Summary. Escape names and values; never present an inherited policy as a managed operation.

Apply retains repository → topics → labels behavior, then reconciles declared Rulesets using a new complete read with the apply credential. Precompute and validate the entire Rulesets operation sequence and all request bodies before its first mutation. Merge declared attributes into the fresh complete writable observation: omission preserves, declared inner membership removes absent members. PUT collections are assembled completely so that an omitted parameter or sibling cannot disappear accidentally. If preservation or a target change cannot be represented by the supported API, fail before Rulesets writes instead of recreating the object.

Execute creates/updates in exact-name order, followed by deletes in exact-name order, using POST for creates, PUT by observed ID for updates, and DELETE by observed ID for removals. Update only when managed desired values differ; unchanged declarations produce no Rulesets mutation. Do not send a name-only PUT for a zero diff. After a successful reconciliation, a fresh plan against the resulting state must have zero Rulesets differences; repeat apply must issue no Rulesets writes absent concurrent changes.

Stop on the first API error, reject redirects on reads/writes, and report completed, failed/uncertain and unattempted operations honestly. Earlier repository/topics/labels/Rulesets writes may remain applied. No rollback, retry, saved-plan execution, lock or drift protection is added. Reads may race with remote changes; a successful response alone does not prove effective protection or authorize bypassing it. Do not disable protection temporarily to make reconciliation or merging succeed.

## API, access and availability

[Repository Rulesets REST endpoints](https://docs.github.com/en/rest/repos/rules) document Metadata read for listing/details and Administration write for CRUD. Detail reads expose bypass actors only to callers with write access to the Ruleset: Metadata read alone is insufficient evidence of a complete export/plan observation. Require their actual presence; report insufficient visibility rather than requesting broader grants automatically. Apply uses the existing declared credential boundary, with no new credential discovery or permission expansion.

[Rulesets availability](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) depends on repository visibility and account plan: public repositories can use branch/tag Rulesets on Free plans, while private repository access requires an applicable Pro/Team/Enterprise plan; evaluate enforcement requires Enterprise. API rejection or unavailable capability is an error, not empty state or a reason to downgrade enforcement. Multiple applicable Rulesets and legacy protection are cumulative.

Push/file restrictions, merge queues, deployments, code-scanning/licensing rules, organization management, legacy protection and repository-wide API coverage remain outside the initial implementation. Selected API version, endpoint-specific writable fields/defaults, and null translation must be explicit in adapter tests. A read-only GET establishes observed state, not that replaying every returned field is accepted or preserves policy.

## Implementation stages and acceptance plan

[#335](https://github.com/daiksud/gh-settings/issues/335) completes this design only. No live policy, permission, workflow or release changes are part of it. Any future live API write verification needs separately authorized targets; local fixtures and read-only evidence do not prove write compatibility.

[#336](https://github.com/daiksud/gh-settings/issues/336) implements one complete basic vertical feature: declaration/codec, complete reads, explicit export opt-in, comparison/CRUD and CLI/Action presentation. The initial supported rule types are `deletion`, `non_fast_forward`, `required_linear_history` and `required_signatures`. It must reject owned observations containing PR/check or other unsupported rules, including during empty-list reconciliation, and cannot claim to codify those repositories yet.

[#337](https://github.com/daiksud/gh-settings/issues/337) adds `pull_request` and `required_status_checks`, including complete writable parameter coverage required for existing configuration. Cover review counts, dismissal/owner/last-push/thread settings, allowed merge methods, reviewer restrictions, required reviewers, check context/integration, strictness and creation behavior when exposed by the selected API. Newly returned parameters such as `require_extra_approval_for_unattributed_changes` require explicit writable/read-only classification and preservation evidence; do not silently drop them or claim a lossless round trip before resolving them. If the API cannot preserve a parameter, report that concrete limitation rather than declaring adoption complete.

| Boundary | Required fixtures and outcomes |
| --- | --- |
| Settings model / codec | Omitted/null/empty at each depth, false/0, partial attributes, duplicate names/types/actor/check identities, case-distinct names, no rename inference, creation requirements and invalid target combinations. |
| Read adapter | Multiple/exact pages, failed later page/detail, wrong owner/ID/target, same-name inheritance, equivalent-case repository ownership, list summaries without target followed by mixed branch/tag/push details, disabled/evaluate policy, missing bypass, null and wrong types. Inject unknown keys separately at every writable response mapping: Ruleset, conditions/ref_name, rule/parameters, bypass actor and nested PR/check objects; require rejection, while discarding only explicitly classified read-only metadata. No plausible partial observation. |
| Export / comparison | Complete writable export → decode → unchanged Rulesets plan equals zero; remote list/map reordering equals zero; a removed outer/inner member yields the exact delete/removal; omitted attributes remain unchanged. Failed export preserves the old file. |
| Apply adapter / sequencing | Exact POST/PUT/DELETE payloads/IDs, preserved siblings/required companion values, no injected false/empty defaults, all Rulesets requests validated before first write, fresh-state recomputation, sorted creates/updates before deletes, no mutation on zero diff, failure/redirect stops later operations. |
| CLI / Action | Explicit export opt-in, default export compatibility, managed count and escaped nested diffs/protection deletion, coherent private report/Summary validation and output on failures. |
| Built-product acceptance | Isolated existing-policy export → zero plan → edit/add/remove → exact CRUD → zero plan and no-write repeat apply; inherited/push/legacy objects untouched; mixed-resource partial failure and read-permission failure. |

Use local HTTP/files/Git fixtures with dummy credentials. Under #337, include a fixture shaped like an existing default-branch policy with bypass `[]`, three basic rules, PR approvals, and two integration-bound required checks, including every observed policy parameter. This must not install or modify the active repository Ruleset. Test mappings are added only when the corresponding tests exist; this design document itself is not executed by CI.
