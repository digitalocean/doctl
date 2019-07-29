#!/usr/bin/env bash

set -o pipefail

tfile=$(mktemp /tmp/doctl-CHANGELOG-XXXXXX)
github-release-notes -org digitalocean -repo doctl -since-latest-release -include-author >"$tfile"

GO111MODULE=on go mod tidy

echo "$tfile"
