#!/bin/sh

ver=$1
name=$2

github-release release --user bryanl --repo doit --tag v$ver --name $name
