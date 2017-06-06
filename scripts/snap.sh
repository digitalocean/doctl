#!/bin/bash

set -eo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUT_DIR="$DIR/../out"

# Build the package
/bin/bash build.sh

# Create the package
version="$(git rev-parse --short HEAD)"
sed -i "s/edge/$version/g" snapcraft.yaml

cp $OUT_DIR/doctl .
snapcraft

# Clean temp stuff
snapcraft clean

