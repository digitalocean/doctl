#!/usr/bin/env sh

GO111MODULE="on" go test -race -mod=vendor ./commands/... ./do/... ./pkg/... .
# disable until integration tests green on windows
#GO111MODULE="on" go test -race -mod=vendor ./integration
