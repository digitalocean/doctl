#!/bin/bash

set -e

go test ./commands/... ./do/... ./pkg/... .
