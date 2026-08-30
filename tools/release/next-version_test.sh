#!/usr/bin/env bash
set -euo pipefail

version_tool=${1:-tools/release/next-version.sh}
scratch=${TEST_TMPDIR:-/tmp}
commit_log="$scratch/start-release-commits.txt"

assert_version() {
  local expected=$1
  local current=$2
  local message=$3
  printf '%s\n' "$message" > "$commit_log"
  local actual
  actual=$("$version_tool" auto "$current" "$commit_log")
  if [[ $actual != "$expected" ]]; then
    echo "expected $expected, got $actual for: $message" >&2
    exit 1
  fi
}

assert_version 1.2.4 v1.2.3 'fix: keep keyboard focus visible'
assert_version 1.3.0 1.2.3 'feat(search): add saved searches'
assert_version 2.0.0 1.2.3 'feat(api)!: replace link identifiers'
assert_version 2.0.0 1.2.3 $'refactor: simplify claims\n\nBREAKING CHANGE: group mappings use IDs'

printf '%s\n' 'docs: clarify setup' > "$commit_log"
[[ $("$version_tool" minor 1.2.3 "$commit_log") == 1.3.0 ]]
