#!/bin/bash

set -eo pipefail

go build -ldflags "-X github.com/bryanl/doit.Build=`git rev-parse --short HEAD`" github.com/bryanl/doit/cmd/doctl
