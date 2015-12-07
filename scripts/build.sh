#!/bin/sh

set -eo pipefail

go build -ldflags "-X github.com/bryanl/doit/commands.Build=`git rev-parse HEAD`" github.com/bryanl/doit/cmd/doit
