#!/usr/bin/env bash

set -euo pipefail

if [[ ! -x $(command -v goreleaser) ]] ; then
  echo "install https://goreleaser.com/install/ then run again"
  exit 1
fi

echo "generating changelog"
release_notes="$(make _changelog)"

goreleaser --rm-dist --release-notes="${release_notes}"

rm -f "$release_notes"
