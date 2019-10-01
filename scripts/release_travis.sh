#!/usr/bin/env bash

set -euo pipefail

echo "generating changelog"
tfile=$(mktemp /tmp/doctl-CHANGELOG-XXXXXX)
github-changelog-generator -org digitalocean -repo doctl >"$tfile"

goreleaser --rm-dist --release-notes="$tfile"

echo "released!"
