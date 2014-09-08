#!/bin/bash

if [[ $# -lt 1 ]]; then
	echo "Usage: release.sh <version>"
	exit 64
fi

VERSION=$1
GHRELEASE=`which github-release`

if [[ "$GHRELEASE" -eq "" ]]; then
	echo "Installing github-release..."
	go get github.com/aktau/github-release
fi

git tag $VERSION && git push --tags

$GHRELEASE release \
    --user slantview \
    --repo doctl \
    --tag $VERSION \
    --name "$VERSION" \
    --description "Release $VERSION"


 find bin -print | grep doctl |
 while read binary
 do
 	if [[ -f $binary ]]; then
	 	OS=`echo $binary | awk -F/ '{print $2}'`
	 	ARCH=`echo $binary | awk -F/ '{print $3}'`
	 	EXE=`echo $binary | awk -F/ '{print $4}'`

	 	$GHRELEASE upload \
		    --user slantview \
		    --repo doctl \
		    --tag $VERSION \
		    --name "$OS-$ARCH-$EXE" \
		    --file $binary
	fi
done