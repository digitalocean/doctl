#!/bin/bash

set -eo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUT_DIR="$DIR/../out"
mkdir -p $OUT_DIR

go build \
  -o $OUT_DIR/doctl \
  -ldflags "-X github.com/digitalocean/doctl/Build=`git rev-parse --short HEAD`" \
  github.com/digitalocean/doctl/cmd/doctl
