#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/../"

< "$DIR/dockerfiles/Dockerfile.snap" docker build -t doctl-snap-base -

cat <<INSTRUCTIONS
Next, test local/doctl-snap-base

1. make _build_snap
2. test the resulting snap
  a. install resulting snap locally, e.g., sudo snap install doctl_vX.XX.XXX*.snap --dangerous
  b. wire up your snap: 
     - sudo snap connect doctl:doctl-config
     - sudo snap connect doctl:kube-config
  c. take it for a spin

Assuming it passes the test, continue!

Push a prerelease version of the image to dockerhub. To get the version number run
'make version'. Take the entire result and add 'pre', e.g., '1.31.2-automate-snap-image-b1582d49-pre'
to get your version.

1. login to dockerhub as sammytheshark (credentials in LastPass)
   docker login -u sammytheshark -p <from LastPass>
2. docker rename local/doctl-snap-base sammytheshark/doctl-snap-base:1.31.2-automate-snap-image-b1582d49-pre
3. docker push sammytheshark/doctl-snap-base:1.31.2-automate-snap-image-b1582d49-pre
4. docker rename sammytheshark/doctl-snap-base:1.31.2-automate-snap-image-b1582d49-pre sammytheshark/doctl-snap-base:latest
5. docker push sammytheshark/doctl-snap-base:latest

submit your PR and merge. But wait! You aren't done yet!

Release a new version and get a new version number.

Finally, docker rename and docker push your image from pre to the new version.
INSTRUCTIONS
