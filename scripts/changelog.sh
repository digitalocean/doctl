#!/usr/bin/env bash

set -o pipefail

github-release-notes -org digitalocean -repo doctl -since-latest-release -include-author
