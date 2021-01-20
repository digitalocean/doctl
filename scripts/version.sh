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
"

semver() {
  git tag -l | sort --version-sort | tail -n1 | cut -c 2-
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

  *)
    echo "$help_message"
    exit 0
    ;;
esac

echo "$version"
