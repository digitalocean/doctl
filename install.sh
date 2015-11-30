#!/bin/sh

set -o pipefail

tmpdir=`mktemp -d`

function cleanup {
  rm -rf $tmpdir
}
trap "cleanup" EXIT

current_version="0.6.0"

echo -e "Installing doit ${current_version}...\n"
echo "Doit installation directory (this will create a doit subdirectory) (${HOME}): "
read install_dir

if [[ -z "$install_dir" ]]; then
	install_dir=$HOME
fi

echo "Creating ${install_dir}/doit"
mkdir -p "${install_dir}/doit/bin"

osarch=$(echo `uname -s`_`uname -m` | awk '{print tolower($0)}')

case "$osarch" in
	darwin_x86_64)
		bin_name="doit-${current_version}-darwin-10.6-amd64"
		;;
	linux_386)
		bin_name="doit-${current_version}-linux-386"
		;;
	linux_x86_64)
		bin_name="doit-${current_version}-linux-amd64"
		;;
	*)
		echo "Unsupported arch $(uname -s) $(uname -m)"
		exit 1
esac

echo "Retrieving binaries"
cd $tmpdir
curl -# -L -O "https://github.com/bryanl/doit/releases/download/v${current_version}/${bin_name}"
echo "Retrieving checksums"
curl -# -L -O "https://github.com/bryanl/doit/releases/download/v${current_version}/${bin_name}.sha256"

case $(uname -s) in
	Linux)
		shasum="sha256sum"
		;;
	Darwin)
		shasum="shasum -p -a 256"
		;;
esac

echo -n "Validating checksums..."
$shasum -c ${bin_name}.sha256 | grep OK 2>&1 > /dev/null
if [ $? -ne 0 ]; then
	echo "${bin_name} has invalid checksum"
	exit 1
fi
echo "Checksums have been validated"

echo "Installing binaries"

cp $tmpdir/$bin_name $install_dir/doit/bin/doit
chmod u+x "${install_dir}/doit/bin/doit"

echo -e "\nInstall complete!\n"
echo -e "doit has been installed to ${install_dir}/doit/bin/doit"

