# Architecture and design

This page explains the design for contributors. The [settings reference](../settings.md), [command reference](../commands.md), and [GitHub Actions guide](../github-actions.md) define the interfaces users work with; [development](development.md) describes how to extend them.

## Purpose and product boundary

gh-settings lets repository maintainers record intended settings in version control, review differences, and apply declared values. It is inspired by [babarot/gh-infra](https://github.com/babarot/gh-infra), [noirbizarre/gh-settings](https://github.com/noirbizarre/gh-settings), and [the GitHub Terraform provider](https://github.com/integrations/terraform-provider-github), rather than being a fork of those implementations.

Configuration lives at the current checkout's root `.github/settings.yml` and targets only that checkout's GitHub.com repository. Current resources are repository fields and topics. Organization settings, inherited organization rulesets, and management of multiple repositories are outside this product boundary.

Written fields are managed and authoritative; omitted fields remain unmanaged. Configuration keys and value types follow the corresponding update request schema, within the explicit versioned resource grouping. A repository GET response does not define the writable configuration schema. The [settings reference](../settings.md#configuration-semantics) owns exact field, omission, null, and grouping behavior.

Observation and mutation are separate operations. Plan compares declared values with observed state; apply sends the declaration directly. A plan is informational and cannot guarantee what later remote state will be. Failures remain visible, without transactional or automatic recovery guarantees.

## Implementation ownership

One Go implementation serves thin CLI and Actions adapters. Keep declaration, observed state, and comparison distinct; the [glossary](glossary.md) defines their types and meanings.

| Owner | Responsibility | Main implementation |
| --- | --- | --- |
| Settings model | Optional declarations, complete observed repository values, topic sets, presence, and ordered native comparisons; no I/O or YAML/JSON/presentation tags | [Repository model](../../internal/settings/repository.go), [resources/topics](../../internal/settings/settings.go), [comparison](../../internal/settings/comparison.go) |
| Workspace YAML | Decode and encode the versioned/grouped contract through private YAML mappings | [Codec](../../internal/workspace/config.go) |
| Workspace files and Git | Fixed configuration path, file reads/writes, checkout root, commit, and checkout-scoped target resolution | [Files](../../internal/workspace/config_file.go), [Git](../../internal/workspace/repository.go) |
| GitHub adapter | Client configuration, complete read normalization, declared-only PATCH, topics GET/PUT, and write redirect handling | [Repository reads](../../internal/github/repository.go), [repository writes](../../internal/github/apply.go), [topics](../../internal/github/topics.go) |
| Shared application | Validation, resource selection, operation sequencing, and typed results | [Application operations](../../internal/app/), [deferred readers/writers](../../internal/app/resources.go) |
| CLI adapter | Arguments, help/version, terminal presentation, diagnostics, and process exit status | [Dispatch](../../internal/cli/run.go), [comparison rendering](../../internal/cli/comparison.go) |
| Actions adapter | Same-version metadata, ephemeral plan reports, validated Summary rendering, and status | [Dispatch](../../internal/actions/run.go), [reports](../../internal/actions/plan.go), [Summary](../../internal/actions/summary.go) |
| Composite Action | Verified installation bootstrap and necessary Actions input/token/status/output wiring | [Root Action](../../action.yml), [installer](../../action/install-cli.sh) |

The [root entry point](../../main.go) binds both adapters to shared application operations and concrete deferred GitHub functions. A needed read or write constructs its API client; local validation, help/version, and empty CLI plan/apply do not eagerly configure authentication. Keep use-case sequencing in the application rather than moving it into root dispatch, shell, or presentation code.

### Workspace and GitHub boundaries

The workspace codec returns a typed declaration. Its file boundary accepts already encoded export bytes and owns writing; it does not contact GitHub. Git resolution uses the checkout's local/worktree origin, including repository-specific configuration, ignores `GH_REPO` and unscoped origins, and retains the direct repository identity even when Git rewrites transport URLs.

The GitHub read adapter requires complete repository values, normalizes permitted null strings to empty strings, and rejects missing fields or null booleans. The write adapter maps only declared fields to its private request DTO and sends no preliminary GET. Read and write clients retain separate 30-second timeouts; writes reject redirects. A network error does not establish whether a remote update occurred.

### Shared operation flow

| Operation | Application behavior |
| --- | --- |
| Validate | Find the checkout root, read configuration, and return a typed declaration; no origin, authentication, API access, or output formatting |
| Plan | Validate; return a local no-op for an empty declaration; otherwise resolve the target, read only managed resources, and return native comparisons |
| Apply | Validate; return a local no-op for an empty declaration; otherwise resolve the target and send declared repository settings followed by declared topics, stopping on the first failure |
| Export | Resolve the checkout/target, read complete repository settings and topics, construct a complete declaration, serialize it, then write the configuration |

`app.PlanError` preserves the original error and identifies validation, target, or read failure; read failures carry the resolved target. `app.ApplyResult.Applied` is true only after all required writes succeed. False with an error does not prove that no earlier write succeeded, and later output failure does not undo an update.

### Adapter boundaries

The CLI owns terminal quoting, arrows, messages, and exit codes. Actions metadata resolves consumer identity and hashes the executing binary without a settings operation. Plan-report generation invokes shared planning once; a separate credential-free Summary entry renders its typed result without planning again.

Reports preserve native values and use categorical review diagnostics separately from original stderr errors. Apply never consumes a report. Failed plans remain nonzero after report/Summary output, and the composite exposes success outputs only after required work succeeds. See [Actions internals](actions-internals.md) for those entry contracts and [testing](testing.md) for their verification boundaries.

## v0 simplification contract

The completed [source-refactoring series](https://github.com/daiksud/gh-settings/issues/133) established shared Go operations and thin adapters while retaining the product requirements below.

- Keep schema versioning, resource grouping, partial declarations, false/empty values, accepted YAML aliases and merges, strict validation, and checkout/target isolation.
- Validate before remote work and omit unmanaged fields from updates. Apply remains direct, without saved-plan execution, drift reconciliation, automatic rollback, or retries.
- Preserve explicit credential ownership, release artifact verification, and the distinction among consumer, Action, and CLI identities.
- Put tests with the owner of a current requirement. Retired wrappers, internal interfaces, and historical fixture combinations do not require compatibility shims merely because they once existed.
- Extend useful concrete structure when a resource is requested; avoid speculative registries or a package for every concept.

Two deliberate behavior simplifications remain current: empty CLI plan/apply succeeds before target/authentication work, and origins must directly identify GitHub.com through supported HTTPS, `git@` SCP, or full SSH forms. Rewrite-only aliases and username-less SCP have no fallback mode. Exact user behavior belongs in the [command reference](../commands.md).

Topics was added as a separate concrete resource after that refactoring. An omitted names pointer is unmanaged, an explicit empty list clears the set, and comparisons/payloads use sorted names. Mixed apply sends repository PATCH before topics PUT and leaves earlier successful writes in place on later failure; the refactoring scope is not a prohibition on separately agreed product additions.

Published tags/assets remain immutable when source changes. Candidate-source tests and published-download verification have different subjects. [Releases](releases.md), [consumer adoption](../github-actions.md#published-package-and-consumer-pins), and [this repository's automation](repository-automation.md) have separate operational procedures.

## Design decisions

ADRs record the context and consequences of individual decisions. Read their status and superseding links when a historical consequence differs from today's contract.

| Concern | Decisions |
| --- | --- |
| Delivery and implementation | [GitHub CLI extension](../adr/0001-github-cli-extension.md), [Go](../adr/0007-go-implementation.md), [official go-gh library](../adr/0008-official-go-gh-library.md), [YAML v4](../adr/0009-yaml-v4-library.md) |
| Managed settings | [Declared-field ownership](../adr/0002-managed-fields.md), [update request schema](../adr/0006-update-api-schema.md), [explicit null](../adr/0019-reject-explicit-null.md), [resource grouping](../adr/0020-group-settings-by-resource.md), [schema version](../adr/0021-require-schema-version.md) |
| Mutation | [Direct apply](../adr/0003-direct-apply.md), [strict validation](../adr/0004-strict-validation.md), [no automatic rollback](../adr/0005-no-automatic-rollback.md) |
| Actions | [Caller approval and credential ownership](../adr/0022-owner-approved-actions.md), [public composite Action](../adr/0023-public-composite-action.md) |
| Distribution | [Release decisions and future versioning](releases.md#release-decisions) |

Prefer the Go standard library where sufficient; no command framework is mandated. The selected YAML implementation is maintained by the YAML organization, not the Go team. Its release-candidate dependency is an explicit choice recorded in ADR 0009; current dependency versions belong in [go.mod](../../go.mod).

## Future direction

Labels, rulesets, Pages, and Actions settings remain outside current support. The following label semantics are agreed for a future extension:

- Omit `labels` to leave the collection unmanaged.
- Provide a list to make membership authoritative, deleting remote labels not declared.
- Provide `labels: []` to delete all remote labels.
- Preserve omitted attributes, such as `description`, on retained labels.
- Identify labels by name; ambiguous matches are errors. There is no rename inference, and explicit renaming is deferred.

Deleting a label loses its assignments. That consequence belongs in the future feature's behavior and user guidance if label support is implemented.

Names also identify outer rulesets, ambiguous matches are errors, and no rename is inferred. Applying label-like membership or nested collection behavior to rulesets was proposed but **not confirmed**. Ruleset deletion semantics and inner-rule identity remain unresolved.

Null handling for future fields, new output options, new-object required fields, and inner-rule identity require their own decisions. Current configuration and authentication contracts do not resolve those future choices. [Automatic version selection](releases.md#future-automatic-versioning) is also future work.
