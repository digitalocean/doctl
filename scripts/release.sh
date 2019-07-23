#!/usr/bin/env bash

tag=$1

if [[ -z "$tag" ]]; then
  echo "usage: $0 <tag>"
fi

gothub release \
  --user digitalocean \
  --repo doctl \
  --name "$tag" \
  --tag "$tag" \
  --pre-release
