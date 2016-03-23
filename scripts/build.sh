#!/bin/bash

set -eo pipefail

go build -ldflags "-X github.com/digitalocean/doctlBuild=`git rev-parse --short HEAD`" github.com/digitalocean/doctlcmd/doctl
