#!/usr/bin/env bash

set -o pipefail

tfile=$(mktemp)
github-release-notes -org digitalocean -repo doctl -since-latest-release -include-author >"$tfile"

GO111MODULE=on go mod tidy

echo "$tfile"
