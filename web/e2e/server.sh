#!/usr/bin/env bash
set -uo pipefail

runfiles_library=bazel_tools/tools/bash/runfiles/runfiles.bash
source "${RUNFILES_DIR:-/dev/null}/${runfiles_library}" 2>/dev/null || \
  source "$(grep -sm1 "^${runfiles_library} " "${RUNFILES_MANIFEST_FILE:-/dev/null}" | cut -d ' ' -f 2-)" 2>/dev/null || \
  source "$0.runfiles/${runfiles_library}" 2>/dev/null || \
  source "$(grep -sm1 "^${runfiles_library} " "$0.runfiles_manifest" | cut -d ' ' -f 2-)" 2>/dev/null || {
    echo >&2 "Unable to load Bazel runfiles support."
    exit 1
  }
set -e

start_binary="$(rlocation "start/cmd/start/start_/start")"
static_dir="$(rlocation "start/web/dist")"
seed_file="$(rlocation "start/links.yaml")"
test_dir="${TEST_TMPDIR:?Bazel must provide TEST_TMPDIR}"
test_port="${START_E2E_PORT:-18081}"

export STARTPAGE_DEV_MODE=true
export STARTPAGE_DEV_USER_EMAIL=e2e-admin@example.test
export STARTPAGE_DEV_USER_NAME="E2E Administrator"
export STARTPAGE_DEV_USER_GROUPS=start-admins
export STARTPAGE_LISTEN_ADDR="127.0.0.1:${test_port}"
export STARTPAGE_BASE_URL="http://127.0.0.1:${test_port}"
export STARTPAGE_DATABASE_PATH="${test_dir}/start-e2e.db"
export STARTPAGE_STATIC_DIR="$static_dir"
export STARTPAGE_SEED_PATH="$seed_file"
export STARTPAGE_METRICS_ENABLED=true
export STARTPAGE_LOG_LEVEL=warn

exec "$start_binary"
