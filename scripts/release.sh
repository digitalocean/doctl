#!/usr/bin/env bash

tag=$1

if [[ -z "$tag" ]]; then
  echo "usage: $0 <tag>"
fi

github-release release \
  --user bryanl \
  --repo doit \
  --name "$tag" \
  --pre-release --tag "$tag"