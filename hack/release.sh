#!/bin/bash

GHRELEASE=`which github-release`

if [[ "$GHRELEASE" -eq "" ]]; then
	echo "Installing github-release..."
	go get github.com/aktau/github-release
fi

