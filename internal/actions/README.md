# GitHub Action reporting

The public composite Action calls private commands in the installed executable to identify the checkout and CLI, produce a plan report and write a job summary. The [Action steps](../../action.yml) show the invocations and [run.go](run.go) dispatches them.

For a change to what a plan records, start with the [plan report](plan.go). For a change to what reviewers see, follow the report through [validation](validated_report.go) to [summary rendering](summary.go). The [Action internals guide](../../docs/development/actions-internals.md) describes detailed and status-only output and how the summary step runs without settings credentials.
