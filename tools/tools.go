//go:build tools

// To install the following tools at the version used by this repo run:
// $ go generate -tags tools tools/tools.go

package tools

//go:generate go install go.uber.org/mock/mockgen

import (
	_ "go.uber.org/mock/mockgen"
)
