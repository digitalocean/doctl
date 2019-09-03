#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/../"

set +e
rm doctl_*_amd64.snap 2>/dev/null
set -e

echo "building snap"
echo ""
cd "$DIR" && docker run --rm -v "$DIR":/build -w /build snapcore/snapcraft:stable \
       bash -c "apt update && snapcraft"
