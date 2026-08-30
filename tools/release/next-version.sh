#!/usr/bin/env bash
set -euo pipefail

bump=${1:-auto}
current=${2:-0.0.0}
commit_log=${3:-/dev/stdin}

current=${current#v}
if [[ ! $current =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "current version must be a stable semantic version, got: $current" >&2
  exit 2
fi
if [[ ! -r $commit_log ]]; then
  echo "commit log is not readable: $commit_log" >&2
  exit 2
fi

major=${BASH_REMATCH[1]}
minor=${BASH_REMATCH[2]}
patch=${BASH_REMATCH[3]}

if [[ $bump == auto ]]; then
  if grep -Eq '(^|[[:space:]])BREAKING CHANGE:|^[[:alpha:]][[:alnum:]_-]*(\([^)]*\))?!:' "$commit_log"; then
    bump="major"
  elif grep -Eq '^feat(\([^)]*\))?:' "$commit_log"; then
    bump="minor"
  else
    bump="patch"
  fi
fi

case "$bump" in
  major)
    printf '%d.0.0\n' "$((major + 1))"
    ;;
  minor)
    printf '%d.%d.0\n' "$major" "$((minor + 1))"
    ;;
  patch)
    printf '%d.%d.%d\n' "$major" "$minor" "$((patch + 1))"
    ;;
  *)
    echo "bump must be auto, patch, minor, or major, got: $bump" >&2
    exit 2
    ;;
esac
