#!/usr/bin/env bash

tag=$1

if [[ -z "$tag" ]]; then
  echo "usage: $0 <tag>"
fi

# ensure tags are up to date
git fetch --tags &>/dev/null

git tag -a -m "release ${tag}" "$tag" && git push --tags

gothub release \
  --user digitalocean \
  --repo doctl \
  --name "$tag" \
  --tag "$tag" \
  --pre-release
