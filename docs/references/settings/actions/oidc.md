---
type: Reference
title: "OIDC subject claims"
---

# Actions OIDC subject template

Use `repository.settings.actions.oidc.subject_claim` for the **Subject claim**
page of a personally owned public repository:

```yaml
repository:
  settings:
    actions:
      oidc:
        subject_claim:
          use_default_template: false
          subject_claim_template: [repo, context, job_workflow_ref]
          use_immutable_subject_claim: true
```

The group has two independently selectable units: the complete active template
and the optional `use_immutable_subject_claim` boolean. Omission or an empty mapping
leaves it unmanaged. An immutable-only declaration is valid. If either template
field is selected, `use_default_template: false` requires an explicit ordered array;
`true` forbids that array and selects GitHub's default template. An array without
an explicit false mode is invalid. Keys are unique nonempty ASCII alphanumeric
or underscore strings; order and case are preserved. There is no closed claim
name list. Null, unsupported keys and coercions are rejected. An explicit
`[]` is accepted locally because neither the official API schema nor the inspected
form establishes a minimum; GitHub acceptance and exact readback are still required.

Selecting default deactivates the custom template. It does not promise to retain
hidden inactive keys that GitHub ignores in default mode. Changing a subject can
stop cloud authentication: coordinate the corresponding cloud trust configuration
before applying it. This tool does not issue tokens, run workflows, or change
cloud trust, credentials, grants or billing.

`use_immutable_subject_claim` requests the API configuration corresponding to the
UI checkbox, including explicit true or false. It does not certify effective token
format, universal UI editability or a reversible transition. GitHub documents
[mandatory immutable subjects](https://docs.github.com/en/actions/reference/security/oidc#immutable-subject-claims)
for some repositories independently of manual opt-in. gh-settings does not infer
eligibility from creation dates, rename/transfer history or the returned prefix.
The editable legacy control was observed unsaved on 2026-11-14; persisted legacy
opt-out has not been live-verified. GitHub decides whether a particular update is
permitted. A false configuration readback is not proof of a name-only cloud token.

Ordinary export acquires this family independently. Known default exports true
without keys; known custom exports false with its ordered keys. Known immutable
configuration exports its boolean, including false. A missing optional immutable
field remains unknown: export refreshes the readable active template and retains
only valid previous immutable intent, or omits that leaf on first export, with
bounded info. Retained intent is not acquired evidence. Whole-family transport,
identity or malformed-data failure retains compatible prior whole intent. Required
diagnostic delivery must succeed before saving.

Plan compares one managed group. Apply reacquires complete personal-public target
identity and active configuration, then skips known matching selected values.
Selecting immutable requires its known boolean even for comparison; unknown is
neither false nor a proven difference. A matching template-only declaration can
still skip with unknown immutable. A needed PUT requires both complete active
template and known immutable evidence. Immutable-only writes replay the fresh
active mode and exact ordered custom keys; template-only writes replay immutable.
Combined intent supplies both. Omission never silently selects a different mode.
Missing or malformed required state fails before writing. A successful PUT must
return HTTP 201 with a bounded empty acknowledgment, followed by independent GET
confirmation of the original repository/owner identity, archive state, requested
values and replayed active companions. The read-only prefix may legitimately
change and is not an equality guard. A failed write or readback reports uncertain effects;
earlier verified progress remains and later work stops. No rollback or automatic
retry occurs, and fresh reads are not a concurrency lock.

The official [repository OIDC API](https://docs.github.com/en/rest/actions/oidc)
uses Actions read for observation and Actions write for updates. Its read-only
subject prefix is neither YAML intent nor identity evidence; it and raw API bodies
are excluded from diagnostics even with `GH_DEBUG=api`. Current-source support
requires adoption through a package that includes this capability.

[All Actions settings](README.md) · [GitHub App permissions](../permissions.md#actions)
