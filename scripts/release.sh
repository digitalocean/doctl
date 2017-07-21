#!/usr/bin/env bash

tag=$1

if [[ -z "$tag" ]]; then
  echo "usage: $0 <tag>"
fi

github-release release \
  --user digitalocean \
  --repo doctl \
  --name "$tag" \
  --pre-release --tag "$tag"