# Release notes generator

When preparing a new draft, the publisher invokes [main.go](main.go) with the repository, chosen version, source commit and a saved GitHub release catalog. The generator reads the committed messages from the selected first-parent history and renders the changes as Markdown.

For changes to commit selection, category order or compatibility footers, start with the [release notes tests](main_test.go). The [withdrawal tests](withdrawals_test.go) cover how a later commit can withdraw or clarify an earlier entry. The [release notes reference](../../docs/development/releases.md#release-notes-from-committed-messages) defines what maintainers should see in the published notes.
