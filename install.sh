#!/bin/sh

set -eo pipefail

tmpdir=`mktemp -d`

function cleanup {
  rm -rf $tmpdir
}
trap "cleanup" EXIT

current_version="0.5.0"

# get directory for installation: ${HOME}/digitalocean and set it doit_home
echo "Doit installation directory (this will create a doit subdirectory) (${HOME}): "
read install_dir

echo "Creating ${install_dir}/doit\n"

bin_name=$(echo "doit_"`/usr/bin/uname -s`_`/usr/bin/uname -m` | awk '{print tolower($0)}' | sed 's/x86_64/amd64/')
curl -o $tmpdir/doit -# -L "https://github.com/bryanl/releases/download/v${current_version}/${bin_name}"

mkdir -p "${install_dir}/doit/bin"
mv $tmpdir/doit "${install_dir}/doit/bin/doit"

echo "Install complete!\n\n"

