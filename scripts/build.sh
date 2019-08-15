#!/bin/bash

set -eou pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUT_D="$DIR/../out"

cd "$DIR/../." && OUT_D="$OUT_D" make native

chmod +x "$OUT_D/doctl"
