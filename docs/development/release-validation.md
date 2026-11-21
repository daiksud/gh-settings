# Release validation

Use the checks below when changing version selection, workflow gates, release notes or the publisher. The [release guide](releases.md) describes how to authorize and run a release, recover a draft and verify published downloads. The published Action's installer has its own [behavior specification](features/actions.feature.md#rule-a-release-selects-the-same-action-and-cli-version).

## Select a version and commit range

Run the following from the repository root. The [unit-test Action](../../.github/actions/unit-test/action.yml) runs both suites in PR screening and main CI. The `-B` option keeps generated Python bytecode out of the checkout before native builds. A [regression test](../../tests/github-actions/unit_test_action_test.go) checks that the compiled binary records the expected revision and `vcs.modified=false`.

```sh
python3 -B -m unittest discover -s .github/scripts -p 'test_release_version.py' -q
python3 -B -m unittest discover -s .github/scripts -p 'test_release_ci.py' -q
```

The [version tests](../../.github/scripts/test_release_version.py) build temporary Git histories and release catalogs. They cover the stable predecessor selected by publication time, a fixed source, substantive tree changes, empty intervals, missing or unrelated history, and malformed metadata. With no published stable predecessor, version resolution stops and the initial release requires a separately chosen baseline. Cases include breaking, feature, performance, deprecation, maintenance and revert signals. The strongest signal in the complete interval determines the increment. Changing a branch name or editing PR metadata does not change committed history.

The promotion cases cover default-off v0.x calculation, explicit promotion from a validated v0.x baseline to v1.0.1, ordinary calculation after v1, and protection against duplicate or lower published versions. An exact v1 retry keeps its recorded version with either checkbox choice. Requesting promotion for a source already published as v0 fails. See the [version selection rules](releases.md#automatic-version-calculation).

The [release notes tests](../../scripts/release-metadata/main_test.go) use fixed Git ranges to check category precedence, formal compatibility footers, escaping, migration guidance and commit links. Each substantive first-parent commit appears once. The [withdrawal tests](../../scripts/release-metadata/withdrawals_test.go) cover formal `Release-withdraws:` and `Release-note:` footers, earlier unique targets, reasons and invalid references. Marker examples in fences and quoted text do not classify or withdraw a commit. An incomplete range or changed predecessor stops generation before remote mutation. The [release guide](releases.md#release-notes-from-committed-messages) defines the categories and footer syntax.

## Validate the source and artifacts

| Case | Expected result and owning check |
| --- | --- |
| Dispatch uses another ref, main moved, or an explicit source pin differs | Resolution requires the dispatched main SHA. The [workflow tests](../../tests/github-actions/release_test.go) use a local `gh` stub to cover matching, omitted, mismatched and invalid pins and lookup failures. The executable fixture requires Bash 5. Older shells run the structural checks only. [Metadata wiring tests](../../tests/github-actions/release_metadata_test.go) check the manual trigger, read-only resolver and frozen predecessor outputs. |
| Main CI exists for the source | The [CI tests](../../.github/scripts/test_release_ci.py) require the latest matching run to have succeeded. Failed, cancelled or pending runs, changed provenance and API errors stop release. An absent run requires full native validation. |
| Reuse a pair of main CI artifacts | Each Linux and macOS product needs the expected archive digest, inventory and manifest binding source, version, target, filename and binary hash. Missing, expired, corrupted or version-mismatched products trigger full build and acceptance. The CI tests include normal v0 artifacts presented for a v1 promotion. |
| Accept products and gate publication | The [local Action tests](../../tests/github-actions/release_actions_test.go) and [workflow tests](../../tests/github-actions/release_test.go) cover artifact transfer, native acceptance, candidate Action identity and required job results. Only the publication job requests Contents write. The product build, native acceptance and artifact verification steps receive no token in their environment. Publication reverifies the release-run products. A fresh check of main and CI blocks publication if main advanced or the selected run stopped qualifying. |

## Exercise publication and recovery locally

```sh
mise exec -- go test ./scripts/release-metadata ./scripts/release-publisher ./tests/github-actions
mise exec -- bash tests/release.sh
```

The [publisher fixture](../../tests/release.sh) runs the real publisher against a local `gh` stand-in. It compares the complete generated body with the expected categories, migration details and installation/checksum guidance. The stand-in rejects requests for PR metadata or GitHub-generated notes. It exercises first publication, generation failure before tag creation, and recovery across these states.

| Starting state | Expected outcome | Owning tests |
| --- | --- | --- |
| Invalid source, version, expected hashes or local product bytes | Stop before remote changes. Checksums and manifest bytes must agree with the accepted products. | [Publisher preflight](../../scripts/release-publisher/main_test.go) |
| Matching draft missing a required asset | Verify the tag, marker, version and existing asset bytes. Upload only missing assets, retain the stored notes and recheck state before publication. | [Asset checks](../../scripts/release-publisher/assets_test.go) and [publisher controller](../../scripts/release-publisher/publish_test.go) |
| Matching published release | Confirm the immutable tag, marker and complete asset set. Retrying preserves the bytes, notes and comparison range. | [Release states](../../scripts/release-publisher/state_test.go) and [publisher fixture](../../tests/release.sh) |
| Conflicting tag or source, duplicate or unknown assets, altered bytes or malformed lookup results | Stop. A failed lookup cannot establish absence. Check the tag again at publication. | [Lookup and tag checks](../../scripts/release-publisher/lookup_test.go) and [asset checks](../../scripts/release-publisher/assets_test.go) |

The [CLI tests](../../internal/cli/run_test.go) own the version command. Only an authorized release run can confirm actual publication, verify the two raw public downloads against the accepted manifests and run the published Action for a separate consumer. See the [evidence table](testing.md#release-and-publication-evidence) when reporting results.
