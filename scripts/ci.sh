#!/bin/bash

set -e

GO15VENDOREXPERIMENT=1 go test ./cmd/... ./commands/... ./do/... ./install/... ./pkg/... ./pluginhost/... .
