#!/usr/bin/env bash

set -Eeuo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/../"

function cleanup {
  echo "cleaning up"
  echo ""
  cd "$DIR" && docker run -v "$DIR":/build -w /build snapcore/snapcraft:stable snapcraft clean
}
trap cleanup EXIT

set +e
rm doctl_*_amd64.snap 2>/dev/null
set -e

echo "building snap"
echo ""
cd "$DIR" && docker run --rm -v "$DIR":/build -w /build snapcore/snapcraft:stable \
       bash -c "apt update && snapcraft"

snap=$(ls doctl_*_amd64.snap)

CHANNEL=${CHANNEL:-stable}

echo "releasing snap"
echo ""
cd "$DIR" && docker run -i -v "$DIR":/build -w /build snapcore/snapcraft:stable \
       bash -c "snapcraft login && snapcraft push --release=${CHANNEL} ${snap}"
