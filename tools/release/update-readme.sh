#!/usr/bin/env bash
set -euo pipefail

version=${1:?usage: update-readme.sh VERSION OWNER/REPOSITORY [README]}
repository=${2:?usage: update-readme.sh VERSION OWNER/REPOSITORY [README]}
readme=${3:-README.md}
version=${version#v}

if [[ ! $version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "version must be a stable semantic version, got: $version" >&2
  exit 2
fi
if ! grep -q '<!-- release:start -->' "$readme" || ! grep -q '<!-- release:end -->' "$readme"; then
  echo "release markers are missing from $readme" >&2
  exit 2
fi

major=${BASH_REMATCH[1]}
minor=${BASH_REMATCH[2]}
image="ghcr.io/${repository,,}"

replacement=$(mktemp)
output=$(mktemp)
trap 'rm -f "$replacement" "$output"' EXIT

cat > "$replacement" <<EOF
<!-- release:start -->
Latest release: [v${version}](https://github.com/${repository}/releases/tag/v${version})

Published multi-architecture container tags: \`${image}:${version}\`,
\`${image}:${major}.${minor}\`, \`${image}:${major}\`, and \`${image}:latest\`.
<!-- release:end -->
EOF

awk -v replacement="$replacement" '
  /<!-- release:start -->/ {
    while ((getline line < replacement) > 0) print line
    close(replacement)
    skipping = 1
    next
  }
  /<!-- release:end -->/ { skipping = 0; next }
  !skipping { print }
' "$readme" > "$output"
mv "$output" "$readme"
