#!/usr/bin/env sh

GO111MODULE="on" go test -race -mod=vendor ./commands/... ./do/... ./pkg/... .
GO111MODULE="on" go test -race -mod=vendor ./integration
