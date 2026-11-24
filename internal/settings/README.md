# Settings model and comparison

This package defines the desired values in a [declaration](settings.go), the state acquired from GitHub and the changes needed to reconcile them. Use the [model glossary](../../docs/development/glossary.md) for the distinction between an omitted setting, an observed value and a planned change.

To change what a setting manages, start with its resource files. For example, [labels.go](labels.go) validates declarations and compares them with an [acquired label inventory](label_observation.go). The [UI model](uimodel/README.md) groups fields the way they appear in the repository configuration.
