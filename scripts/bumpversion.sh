#!/usr/bin/env bash

set -euo pipefail

ORIGIN=${ORIGIN:-origin}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
version="$("$DIR"/../scripts/version.sh -s)"
IFS='.' read -r major minor patch <<< "$version"

# Bump defaults to patch. We provide friendly aliases
# for patch, minor and major
BUMP=${BUMP:-patch}
case "$BUMP" in
  feature | minor)
    minor=$((minor + 1))
    patch=0
    ;;
  breaking | major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  *)
    patch=$((patch + 1))
    ;;
esac

if [[ $(git status --porcelain) != "" ]]; then
  echo "Error: repo is dirty. Run git status, clean repo and try again."
  exit 1
elif [[ $(git status --porcelain -b | grep -e "ahead" -e "behind") != "" ]]; then
  echo "Error: repo has unpushed commits. Push commits to remote and try again."
  exit 1
fi  

echo
new_version="v${major}.${minor}.${patch}"

git tag -m "release ${new_version}" -a "$new_version" && git push "${ORIGIN}" tag "$new_version"

echo "Bumped version to ${new_version}"
 