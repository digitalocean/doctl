#!/bin/bash

# regenerated generated mocks

set -euo pipefail

GO111MODULE=off go get github.com/vektra/mockery/.../

cd "do" && mockery -all -note "Generated: please do not edit by hand"
