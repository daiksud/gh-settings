# GitHub Action reference

Call `daiksud/gh-settings` from a workflow to run one settings command against the repository checked out at `GITHUB_WORKSPACE`. The Action pin selects the CLI release and therefore the configuration fields the workflow accepts. The [setup guide](../guides/github-actions.md) covers runner requirements, installation tokens and approvals. See [package selection](../guides/github-actions.md#published-package-and-consumer-pins) before adopting fields from current source.

## Inputs

| Input | Accepted values | When to supply it |
| --- | --- | --- |
| `command` | `export`, `validate`, `plan`, `apply` | Required for every call. |
| `token` | A target-scoped GitHub App installation token | Required for export, plan and apply, including when the declaration manages no fields. Validate runs without API authentication. |
| `plan-reporting` | `detailed` (default), `status-only` | Choose `status-only` for a plan whose settings must stay out of the published Summary and command logs. |
| `prune` | One comma-separated list of [prune scopes](settings/pruning.md#collection-pruning), or `all` alone | Optional for plan and apply. Export and validate reject a nonempty value before installation. |

The checkout's direct GitHub.com `origin` selects the target and its root `.github/settings.yml` is the declaration. Even validate needs a resolvable checkout commit and origin for its outputs. The caller supplies the token explicitly. Its installation and job-scoped grants must cover the resources the command reads or writes. Some complete reads require write access, including observation of Ruleset bypass actors. [GitHub App permission matrix](settings/permissions.md)

## Outputs

Read `steps.<id>.outputs.<name>` after a successful step. Every successful command reports the following identities.

| Output | Value |
| --- | --- |
| `config-sha` | Full checkout `HEAD` commit. This identifies the checked-out revision, including when export writes a new local file. |
| `target` | Repository selected by the checkout, in `OWNER/REPO` form. |
| `cli-version` | Verified installed CLI release. |
| `cli-sha256` | SHA-256 of the installed executable. |

Two outputs are specific to a command and are empty for other invocations.

| Output | Available after | Value |
| --- | --- | --- |
| `config-path` | Successful export | `.github/settings.yml` relative to the checkout. The file stays in this job's workspace. |
| `plan-outcome` | Successful status-only plan | `changed`, `no-difference`, or `unmanaged`. Each is a successful comparison. |

## Plan reporting

Detailed mode shows the settings changes in the job Summary alongside the workflow, configuration, Action and CLI identities. Depending on the fields declared, the plan and logs can reveal names, plaintext variable values, collaborator identities, webhook URLs and removal consequences.

Status-only mode publishes the outcome and a bounded managed count. It suppresses settings details and raw diagnostics in the command logs and Summary. Checkout and package identity outputs remain available. Use the same trusted workflow and scoped credential for either mode.

Both modes use a private temporary report on the runner. It can contain detailed values even in status-only mode. The Action restricts the file to the current user, renders the Summary, and removes the report before publishing success outputs. The cleanup step also runs after ordinary failures. Forced runner termination can interrupt cleanup. Keep the runner temporary directory out of caches and artifacts.

## Apply, pruning and ID recovery

The Action runs the same reconciliation as the [CLI](commands.md#apply). To preview deletions, use the same `prune` input and declaration in plan and apply. The Action checks CLI support before pruning. An omitted collection stays unmanaged. A present empty list can select every object in that scope for deletion. [Collection ownership and pruning](settings/pruning.md#collection-pruning)

Apply leaves the workflow checkout's declaration untouched. For resources that return creation IDs, record verified IDs in a reviewed change to the declaration. If a failure reports a recovery ID, confirm the object in the target repository before recording or retrying it. Repeating an unbound create can collide with the object that was just created. [CLI ID recovery](commands.md#recording-ids)

## Failures

An invalid input, installation or checkout failure, command error, or failure writing the Summary or outputs fails the step. For plan, a command failure retains its nonzero status. After a successful comparison, Summary rendering and report cleanup must also succeed. Export, validate and apply report through the step logs and outcome.

A failed step supplies no reliable success outputs. If apply or output publication fails, check the remote state and any reported IDs before retrying. Earlier writes may already have taken effect. [Recovery guide](../guides/github-actions.md#failure-and-recovery)
