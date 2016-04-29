#!/bin/bash

# regenerated generated mocks

set -e

go get github.com/vektra/mockery/.../

cd do
mockery -all -note "Generated: please do not edit by hand"