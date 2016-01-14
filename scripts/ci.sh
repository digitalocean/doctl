#!/bin/bash

set -e

VER=0.8.3

curl -L -o /tmp/glide-${VER}-linux-amd64.tar.gz https://github.com/Masterminds/glide/releases/download/${VER}/glide-${VER}-linux-amd64.tar.gz
tar -C /tmp -xf /tmp/glide-${VER}-linux-amd64.tar.gz

GLIDE=/tmp/linux-amd64/glide

${GLIDE} install
go test $(${GLIDE} nv)

