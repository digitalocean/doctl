#!/bin/bash

set -e

curl -L -o /tmp/glide-0.8.0-linux-amd64.tar.gz https://github.com/Masterminds/glide/releases/download/0.8.0/glide-0.8.0-linux-amd64.tar.gz
tar -C /tmp -xf /tmp/glide-0.8.0-linux-amd64.tar.gz
cp /tmp/linux-amd64/glide /usr/local/bin

glide install
go test $(glide nv)

