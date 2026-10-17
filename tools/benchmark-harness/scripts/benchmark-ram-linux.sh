#!/usr/bin/env bash
# Manage a measured Linux tmpfs mount for benchmark workspaces.
set -euo pipefail
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec "$script_dir/benchmark-ram-common.sh" linux "$@"
