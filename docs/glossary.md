---
type: Glossary
title: Repository settings model
description: Distinguish repository declarations, observed state and comparison results.
sources:
  - id: ownership
    resource: concept.md#implementation-ownership
---

These terms describe the five managed repository fields in the [settings model](../internal/settings/repository.go). YAML and GitHub request/response conversions belong at their respective boundaries, outside this I/O-free model.

| Term | Meaning | Go type |
| --- | --- | --- |
| Repository declaration | Only the fields the maintainer chose to manage. A nil field is omitted; explicit false and empty strings remain declared values. `Empty` means all fields are omitted. | `settings.RepositoryDeclaration` |
| Observed repository state | All five normalized values from a successful repository read. Ordinary string and boolean values represent the complete state; empty strings and false are observed values. | `settings.RepositoryState` |
| Repository comparison | Managed-field count plus ordered changes between observed state and a declaration. A positive count with no changes differs from no managed fields. | `settings.RepositoryComparison` with `settings.RepositoryChange` entries |

The [GitHub read adapter](../internal/github/repository.go) owns client construction and response normalization. `NewReadClient` retains gh authentication and the 30-second read timeout; `Client.ReadRepository` reads an explicitly resolved target, checks completeness and normalizes permitted null strings to empty strings in the observed state. Export declares every observed field; PATCH includes only declared fields. The [configuration codec](../internal/workspace/config.go) validates and encodes the [existing versioned/grouped YAML contract](settings.md#configuration-semantics), returning only a repository declaration to callers.

The [GitHub write adapter](../internal/github/apply.go) owns declaration-to-JSON conversion and direct PATCH to an explicitly resolved target, without a preliminary GET. `NewWriteClient` retains the separate 30-second write timeout and rejects redirects; `Client.ApplyRepository` sends only declared values. A network error does not establish whether the remote update happened.

The [workspace file boundary](../internal/workspace/config_file.go) owns the fixed checkout-root configuration path, reading and export writing. It delegates decoding to the pure codec and accepts already-encoded export bytes; GitHub access stays with its caller.

The [workspace Git boundary](../internal/workspace/repository.go) owns checkout-root, commit and checkout-scoped GitHub target resolution through `GitRoot`, `CurrentCommit` and `CurrentRepository`. It ignores `GH_REPO` and unscoped origins, retaining the [existing origin and transport-rewrite contract](commands.md#export-behavior).

The shared [local validation operation](../internal/app/validate.go), `app.Validate`, locates the checkout root and reads a typed repository declaration or returns an error. It does not resolve origin, configure authentication, access GitHub, write files or format CLI output. CLI validate adapts the error; plan and apply consume the validated declaration.

The shared [plan operation](../internal/app/plan.go) returns `app.PlanResult`: a native repository comparison and the target when fields are managed. After validation, an empty declaration returns a zero result without target resolution or a read. `app.PlanError` identifies validation, target or read failure while preserving original error text and unwrapping; read failures carry the already-resolved target while the result remains zero. Root wiring supplies the deferred GitHub read; the CLI renders the comparison. Plan and apply use the declaration's single `Empty` definition before remote work; explicit false and empty values remain managed.

The shared [apply operation](../internal/app/apply.go) returns `app.ApplyResult`: `Applied` is true only after writer success, while false without an error means local no-op. Validation and the empty-declaration guard precede target/writer access; root wiring supplies the deferred write and the CLI formats the message. False with an error does not prove no remote mutation, and later output failure does not roll back success.

The shared [export operation](../internal/app/export.go) converts complete observed state into a declaration of all five fields, serializes it and only then hands bytes to workspace storage. It does not validate the old file or format stdout. Plan/export share [deferred read wiring](../main.go), with no eager client initialization.

The [CLI adapter](../internal/cli/run.go), `cli.Run`, owns argument dispatch, help/version, typed Plan/Apply rendering, diagnostics and exit status. Its `Operations` value binds the four application signatures; export and validate remain silent. The [root entry point](../main.go) retains linker-injected `main.version` and concrete deferred dependencies, without command wrappers or application sequencing.

The [Actions adapter](../internal/actions/run.go), `actions.Run`, owns source-only metadata, plan-report and Summary arguments, diagnostics and status. Root dispatch passes its injected version and deferred shared planning operation directly. Metadata emits four nonsecret records with single-line identity validation; shared workspace functions resolve the consumer commit/target, and the adapter hashes the actual executing binary without a settings operation, authentication or API client. Actions owns [PlanReport and PlanChange](../internal/actions/plan.go), the same-version ephemeral JSON presentation of typed planning outcomes and native values. Static categorical JSON diagnostics are separate from original stderr errors. Apply never consumes the report. Actions-owned [RunMetadata and pure RenderSummary](../internal/actions/summary.go) separate explicit nonsecret run/identity context from typed plan data. The Summary adapter loads and validates before appending, keeps failed plans nonzero, and does not establish GitHub publication. The current unreleased v0.7.0 composite adopts metadata, authenticated plan-report generation and separate credential-free Summary rendering through its verified executable. Original nonzero plan status is preserved; otherwise Summary failure prevents success outputs. Released v0.6.1 callers retain their original implementation.

The pure [comparison](../internal/settings/comparison.go) retains native string/string or bool/bool value pairs. Terminal messages, quoting and arrows belong to the [CLI renderer](../internal/cli/comparison.go); apply consumes the declaration directly.
