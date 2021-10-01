//go:build !windows
// +build !windows

package integration

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("auth/remove", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect     *require.Assertions
		tmpDir     string
		testConfig string
	)

	it.Before(func() {
		expect = require.New(t)

		var err error
		tmpDir, err = ioutil.TempDir("", "")
		expect.NoError(err)

		testConfig = filepath.Join(tmpDir, "test-config.yml")
		var testConfigBytes = []byte(`access-token: first-token
auth-contexts:
  second: second-token
context: default
`)

		expect.NoError(ioutil.WriteFile(testConfig, testConfigBytes, 0644))

	})

	when("a context is not provided", func() {
		it("should error", func() {

			cmd := exec.Command(builtBinaryPath,
				"auth",
				"remove",
				"--config", testConfig,
			)
			output, err := cmd.CombinedOutput()
			expect.Error(err)

			expect.Equal("Error: You must provide a context name\n", string(output[:]))
		})
	})

	when("default context is provided", func() {
		it("allows you to remove that context", func() {
			removeContext := "default"

			cmd := exec.Command(builtBinaryPath,
				"auth",
				"remove",
				"--config", testConfig,
				"--context",
				removeContext,
			)
			_, err := cmd.CombinedOutput()
			expect.NoError(err)

			fileBytes, err := ioutil.ReadFile(testConfig)
			expect.NoError(err)
			expect.NotContains(string(fileBytes), "first-token")
		})
	})

	when("a valid context is provided", func() {
		it("allows you to remove that context", func() {
			removeContext := "second"

			cmd := exec.Command(builtBinaryPath,
				"auth",
				"remove",
				"--config", testConfig,
				"--context",
				removeContext,
			)
			_, err := cmd.CombinedOutput()
			expect.NoError(err)

			fileBytes, err := ioutil.ReadFile(testConfig)
			expect.NoError(err)
			expect.NotContains(string(fileBytes), "second-token")
		})
	})
})
