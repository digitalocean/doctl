package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var (
	testSpellingError = `Error: unknown command "apa" for "doctl"

Did you mean this?
	apps

Run 'doctl --help' for usage.`
)

var _ = suite("errors", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
	)
	it.Before(func() {
		expect = require.New(t)
	})
	when("the command is spelled incorrectly", func() {
		it("spell checks once", func() {
			cmd := exec.Command(builtBinaryPath, "apa")
			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(testSpellingError, strings.TrimSpace(string(output)))
		})
	})
})
