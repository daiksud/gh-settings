#!/usr/bin/env bash
set -euo pipefail
: "${GH_TOKEN:?missing App token}" "${SETTINGS_CLI:?missing verified CLI}"
"$SETTINGS_CLI" apply
