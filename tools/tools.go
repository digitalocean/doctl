//go:build tools

// To install the following tools at the version used by this repo run:
// $ go generate -tags tools tools/tools.go

package tools

//go:generate go install github.com/golang/mock/mockgen

import (
	_ "github.com/golang/mock/mockgen"
)
