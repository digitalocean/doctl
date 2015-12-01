#!/bin/sh

set -eo pipefail

ver=$1

if [[ -z "$ver" ]]; then
  echo "usage: $0 <version>"
  exit 1
fi


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUTPUT_DIR="${DIR}/../builds/${ver}"

if [[ -z $SKIPBUILD ]]; then
  mkdir -p $OUTPUT_DIR

  xgo \
    --dest $OUTPUT_DIR \
    --targets='windows/*,darwin/*,linux/*' \
    -ldflags "-X commands.verson=${ver}" \
    -out doit-0.6.0 github.com/bryanl/doit/cmd/doit

fi

# FIXME mac only for now
cd $OUTPUT_DIR
echo $CWD
for f in $(find . -maxdepth 1 -perm -111 -type f); do
  fn=$(basename $f)
  echo "generating sha256 checksum for $fn"
  shasum -a 256 ${fn} > ${f}.sha256
done

