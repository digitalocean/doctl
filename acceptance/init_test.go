package acceptance

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const packagePath string = "github.com/digitalocean/doctl"

var (
	suite           spec.Suite
	builtBinaryPath string
)

func init() {
	suite = spec.New("acceptance", spec.Report(report.Terminal{}))
	suite("account/get", testAccountGet)
	suite("account/ratelimit", testAccountRateLimit)
}

func TestAll(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "acceptance-doctl")
	if err != nil {
		t.Fatal("failed to create temp dir")
	}

	builtBinaryPath = filepath.Join(tmpDir, path.Base(packagePath))

	cmd := exec.Command("go", "build", "-o", builtBinaryPath, packagePath)
	err = cmd.Run()
	if err != nil {
		t.Fatal("failed to build doctl")
	}

	suite.Run(t)

	err = os.RemoveAll(tmpDir)
	if err != nil {
		t.Fatal("failed to cleanup the doctl acceptance artifacts")
	}
}
