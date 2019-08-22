#!/usr/bin/env bash

set -o pipefail

tfile=$(mktemp /tmp/doctl-CHANGELOG-XXXXXX)
github-changelog-generator -org digitalocean -repo doctl >"$tfile"

echo "$tfile"
