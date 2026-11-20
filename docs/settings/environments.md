---
type: Reference
title: "Environment settings"
description: "Environment settings declaration, ownership and API contracts."
---

[Settings index](README.md) · [GitHub App permissions](permissions.md)

## Environments

`repository.settings.environments` selects supported policies and variables by `name`, without numeric Environment/pattern ID writeback. Names are nonempty, have no leading/trailing whitespace/control characters and use Unicode simple folding for client identity. Spelling remains; slash/percent are URL-encoded as one segment. Without prune, omission/`[]` selects nothing and unlisted Environments are preserved. Explicit [Environment pruning](pruning.md#environment-pruning) adds bounded parent deletion; omission remains unmanaged. Deletion/rename markers and secret values remain unsupported. Already installed custom protection Apps use the separate checkbox ownership below; installation and grants remain excluded. Selected non-sensitive variables have the separate ownership described below.

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: production
        deployment_protection_rules:
          wait_timer: 0
          required_reviewers: []
          prevent_self_review: false
          allow_administrators_to_bypass_configured_protection_rules: false
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules:
          - {ref_type: branch, name_pattern: 'release/*'}
          - {ref_type: tag, name_pattern: 'v*'}
```

For an existing policy-selected Environment, only declared built-in controls are owned. Matching partial and name-only entries are no-ops. When a built-in change needs PUT, apply overlays the original partial request onto the latest complete supported timer, reviewers, self-review, administrator-bypass and deployment-mode observation. It sends and independently verifies those preservation companions without adding them to desired intent. Plan and Summary show the partial request, not an invented final state. Creation still sends only declared controls; omitted creation values remain API-controlled, and a missing name-only entry sends an empty create-or-update body.

`deployment_protection_rules` contains optional `wait_timer` (0–43200 minutes), up to six unique `required_reviewers` (each `{person: positive-id}` or `{team: positive-id}`), `prevent_self_review` and `allow_administrators_to_bypass_configured_protection_rules`. True prevent-self-review cannot accompany an explicitly empty reviewer set. An existing partial update that conflicts with freshly preserved reviewers or self-review fails before writing; apply never clears an unowned dependency implicitly. No null is accepted. Reviewer eligibility, grants and entitlement remain GitHub prerequisites; principals are not invited.

`deployment_branches_and_tags` is `no_restriction`, `protected_branches_only` or `selected_branches_and_tags`. There is no public raw boolean-pair/null alias. Protected mode may permit all branches when no branch protections exist; configured intent does not prove protection.

`deployment_branch_or_tag_rules` is an authoritative list of `{ref_type: branch|tag, name_pattern: pattern}`. Type+pattern is case-sensitive identity; branch and tag may share a pattern. Omission makes no pattern CRUD; `[]` removes all observed patterns. Omitted mode plus a list requires existing selected mode; creating with patterns needs explicit selected mode. Explicit incompatible modes fail. Patterns are a separate API collection, not PUT members. When selected mode stays selected and the list is omitted, complete raw pattern IDs, types and names are retained and verified without pattern CRUD or unowned warnings, including unsupported ref types and projection collisions. Changing from a raw custom mode to No restriction or Protected branches only is rejected while any raw pattern exists and the list is omitted. To remove those patterns deliberately, first apply `deployment_branch_or_tag_rules: []` while Selected, then change mode. A complete known-empty collection permits the transition. GitHub can discard the patterns when custom mode is disabled; the client does not probe inactive state by temporarily changing mode or treat an inaccessible pattern endpoint as empty.

Reads require complete inventory, built-in/custom-protection details and typed paginated patterns when applicable; missing type is not assumed branch. Duplicate JSON members, ambiguous identity aliases, malformed Unicode and incomplete or oversized responses cannot establish equality or preservation. Parent/pattern GETs and parent PUT/pattern POST acknowledgments require HTTP 200. Unknown or malformed protection fails instead of being erased. Apply preflights and reconciles in stable name order. A pattern-only change skips policy PUT, verifies existing custom mode, creates missing patterns and deletes obsolete patterns by verified IDs. Retained pairs keep IDs. Complete fresh readback verifies original requested values/list and the preserved built-in snapshot; after a policy PUT it also verifies every sent companion. Duplicate-create 303, unavailable reads and mismatches fail; earlier/uncertain effects remain without retry, rollback or an intermediate-enforcement guarantee.

Administrator bypass is a real writable setting. Export requires/retains its complete boolean, even false. A prior temporary public-Environment experiment confirmed explicit false/true and retained preceding values on omission with other members fixed; this is not a platform-wide omission or entitlement guarantee. No default true is inserted.

Ordinary `export` observes supported Environment policy, active selected patterns and enabled custom App IDs, without Environment/pattern/binding IDs. Variables are inventoried independently after complete name acquisition; policy failure does not prevent listing accessible variables. Existing Actions read, Administration write, custom-protection visibility and account eligibility still apply. See [Environment behavior](../features/environments.feature.md).

### Custom protection Apps

`deployment_protection_rules.custom_rules` is an optional set of up to six unique
positive GitHub App IDs. It selects the already installed Apps checked under
**Deployment protection rules** in the Environment UI:

```yaml
repository:
  settings:
    environments:
      - name: production
        deployment_protection_rules:
          custom_rules: [12345, 67890]
```

Omission preserves every freshly acquired binding. A present list owns the full
checkbox set; `[]` explicitly disables every supported binding in that Environment
without a prune flag. Matching and reordered sets make no binding write. App IDs
identify integrations; rule IDs identify individual enabled bindings and never
enter YAML or ID writeback. This does not install or uninstall Apps, grant access,
approve deployments or configure webhooks. Enabling a rule affects later deployment
gates according to that App. [GitHub documents the UI and prerequisites](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/configure-custom-protection-rules).

Export reads enabled bindings independently of App availability and emits `[]`
only after a complete known-empty observation. Missing/null fields, unsupported
shapes, ambiguous identities or failed reads retain the whole previous Environment
policy family, or omit it on first export. A retained enabled App does not need to
appear in the available list. Export never enables an App or follows integration URLs.

For an existing parent, apply validates all additions against complete available-App
pagination before the first Environment write. Parent creation can precede that
availability read; later failure reports the confirmed parent progress. App-only
changes send no parent PUT. Built-in and pattern changes preserve exact omitted
binding identities, and each App write preserves complete built-ins and patterns.
A changed repository, parent, App or binding identity stops later operations.

Apply adds before removing. A temporary custom-App union above six fails before
any Environment write: first apply an explicit reduced set, then apply the desired
replacement set, accounting for the protection change between those runs. The
published six-rule limit does not establish how built-ins count; no combined
capacity formula is invented. A server capacity refusal never triggers deletion
as recovery. This sequence is nontransactional and has no intermediate-enforcement
or atomic drift guarantee.

Enable sends only `integration_id`, requires 201 with matching identities, then
independently verifies the exact binding and complete enabled inventory. Disable
uses the fixed rule ID, requires an empty 204 and complete fresh absence with all
retained bindings unchanged. A recreated binding for the same App is drift, never
a fallback deletion target. Failed verification returns nonzero, retains any
positive acknowledged rule ID in recovery diagnostics and stops later operations;
no binding is written to YAML and no retry or rollback occurs. Read actual state
before retrying uncertain effects. See the [REST contracts](https://docs.github.com/en/rest/deployments/protection-rules).

Enabled reads need Actions read where authentication is required. Additions also
need Administration read for availability; changes need Administration write.
Fixture checks do not establish actual token grants or live nonempty App behavior.
Parent pruning still requires an explicitly known-empty custom protection list;
ordinary App support does not expand parent-deletion ownership.

### Environment variables

`repository.settings.environments[].variables` selects explicit non-sensitive values in **Settings → Environments → the selected Environment → Environment variables**:

```yaml
version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: production
        variables:
          - {name: DISPLAY_NAME, value: 'Production'}
```

The [variable name/value rules](actions.md#actions-repository-variables) apply: canonical uppercase names, unique ignoring ASCII case within each Environment, required nonempty string values and no YAML delete/rename member. The same name in different Environments or repository variable families remains independent. Without prune, omitted `variables` and `variables: []` make no variable requests and unlisted variables are preserved. Explicit `--prune environment-variables` follows the [variable pruning contract](actions.md#variable-pruning); omission stays unmanaged, while [] selects all variables in that listed Environment.

An entry containing `name` and `variables` alone selects **only variables**, including when the variable list is empty. It never sends an Environment policy PUT, changes protection or secrets, or creates a missing Environment implicitly. A nonempty variable selection requires that Environment to exist. To manage policy and variables together, include the desired policy controls in the same entry; policy upserts run first, then selected variable upserts. Existing name-only entries select no policy mutation; a missing name-only entry still selects creation. Combined declarations retain the existing-policy preservation and creation-omission rules above.

Without prune, plan reads selected names only. Prune additionally acquires complete selected inventories. When the same plan explicitly creates a missing Environment policy, its selected variables are shown as creates without requesting nonexistent variable endpoints. Apply checks the selected Environment identity, reads each variable fresh, creates an absent variable, updates a changed value, skips equality and verifies independent readback. The parent identity must have a positive ID and matching name; duplicate or ambiguous JSON fields, malformed Unicode, oversized or incomplete responses fail the check. Unrelated valid protection metadata remains unmanaged. Writes run in stable Environment/name order. Failed reads do not become empty state; later failures stop execution without rolling back earlier writes, and a failed write or verification can have uncertain effects.

Ordinary `export` inventories variables for every completely listed Environment. Each parent variable list is atomic; failed acquisition retains that parent’s prior declarations with info and other parents continue. Unknown parent inventory retains all prior scopes. Variable-only entries do not invent deployment policy. Values are **plaintext** in YAML, exports and detailed output; secret APIs are not used.

The [Environment Variables REST API](https://docs.github.com/en/rest/actions/variables#get-an-environment-variable) uses name-addressed `/repos/{owner}/{repo}/environments/{environment_name}/variables` endpoints. Fine-grained/App tokens need **Environments** read for variables and write for mutations, plus **Actions** read for the Environment identity check. No access is granted automatically. UI creation/update/deletion and API persistence were verified on a Free, personally owned public test repository on 2026-11-11; both rejected an empty value. This is bounded adoption evidence, not a runtime entitlement check or proof of workflow consumption. See the [behavior specification](../features/environments.feature.md#selected-environment-variables).

### Environment deployment mode observation normalization

Complete typed equal policy booleans `(false,false)` or `(true,true)` are outside the three UI choices. Project the named `no_restriction` fallback while retaining both acquired booleans separately from effective policy. A valid API null remains exact No restriction without provenance. Missing policy, missing/null/wrong-type boolean members, unknown controls, failed reads and incomplete pages remain acquisition errors.

Raw `custom_branch_policies:true` still requires complete paginated patterns and custom-protection checks before publishing an observation, including a projected equal pair. Keep those latent pattern IDs/values in separate acquired state; effective No restriction never exports or replays them as selected patterns. Acquired empty patterns differ from an incomplete read. Pattern ref-type observations follow the policy below; unrepresentable principal identities are not invented.

Export warns in folded environment-name order and saves only effective policy. Plan/detailed Summary label every Environment before-object as **effective before policy**. An explicitly selected normalized mode adds a contextual raw-pair warning and normalized/write-required UI decision, including fallback-equal No restriction. Only selected names are compared; unlisted names stay undisclosed and unwritten. Private reports retain closed typed pair metadata, actual partial requests, status-only privacy and the existing collection count one; they never supply apply input.

Apply verifies an explicit mode against the original nullable pair in the acknowledgment and an independent complete fresh read. Retained equal booleans cannot confirm canonical No restriction. Pattern intent under an unknown mode needs explicit Selected; neither fallback nor the raw custom flag proves active applicability. A canonical mode acknowledgment alone does not permit pattern CRUD if the intermediate fresh read retains an unknown mode. Writes remain ordered create-before-delete with verified pattern identities and honest partial effects.

An omitted mode remains unowned and emits no mode warning. A needed existing-policy PUT preserves its freshly observed canonical nullable pair as a companion. Equal raw booleans cannot be safely replayed through the supported PUT contract: when another built-in needs changing, apply fails before writing and requires an explicit supported mode. Matching partial or name-only intent may still skip without changing that raw pair. No effective fallback is used as preservation evidence. Malformed acquisition preserves export destinations and makes no Environment write; later failures retain actual prior resource and uncertain effects.
