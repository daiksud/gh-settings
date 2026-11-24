# Release publisher

The [publication code](main.go) binds a release to its source commit and expected asset digests. It checks the tag and draft, uploads missing assets, and verifies the downloaded bytes before publishing. On a retry it can finish a matching draft or verify an already published release.

Start with the [asset tests](assets_test.go) when changing checksum verification, and the [publication tests](publish_test.go) when changing the order of uploads and final checks. The [offline release fixture](../../tests/release.sh) exercises the shell entry point with a simulated GitHub CLI. For operational recovery, use the [publication and recovery guide](../../docs/development/releases.md#publication-and-recovery).
