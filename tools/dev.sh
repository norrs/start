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
workspace_dir="${BUILD_WORKSPACE_DIRECTORY:-$PWD}"
development_dir="${STARTPAGE_DEV_DIR:-${workspace_dir}/.start-dev}"

mkdir -p "$development_dir"

export STARTPAGE_DEV_MODE="${STARTPAGE_DEV_MODE:-true}"
export STARTPAGE_LISTEN_ADDR="${STARTPAGE_LISTEN_ADDR:-127.0.0.1:8080}"
export STARTPAGE_BASE_URL="${STARTPAGE_BASE_URL:-http://127.0.0.1:8080}"
export STARTPAGE_DATABASE_PATH="${STARTPAGE_DATABASE_PATH:-${development_dir}/start.db}"
export STARTPAGE_STATIC_DIR="${STARTPAGE_STATIC_DIR:-$static_dir}"
export STARTPAGE_SEED_PATH="${STARTPAGE_SEED_PATH:-$seed_file}"
export STARTPAGE_METRICS_ENABLED="${STARTPAGE_METRICS_ENABLED:-true}"

echo "Start development server: ${STARTPAGE_BASE_URL}"
echo "SQLite database: ${STARTPAGE_DATABASE_PATH}"
exec "$start_binary" "$@"
