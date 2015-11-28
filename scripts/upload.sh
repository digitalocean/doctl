#!/bin/bash

ver=$1
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUTPUT_DIR="${DIR}/../builds/${ver}"

for r in $(ls ${OUTPUT_DIR}/*.{gz,zip}); do
	name=$(basename $r)
	github-release upload \
		--user bryanl \
		--repo doit \
		--tag v${ver} \
		--name $name \
		--file $r
done

