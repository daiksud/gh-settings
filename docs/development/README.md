# Maintainer documentation

Start with [setup](setup.md) to build gh-settings locally. Use the [architecture guide](architecture.md) to find the package that owns a change and the [testing guide](testing.md) to find the checks that exercise it.

| When changing | Where to look |
| --- | --- |
| Behavior a user can observe | [Feature specifications](features/README.md) describe scenarios and point to their tests. The [settings test guide](testing/settings.md) maps individual controls to their checks. Update the matching [public reference](../references/README.md) alongside a behavior change. |
| Support for a repository setting | The [coverage ledger](settings-coverage.md) records what the target GitHub UI and APIs support. The UI mapping documents in this directory trace individual controls to their implementation. |
| A design constraint | [Development policies](policies.md) state the current rules. [Architecture decision records](adr/README.md) explain how past decisions were reached. |
| CI or releases | [Repository automation](repository-automation.md) covers this repository's workflows and settings. The [release procedure](releases.md) covers publication and recovery. |
