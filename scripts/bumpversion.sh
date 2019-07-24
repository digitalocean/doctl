#!/usr/bin/env bash

set -euo pipefail

# Bump defaults to patch. We provide friendly aliases
# for patch, minor and major
BUMP=${BUMP:-patch}
case "$BUMP" in
  feature | minor)
    BUMP="minor"
    ;;
  breaking | major)
    BUMP="major"
    ;;
  *)
    BUMP="patch"
    ;;
esac

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

version="$("$DIR"/../scripts/version.sh -s)"
new_version="v$(sembump --kind "$BUMP" "$version")"

echo "Bumping version from v${version} to ${new_version}"

git tag -m "release ${new_version}" -a "$new_version" && git push --tags

