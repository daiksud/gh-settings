# Internal GitHub Actions interfaces

## Scope

This page is for contributors changing the composite Action or its Go adapters. Consumers should use the [public Action guide](../github-actions.md) and the four [settings commands](../commands.md); the internal interfaces below are same-version wiring, not an independent compatibility API.

This page describes the Go interfaces in current source and published v0.11.0, including typed label operations and merge-method booleans. See the [consumer package boundary](../github-actions.md#labels-and-permissions). Source protocol tests use the built candidate binary.

The [composite Action](../../action.yml) installs one verified Go executable, resolves metadata, runs the selected command, renders a plan Summary when needed, then publishes success outputs. The plan report is temporary presentation data. The same executable owns metadata and Summary rendering; shell owns installation bootstrap and Actions wiring. [ADR 0023](../adr/0023-public-composite-action.md) records the public packaging decision, and the [testing guide](testing.md) describes verification boundaries.

## Source Actions metadata

The CLI provides `gh settings action metadata` for same-version CLI/Action wiring. This is separate from the four settings commands. The composite invokes it through the verified executable. It resolves the current consumer checkout, including subdirectories, using the shared Go origin rules and ignoring `GH_REPO`. Neither the executable directory nor an Action source directory selects the target; no `GITHUB_WORKSPACE` override mode is added. No settings file, token or API request is needed.

Success emits exactly four single-line `key=value` records: `config-sha` is the checkout's resolved commit, `target` is its origin's owner/repository, `cli-version` is the injected executable version and `cli-sha256` is the SHA-256 of the actual executing binary. No binary-path or version argument selects a different identity. Missing identity or CR/LF in its components fails before output; output errors also return a nonzero status with a stderr diagnostic. Write failures can leave partial output. This interface adds no old-CLI fallback or multi-version protocol. See the [metadata behavior and tests](../features/actions.feature.md#rule-source-metadata-identifies-the-consumer-and-executing-binary).

On the supported Linux runner, the digest reads the running image through `/proc/self/exe`, so replacement or deletion of its launch pathname does not select other bytes. Failure to read that reference fails the command. Other platforms use the executable pathname for native development checks and require that path to remain unchanged; this does not extend the Action's supported platforms.

## Source Actions plan report

The CLI provides `gh settings action plan REPORT_PATH` for same-version Actions presentation. It runs shared application planning once and writes JSON instead of terminal plan text. Supply a caller-owned runner-temporary path with an existing parent; new files use mode 0600. Success is silent and exits zero. Missing arguments or unavailable storage fail visibly; a planning failure remains nonzero even after its report is written. Write failures can leave partial output.

The [Actions-owned report](../../internal/actions/plan.go) has these fields:

| Field | Meaning |
| --- | --- |
| `outcome` | `changed`, `no-difference`, `unmanaged`, `invalid-config`, `target-failure` or `read-failure`. |
| `target` | Available origin owner/repository; omitted for unmanaged, validation or target-resolution failure. |
| `managed` | Number of managed fields/collections in a successful comparison; labels counts once, including an empty declared list. Zero on failure does not mean unmanaged. |
| `changes` | Ordered repository/topic field/before/after entries; values remain native strings, booleans or topic string arrays. Field paths are `repository.FIELD` or `topics.names`. Empty comparisons, failures and label-only changes use an empty array. |
| `labels` | Optional typed create/update/delete entries with `operation`, `name` and applicable `before`/`after` attribute objects. Omitted when no label operations exist; available in v0.9.0. |
| `diagnostic` | Static categorical failure text; omitted on success. No raw error, token, environment dump or request header is serialized. |

Validation precedes target resolution and API access. Valid unmanaged declarations succeed locally without origin or credentials. Read failures include the already-resolved target; ordinary stderr diagnostics retain their existing text. The report is ephemeral presentation data, never apply input. The composite calls this entry with its explicit read token, then passes the report to a separate credential-free Summary step. Shared planning owns validation; no separate CLI validation call, multi-version format or saved-plan execution is introduced. See the [report behavior](../features/actions.feature.md#rule-source-planning-writes-ephemeral-typed-presentation-data).

### Actions execution-policy entries

Current source adds optional `actions`, an ordered array of the six closed `actions.FIELD` paths and native `before`/`after` values. Each declared field counts once, including an empty patterns list. Only inactive selected fields may have `before: null`: it means unobservable, not YAML null or an observed default. All three unavailable fields must accompany a nonselected-to-selected policy change. Summary validates types, exact enums, counts, duplicates, no-op changes and this transition as a unit. Pattern arrays preserve order and duplicates; they do not use topic-set canonicalization. This is same-version presentation data, never apply input.

### Label plan entries

The [labels report adapter](../../internal/actions/labels.go) keeps operation identity separate from attribute objects. `before` contains both observed `color` and `description`; `after` includes only attributes represented by the operation. Omitted attributes remain absent rather than being converted to empty strings or defaults.

| Operation | `before` | `after` |
| --- | --- | --- |
| `create` | Absent | Present, with only declared attributes; an empty object is valid. |
| `update` | Present, with both observed attributes | Present, with at least one changed declared attribute. |
| `delete` | Present, with both observed attributes | Absent |

The application emits creates/updates in label identity order, then deletes in identity order. Report validation rejects unknown/null attributes, malformed operation fields, duplicate identities, incomplete before values, invalid colors/descriptions, no-op updates and rename attempts. Colors must be lowercase. When label operations exist, the managed count must be at least the number of repository/topic change entries plus one; the actual count includes all declared fields and collections. A changed outcome may contain label operations alone. This is ephemeral presentation data, never apply input.

## Source Actions Summary

The CLI provides `gh settings action summary REPORT_PATH SUMMARY_PATH`. It reads the same-version typed report and explicit nonsecret run metadata, then appends review content to the caller-owned Summary path. It requires no checkout, settings file, credentials or API access and writes no GitHub outputs. The composite invokes this verified entry in a separate step with both token variables unset. Original nonzero plan status is preserved; otherwise Summary status determines success. The four public outputs follow both required stages.

The metadata allowlist is `SETTINGS_ACTOR`, `SETTINGS_TRIGGERING_ACTOR`, `SETTINGS_EVENT`, `SETTINGS_RUN_ID`, `SETTINGS_RUN_ATTEMPT`, `SETTINGS_WORKFLOW_REF`, `SETTINGS_WORKFLOW_SHA`, `SETTINGS_CONFIG_SHA`, `SETTINGS_TARGET`, `SETTINGS_CLI_VERSION` and `SETTINGS_CLI_SHA256`. All are required. `SETTINGS_ACTION_REPOSITORY` and `SETTINGS_ACTION_REF` must either both be present or both absent; absence displays an honest local-checkout identity. Other environment values are not inspected or included.

Run ID and attempt must be positive integers, workflow/configuration revisions 40 hex characters and the CLI digest 64 hex characters. Workflow references contain an owner/repository/path and `@refs/` reference; target and optional Action repository contain owner/repository. The file entry requires an explicit non-null `managed` count from zero through thirty-one in current source (twenty-two repository fields, topics, labels, Rulesets and six Actions execution-policy fields); published v0.16.0 retains twenty-one; published v0.15.x retains its eighteen-item limit; published v0.14.0 retains its seventeen-item limit; published v0.13.0 retains its fifteen-item limit; published v0.12.0 retains its thirteen-item limit; published v0.11.0 retains its eleven-item limit; published v0.10.0 retains its ten-item limit and v0.9.0 its seven-item limit. The renderer rejects inconsistent report states, counts or targets, mismatched categorical failure diagnostics, unknown or duplicate fields, no-op changes and field/value-type mismatches for the supported fields. Topic arrays must be sorted, unique and valid, and equality is checked without comparing slices as interface values. Empty strings and false remain valid change values for their existing fields. Squash and normal merge commit title/message changes require supported string enums on both sides; empty, unknown or wrongly typed values are invalid. Workflow links use the workflow repository/revision; configuration links use the explicit target/configuration revision. Original and rerun actors, Action identity and CLI identity remain distinct.

Repository field names, native types, enum policies, and the scalar field count come from the [static settings definitions](../../internal/settings/repository_fields.go). Report validation adds one managed unit each for topics and labels; report-state and collection validation remain in `internal/actions`.

The JSON DTO is converted once into a private validated report containing native `settings.Change` values and a `settings.LabelChanges` sequence. Validation retains the checked outcome, diagnostic, counts, topic arrays, and label attribute presence. Rendering consumes this result through typed visitors, without inferring types from JSON values or reconstructing label operations. Metadata validation and Summary file appending retain their separate boundaries.

The [pure renderer](../../internal/actions/summary.go) classifies typed outcomes rather than terminal sentences. It quotes HTML and multiline data, retaining native observed/desired values and the apply overwrite warning. Label operations use explicit create/update/delete names, with an issue/PR association-loss warning for deletes. Label membership is reconciled against fresh state during apply, so the report does not bind later operations. Missing/malformed input or required metadata fails before opening the destination, preserving existing content. A blank-line separator starts the appended section even when prior text has no trailing newline. New files use mode 0600; append, write and close failures return nonzero, without atomicity or rollback guarantees. Success is silent. A valid failed-plan report is rendered and appended but still exits nonzero. Rendering/writing does not prove that GitHub published the Summary; review it before separately authorizing apply. See the [Summary behavior](../features/actions.feature.md#rule-source-summary-renders-typed-review-data-without-credentials).

Current-source Rulesets operations use a separate `rulesets` array of operation names and complete typed before/after policies, without transport IDs. The validator rejects malformed/unknown policy fields, nulls, incomplete values, no-op updates, duplicate identities, incorrect operation ordering and inconsistent managed counts. CLI and Summary display nested policy changes and outer protection deletions; reports remain presentation data and never enter apply.
