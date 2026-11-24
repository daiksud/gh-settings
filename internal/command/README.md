# Executable startup

The executable enters [Run](run.go) with the command-line arguments and build version. It creates a checkout, dispatches the public CLI or the private Action commands, and supplies the application operations each needs.

For changes to how a file becomes a settings declaration, or how an exported declaration is saved, follow the [checkout adapters](checkout_config.go). Export connects its diagnostics and save step in [export.go](export.go). The [bindings package](bindings/README.md) connects resource reads and writes to GitHub clients.
