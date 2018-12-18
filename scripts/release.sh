#!/usr/bin/env bash

tag=$1

if [[ -z "$tag" ]]; then
  echo "usage: $0 <tag>"
fi

git tag -a -m "release ${tag}" && git push --tags

gothub release \
  --user digitalocean \
  --repo doctl \
  --name "$tag" \
  --tag "$tag" \
  --pre-release
