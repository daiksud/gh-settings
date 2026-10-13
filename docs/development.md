# Local development

Use Go 1.25 or later and install [mise](https://mise.jdx.dev/getting-started.html). From the repository root, use the commands below to install the exact tool versions in [mise.toml](../mise.toml), including Go 1.26.0, and run checks through mise so the local versions match CI.

## Development checks

```sh
mise install
mise exec -- go test -race -count=1 -vet=off ./...
mise exec -- go vet ./...
mise exec -- go build ./...
mise exec -- go mod verify
mise exec -- go mod tidy -diff
mise exec -- actionlint -shellcheck shellcheck .github/workflows/*.yml
mise exec -- shellcheck scripts/publish-release.sh tests/release.sh
mise exec -- bash tests/release.sh
mise exec -- rumdl check --no-cache --deny-config-warnings README.md docs
```

## Local process acceptance

Build the product once, then run the tagged Go acceptance tests against that executable:

```sh
mise exec -- go build -o /tmp/gh-settings-acceptance .
mise exec -- go test -tags=acceptance -count=1 ./tests/acceptance -args -binary /tmp/gh-settings-acceptance
```

The [acceptance tests](../tests/acceptance/acceptance_test.go) require an explicit existing binary path and use a local Unix-socket HTTP fixture and go-gh v2.16.2's `http_unix_socket` configuration in a temporary `GH_CONFIG_DIR`. Their dummy token never authenticates with GitHub. Checks cover YAML values, empty strings and false, GET null normalization, current-repository selection, exit codes, stderr, and failure paths. Temporary repositories/configuration are isolated and cleaned up. Product code needs no testing hook or test-only modification; Unix sockets are required.
