#!/usr/bin/env bash

set -o pipefail

go get -u github.com/buchanae/github-release-notes

github-release-notes -org digitalocean -repo doctl -since-latest-release -include-author

GO111MODULE=on go mod tidy
