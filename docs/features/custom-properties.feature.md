# Custom property values

Published v0.18.0 implements the bounded value contract in [ADR0030](../adr/0030-schema-bound-custom-property-values.md). Published v0.17.0 and older packages reject `settings.custom_properties` and `export --custom-properties`. Required/default-backed/explicitly-required properties and the other unadopted constraints remain tracked in #354. See the [configuration reference](../settings.md#custom-property-values) for syntax.

## Ownership and constraints

The map owns named values only. Omission and `{}` own nothing; null explicitly unsets one property; `""` and `[]` remain distinct values. Names are case-sensitive, unique and validated locally. Multi-select values are unique, sorted sets. True/false properties use quoted `"true"` or `"false"`, not YAML booleans. All requested values must match a completely observed schema before comparison or mutation.

Adopted schemas are organization-sourced, editable by organization and repository actors, optional, not explicitly required, and have no default. Supported types are string, single-select, multi-select and true/false. Selections must belong to observed allowed values. String regex is read through GraphQL, with organization ID/login and property name/type checks. Nonempty regex is unsupported; missing data or partial GraphQL errors cannot establish an unconstrained string.

Required/default-backed values, explicit-value requirements, enterprise schemas, organization-only editing, URL and regex-constrained types remain outside the adopted subset. In particular, effective values do not establish explicit-versus-inherited provenance. Schema management never accompanies a value update.

## Observation and apply

Read the repository identity, organization schema, selected string constraints and complete repository values, then recheck identity. Denied/404/redirected/malformed/incomplete collections fail; they do not mean no properties. Unknown names, duplicate schemas/values and values without a schema fail. Full export validates every observed property before saving. Repeatable `--custom-property NAME` validates and exports only named schemas, while still requiring complete collections; unsupported unselected schemas and unreadable unselected string constraints are not declared or validated. Plan and apply carry the declaration's selected names into their constraint reads. A missing, unsupported or unreadable selected name fails before mutation. Both export modes preserve an existing file on read/validation failure, and their successful output validates and plans with zero differences.

Plan is read-only, waits for all required reads, and shows sorted native values; properties count as one managed collection. Apply follows existing resource writes and precedes final archive. Explicit unarchive occurs first. Fresh apply comparison skips a match, otherwise sends only declared property entries using actual JSON null or empty arrays. A second complete observation immediately before PATCH rejects changed schema, values or identity. Successful readback must preserve unowned values and confirm selected schemas, target identity and requested values.

Failure before the property mutation is reported separately from an attempted update with uncertain outcome. Earlier successful writes remain applied. Failed/ignored updates, unexpected status, redirected writes, mismatching readback and concurrent changes stop processing without retry or rollback. These checks do not provide a transaction or an atomic schema/identity lock.

## Evidence

| Boundary | Scenarios | Tests |
| --- | --- | --- |
| Declaration/model | Null versus empty values, canonical selections, unknown observation, schema/default/editability/type restrictions and defensive copies | [Model](../../internal/settings/properties_test.go), [codec](../../internal/workspace/properties_test.go) |
| API | Declared-only PATCH, complete constraints, GraphQL partial/missing/identity failures, fresh schema/value checks, permissions/404, redirects, archive, readback and unowned preservation | [Adapter](../../internal/github/properties_test.go) |
| Application | Selective reads, managed count, local empty map, lifecycle ordering and honest partial results | [Application](../../internal/app/properties_test.go) |
| CLI/Action | Full versus repeatable named export selection, invalid/duplicate/mixed selection rejection, typed null/array report and closed Summary validation | [CLI](../../internal/cli/properties_test.go), [Action](../../internal/actions/properties_test.go) |
| Product binary | Full and selected export→validate→zero plan, unsupported-unselected isolation, selected missing/unsupported errors, explicit unset/empty apply, idempotence, failed-export preservation, uncertainty, earlier-write preservation and credential-free private Summary | [Acceptance](../../tests/acceptance/properties_test.go) |

These tests use local fixtures and dummy credentials. They do not establish live permissions, actual GitHub schema completeness, live unset/empty-value behavior, organization policy or released package support. No real API mutation is performed.

## Official contracts

- [Repository custom properties REST](https://docs.github.com/en/rest/repos/custom-properties): Metadata read for values; Custom properties write for declared value updates; string/array/null union and null unset.
- [Organization schema REST](https://docs.github.com/en/rest/orgs/custom-properties): organization Custom properties read, schema constraints and editability; no schema write is used.
- [GraphQL repository types](https://docs.github.com/en/graphql/reference/repos#repositorycustomproperty) and [organization lookup](https://docs.github.com/en/graphql/reference/orgs#organization): nullable string regex and named schema lookup. Complete authenticated visibility is required.
- [Managing custom properties](https://docs.github.com/en/organizations/managing-organization-settings/managing-custom-properties-for-repositories-in-your-organization): names/value limits, required defaults and explicit-value rules.
