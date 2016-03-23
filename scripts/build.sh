#!/bin/bash

set -eo pipefail

go build -ldflags "-X github.com/digitalocean/doctl/Build=`git rev-parse --short HEAD`" github.com/digitalocean/doctlcmd/doctl
