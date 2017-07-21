#!/bin/bash

set -e

go test ./cmd/... ./commands/... ./do/... ./install/... ./pkg/... ./pluginhost/... .
