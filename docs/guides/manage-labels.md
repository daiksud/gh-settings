# Manage one repository label

Use this procedure to change the color of an existing label while keeping its
issue and pull request assignments. Work in the repository's checkout with the
CLI installed and credentials that can read and update labels. See the
[CLI setup](../../README.md#install) and [label permissions](../references/settings/permissions.md#general-about-and-labels).

## Find the label's ID and current values

Check the [target repository](../references/commands.md#repository-and-authentication).
Replace `OWNER/REPO` with the repository from the checkout's direct GitHub.com
origin. For the existing `bug` label, run this read-only command:

```sh
gh api repos/OWNER/REPO/labels/bug --jq '{id, name, color, description}'
```

Keep the numeric `id`, name and description from the response. If you already
have a recent `gh settings export`, the label entry in `.github/settings.yml`
also contains these values.

## Declare the change

Edit `.github/settings.yml` at the checkout root. A complete declaration for
a repository whose `bug` label has ID `42` could look like this:

```yaml
version: daiksud/gh-settings@v0
repository:
  labels:
    - id: 42
      name: bug
      description: Something isn't working
      color: '#b60205'
```

Use your label's actual ID, name and description. Choose a different color
from the current one. The API returns six hex digits. YAML requires a quoted
string with a leading `#`. Omitting `description` clears it on update. For a
label with no description, use `description: ''`.

If the file already declares other settings, edit the matching label entry or
add it to the existing `repository.labels` list. Ordinary apply preserves
unlisted labels. You can manage a single label without copying all labels
into the declaration.

## Preview, apply and verify

```sh
gh settings validate
gh settings plan
```

Look for an **update** to the expected ID and a change to its color. Review
the entire plan, since apply includes any other settings in the declaration.
Continue when the plan shows only the changes you intend.

```sh
gh settings apply
gh settings plan
```

Check the label in GitHub. The follow-up plan should show no outstanding color
change. A rename works the same way: keep the ID and change `name` to an
unused name to preserve assignments.

## Create a new label

To create a label, omit `id` and declare an unused name, color and description.
An ID-less entry is create-only. If its name matches an existing label, the
command reports a collision and its numeric ID. Inspect that label before
binding the declaration to it.

After verifying a creation, ordinary CLI apply writes the new ID into the
original YAML. Review and save the change so the next apply targets that
label. With `--no-writeback` or `GITHUB_ACTIONS=true`, record the verified ID
yourself. If creation or writeback fails, check the actual label and retain
any reported ID before retrying. [ID recording and recovery](../references/commands.md#recording-ids)

## Preview deletions separately

This procedure uses ordinary apply. To intentionally delete unlisted labels,
first declare every label you want to retain, then run
`gh settings plan --prune labels`. Deleting a label removes its assignments
from issues and pull requests. Review the deletion candidates before choosing
whether to apply the same scope.

| Declaration | Ordinary apply | Apply with `--prune labels` |
| --- | --- | --- |
| Omit `repository.labels` | Leave all labels alone. | Leave all labels alone. |
| Set `repository.labels: []` | Leave all labels alone. | Delete every current label. |

See the [label reference](../references/settings/labels.md) for name collisions,
ownership and write verification.
