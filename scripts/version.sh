#!/usr/bin/env bash

# do NOT use pipefail in this script, as it will cause problems with the version
# line

set -e

me=$(basename "$0")

help_message="\
Usage: $me [<options>]
Display doctl version

Options:

  -h, --help  Show this help information.
  -s, --short major.minor.patch only
"

parse_args() {
  while : ; do
    if [[ $1 = "-h" || $1 = "--help" ]]; then
      echo "$help_message"
      return 0
    elif [[ $1 = "-s" || $1 = "--short" ]]; then
      short=true
      shift
    else
      break
    fi
  done
}

parse_args "$@"

ORIGIN=${ORIGIN:-origin}

version=$(git fetch --tags "${ORIGIN}" &>/dev/null | git tag -l | sort --version-sort | tail -n1 | cut -c 2-)

if [[ $short = true ]]; then
  echo "$version"
  exit 0
fi

branch=$(git rev-parse --abbrev-ref HEAD)
if [[ $branch != 'master' && $branch != HEAD ]]; then
  version=${version}-${branch}
fi

if [[ $(git status --porcelain) != "" ]]; then
  commit=$(git rev-parse --short HEAD)
  version=${version}-${commit}
fi

echo "$version"
