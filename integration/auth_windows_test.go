//go:build windows
// +build windows

package integration

import (
	"testing"

	"github.com/sclevine/spec"
)

var _ = suite("auth/init", func(t *testing.T, when spec.G, it spec.S) {
	it.Pend("this is not implemented on windows", func() {})
})
