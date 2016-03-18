#!/usr/bin/env

set -eo pipefail

ver=$1

if [[ -z "$ver" ]]; then
  echo "usage: $0 <version>"
  exit 1
fi

RELEASE_PACKAGE=github.com/bryanl/doit/cmd/doctl
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUTPUT_DIR="${DIR}/../builds/${ver}"
STAGE_DIR=$OUTPUT_DIR/stage
RELEASE_DIR=$OUTPUT_DIR/release

mkdir -p $OUTPUT_DIR/stage $OUTPUT_DIR/release

rm -f $STAGE_DIR/doctl $STAGE_DIR/doctl.exe

if [[ -z $SKIPBUILD ]]; then
  xgo \
    --dest $OUTPUT_DIR/stage \
    --targets='windows/*,darwin/amd64,linux/amd64,linux/386' \
    -ldflags "-X github.com/bryanl/doit.Build=$(git rev-parse HEAD)" \
    -out doit-${ver} $RELEASE_PACKAGE
fi

cd $RELEASE_DIR

for f in $STAGE_DIR/*; do
  distfile=$(basename ${f%".exe"})
  if [[ $f == *"windows"* ]]; then
    distfile=${distfile}.zip
  else
    distfile=${distfile}.tar.gz
  fi
  
  distbin=$(basename $RELEASE_PACKAGE)
  if [[ $f == *.exe ]]; then
    distbin=$distbin.exe
  fi
  
  bindir=$STAGE_DIR/$distbin
  cp $f $bindir
  
  if [[ $distfile == *.zip ]]; then
    zip -j $distfile $bindir
  else
    tar cvzhf $distfile -C $STAGE_DIR $distbin
  fi
  
  pushd $STAGE_DIR
  shasum -a 256 $(basename $distbin) > ${RELEASE_DIR}/$(basename ${f%".exe"}).sha256
  popd
  
  rm $bindir
done
