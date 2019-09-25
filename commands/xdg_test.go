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
	switch runtime.GOOS {
	case goosDarwin, goosLinux:
		expectedConfigDir := "/.config/doctl"
		actualConfigDir := findConfigDir()

		if !strings.Contains(actualConfigDir, expectedConfigDir) {
			t.Fatalf("expected %s to contain %s", actualConfigDir, expectedConfigDir)
		}
	}
}
