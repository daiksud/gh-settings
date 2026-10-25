# Custom property values

Removed from current source when product scope was narrowed to settings usable by personally owned repositories. Published v0.18.0 and v0.18.1 retain the earlier bounded feature. [ADR](../adr/0030-schema-bound-custom-property-values.md) preserves its historical design; the [configuration reference](../settings.md) owns migration instructions.

Removed YAML keys and export options fail locally, including empty/null declarations. Removing configuration leaves remote state unmanaged and performs no cleanup writes. Actions reject old custom-property reports and removed forking field changes rather than silently dropping them.

[Workspace regressions](../../internal/workspace/removed_settings_test.go), [CLI rejection tests](../../internal/cli/run_test.go), [Summary rejection tests](../../internal/actions/summary_test.go) and [binary acceptance](../../tests/acceptance/removed_settings_test.go) verify rejection and preservation boundaries using local fixtures. No live schema/value mutation is performed.
