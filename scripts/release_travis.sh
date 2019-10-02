#!/usr/bin/env bash

set -euo pipefail

echo "generating changelog"

# work in temp dir to prevent polluting our go.mod
current_dir=$(pwd)
cd "$(mktemp -d)"

go get -u github.com/digitalocean/github-changelog-generator

tfile=$(mktemp)
github-changelog-generator -org digitalocean -repo doctl >"$tfile"

cd "$current_dir"
goreleaser --rm-dist --release-notes="$tfile"

echo "released!"
