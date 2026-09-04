package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const packagePath string = "github.com/digitalocean/doctl/cmd/doctl"

var (
	suite           = spec.New("doctl", spec.Report(report.Terminal{}), spec.Random(), spec.Parallel())
	builtBinaryPath string
)

const (
	confirmNonInteractiveOutput = "Warning: Requires confirmation. Use the `--force` flag to continue without confirmation.\nError: Operation aborted."
)

// waitElapsed matches the elapsed time a wait reports on each of its progress
// lines, as in "Database (some-id) is online (10s)". Go renders a duration in
// as many units as it needs, so the unit repeats: a wait that crosses a minute
// reports "(1m40s)".
var waitElapsed = regexp.MustCompile(`\((?:\d+[a-z]+)+\)$`)

// normalizeWaitElapsed replaces the elapsed time on wait progress lines with a
// fixed token. A wait that polls on a ten second interval reports either 10s
// or 11s depending on how loaded the machine is, which is not something a test
// should be asserting on.
func normalizeWaitElapsed(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = waitElapsed.ReplaceAllString(line, "(elapsed)")
	}

	return strings.Join(lines, "\n")
}

func TestRun(t *testing.T) {
	suite.Run(t)
}

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "integration-doctl")
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
		contents, err = os.ReadFile(location)
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
		err = os.WriteFile(location, contents, 0644)
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
