# Git checkout and settings file

This package works with the checkout that holds `.github/settings.yml`. [Repository resolution](repository.go) identifies the target from the checkout's Git origin. The [configuration codec](config.go) decodes and encodes the UI model, and the [settings document](ui_settings_document.go) defines how its sections appear in YAML.

For export changes, start with [Checkout.PrepareSave](checkout.go), which binds a save to the destination inspected before remote reads begin. For apply changes that record IDs returned by GitHub, start with the [created-ID recorder](created_ids.go). The [writeback reference](../../docs/references/settings/labels.md#apply-id-writeback) describes what a caller can expect when the settings file changes during an apply.
