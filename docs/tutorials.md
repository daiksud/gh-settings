# Manage a repository description

Use the CLI to change the description in a practice repository's **About** section. You will export a starting configuration, narrow it to one field, preview the change, apply it, and check the result.

Start with an active, personally owned public repository you can administer. Use a checkout with a direct GitHub.com `origin` and no existing `.github/settings.yml`. Follow the [installation instructions](../README.md#install), then run these commands from the checkout root.

## Export the starting configuration

```sh
git config --show-scope --get-all remote.origin.url
gh auth status
```

Check that the first `local` or `worktree` URL names your practice repository. That is the origin gh-settings uses. Then export its settings.

```sh
gh settings export
```

Export reads the supported settings it can access and writes `.github/settings.yml`. Read any `info: export` diagnostics for settings it could not acquire. The file can include plaintext repository or Environment variables, so review it before sharing.

## Declare one change

Note the original description in the repository's **About** section. Replace the contents of the exported file with this declaration. If the repository already uses this description, choose a different one so the preview shows a change.

```yaml
version: daiksud/gh-settings@v0
repository:
  about:
    description: Practice repository for gh-settings
```

With only `description` declared, this exercise manages that one field. The `version` string identifies the YAML format.

## Preview the change

```sh
gh settings validate
gh settings plan
```

Validation succeeds silently. If the repository description was empty and you used the declaration above, the plan should show one line:

```text
repository.about.description: "" → "Practice repository for gh-settings"
```

Check that the left side matches your original description, the right side matches your edit, and the plan lists only this change.

## Apply and verify

```sh
gh settings apply
gh settings plan
```

Apply uses the declaration in the checkout at execution time. Check the description in **About**. The second plan should report `No differences for managed repository settings.` If apply reports an error, inspect the remote state before retrying. [Apply behavior and partial failures](references/commands.md#recovering-from-a-failed-apply)

To restore the original description, put it back in the declaration (use `description: ''` if it was empty), then plan, apply and verify again. To manage another field, look it up in the [settings reference](references/settings/README.md).
