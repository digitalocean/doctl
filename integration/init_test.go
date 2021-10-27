package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const packagePath string = "github.com/digitalocean/doctl/cmd/doctl"

var (
	suite           = spec.New("doctl", spec.Report(report.Terminal{}), spec.Random(), spec.Parallel())
	builtBinaryPath string
)

func TestRun(t *testing.T) {
	suite.Run(t)
}

func TestMain(m *testing.M) {
	tmpDir, err := ioutil.TempDir("", "integration-doctl")
	if err != nil {
		panic("failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir) // yes, this is the best effort only

	builtBinaryPath = filepath.Join(tmpDir, path.Base(packagePath))
	if runtime.GOOS == "windows" {
		builtBinaryPath += ".exe"
	}

	// tried to use -mod=vendor but it blew up
	cmd := exec.Command("go", "build", "-o", builtBinaryPath, packagePath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("failed to build doctl: %s", output))
	}

	location, err := getDefaultConfigLocation()
	if err != nil {
		panic(fmt.Sprintf("failed to get config location: %s", err))
	}

	var contents []byte
	if _, err := os.Stat(location); !os.IsNotExist(err) {
		contents, err = ioutil.ReadFile(location)
		if err != nil {
			panic("failed to copy config")
		}

		err = os.Remove(location)
		if err != nil {
			panic("failed to delete initial config")
		}
	}

	code := m.Run()

	if len(contents) != 0 {
		err = ioutil.WriteFile(location, contents, 0644)
		if err != nil {
			panic("failed to restore contents of config")
		}
	}

	os.Exit(code)
}

func getDefaultConfigLocation() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %s", err)
	}

	return filepath.Join(configDir, "doctl", "config.yaml"), nil
}
