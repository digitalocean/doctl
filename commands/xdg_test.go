package commands

import (
	"runtime"
	"strings"
	"testing"
)

const (
	goosDarwin = "darwin"
	goosLinux  = "linux"
)

func Test_findConfigDir(t *testing.T) {
	var expectedConfigDir string
	actualConfigDir := findConfigDir()

	switch runtime.GOOS {
	case goosDarwin, goosLinux:
		expectedConfigDir = "/.config/doctl"
	}

	if !strings.Contains(actualConfigDir, expectedConfigDir) {
		t.Fatalf("expected %s to contain %s", actualConfigDir, expectedConfigDir)
	}
}
