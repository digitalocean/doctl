#!/usr/bin/env bash

# do NOT use pipefail in this script, as it will cause problems with the version
# line

set -e

me=$(basename "$0")

help_message="\
Usage: $me [<options>]
Display doctl version

Options:

  -h, --help   Show this help information.
  -s, --short  major.minor.patch only
  -i, --image  snap image version
  -b, --branch branch only
  -c, --commit commit only
  --snap, returns the version formatted for a snap release
"

semver() {
  # Prefer the latest GA tag (vX.Y.Z). Beta tags must not be reported as
  # "latest" so callers like snap/snapcraft.yaml and scripts/_build.sh keep
  # producing GA-based versions while a beta line is in flight.
  local v
  v="$(git tag -l | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort --version-sort | tail -n1 | cut -c 2-)"
  if [[ -z "$v" ]]; then
    v="$(git tag -l | sort --version-sort | tail -n1 | cut -c 2-)"
  fi
  echo "$v"
}

branch() {
  local branch
  branch=$(git rev-parse --abbrev-ref HEAD)
  if [[ $branch != 'main' && $branch != HEAD ]]; then
    echo "${branch}"
  fi
}

commit() {
  if [[ $(git status --porcelain) != "" ]]; then
    git rev-parse --short HEAD
  fi
}

image() {
  echo "$(semver)-$(commit)-pre"
}

snap() {
  version=$(semver)
  if [[ $(git tag --points-at) != "" ]]; then
    echo "v$version"
  else
    local_commit=$(git rev-parse --short HEAD)
    echo "v$version+git$local_commit"
  fi
}

ORIGIN=${ORIGIN:-origin}
set +e
git fetch --tags "${ORIGIN}" &>/dev/null
set -e

if [[ "$#" -eq 0 ]]; then

  SNAP_IMAGE=${SNAP_IMAGE:-false}
  if [[ "${SNAP_IMAGE}" != 'false' ]]; then
    image
    exit 0
  fi

  version=$(semver)

  br=$(branch)
  if [[ -n "$br" ]]; then
    version="${version}-${br}"
  fi

  cm=$(commit)
  if [[ -n "$cm" ]]; then
    version="${version}-${cm}"
  fi

  echo "$version"
  exit 0
fi

case "$1" in
  "-b"|"--branch")
    version=$(branch)
    ;;

  "-c"|"--commit")
    version=$(commit)
    ;;

  "-s"|"--short")
    version=$(semver)
    ;;

  "-i"|"--image")
    version=$(image)
    ;;

  "--snap")
    version=$(snap)
    ;;

  *)
    echo "$help_message"
    exit 0
    ;;
esac

echo "$version"
