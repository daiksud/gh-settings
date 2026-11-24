# Release tools

This directory contains the programs the [manual release workflow](../.github/workflows/release.yml) uses to prepare release notes and publish tested binaries.

- [release-metadata](release-metadata/README.md) generates the change list from committed messages. The workflow calculates the release version separately in the [version workflow](../.github/workflows/version.yml).
- [release-publisher](release-publisher/README.md) checks the tag, draft and assets before publishing. The [shell entry point](publish-release.sh) builds and runs it in a temporary directory.

For the sequence from selecting a commit to verifying the published package, see the [maintainer release guide](../docs/development/releases.md).
