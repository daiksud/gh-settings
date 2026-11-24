# Configuration UI types

These Go types group settings using the pages, sections and controls a contributor will recognize in the YAML configuration. The [repository type](repository.go) connects the groups. Other files define individual controls. For example, [Pages](pages.go) represents the deployment source, branch, folder, custom domain and HTTPS setting.

Follow a field from its UI type to the [workspace codec](../../workspace/config.go) for its YAML representation and to the [GitHub declaration adapter](../../github/ui_declaration.go) for its native settings. The [settings reference](../../../docs/references/settings/README.md) documents the keys accepted from users.
