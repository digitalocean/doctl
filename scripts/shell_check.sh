#!/usr/bin/env bash
#
# https://github.com/koalaman/shellcheck

set -euo pipefail

if [[ ! -x $(command -v shellcheck) ]] ; then
  echo "install https://github.com/koalaman/shellcheck then run again"
  exit 1
fi

shellcheck scripts/*.sh
