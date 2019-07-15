#!/usr/bin/env bash

set -eou pipefail

version=`git fetch --tags &>/dev/null | git tag -l | sort --version-sort | tail -n1 | cut -c 2-`

branch=`git rev-parse --abbrev-ref HEAD`
if [[ $branch != 'master' ]]; then
  version=${version}-${branch}
fi

num_changes=`git status --porcelain | wc -l`
if [[ $num_changes -ne 0 ]]; then
  commit=`git rev-parse --short HEAD`
  version=${version}-${commit}
fi

echo $version
