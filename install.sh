#!/bin/sh

set -eo pipefail

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
		bin_name="doit_${current_version}_darwin_amd64"
		ext="zip"
		;;
	linux_386)
		bin_name="doit_${current_version}_linux_386"
		ext="tar.gz"
		;;
	linux_x86_64)
		bin_name="doit_${current_version}_linux_amd64"
		ext="tar.gz"
		;;
	*)
		echo "Unsupported arch $(uname -s) $(uname -m)"
		exit 1
esac

cd $tmpdir
curl -# -L -O "https://github.com/bryanl/doit/releases/download/v${current_version}/${bin_name}.${ext}"

case $(uname -s) in
	Darwin)
		unzip -q "${bin_name}.${ext}"
		;;
	Linux)
		tar xzf "${bin_name}.${ext}"
		;;
esac

cp "${bin_name}/${bin_name}" "${install_dir}/doit/bin/doit"
chmod u+x "${install_dir}/doit/bin/doit"

echo -e "\nInstall complete!\n"
echo -e "doit has been installed to ${install_dir}/doit/bin/doit"

