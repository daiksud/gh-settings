# Shared settings presentation

The CLI and the public Action use this package to describe settings changes in the same terms. [Path mapping](paths.go) turns internal field names into visible configuration paths. The [comparison renderer](comparison.go) groups fields that belong to one UI control. Resource-specific files format entries such as [label changes](labels.go).

Change shared wording here when a difference should read the same in both outputs. The [CLI comparison](../cli/comparison.go) arranges terminal output, and the [Action summary](../actions/summary.go) handles Markdown layout and escaping.
