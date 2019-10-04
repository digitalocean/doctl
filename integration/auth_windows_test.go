// +build windows

package integration

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestAuth(t *testing.T) {
	spec.Run(t, "auth/init", testAuthInit, spec.Report(report.Terminal{}))
}

func testAuthInit(t *testing.T, when spec.G, it spec.S) {
	it.Pend("this is not implemented on windows", func() {})
}
